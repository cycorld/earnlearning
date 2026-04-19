package llmproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatMessage 은 OpenAI-compatible chat completion 의 단일 메시지.
// content 는 string 또는 (vision 용) []ContentBlock. 여기선 텍스트만 노출.
type ChatMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []ChatToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Name       string            `json:"name,omitempty"` // for role=tool
}

// ChatToolCall 은 assistant 응답 속 tool_calls[] 항목.
type ChatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ChatToolFunction `json:"function"`
}

type ChatToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 문자열
}

// ChatToolSpec 은 request 로 전달할 tool 정의.
type ChatToolSpec struct {
	Type     string                `json:"type"` // "function"
	Function ChatToolSpecFunction  `json:"function"`
}

type ChatToolSpecFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
}

type ChatRequest struct {
	Model            string         `json:"model"`
	Messages         []ChatMessage  `json:"messages"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	ReasoningEffort  string         `json:"reasoning_effort,omitempty"`
	Tools            []ChatToolSpec `json:"tools,omitempty"`
	ToolChoice       any            `json:"tool_choice,omitempty"` // "auto" | "none" | {"type":"function","function":{"name":"..."}}
	Stream           bool           `json:"stream,omitempty"`
	StreamOptions    *StreamOptions `json:"stream_options,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// ChatStreamEvent — SSE chunk 의 추상화. 한 번에 하나만 채워짐.
type ChatStreamEvent struct {
	TextDelta    string     // assistant content delta
	FinishReason string     // "stop" | "length" | ...
	Usage        *ChatUsage // 마지막 chunk 의 usage 정보 (nil if not yet)
	Err          error
}

type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	// OpenAI-compatible extensions
	PromptCachedTokens int `json:"prompt_cached_tokens,omitempty"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	// Qwen extension: reasoning content (thinking 토큰)
	// OpenAI 일부 구현에선 message.reasoning_content 이기도 해서 두 위치 모두 체크
}

type ChatResponse struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   ChatUsage    `json:"usage"`
}

// ChatComplete 은 /v1/chat/completions 를 non-streaming 으로 호출.
// streaming 은 ChatCompleteStream 으로 추후 추가.
func (c *Client) ChatComplete(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}
	// Tools 포함 요청은 내부적으로 더 길 수 있어 타임아웃 여유롭게
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	key := c.userKey
	if key == "" {
		key = c.adminKey
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chat http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("chat %d: %s", resp.StatusCode, string(raw))
	}
	var out ChatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode chat response: %w (body: %s)", err, truncate(raw, 300))
	}
	return &out, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

// ChatCompleteStream — /v1/chat/completions 를 stream=true 로 호출.
// 반환된 채널은 끝나면 close. 호출자는 ctx 로 취소 가능.
//
// 본 구현은 tool_calls 가 없는 "최종 응답" 전용 — caller (use case) 가 tool loop
// 마지막 turn 에서만 호출. tool_calls delta 는 무시.
func (c *Client) ChatCompleteStream(ctx context.Context, req *ChatRequest) (<-chan ChatStreamEvent, error) {
	streamReq := *req // shallow copy
	streamReq.Stream = true
	streamReq.StreamOptions = &StreamOptions{IncludeUsage: true}

	body, err := json.Marshal(&streamReq)
	if err != nil {
		return nil, fmt.Errorf("marshal chat stream request: %w", err)
	}
	// stream 은 길게 둠 — Cloudflare free 100s 제한과 별개로 origin 에선 여유롭게
	streamCtx, cancel := context.WithTimeout(ctx, 180*time.Second)

	httpReq, err := http.NewRequestWithContext(streamCtx, http.MethodPost,
		c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		cancel()
		return nil, err
	}
	key := c.userKey
	if key == "" {
		key = c.adminKey
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Content-Type", "application/json")

	// stream 전용 클라이언트: HTTP 자체 timeout 끄고 ctx 로만 제어
	streamHTTP := &http.Client{Timeout: 0}
	resp, err := streamHTTP.Do(httpReq)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("chat stream http: %w", err)
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		return nil, fmt.Errorf("chat stream %d: %s", resp.StatusCode, string(raw))
	}

	out := make(chan ChatStreamEvent, 16)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		defer cancel()
		parseSSEStream(resp.Body, out)
	}()
	return out, nil
}

// parseSSEStream — SSE chunked response 를 line 단위로 파싱해 ChatStreamEvent 생성.
// OpenAI/Qwen 공통 포맷: "data: {...}\n\n" 와 종료 "data: [DONE]\n\n".
func parseSSEStream(body io.Reader, out chan<- ChatStreamEvent) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			for {
				// SSE event 는 \n\n 로 분리. 일부 서버는 \r\n\r\n 도 사용 → 둘 다 시도
				idx := bytes.Index(buf, []byte("\n\n"))
				altIdx := bytes.Index(buf, []byte("\r\n\r\n"))
				if idx == -1 && altIdx == -1 {
					break
				}
				cut := idx
				skip := 2
				if altIdx != -1 && (idx == -1 || altIdx < idx) {
					cut = altIdx
					skip = 4
				}
				event := buf[:cut]
				buf = buf[cut+skip:]
				processSSEEvent(event, out)
			}
		}
		if err == io.EOF {
			return
		}
		if err != nil {
			out <- ChatStreamEvent{Err: fmt.Errorf("sse read: %w", err)}
			return
		}
	}
}

func processSSEEvent(event []byte, out chan<- ChatStreamEvent) {
	// event 는 여러 line 일 수 있음 (data:, event:, id: ...)
	// 우리는 data: 만 처리.
	for _, line := range bytes.Split(event, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(line[len("data:"):])
		if len(payload) == 0 {
			continue
		}
		if bytes.Equal(payload, []byte("[DONE]")) {
			return
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *ChatUsage `json:"usage"`
		}
		if err := json.Unmarshal(payload, &chunk); err != nil {
			// malformed chunk — skip silently (stream 도중 keepalive 등 가능)
			continue
		}
		if len(chunk.Choices) > 0 {
			ch := chunk.Choices[0]
			if ch.Delta.Content != "" {
				out <- ChatStreamEvent{TextDelta: ch.Delta.Content}
			}
			if ch.FinishReason != nil && *ch.FinishReason != "" {
				out <- ChatStreamEvent{FinishReason: *ch.FinishReason}
			}
		}
		if chunk.Usage != nil {
			out <- ChatStreamEvent{Usage: chunk.Usage}
		}
	}
}
