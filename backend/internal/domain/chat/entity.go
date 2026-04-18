// Package chat contains domain entities for the in-app chatbot TA (#071).
package chat

import "time"

// Session 은 한 학생의 챗봇 대화 세션.
type Session struct {
	ID            int       `json:"id"`
	UserID        int       `json:"-"`
	Title         string    `json:"title"`
	ActiveSkillID *int      `json:"active_skill_id,omitempty"`
	TokensUsed    int       `json:"tokens_used"`
	CreatedAt     time.Time `json:"created_at"`
	LastMessageAt time.Time `json:"last_message_at"`

	// Nested
	ActiveSkill *Skill     `json:"active_skill,omitempty"`
	Messages    []*Message `json:"messages,omitempty"`
}

// Role 은 챗 메시지 역할 (OpenAI chat completions 과 일치).
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall 은 assistant 메시지가 지시한 도구 호출 하나.
// OpenAI function-calling 스펙과 동일.
type ToolCall struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Args     map[string]any    `json:"args,omitempty"`      // 파싱된 인자 (tool 측에서 사용)
	RawArgs  string            `json:"raw_args,omitempty"`  // 원본 JSON 문자열
}

type Message struct {
	ID               int        `json:"id"`
	SessionID        int        `json:"-"`
	Role             Role       `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	Model            string     `json:"model,omitempty"`
	PromptTokens     int        `json:"prompt_tokens,omitempty"`
	CompletionTokens int        `json:"completion_tokens,omitempty"`
	CacheTokens      int        `json:"cache_tokens,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"` // role=tool 일 때 어떤 호출에 대한 응답인지
	CreatedAt        time.Time  `json:"created_at"`
}

// Skill 은 챗봇의 "페르소나 + 도구 + 위키 스코프" 번들.
type Skill struct {
	ID                     int       `json:"id"`
	Slug                   string    `json:"slug"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	SystemPrompt           string    `json:"system_prompt"`
	DefaultModel           string    `json:"default_model"`
	DefaultReasoningEffort string    `json:"default_reasoning_effort,omitempty"`
	ToolsAllowed           []string  `json:"tools_allowed"`
	WikiScope              []string  `json:"wiki_scope"` // glob 리스트 (예: "notion-manuals/wallet*")
	Enabled                bool      `json:"enabled"`
	AdminOnly              bool      `json:"admin_only"`
	CreatedBy              *int      `json:"created_by,omitempty"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// WikiDocMeta 는 위키 문서의 DB 저장 메타 (실제 본문은 FTS5 가상 테이블 + 파일).
type WikiDocMeta struct {
	Slug         string    `json:"slug"`
	Path         string    `json:"path"`           // git 저장소 내 상대 경로
	Title        string    `json:"title"`
	NotionPageID string    `json:"notion_page_id,omitempty"`
	SyncedAt     time.Time `json:"synced_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WikiSearchHit 는 BM25 검색 결과 1건.
type WikiSearchHit struct {
	Slug    string  `json:"slug"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// AskMode 는 학생이 선택할 수 있는 응답 속도/깊이 모드.
type AskMode string

const (
	ModeFast AskMode = "fast" // qwen-chat
	ModeDeep AskMode = "deep" // qwen-reasoning + medium effort
)

// UsageDay 는 관리자 대시보드용 일별 챗봇 비용 집계 (학교 부담).
type UsageDay struct {
	Date             time.Time `json:"date"`
	Requests         int       `json:"requests"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CacheTokens      int       `json:"cache_tokens"`
	CostKRW          int       `json:"cost_krw"`
}
