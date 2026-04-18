// Package context7 는 context7.com 의 HTTP API 를 호출하는 클라이언트.
// 공식 라이브러리 문서 검색 + 최신 문서 페이지 fetch 용도.
package context7

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	apiKey string
	http   *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

type SearchResult struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	TrustScore     float64 `json:"trustScore"`
	BenchmarkScore float64 `json:"benchmarkScore"`
	Stars          int     `json:"stars"`
	TotalSnippets  int     `json:"totalSnippets"`
	Verified       bool    `json:"verified"`
	LastUpdateDate string  `json:"lastUpdateDate"`
}

type searchResponse struct {
	Results []SearchResult `json:"results"`
}

// Search — context7 가 인덱싱한 라이브러리 검색. top N 반환.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("context7 api key not configured")
	}
	if limit <= 0 {
		limit = 5
	}
	endpoint := "https://context7.com/api/v1/search?query=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ctx7 search: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2_000_000))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ctx7 %d: %s", resp.StatusCode, string(raw))
	}
	var out searchResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("ctx7 decode: %w", err)
	}
	if len(out.Results) > limit {
		out.Results = out.Results[:limit]
	}
	return out.Results, nil
}

// Docs — 라이브러리 ID 에 대한 문서 스니펫 반환. topic 으로 주제 범위 지정.
// tokens 는 응답 최대 길이 (기본 3000).
func (c *Client) Docs(ctx context.Context, libraryID, topic string, tokens int) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("context7 api key not configured")
	}
	if libraryID == "" {
		return "", fmt.Errorf("library id required")
	}
	if tokens <= 0 || tokens > 10000 {
		tokens = 3000
	}
	// libraryID 는 /websites/foo 또는 /tanstack/query 등 슬래시 시작 형태
	if libraryID[0] == '/' {
		libraryID = libraryID[1:]
	}
	q := url.Values{}
	q.Set("tokens", fmt.Sprintf("%d", tokens))
	if topic != "" {
		q.Set("topic", topic)
	}
	endpoint := "https://context7.com/api/v1/" + libraryID + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("ctx7 docs: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2_000_000))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("ctx7 docs %d: %s", resp.StatusCode, string(raw))
	}
	return string(raw), nil
}
