// Package notion — Notion API 클라이언트 + 페이지 → 마크다운 변환 (#082).
//
// 설계:
//   - 인증: NOTION_INTEGRATION_TOKEN (env). Bearer 토큰.
//   - API: notion-version: 2022-06-28
//   - FetchPageMarkdown(pageID) → Notion 페이지의 모든 block 을 재귀로 가져와
//     markdown 으로 변환. 표 / 리스트 / 코드 / 인용 / 헤딩 / 단락 처리.
//
// 트레이드오프 (#082 A1):
//   - 마크다운 표는 Notion 의 alignment / cell merge 등 일부 손실 허용
//   - 콜아웃 / 토글 / 임베드는 텍스트로만 (이모지 포함, 본문은 그대로)
//   - 이미지 / 임베드 URL 은 그대로 보존
package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	apiBase       = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
)

type Client struct {
	token string
	http  *http.Client
}

func New(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchPageMarkdown — pageID 의 모든 블록을 재귀로 가져와 markdown 문자열 반환.
// pageID 는 dashed UUID 또는 32-char hex 모두 OK.
func (c *Client) FetchPageMarkdown(ctx context.Context, pageID string) (string, error) {
	if c.token == "" {
		return "", fmt.Errorf("notion: NOTION_INTEGRATION_TOKEN not set")
	}
	id := strings.ReplaceAll(pageID, "-", "")
	if len(id) != 32 {
		return "", fmt.Errorf("notion: invalid page id %q", pageID)
	}
	blocks, err := c.fetchChildren(ctx, id)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	c.renderBlocks(ctx, blocks, &sb, 0)
	return sb.String(), nil
}

// FetchPageTitle — Notion 페이지의 제목 (`/v1/pages/{id}` 의 properties.title).
func (c *Client) FetchPageTitle(ctx context.Context, pageID string) (string, error) {
	id := strings.ReplaceAll(pageID, "-", "")
	if len(id) != 32 {
		return "", fmt.Errorf("notion: invalid page id %q", pageID)
	}
	url := apiBase + "/pages/" + id
	body, err := c.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	var page struct {
		Properties map[string]struct {
			Title []richTextItem `json:"title"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return "", fmt.Errorf("notion: decode page: %w", err)
	}
	for _, prop := range page.Properties {
		if len(prop.Title) > 0 {
			return concatRichText(prop.Title), nil
		}
	}
	return "", nil
}

// fetchChildren — `/v1/blocks/{id}/children` paginated.
func (c *Client) fetchChildren(ctx context.Context, blockID string) ([]block, error) {
	var all []block
	cursor := ""
	for {
		url := apiBase + "/blocks/" + blockID + "/children?page_size=100"
		if cursor != "" {
			url += "&start_cursor=" + cursor
		}
		body, err := c.do(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		var resp blockListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("notion: decode blocks: %w", err)
		}
		all = append(all, resp.Results...)
		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}
	return all, nil
}

// do — 공통 HTTP 호출. body 는 nil 허용. 응답 raw bytes 반환.
func (c *Client) do(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", notionVersion)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notion http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("notion %s %s: %d %s", method, url, resp.StatusCode, truncate(raw, 200))
	}
	return raw, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

// ============================================================================
// Block / rich text 모델 — Notion API 응답 부분 매핑
// ============================================================================

type blockListResponse struct {
	Results    []block `json:"results"`
	HasMore    bool    `json:"has_more"`
	NextCursor string  `json:"next_cursor"`
}

type block struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	HasChildren    bool            `json:"has_children"`
	Paragraph      *richTextBlock  `json:"paragraph,omitempty"`
	Heading1       *richTextBlock  `json:"heading_1,omitempty"`
	Heading2       *richTextBlock  `json:"heading_2,omitempty"`
	Heading3       *richTextBlock  `json:"heading_3,omitempty"`
	BulletedList   *richTextBlock  `json:"bulleted_list_item,omitempty"`
	NumberedList   *richTextBlock  `json:"numbered_list_item,omitempty"`
	ToDo           *toDoBlock      `json:"to_do,omitempty"`
	Toggle         *richTextBlock  `json:"toggle,omitempty"`
	Quote          *richTextBlock  `json:"quote,omitempty"`
	Callout        *calloutBlock   `json:"callout,omitempty"`
	Code           *codeBlock      `json:"code,omitempty"`
	Image          *imageBlock     `json:"image,omitempty"`
	Bookmark       *bookmarkBlock  `json:"bookmark,omitempty"`
	Table          *tableMeta      `json:"table,omitempty"`
	TableRow       *tableRowBlock  `json:"table_row,omitempty"`
	Divider        *struct{}       `json:"divider,omitempty"`
	ChildPage      *childPageBlock `json:"child_page,omitempty"`
}

type richTextBlock struct {
	RichText []richTextItem `json:"rich_text"`
}

type toDoBlock struct {
	RichText []richTextItem `json:"rich_text"`
	Checked  bool           `json:"checked"`
}

type calloutBlock struct {
	RichText []richTextItem `json:"rich_text"`
	Icon     *struct {
		Emoji string `json:"emoji"`
	} `json:"icon"`
}

type codeBlock struct {
	RichText []richTextItem `json:"rich_text"`
	Language string         `json:"language"`
}

type imageBlock struct {
	Type     string `json:"type"`
	External *struct {
		URL string `json:"url"`
	} `json:"external"`
	File *struct {
		URL string `json:"url"`
	} `json:"file"`
	Caption []richTextItem `json:"caption"`
}

type bookmarkBlock struct {
	URL     string         `json:"url"`
	Caption []richTextItem `json:"caption"`
}

type tableMeta struct {
	TableWidth      int  `json:"table_width"`
	HasColumnHeader bool `json:"has_column_header"`
}

type tableRowBlock struct {
	Cells [][]richTextItem `json:"cells"`
}

type childPageBlock struct {
	Title string `json:"title"`
}

type richTextItem struct {
	Type string `json:"type"`
	Text *struct {
		Content string `json:"content"`
		Link    *struct {
			URL string `json:"url"`
		} `json:"link"`
	} `json:"text"`
	PlainText   string `json:"plain_text"`
	Annotations struct {
		Bold          bool   `json:"bold"`
		Italic        bool   `json:"italic"`
		Strikethrough bool   `json:"strikethrough"`
		Underline     bool   `json:"underline"`
		Code          bool   `json:"code"`
		Color         string `json:"color"`
	} `json:"annotations"`
	Href string `json:"href"`
}

// concatRichText — 단순 텍스트 (annotation 무시)
func concatRichText(items []richTextItem) string {
	var sb strings.Builder
	for _, it := range items {
		sb.WriteString(it.PlainText)
	}
	return sb.String()
}

// renderRichText — annotation (bold/italic/code) + link 보존한 markdown 문자열
func renderRichText(items []richTextItem) string {
	var sb strings.Builder
	for _, it := range items {
		text := it.PlainText
		if text == "" && it.Text != nil {
			text = it.Text.Content
		}
		// markdown 특수문자 escape 는 안 함 — 가독성 우선, 깨지면 admin 이 수정
		if it.Annotations.Code {
			text = "`" + text + "`"
		} else {
			if it.Annotations.Bold {
				text = "**" + text + "**"
			}
			if it.Annotations.Italic {
				text = "*" + text + "*"
			}
			if it.Annotations.Strikethrough {
				text = "~~" + text + "~~"
			}
		}
		// link
		href := it.Href
		if href == "" && it.Text != nil && it.Text.Link != nil {
			href = it.Text.Link.URL
		}
		if href != "" {
			text = "[" + text + "](" + href + ")"
		}
		sb.WriteString(text)
	}
	return sb.String()
}

// renderBlocks — block 슬라이스를 markdown 으로 변환. indent 는 list nesting 용 (스페이스 N개).
// has_children=true 인 블록은 fetchChildren 로 한 번 더 들어감.
func (c *Client) renderBlocks(ctx context.Context, blocks []block, sb *strings.Builder, indent int) {
	pad := strings.Repeat("  ", indent)
	tableBuf := []block{} // table_row 누적용

	flushTable := func(meta *tableMeta) {
		if len(tableBuf) == 0 {
			return
		}
		c.renderTable(meta, tableBuf, sb)
		tableBuf = nil
	}

	for i, b := range blocks {
		// table 끝났으면 flush
		if b.Type != "table_row" && len(tableBuf) > 0 {
			// previous block 이 table 이었어야 함
			flushTable(prevTableMeta(blocks, i))
		}

		switch b.Type {
		case "paragraph":
			if b.Paragraph != nil {
				sb.WriteString(pad)
				sb.WriteString(renderRichText(b.Paragraph.RichText))
				sb.WriteString("\n\n")
			}
		case "heading_1":
			if b.Heading1 != nil {
				sb.WriteString("# " + renderRichText(b.Heading1.RichText) + "\n\n")
			}
		case "heading_2":
			if b.Heading2 != nil {
				sb.WriteString("## " + renderRichText(b.Heading2.RichText) + "\n\n")
			}
		case "heading_3":
			if b.Heading3 != nil {
				sb.WriteString("### " + renderRichText(b.Heading3.RichText) + "\n\n")
			}
		case "bulleted_list_item":
			if b.BulletedList != nil {
				sb.WriteString(pad + "- " + renderRichText(b.BulletedList.RichText) + "\n")
				if b.HasChildren {
					children, _ := c.fetchChildren(ctx, strings.ReplaceAll(b.ID, "-", ""))
					c.renderBlocks(ctx, children, sb, indent+1)
				}
			}
		case "numbered_list_item":
			if b.NumberedList != nil {
				sb.WriteString(pad + "1. " + renderRichText(b.NumberedList.RichText) + "\n")
				if b.HasChildren {
					children, _ := c.fetchChildren(ctx, strings.ReplaceAll(b.ID, "-", ""))
					c.renderBlocks(ctx, children, sb, indent+1)
				}
			}
		case "to_do":
			if b.ToDo != nil {
				mark := "[ ]"
				if b.ToDo.Checked {
					mark = "[x]"
				}
				sb.WriteString(pad + "- " + mark + " " + renderRichText(b.ToDo.RichText) + "\n")
			}
		case "toggle":
			if b.Toggle != nil {
				sb.WriteString(pad + "- " + renderRichText(b.Toggle.RichText) + "\n")
				if b.HasChildren {
					children, _ := c.fetchChildren(ctx, strings.ReplaceAll(b.ID, "-", ""))
					c.renderBlocks(ctx, children, sb, indent+1)
				}
			}
		case "quote":
			if b.Quote != nil {
				lines := strings.Split(renderRichText(b.Quote.RichText), "\n")
				for _, line := range lines {
					sb.WriteString("> " + line + "\n")
				}
				sb.WriteString("\n")
			}
		case "callout":
			if b.Callout != nil {
				icon := ""
				if b.Callout.Icon != nil && b.Callout.Icon.Emoji != "" {
					icon = b.Callout.Icon.Emoji + " "
				}
				sb.WriteString("> " + icon + renderRichText(b.Callout.RichText) + "\n\n")
			}
		case "code":
			if b.Code != nil {
				lang := b.Code.Language
				if lang == "plain text" || lang == "" {
					lang = ""
				}
				sb.WriteString("```" + lang + "\n")
				sb.WriteString(concatRichText(b.Code.RichText))
				sb.WriteString("\n```\n\n")
			}
		case "image":
			if b.Image != nil {
				url := ""
				if b.Image.External != nil {
					url = b.Image.External.URL
				} else if b.Image.File != nil {
					url = b.Image.File.URL
				}
				caption := concatRichText(b.Image.Caption)
				sb.WriteString("![" + caption + "](" + url + ")\n\n")
			}
		case "bookmark":
			if b.Bookmark != nil {
				caption := concatRichText(b.Bookmark.Caption)
				if caption == "" {
					caption = b.Bookmark.URL
				}
				sb.WriteString("[" + caption + "](" + b.Bookmark.URL + ")\n\n")
			}
		case "divider":
			sb.WriteString("---\n\n")
		case "child_page":
			if b.ChildPage != nil {
				sb.WriteString("📄 _하위 페이지: " + b.ChildPage.Title + "_\n\n")
			}
		case "table":
			// 다음 children 으로 table_row 들이 들어옴
			if b.HasChildren {
				children, _ := c.fetchChildren(ctx, strings.ReplaceAll(b.ID, "-", ""))
				c.renderTable(b.Table, children, sb)
			}
		case "table_row":
			tableBuf = append(tableBuf, b)
		default:
			// 알 수 없는 블록 타입은 skip
		}
	}
	if len(tableBuf) > 0 {
		flushTable(nil)
	}
}

