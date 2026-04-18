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
	httpReq.Header.Set("Authorization", "Bearer "+c.adminKey)
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
