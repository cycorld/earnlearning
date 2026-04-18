package application

// LLM 호출 관련 타입을 application 레이어에 정의해서 llmproxy 와의 import cycle 을
// 피한다. llmproxy 는 이 타입을 구현하는 어댑터를 제공한다.

type LLMChatMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content,omitempty"`
	ToolCalls  []LLMChatToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	Name       string             `json:"name,omitempty"`
}

type LLMChatToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function LLMChatToolFunc    `json:"function"`
}

type LLMChatToolFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type LLMChatToolSpec struct {
	Type     string               `json:"type"`
	Function LLMChatToolSpecFunc  `json:"function"`
}

type LLMChatToolSpecFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type LLMChatRequest struct {
	Model           string
	Messages        []LLMChatMessage
	MaxTokens       int
	ReasoningEffort string
	Tools           []LLMChatToolSpec
	ToolChoice      any
}

type LLMChatUsage struct {
	PromptTokens       int
	CompletionTokens   int
	PromptCachedTokens int
}

type LLMChatChoice struct {
	Message      LLMChatMessage
	FinishReason string
}

type LLMChatResponse struct {
	Model   string
	Choices []LLMChatChoice
	Usage   LLMChatUsage
}