func prevTableMeta(blocks []block, currentIdx int) *tableMeta {
	for i := currentIdx - 1; i >= 0; i-- {
		if blocks[i].Type == "table" {
			return blocks[i].Table
		}
	}
	return nil
}

// renderTable — table_row 슬라이스 + meta 를 markdown 표로 변환.
// has_column_header=true 면 첫 행을 헤더로.
func (c *Client) renderTable(meta *tableMeta, rows []block, sb *strings.Builder) {
	if len(rows) == 0 {
		return
	}
	hasHeader := meta != nil && meta.HasColumnHeader
	// 모든 셀의 markdown 변환
	cellsRows := make([][]string, 0, len(rows))
	maxCols := 0
	for _, r := range rows {
		if r.TableRow == nil {
			continue
		}
		row := make([]string, 0, len(r.TableRow.Cells))
		for _, cell := range r.TableRow.Cells {
			text := renderRichText(cell)
			text = strings.ReplaceAll(text, "\n", " ")
			text = strings.ReplaceAll(text, "|", "\\|")
			row = append(row, text)
		}
		if len(row) > maxCols {
			maxCols = len(row)
		}
		cellsRows = append(cellsRows, row)
	}
	if maxCols == 0 {
		return
	}
	for i, row := range cellsRows {
		// pad
		for len(row) < maxCols {
			row = append(row, "")
		}
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
		if i == 0 {
			// 헤더 / 본문 구분선
			sep := make([]string, maxCols)
			for j := range sep {
				sep[j] = "---"
			}
			if hasHeader {
				sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
			} else {
				// 헤더 없으면 첫 행 위에 빈 헤더 + 구분선 추가하는 방식 대신 그냥 구분선 출력
				// (markdown 표는 header 가 필수라 일단 첫 행을 header 로 취급)
				sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")
			}
		}
	}
	sb.WriteString("\n")
}

// _ unused import guard — bytes 는 향후 PUT/POST body 위해 보존
var _ = bytes.NewReader
