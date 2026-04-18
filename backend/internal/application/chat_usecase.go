package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/chat"
	"github.com/earnlearning/backend/internal/domain/llm"
)

// ChatLLMClient 은 LLM proxy 호출에 필요한 최소 인터페이스 (test 에서 fake 주입).
// 구체 구현은 infrastructure/llmproxy 의 ChatAdapter.
type ChatLLMClient interface {
	ChatComplete(ctx context.Context, req *LLMChatRequest) (*LLMChatResponse, error)
}

// ChatUseCase — 세션 관리 + 메시지 전송 + 도구 실행 루프.
type ChatUseCase struct {
	sessionRepo chat.SessionRepository
	messageRepo chat.MessageRepository
	skillRepo   chat.SkillRepository
	wikiRepo    chat.WikiRepository
	usageRepo   chat.UsageRepository
	tools       *ChatToolRegistry
	llm         ChatLLMClient
	loader      WikiLoader // optional; used for manual reindex
	maxToolHops int
}

// WikiLoader 는 관리자 재인덱스 요청 시 호출하는 선택적 의존성.
type WikiLoader interface {
	Sync() (int, error)
}

func NewChatUseCase(
	sessionRepo chat.SessionRepository,
	messageRepo chat.MessageRepository,
	skillRepo chat.SkillRepository,
	wikiRepo chat.WikiRepository,
	usageRepo chat.UsageRepository,
	tools *ChatToolRegistry,
	llmClient ChatLLMClient,
	loader WikiLoader,
) *ChatUseCase {
	return &ChatUseCase{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		skillRepo:   skillRepo,
		wikiRepo:    wikiRepo,
		usageRepo:   usageRepo,
		tools:       tools,
		llm:         llmClient,
		loader:      loader,
		maxToolHops: 4,
	}
}

// ============================================================================
// Session management
// ============================================================================

func (uc *ChatUseCase) CreateSession(userID int, skillSlug string) (*chat.Session, error) {
	s := &chat.Session{
		UserID:        userID,
		Title:         "새 대화",
		LastMessageAt: time.Now(),
	}
	if skillSlug != "" {
		sk, err := uc.skillRepo.FindBySlug(skillSlug)
		if err != nil {
			return nil, err
		}
		s.ActiveSkillID = &sk.ID
		s.ActiveSkill = sk
	}
	if _, err := uc.sessionRepo.Create(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (uc *ChatUseCase) GetSession(userID, sessionID int) (*chat.Session, error) {
	s, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}
	if s.UserID != userID {
		return nil, chat.ErrForbidden
	}
	// attach active skill
	if s.ActiveSkillID != nil {
		sk, err := uc.skillRepo.FindByID(*s.ActiveSkillID)
		if err == nil {
			s.ActiveSkill = sk
		}
	}
	msgs, err := uc.messageRepo.ListBySession(sessionID, 200)
	if err != nil {
		return nil, err
	}
	s.Messages = msgs
	return s, nil
}

func (uc *ChatUseCase) ListSessions(userID, page int) ([]*chat.Session, int, error) {
	return uc.sessionRepo.ListByUser(userID, page, 20)
}

// AdminListAllSessions — 관리자 전용. userID 필터 옵션 (0 이면 전체).
func (uc *ChatUseCase) AdminListAllSessions(userID, page int) ([]*chat.Session, int, error) {
	if userID > 0 {
		return uc.sessionRepo.ListByUser(userID, page, 50)
	}
	return uc.sessionRepo.ListAll(page, 50)
}

// AdminGetSession — 관리자 전용. 다른 유저의 세션도 열람 가능.
func (uc *ChatUseCase) AdminGetSession(sessionID int) (*chat.Session, error) {
	s, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, err
	}
	if s.ActiveSkillID != nil {
		sk, err := uc.skillRepo.FindByID(*s.ActiveSkillID)
		if err == nil {
			s.ActiveSkill = sk
		}
	}
	msgs, err := uc.messageRepo.ListBySession(sessionID, 500)
	if err != nil {
		return nil, err
	}
	s.Messages = msgs
	return s, nil
}

func (uc *ChatUseCase) DeleteSession(userID, sessionID int) error {
	s, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return err
	}
	if s.UserID != userID {
		return chat.ErrForbidden
	}
	return uc.sessionRepo.Delete(sessionID)
}

// ============================================================================
// Skills
// ============================================================================

