// Package websearch 는 챗봇이 사용하는 외부 웹 검색 / 문서 fetch 유틸.
//
// 설계:
//   - `Search(query)` — DuckDuckGo HTML 검색 파싱. API key 불필요.
//   - `Fetch(url)` — 임의 URL GET → HTML 태그 제거 후 plain text 반환.
//
// DuckDuckGo HTML 파싱은 공식 API 가 아니므로 사이트 구조 변경에 취약.
// 프로덕션급 신뢰도가 필요하면 Brave Search API 로 교체 예정.
package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type Client struct {
	http      *http.Client
	userAgent string
}

func New() *Client {
	return &Client{
		http:      &http.Client{Timeout: 10 * time.Second},
		userAgent: "EarnLearning-Chatbot/1.0 (+https://earnlearning.com)",
	}
}

// Search — DuckDuckGo HTML 결과 파싱 (top N).
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty query")
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}
	endpoint := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ddg http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1_000_000))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ddg %d", resp.StatusCode)
	}
	return parseDuckDuckGoResults(string(raw), limit), nil
}

// Fetch — 임의 URL 을 가져와 HTML 태그 제거 후 일부 plain text 반환.
// maxChars <= 0 이면 기본 6000.
func (c *Client) Fetch(ctx context.Context, target string, maxChars int) (string, error) {
	if target == "" {
		return "", fmt.Errorf("empty url")
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", fmt.Errorf("bad url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("only http/https allowed")
	}
	if maxChars <= 0 || maxChars > 20000 {
		maxChars = 6000
	}

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,text/plain,application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1_500_000))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("fetch %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	text := string(raw)
	if strings.Contains(ct, "html") {
		text = stripHTML(text)
	}
	text = collapseWhitespace(text)
	if len([]rune(text)) > maxChars {
		runes := []rune(text)
		text = string(runes[:maxChars]) + "...[truncated]"
	}
	return text, nil
}

// ============================================================================
// HTML parsing helpers
// ============================================================================

var (
	reStyleBlock  = regexp.MustCompile(`(?is)<(style|script)[^>]*>.*?</(?:style|script)>`)
	reTag         = regexp.MustCompile(`<[^>]+>`)
	reWhitespace  = regexp.MustCompile(`[\t\r]+|  +`)
	reBlankLines  = regexp.MustCompile(`\n{3,}`)
	reDDGLinkTag  = regexp.MustCompile(`(?is)<a[^>]+class="result__a"[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	reDDGSnippet  = regexp.MustCompile(`(?is)<a[^>]+class="result__snippet"[^>]*>(.*?)</a>`)
)

func stripHTML(s string) string {
	s = reStyleBlock.ReplaceAllString(s, "")
	s = reTag.ReplaceAllString(s, " ")
	s = replaceHTMLEntities(s)
	return s
}

func collapseWhitespace(s string) string {
	s = reWhitespace.ReplaceAllString(s, " ")
	s = reBlankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func replaceHTMLEntities(s string) string {
	repl := map[string]string{
		"&nbsp;": " ",
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&#39;":  "'",
		"&#x2F;": "/",
	}
	for k, v := range repl {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

// parseDuckDuckGoResults — DDG HTML 페이지에서 top N 결과 파싱.
func parseDuckDuckGoResults(html string, limit int) []SearchResult {
	links := reDDGLinkTag.FindAllStringSubmatch(html, -1)
	snippets := reDDGSnippet.FindAllStringSubmatch(html, -1)

	out := make([]SearchResult, 0, limit)
	for i, m := range links {
		if i >= limit {
			break
		}
		href := m[1]
		title := collapseWhitespace(stripHTML(m[2]))
		snippet := ""
		if i < len(snippets) {
			snippet = collapseWhitespace(stripHTML(snippets[i][1]))
		}
		// DDG 가 결과 URL 을 `/l/?kh=-1&uddg=<encoded>` 로 래핑함 — 원 URL 추출
		if strings.HasPrefix(href, "//duckduckgo.com/l/") || strings.HasPrefix(href, "/l/") {
			parsed, _ := url.Parse("https:" + strings.TrimPrefix(href, "//"))
			if parsed == nil {
				parsed, _ = url.Parse("https://duckduckgo.com" + href)
			}
			if parsed != nil {
				if u := parsed.Query().Get("uddg"); u != "" {
					href = u
				}
			}
		}
		if title == "" || href == "" {
			continue
		}
		out = append(out, SearchResult{Title: title, URL: href, Snippet: snippet})
	}
	return out
}
