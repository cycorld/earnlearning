package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/chat"
	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/grant"
	"github.com/earnlearning/backend/internal/domain/llm"
	userdom "github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

// ChatToolCtx 는 툴 실행 시 필요한 사용자/권한 컨텍스트.
type ChatToolCtx struct {
	UserID    int
	IsAdmin   bool
	SessionID int
}

// ChatTool 은 챗봇이 호출할 수 있는 함수 하나.
type ChatTool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema (OpenAI tool format)
	AdminOnly   bool
	// Run 은 argsJSON (문자열) 을 받아 결과를 문자열(JSON or 평문) 로 반환.
	Run func(ctx context.Context, tctx ChatToolCtx, argsJSON string) (string, error)
}

// ChatToolRegistry 는 이름 → ChatTool 매핑.
type ChatToolRegistry struct {
	tools map[string]*ChatTool
}

func NewChatToolRegistry() *ChatToolRegistry {
	return &ChatToolRegistry{tools: map[string]*ChatTool{}}
}

func (r *ChatToolRegistry) Register(t *ChatTool) { r.tools[t.Name] = t }

func (r *ChatToolRegistry) Get(name string) (*ChatTool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Filter 는 허용 목록과 관리자 권한에 맞는 도구만 반환.
func (r *ChatToolRegistry) Filter(allowed []string, isAdmin bool) []*ChatTool {
	out := make([]*ChatTool, 0, len(allowed))
	allowSet := map[string]bool{}
	for _, a := range allowed {
		allowSet[a] = true
	}
	for name, t := range r.tools {
		if !allowSet[name] {
			continue
		}
		if t.AdminOnly && !isAdmin {
			continue
		}
		out = append(out, t)
	}
	return out
}

// ============================================================================
// Built-in tools
// ============================================================================

func BuildChatTools(
	walletRepo wallet.Repository,
	userRepo userdom.Repository,
	companyRepo company.CompanyRepository,
	grantRepo grant.Repository,
	llmRepo llm.Repository,
	wikiRepo chat.WikiRepository,
	skillRepo chat.SkillRepository,
) *ChatToolRegistry {
	_ = companyRepo
	_ = grantRepo
	r := NewChatToolRegistry()

	r.Register(&ChatTool{
		Name:        "search_wiki",
		Description: "내부 언러닝 가이드/LLM 위키에서 키워드로 문서를 검색합니다. 모르는 용어, LMS 기능 사용법, 수업 정책 관련 질문에 먼저 호출하세요.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "검색어 (한국어 가능)"},
				"limit": map[string]any{"type": "integer", "description": "최대 결과 수 (기본 5)", "default": 5},
			},
			"required": []string{"query"},
		},
		Run: func(ctx context.Context, tctx ChatToolCtx, argsJSON string) (string, error) {
			var args struct {
				Query string `json:"query"`
				Limit int    `json:"limit"`
			}
			_ = json.Unmarshal([]byte(argsJSON), &args)
			if args.Limit <= 0 {
				args.Limit = 5
			}
			hits, err := wikiRepo.Search(args.Query, nil, args.Limit)
			if err != nil {
				return "", err
			}
			if len(hits) == 0 {
				return "결과 없음. 다른 검색어를 시도해보세요.", nil
			}
			var sb strings.Builder
			for _, h := range hits {
				sb.WriteString("## " + h.Title + " (" + h.Slug + ")\n")
				sb.WriteString(h.Snippet + "\n\n")
			}
			return sb.String(), nil
		},
	})

	r.Register(&ChatTool{
		Name:        "get_my_wallet_balance",
		Description: "학생 본인의 지갑 잔액을 조회합니다.",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Run: func(ctx context.Context, tctx ChatToolCtx, _ string) (string, error) {
			w, err := walletRepo.FindByUserID(tctx.UserID)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"balance_krw": %d}`, w.Balance), nil
		},
	})

	r.Register(&ChatTool{
		Name:        "get_my_recent_transactions",
		Description: "학생 본인의 최근 거래내역을 조회합니다. 최근 N건.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{"type": "integer", "description": "가져올 거래 수 (기본 10)", "default": 10},
			},
		},
		Run: func(ctx context.Context, tctx ChatToolCtx, argsJSON string) (string, error) {
			var args struct {
				Limit int `json:"limit"`
			}
			_ = json.Unmarshal([]byte(argsJSON), &args)
			if args.Limit <= 0 || args.Limit > 50 {
				args.Limit = 10
			}
			w, err := walletRepo.FindByUserID(tctx.UserID)
			if err != nil {
				return "", err
			}
			txs, _, err := walletRepo.GetTransactions(w.ID, wallet.TransactionFilter{}, 1, args.Limit)
			if err != nil {
				return "", err
			}
			out := make([]map[string]any, 0, len(txs))
			for _, t := range txs {
				out = append(out, map[string]any{
					"created_at":     t.CreatedAt.Format(time.RFC3339),
					"amount":         t.Amount,
					"balance_after":  t.BalanceAfter,
					"type":           string(t.TxType),
					"description":    t.Description,
					"reference_type": t.ReferenceType,
				})
			}
			b, _ := json.Marshal(out)
			return string(b), nil
		},
	})

	r.Register(&ChatTool{
		Name:        "get_my_companies",
		Description: "학생이 설립하거나 지분을 가진 회사 목록을 조회합니다.",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Run: func(ctx context.Context, tctx ChatToolCtx, _ string) (string, error) {
			// 간단 구현: 사용자 활동 통해 회사 ID 를 얻기보다, company 도메인의
			// MyCompanies 엔드포인트 로직과 유사하게. 여기선 UserActivity 사용.
			activity, err := userRepo.GetUserActivity(tctx.UserID)
			if err != nil {
				return "", err
			}
			if activity == nil {
				return "[]", nil
			}
			b, _ := json.Marshal(activity.FreelanceJobs) // 일단 프리랜스/그랜트 같이 보여주는 편의
			return string(b), nil
		},
	})

	r.Register(&ChatTool{
		Name:        "get_my_grant_applications",
		Description: "학생이 지원한 정부과제(Grant) 목록과 상태를 조회합니다.",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Run: func(ctx context.Context, tctx ChatToolCtx, _ string) (string, error) {
			activity, err := userRepo.GetUserActivity(tctx.UserID)
			if err != nil {
				return "", err
			}
			if activity == nil || len(activity.GrantApps) == 0 {
				return "지원한 과제가 없습니다.", nil
			}
			b, _ := json.Marshal(activity.GrantApps)
			return string(b), nil
		},
	})

	r.Register(&ChatTool{
		Name:        "get_my_llm_usage_summary",
		Description: "학생 본인의 LLM API 누적 비용과 최근 일자별 사용량을 조회합니다.",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Run: func(ctx context.Context, tctx ChatToolCtx, _ string) (string, error) {
			totalCost, totalDebt, err := llmRepo.SumUsageAllTime(tctx.UserID)
			if err != nil {
				return "", err
			}
			since := time.Now().In(llm.KST).AddDate(0, 0, -7)
			weekCost, _, err := llmRepo.SumUsageSince(tctx.UserID, since)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"cumulative_cost_krw": %d, "cumulative_debt_krw": %d, "last_7d_cost_krw": %d}`,
				totalCost, totalDebt, weekCost), nil
		},
	})

	// 관리자 전용 툴: 스킬 편집 draft 저장
	r.Register(&ChatTool{
		Name:        "save_skill_draft",
		Description: "관리자 전용. 새로운 챗봇 스킬 초안을 저장합니다. 저장하기 전 사용자에게 한 번 더 확인하세요.",
		AdminOnly:   true,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"slug":                     map[string]any{"type": "string", "description": "고유 영문 슬러그 (예: 'homework_helper')"},
				"name":                     map[string]any{"type": "string", "description": "화면에 보일 이름 (한글)"},
				"description":              map[string]any{"type": "string", "description": "이 스킬이 무엇을 돕는지 1~2문장"},
				"system_prompt":            map[string]any{"type": "string", "description": "LLM 에게 줄 시스템 프롬프트 전체"},
				"default_model":            map[string]any{"type": "string", "enum": []string{"qwen", "qwen-chat", "qwen-reasoning"}},
				"default_reasoning_effort": map[string]any{"type": "string", "enum": []string{"", "low", "medium", "high"}},
				"tools_allowed":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"wiki_scope":               map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "glob 리스트 (예: 'notion-manuals/wallet*'). 비우면 전체."},
				"admin_only":               map[string]any{"type": "boolean", "default": false},
			},
			"required": []string{"slug", "name", "system_prompt"},
		},
		Run: func(ctx context.Context, tctx ChatToolCtx, argsJSON string) (string, error) {
			if !tctx.IsAdmin {
				return "", fmt.Errorf("admin only")
			}
			var args struct {
				Slug                   string   `json:"slug"`
				Name                   string   `json:"name"`
				Description            string   `json:"description"`
				SystemPrompt           string   `json:"system_prompt"`
				DefaultModel           string   `json:"default_model"`
				DefaultReasoningEffort string   `json:"default_reasoning_effort"`
				ToolsAllowed           []string `json:"tools_allowed"`
				WikiScope              []string `json:"wiki_scope"`
				AdminOnly              bool     `json:"admin_only"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("parse args: %w", err)
			}
			if args.Slug == "" || args.Name == "" || args.SystemPrompt == "" {
				return "", fmt.Errorf("slug/name/system_prompt required")
			}
			if args.DefaultModel == "" {
				args.DefaultModel = "qwen-chat"
			}
			sk := &chat.Skill{
				Slug:                   args.Slug,
				Name:                   args.Name,
				Description:            args.Description,
				SystemPrompt:           args.SystemPrompt,
				DefaultModel:           args.DefaultModel,
				DefaultReasoningEffort: args.DefaultReasoningEffort,
				ToolsAllowed:           args.ToolsAllowed,
				WikiScope:              args.WikiScope,
				Enabled:                true, // draft 저장 후 관리자 UI 에서 disable 가능
				AdminOnly:              args.AdminOnly,
				CreatedBy:              &tctx.UserID,
			}
			id, err := skillRepo.Upsert(sk)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"skill_id": %d, "slug": %q, "status": "saved"}`, id, args.Slug), nil
		},
	})

	return r
}

// ToToolSpecs 은 도구 목록을 OpenAI tool format 으로 변환.
func ToToolSpecs(tools []*ChatTool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}
	return out
}