func (uc *ChatUseCase) ListSkills(isAdmin bool) ([]*chat.Skill, error) {
	return uc.skillRepo.List(isAdmin, isAdmin) // admin 이면 disabled + admin_only 포함
}

func (uc *ChatUseCase) AdminCreateSkill(actorID int, s *chat.Skill) (int, error) {
	if s.Slug == "" || s.Name == "" || s.SystemPrompt == "" {
		return 0, chat.ErrInvalidSlug
	}
	s.CreatedBy = &actorID
	if s.DefaultModel == "" {
		s.DefaultModel = "qwen-chat"
	}
	return uc.skillRepo.Upsert(s)
}

func (uc *ChatUseCase) AdminUpdateSkill(s *chat.Skill) error { return uc.skillRepo.Update(s) }
func (uc *ChatUseCase) AdminDeleteSkill(id int) error        { return uc.skillRepo.Delete(id) }

// ============================================================================
// Wiki
// ============================================================================

func (uc *ChatUseCase) AdminReindexWiki() (int, error) {
	if uc.loader == nil {
		return 0, fmt.Errorf("wiki loader not configured")
	}
	return uc.loader.Sync()
}

func (uc *ChatUseCase) ListWikiDocs() ([]*chat.WikiDocMeta, error) {
	return uc.wikiRepo.ListMeta()
}

// ============================================================================
// Ask flow (the main entry)
// ============================================================================

type AskInput struct {
	SessionID int
	UserID    int
	IsAdmin   bool
	Message   string
	Mode      chat.AskMode // "fast" | "deep", 빈 값이면 skill default
	SkillSlug string       // 선택적으로 이 세션의 스킬 override
}

type AskOutput struct {
	Message  *chat.Message  // final assistant message 저장본
	ToolLogs []chat.Message // 실행된 툴 결과들 (UI 표시용 부가 정보)
}

// Ask 는 한 사용자 질문에 대해 LLM 을 호출하고, 필요시 도구를 실행하고,
// 최종 어시스턴트 응답을 DB 에 저장하여 반환.
//
// 단순화: non-streaming. 도구 호출 루프는 최대 maxToolHops 번까지.
func (uc *ChatUseCase) Ask(ctx context.Context, in AskInput) (*AskOutput, error) {
	if in.UserID <= 0 || in.SessionID <= 0 || strings.TrimSpace(in.Message) == "" {
		return nil, fmt.Errorf("invalid ask input")
	}
	sess, err := uc.sessionRepo.FindByID(in.SessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != in.UserID {
		return nil, chat.ErrForbidden
	}

	// 스킬 결정: slug override > session.active > 기본 general_ta
	skill, err := uc.resolveSkill(sess, in.SkillSlug, in.IsAdmin)
	if err != nil {
		return nil, err
	}

	// 사용자 입력 메시지 저장
	userMsg := &chat.Message{
		SessionID: sess.ID,
		Role:      chat.RoleUser,
		Content:   in.Message,
		CreatedAt: time.Now(),
	}
	if _, err := uc.messageRepo.Create(userMsg); err != nil {
		return nil, err
	}

	// 모델 선택
	model, effort := uc.resolveModelAndEffort(skill, in.Mode, in.IsAdmin)

	// OpenAI-format 메시지 배열 조립 — system + 과거 히스토리 + 이번 user
	history, err := uc.messageRepo.ListBySession(sess.ID, 50)
	if err != nil {
		return nil, err
	}
	messages := buildChatMessages(skill, history)

	// 도구 조립
	allowedTools := uc.tools.Filter(skill.ToolsAllowed, in.IsAdmin)
	toolSpecs := buildToolSpecs(allowedTools)

	toolCtx := ChatToolCtx{UserID: in.UserID, IsAdmin: in.IsAdmin, SessionID: sess.ID}

	// 도구 호출 루프
	var finalAssistant *chat.Message
	toolLogs := []chat.Message{}
	totalPrompt, totalCompletion, totalCache := 0, 0, 0

	for hop := 0; hop < uc.maxToolHops; hop++ {
		req := &LLMChatRequest{
			Model:           model,
			Messages:        messages,
			MaxTokens:       pickMaxTokens(model, effort),
			ReasoningEffort: effort,
		}
		if len(toolSpecs) > 0 {
			req.Tools = toolSpecs
			req.ToolChoice = "auto"
		}
		resp, err := uc.llm.ChatComplete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("llm call: %w", err)
		}
		totalPrompt += resp.Usage.PromptTokens
		totalCompletion += resp.Usage.CompletionTokens
		totalCache += resp.Usage.PromptCachedTokens

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("empty choices from llm")
		}
		choice := resp.Choices[0]

		// Assistant 메시지 저장 (tool_calls 포함)
		assistantMsg := &chat.Message{
			SessionID:        sess.ID,
			Role:             chat.RoleAssistant,
			Content:          choice.Message.Content,
			Model:            resp.Model,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			CacheTokens:      resp.Usage.PromptCachedTokens,
			CreatedAt:        time.Now(),
		}
		for _, tc := range choice.Message.ToolCalls {
			var parsed map[string]any
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &parsed)
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, chat.ToolCall{
				ID:      tc.ID,
				Name:    tc.Function.Name,
				Args:    parsed,
				RawArgs: tc.Function.Arguments,
			})
		}
		if _, err := uc.messageRepo.Create(assistantMsg); err != nil {
			return nil, err
		}
		// messages 배열에도 반영 (다음 루프를 위해)
		messages = append(messages, LLMChatMessage{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		})

		// 도구 호출이 없으면 종료
		if len(choice.Message.ToolCalls) == 0 {
			finalAssistant = assistantMsg
			break
		}

		// 각 도구 실행 → tool message 로 추가
		for _, tc := range choice.Message.ToolCalls {
			result, tErr := uc.runTool(ctx, tc.Function.Name, tc.Function.Arguments, toolCtx, skill)
			if tErr != nil {
				result = fmt.Sprintf(`{"error": %q}`, tErr.Error())
			}
			toolMsg := &chat.Message{
				SessionID:  sess.ID,
				Role:       chat.RoleTool,
				Content:    result,
				ToolCallID: tc.ID,
				CreatedAt:  time.Now(),
			}
			if _, err := uc.messageRepo.Create(toolMsg); err != nil {
				return nil, err
			}
			toolLogs = append(toolLogs, *toolMsg)
			messages = append(messages, LLMChatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
		// 루프 계속
	}

	if finalAssistant == nil {
		// 도구 호출 루프가 maxToolHops 도달 — 마지막 응답을 강제로 최종으로 취급
		// (마지막 추가된 assistant msg 를 가져오는 가장 최근 메시지 조회)
		msgs, _ := uc.messageRepo.ListBySession(sess.ID, 5)
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == chat.RoleAssistant {
				finalAssistant = msgs[i]
				break
			}
		}
	}

	// 세션 업데이트 (title 자동 설정 + tokens + last_message_at)
	now := time.Now()
	totalTokens := totalPrompt + totalCompletion
	if err := uc.sessionRepo.UpdateLastMessageAt(sess.ID, now, totalTokens); err != nil {
		log.Printf("[chat] update session: %v", err)
	}
	if strings.TrimSpace(sess.Title) == "" || sess.Title == "새 대화" {
		title := truncateForTitle(in.Message)
		_ = uc.sessionRepo.UpdateTitle(sess.ID, title)
	}

	// 사용량 기록 (학교 부담, 관리자 대시보드용)
	cost := llm.CostKRW(totalPrompt, totalCompletion, totalCache)
	if err := uc.usageRepo.AddUsage(in.UserID, now, 1, totalPrompt, totalCompletion, totalCache, cost); err != nil {
		log.Printf("[chat] usage record: %v", err)
	}

	return &AskOutput{Message: finalAssistant, ToolLogs: toolLogs}, nil
}

// ============================================================================
// internal helpers
// ============================================================================

func (uc *ChatUseCase) resolveSkill(sess *chat.Session, override string, isAdmin bool) (*chat.Skill, error) {
	if override != "" {
		sk, err := uc.skillRepo.FindBySlug(override)
		if err != nil {
			return nil, err
		}
		if sk.AdminOnly && !isAdmin {
			return nil, chat.ErrAdminOnly
		}
		if !sk.Enabled && !isAdmin {
			return nil, chat.ErrSkillDisabled
		}
		_ = uc.sessionRepo.UpdateActiveSkill(sess.ID, &sk.ID)
		return sk, nil
	}
	if sess.ActiveSkillID != nil {
		sk, err := uc.skillRepo.FindByID(*sess.ActiveSkillID)
		if err == nil {
			return sk, nil
		}
		if !errors.Is(err, chat.ErrSkillNotFound) {
			return nil, err
		}
		// 삭제된 스킬 → fallback
	}
	// 기본 skill
	sk, err := uc.skillRepo.FindBySlug("general_ta")
	if err != nil {
		if errors.Is(err, chat.ErrSkillNotFound) {
			return defaultGeneralSkill(), nil
		}
		return nil, err
	}
	return sk, nil
}

func (uc *ChatUseCase) resolveModelAndEffort(skill *chat.Skill, mode chat.AskMode, isAdmin bool) (model, effort string) {
	// 관리자 기본: reasoning high
	if isAdmin {
		return "qwen-reasoning", "high"
	}
	// 학생 explicit mode override
	switch mode {
	case chat.ModeFast:
		return "qwen-chat", ""
	case chat.ModeDeep:
		return "qwen-reasoning", "medium"
	}
	// skill 기본값
	if skill.DefaultModel != "" {
		return skill.DefaultModel, skill.DefaultReasoningEffort
	}
	return "qwen-chat", ""
}

func (uc *ChatUseCase) runTool(ctx context.Context, name, argsJSON string, tctx ChatToolCtx, skill *chat.Skill) (string, error) {
	tool, ok := uc.tools.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	// skill 의 허용 목록 재확인 (llm 이 제멋대로 호출하는 걸 방지)
	allowed := false
	for _, t := range skill.ToolsAllowed {
		if t == name {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("tool %q not allowed for skill %q", name, skill.Slug)
	}
	if tool.AdminOnly && !tctx.IsAdmin {
		return "", fmt.Errorf("tool %q is admin-only", name)
	}
	return tool.Run(ctx, tctx, argsJSON)
}

func buildChatMessages(skill *chat.Skill, history []*chat.Message) []LLMChatMessage {
	out := make([]LLMChatMessage, 0, len(history)+2)
	// 시스템 프롬프트: skill 프롬프트 + 공통 보조 지침
	sys := strings.TrimSpace(skill.SystemPrompt)
	if sys == "" {
		sys = "너는 이화여대 창업 수업 LMS 의 친절한 조교야."
	}
	sys += "\n\n# 공통 지침\n" +
		"- 한국어로 답변해.\n" +
		"- LMS 내부 용어(지갑, 회사, 정부과제, 공시 등) 질문은 먼저 search_wiki 도구로 관련 문서를 찾아보고 근거 있는 답을 줘.\n" +
		"- 사용자의 개인 데이터(지갑 잔액, 거래내역) 가 필요한 질문엔 해당 도구를 호출해.\n" +
		"- 잘 모르면 모른다고 말하고 관련 문서·담당자에게 문의하라고 안내해.\n" +
		"- 학생이 개인정보를 입력하려 하면 정중히 만류해."
	out = append(out, LLMChatMessage{Role: "system", Content: sys})

	for _, m := range history {
		switch m.Role {
		case chat.RoleUser:
			out = append(out, LLMChatMessage{Role: "user", Content: m.Content})
		case chat.RoleAssistant:
			msg := LLMChatMessage{Role: "assistant", Content: m.Content}
			for _, tc := range m.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, LLMChatToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: LLMChatToolFunc{
						Name:      tc.Name,
						Arguments: tc.RawArgs,
					},
				})
			}
			out = append(out, msg)
		case chat.RoleTool:
			out = append(out, LLMChatMessage{
				Role:       "tool",
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})
		}
	}
	return out
}

func buildToolSpecs(tools []*ChatTool) []LLMChatToolSpec {
	out := make([]LLMChatToolSpec, 0, len(tools))
	for _, t := range tools {
		out = append(out, LLMChatToolSpec{
			Type: "function",
			Function: LLMChatToolSpecFunc{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return out
}

func pickMaxTokens(model, effort string) int {
	switch model {
	case "qwen-chat":
		return 2048
	case "qwen-reasoning":
		switch effort {
		case "low":
			return 2048
		case "medium":
			return 8192
		case "high":
			return 16384
		}
		return 4096
	case "qwen":
		return 8192
	}
	return 2048
}

func truncateForTitle(s string) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > 30 {
		return string(runes[:30]) + "..."
	}
	return s
}

func defaultGeneralSkill() *chat.Skill {
	return &chat.Skill{
		ID:           0,
		Slug:         "general_ta",
		Name:         "일반 조교",
		SystemPrompt: "너는 이화여대 창업 수업 LMS 의 친절한 조교야. 학생 질문에 간결하고 근거 있게 답해.",
		DefaultModel: "qwen-chat",
		ToolsAllowed: []string{"search_wiki"},
		Enabled:      true,
	}
}
