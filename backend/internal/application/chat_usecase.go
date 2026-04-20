package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/chat"
	"github.com/earnlearning/backend/internal/domain/llm"
)

// ChatLLMClient 은 LLM proxy 호출에 필요한 최소 인터페이스 (test 에서 fake 주입).
// 구체 구현은 infrastructure/llmproxy 의 ChatAdapter.
type ChatLLMClient interface {
	ChatComplete(ctx context.Context, req *LLMChatRequest) (*LLMChatResponse, error)
	ChatCompleteStream(ctx context.Context, req *LLMChatRequest) (<-chan LLMStreamEvent, error)
	// Stats 는 현재 동시 호출 메트릭 (#088 큐잉 진행률용).
	Stats() LLMStats
}

type LLMStats struct {
	InFlight int
	Waiting  int
	Cap      int
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
	wikiRootDir string     // optional; used by admin wiki editor for file write-through
	notion      NotionFetcher // optional; used by Notion 자동 동기화 (#082)
	maxToolHops int
}

// NotionFetcher — Notion API 에서 페이지 markdown 을 가져오는 인터페이스 (test fake 용).
type NotionFetcher interface {
	FetchPageMarkdown(ctx context.Context, pageID string) (string, error)
}

// SetNotion — main.go 에서 Notion client 주입 (NotionToken 있을 때만).
func (uc *ChatUseCase) SetNotion(n NotionFetcher) { uc.notion = n }

// SetWikiRootDir — main.go 에서 wiki 루트 디렉토리를 주입 (admin 편집기 파일 쓰기용).
func (uc *ChatUseCase) SetWikiRootDir(dir string) { uc.wikiRootDir = dir }

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

// AdminListAllSessions — 관리자 전용. userID 필터 옵션 (0 이면 전체), query 는 title LIKE.
func (uc *ChatUseCase) AdminListAllSessions(userID int, query string, page int) ([]*chat.Session, int, error) {
	return uc.sessionRepo.ListAll(userID, query, page, 50)
}

// AdminLLMStats — 현재 LLM 동시 호출 메트릭 (#089).
func (uc *ChatUseCase) AdminLLMStats() LLMStats { return uc.llm.Stats() }

// AdminUsageDashboard — 관리자 비용 대시보드. days 일치 일별 합계 + 상위 지출 학생.
func (uc *ChatUseCase) AdminUsageDashboard(days int) (map[string]any, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	to := time.Now().UTC()
	from := to.AddDate(0, 0, -days+1)
	daily, err := uc.usageRepo.SumForRange(from, to)
	if err != nil {
		return nil, err
	}
	top, err := uc.usageRepo.TopUsersForRange(from, to, 20)
	if err != nil {
		return nil, err
	}
	if daily == nil {
		daily = []*chat.UsageDay{}
	}
	if top == nil {
		top = []*chat.UserUsageTotal{}
	}
	return map[string]any{
		"days":     days,
		"daily":    daily,
		"top_users": top,
	}, nil
}

// AdminGetSession — 관리자 전용. 다른 유저의 세션도 열람 가능. user_name 포함.
func (uc *ChatUseCase) AdminGetSession(sessionID int) (*chat.Session, error) {
	s, err := uc.sessionRepo.FindByIDWithUser(sessionID)
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

// AdminUpdateWikiDocCallable — handler 가 rootDir 모르고 호출하도록.
func (uc *ChatUseCase) AdminUpdateWikiDocAt(slug, title, body string) error {
	return uc.AdminUpdateWikiDoc(slug, title, body, uc.wikiRootDir)
}

// AdminSyncNotionOne — Notion 에서 한 wiki 문서 fetch → DB + .md 갱신 (#082).
func (uc *ChatUseCase) AdminSyncNotionOne(ctx context.Context, slug string) error {
	if uc.notion == nil {
		return fmt.Errorf("notion 통합이 설정되지 않았습니다 (NOTION_INTEGRATION_TOKEN)")
	}
	meta, err := uc.wikiRepo.FindMeta(slug)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("wiki doc not found: %s", slug)
	}
	if meta.NotionPageID == "" {
		return fmt.Errorf("이 문서는 notion_page_id 가 없습니다: %s", slug)
	}
	body, err := uc.notion.FetchPageMarkdown(ctx, meta.NotionPageID)
	if err != nil {
		return fmt.Errorf("notion fetch: %w", err)
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("notion 응답이 비어있음 (페이지 권한 / 빈 페이지 확인)")
	}
	// 제목은 기존 것 유지 (Notion 제목이 안정적이지 않은 경우 admin 의도가 깨질 수 있음)
	return uc.AdminUpdateWikiDoc(slug, meta.Title, body, uc.wikiRootDir)
}

// AdminSyncNotionAll — notion_page_id 가 있는 모든 wiki 문서를 일괄 동기화.
// 각 문서별 결과 (성공/실패) 반환. 한 개 실패해도 나머지 계속.
func (uc *ChatUseCase) AdminSyncNotionAll(ctx context.Context) (results []NotionSyncResult, err error) {
	if uc.notion == nil {
		return nil, fmt.Errorf("notion 통합이 설정되지 않았습니다 (NOTION_INTEGRATION_TOKEN)")
	}
	metas, err := uc.wikiRepo.ListMeta()
	if err != nil {
		return nil, err
	}
	for _, m := range metas {
		if m.NotionPageID == "" {
			results = append(results, NotionSyncResult{Slug: m.Slug, Skipped: true, Error: "no notion_page_id"})
			continue
		}
		if err := uc.AdminSyncNotionOne(ctx, m.Slug); err != nil {
			results = append(results, NotionSyncResult{Slug: m.Slug, Error: err.Error()})
			log.Printf("[notion-sync] %s: %v", m.Slug, err)
		} else {
			results = append(results, NotionSyncResult{Slug: m.Slug, OK: true})
		}
	}
	return results, nil
}

type NotionSyncResult struct {
	Slug    string `json:"slug"`
	OK      bool   `json:"ok"`
	Skipped bool   `json:"skipped,omitempty"`
	Error   string `json:"error,omitempty"`
}

// AdminGetWikiDoc — admin 전용. meta + body 반환.
func (uc *ChatUseCase) AdminGetWikiDoc(slug string) (*chat.WikiDocMeta, string, error) {
	meta, err := uc.wikiRepo.FindMeta(slug)
	if err != nil {
		return nil, "", err
	}
	if meta == nil {
		return nil, "", fmt.Errorf("wiki doc not found: %s", slug)
	}
	_, body, err := uc.wikiRepo.GetDoc(slug)
	if err != nil {
		return nil, "", err
	}
	return meta, body, nil
}

// AdminUpdateWikiDoc — admin 전용. FTS5 + meta 업데이트, 가능하면 파일도 덮어씀.
// rootDir 가 비어있거나 파일 쓰기 실패하면 DB 만 업데이트 (warn 로그). 다음 재배포 시
// .md 파일에서 다시 동기화되므로, 영구화하려면 PR 머지 필요.
func (uc *ChatUseCase) AdminUpdateWikiDoc(slug, title, body, rootDir string) error {
	if strings.TrimSpace(slug) == "" {
		return fmt.Errorf("empty slug")
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("empty body")
	}
	meta, err := uc.wikiRepo.FindMeta(slug)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("wiki doc not found: %s", slug)
	}
	if title == "" {
		title = meta.Title
	}
	if err := uc.wikiRepo.UpsertDoc(slug, title, body); err != nil {
		return fmt.Errorf("upsert doc: %w", err)
	}
	meta.Title = title
	if err := uc.wikiRepo.UpsertMeta(meta); err != nil {
		return fmt.Errorf("upsert meta: %w", err)
	}
	// best-effort 파일 쓰기 (개발 환경에서 영구화). 컨테이너 환경에선 ephemeral.
	if rootDir != "" && meta.Path != "" {
		full := filepath.Join(rootDir, meta.Path)
		// frontmatter 보존: 기존 파일이 있으면 frontmatter 영역만 유지하고 body 만 교체
		newContent := composeMarkdown(full, title, body)
		if err := os.WriteFile(full, []byte(newContent), 0o644); err != nil {
			log.Printf("[chat] admin wiki file write failed (DB 는 갱신됨): %v", err)
		}
	}
	return nil
}

// composeMarkdown — 기존 파일의 frontmatter 를 보존하면서 body 만 교체.
// 파일이 없거나 frontmatter 가 없으면 새로 만들어 frontmatter + body 형식으로 작성.
func composeMarkdown(path, title, body string) string {
	existing, err := os.ReadFile(path)
	if err == nil {
		s := string(existing)
		if strings.HasPrefix(s, "---\n") {
			if end := strings.Index(s[4:], "\n---\n"); end >= 0 {
				return s[:4+end+5] + body
			}
		}
	}
	return "---\ntitle: " + title + "\n---\n" + body
}

// ============================================================================
// Ask flow (the main entry)
// ============================================================================

type AskInput struct {
	SessionID   int
	UserID      int
	IsAdmin     bool
	Message     string
	Mode        chat.AskMode // "fast" | "deep", 빈 값이면 skill default
	SkillSlug   string       // 선택적으로 이 세션의 스킬 override
	Attachments []string     // #106 학생 첨부 이미지 URL (uploads/xxx.png)
}

type AskOutput struct {
	Message  *chat.Message  `json:"message"`   // final assistant message 저장본
	ToolLogs []chat.Message `json:"tool_logs"` // 실행된 툴 결과들 (UI 표시용 부가 정보)
}

// StreamEventType — 프런트로 보내는 SSE event 종류.
type StreamEventType string

const (
	StreamEventToolCall   StreamEventType = "tool_call"     // 어시스턴트가 도구 호출 결정 (이름 + args)
	StreamEventToolResult StreamEventType = "tool_result"   // 도구 실행 결과 (id + content)
	StreamEventTextDelta  StreamEventType = "text_delta"    // 최종 응답 토큰 chunk
	StreamEventDone       StreamEventType = "done"          // 완료 (총 token / 최종 message id 포함)
	StreamEventError      StreamEventType = "error"
	StreamEventQueued     StreamEventType = "queued"        // #088: LLM 큐 대기 중 (waiting 인원)
)

// AskStreamEvent — handler 가 SSE 로 직렬화해 클라에 push.
type AskStreamEvent struct {
	Type        StreamEventType `json:"type"`
	Delta       string          `json:"delta,omitempty"`
	ToolName    string          `json:"tool_name,omitempty"`
	ToolID      string          `json:"tool_id,omitempty"`
	ToolArgs    string          `json:"tool_args,omitempty"`
	ToolContent string          `json:"tool_content,omitempty"`
	MessageID   int             `json:"message_id,omitempty"`
	Tokens      int             `json:"tokens,omitempty"`
	Error       string          `json:"error,omitempty"`
	// #088 queued: waiting > 0 일 때 현재 대기 인원. 0 이면 큐에서 빠져나옴.
	QueueWaiting int `json:"queue_waiting,omitempty"`
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

	// 사용자 입력 메시지 저장 (#106 첨부 이미지 포함)
	userMsg := &chat.Message{
		SessionID:   sess.ID,
		Role:        chat.RoleUser,
		Content:     in.Message,
		Attachments: in.Attachments,
		CreatedAt:   time.Now(),
	}
	if _, err := uc.messageRepo.Create(userMsg); err != nil {
		return nil, err
	}

	// 모델 선택
	model, effort := uc.resolveModelAndEffort(skill, in.Mode, in.IsAdmin)

	// OpenAI-format 메시지 배열 조립 — system + 과거 히스토리 + 이번 user
	// (이번 user 는 위에서 저장됐으니 history 에 포함됨)
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

// AskStream — Ask 와 동일한 도구 루프지만 결과를 channel 로 흘려보냄.
//
// 설계:
//   - 도구 호출 hop 은 기존처럼 non-streaming (단일 호출 → 도구 실행 → tool_result event 발행)
//   - tool_calls 가 없는 "최종 응답" turn 만 ChatCompleteStream 으로 전환
//   - 각 text delta 를 channel 에 push
//   - 마지막에 assistant 메시지 DB 저장 후 "done" event
//
// channel 은 끝나면 close. 호출자(handler)가 ctx 로 취소.
func (uc *ChatUseCase) AskStream(ctx context.Context, in AskInput) (<-chan AskStreamEvent, error) {
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
	skill, err := uc.resolveSkill(sess, in.SkillSlug, in.IsAdmin)
	if err != nil {
		return nil, err
	}
	userMsg := &chat.Message{
		SessionID: sess.ID,
		Role:      chat.RoleUser,
		Content:   in.Message,
		CreatedAt: time.Now(),
	}
	if _, err := uc.messageRepo.Create(userMsg); err != nil {
		return nil, err
	}

	// FAQ shortcut (#090) — 짧은 인사/감사 류는 LLM 안 거치고 즉시 응답.
	// LLM 슬롯 점유 + 비용 절약. 매칭 못 한 모든 메시지는 평소처럼 LLM 호출.
	if faqResp, ok := lookupFAQ(in.Message); ok {
		out := make(chan AskStreamEvent, 4)
		go func() {
			defer close(out)
			uc.respondFAQ(sess, in, faqResp, out)
		}()
		return out, nil
	}

	model, effort := uc.resolveModelAndEffort(skill, in.Mode, in.IsAdmin)
	history, err := uc.messageRepo.ListBySession(sess.ID, 50)
	if err != nil {
		return nil, err
	}
	messages := buildChatMessages(skill, history)
	allowedTools := uc.tools.Filter(skill.ToolsAllowed, in.IsAdmin)
	toolSpecs := buildToolSpecs(allowedTools)
	toolCtx := ChatToolCtx{UserID: in.UserID, IsAdmin: in.IsAdmin, SessionID: sess.ID}

	out := make(chan AskStreamEvent, 64)
	go func() {
		defer close(out)
		uc.runAskStream(ctx, sess, skill, model, effort, messages, toolSpecs, toolCtx, in, out)
	}()
	return out, nil
}

// runAskStream — AskStream 의 내부 루프. 채널 close 는 caller 에서.
func (uc *ChatUseCase) runAskStream(
	ctx context.Context,
	sess *chat.Session,
	skill *chat.Skill,
	model, effort string,
	messages []LLMChatMessage,
	toolSpecs []LLMChatToolSpec,
	toolCtx ChatToolCtx,
	in AskInput,
	out chan<- AskStreamEvent,
) {
	totalPrompt, totalCompletion, totalCache := 0, 0, 0
	emit := func(ev AskStreamEvent) {
		select {
		case out <- ev:
		case <-ctx.Done():
		}
	}

	// 도구 호출 hop (non-streaming)
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
		qStop := uc.startQueueProgress(ctx, emit)
		resp, err := uc.llm.ChatComplete(ctx, req)
		qStop()
		if err != nil {
			emit(AskStreamEvent{Type: StreamEventError, Error: fmt.Sprintf("llm: %v", err)})
			return
		}
		totalPrompt += resp.Usage.PromptTokens
		totalCompletion += resp.Usage.CompletionTokens
		totalCache += resp.Usage.PromptCachedTokens
		if len(resp.Choices) == 0 {
			emit(AskStreamEvent{Type: StreamEventError, Error: "empty choices"})
			return
		}
		choice := resp.Choices[0]

		// 도구 호출 없으면 → 최종 응답
		if len(choice.Message.ToolCalls) == 0 {
			if choice.Message.Content != "" {
				// 첫 hop 에서 도구 없이 바로 답한 경우 — 전체 content 를 한 번의 text_delta 로
				// 흘려보냄 (재호출하지 않음). UX 차원에서 "한 번에 도착" 하지만 정확한 정보.
				emit(AskStreamEvent{Type: StreamEventTextDelta, Delta: choice.Message.Content})
				uc.finalizeStreamFromText(sess, in, choice.Message.Content, model,
					resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.PromptCachedTokens,
					totalPrompt, totalCompletion, totalCache, emit)
				return
			}
			// content 가 비었으면 → 도구 결과 후 후속 응답이 필요한 상황. streaming 으로 재호출
			uc.streamFinalAnswer(ctx, sess, skill, model, effort, messages, in,
				totalPrompt, totalCompletion, totalCache, emit)
			return
		}

		// assistant 메시지 저장 (tool_calls 포함, content 가 있다면 함께)
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
			emit(AskStreamEvent{
				Type:     StreamEventToolCall,
				ToolID:   tc.ID,
				ToolName: tc.Function.Name,
				ToolArgs: tc.Function.Arguments,
			})
		}
		if _, err := uc.messageRepo.Create(assistantMsg); err != nil {
			emit(AskStreamEvent{Type: StreamEventError, Error: err.Error()})
			return
		}
		messages = append(messages, LLMChatMessage{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		})

		// 각 도구 실행 → tool message 저장 + tool_result event
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
				emit(AskStreamEvent{Type: StreamEventError, Error: err.Error()})
				return
			}
			emit(AskStreamEvent{
				Type:        StreamEventToolResult,
				ToolID:      tc.ID,
				ToolName:    tc.Function.Name,
				ToolContent: result,
			})
			messages = append(messages, LLMChatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}
	// maxToolHops 도달 — 마지막 응답을 streaming 으로 한 번 더 시도
	uc.streamFinalAnswer(ctx, sess, nil, model, effort, messages, in,
		totalPrompt, totalCompletion, totalCache, emit)
}

// startQueueProgress — LLM 호출 직전에 시작, 1.5s 안 끝나면 매 2s 마다 현재
// waiting 인원 push. 반환된 stop() 은 goroutine 이 완전히 종료할 때까지 블로킹 —
// caller 가 emit 채널을 close 하기 전에 호출해야 race-free.
func (uc *ChatUseCase) startQueueProgress(ctx context.Context, emit func(AskStreamEvent)) func() {
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-time.After(1500 * time.Millisecond):
		}
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		lastWaiting := -1
		for {
			select {
			case <-done:
				if lastWaiting > 0 {
					emit(AskStreamEvent{Type: StreamEventQueued, QueueWaiting: 0})
				}
				return
			case <-ctx.Done():
				return
			case <-t.C:
				s := uc.llm.Stats()
				if s.Waiting != lastWaiting && s.Waiting > 0 {
					emit(AskStreamEvent{Type: StreamEventQueued, QueueWaiting: s.Waiting})
					lastWaiting = s.Waiting
				}
			}
		}
	}()
	return func() {
		close(done)
		<-exited
	}
}

// streamFinalAnswer — 도구 없이 최종 응답만 streaming 으로 받음.
func (uc *ChatUseCase) streamFinalAnswer(
	ctx context.Context,
	sess *chat.Session,
	_ *chat.Skill,
	model, effort string,
	messages []LLMChatMessage,
	in AskInput,
	prevPrompt, prevCompletion, prevCache int,
	emit func(AskStreamEvent),
) {
	req := &LLMChatRequest{
		Model:           model,
		Messages:        messages,
		MaxTokens:       pickMaxTokens(model, effort),
		ReasoningEffort: effort,
	}
	qStop := uc.startQueueProgress(ctx, emit)
	stream, err := uc.llm.ChatCompleteStream(ctx, req)
	if err != nil {
		qStop()
		emit(AskStreamEvent{Type: StreamEventError, Error: fmt.Sprintf("stream: %v", err)})
		return
	}
	qStop()
	var contentBuf strings.Builder
	var thisPrompt, thisCompletion, thisCache int
	for ev := range stream {
		if ev.Err != nil {
			emit(AskStreamEvent{Type: StreamEventError, Error: ev.Err.Error()})
			return
		}
		if ev.TextDelta != "" {
			contentBuf.WriteString(ev.TextDelta)
			emit(AskStreamEvent{Type: StreamEventTextDelta, Delta: ev.TextDelta})
		}
		if ev.Usage != nil {
			thisPrompt = ev.Usage.PromptTokens
			thisCompletion = ev.Usage.CompletionTokens
			thisCache = ev.Usage.PromptCachedTokens
		}
	}
	final := contentBuf.String()
	uc.finalizeStreamFromText(sess, in, final, model,
		thisPrompt, thisCompletion, thisCache,
		prevPrompt+thisPrompt, prevCompletion+thisCompletion, prevCache+thisCache, emit)
}

// finalizeStreamFromText — 최종 텍스트가 결정된 후 DB/usage 저장 + done event.
func (uc *ChatUseCase) finalizeStreamFromText(
	sess *chat.Session,
	in AskInput,
	finalText, model string,
	thisPrompt, thisCompletion, thisCache int,
	totalPrompt, totalCompletion, totalCache int,
	emit func(AskStreamEvent),
) {
	assistantMsg := &chat.Message{
		SessionID:        sess.ID,
		Role:             chat.RoleAssistant,
		Content:          finalText,
		Model:            model,
		PromptTokens:     thisPrompt,
		CompletionTokens: thisCompletion,
		CacheTokens:      thisCache,
		CreatedAt:        time.Now(),
	}
	if _, err := uc.messageRepo.Create(assistantMsg); err != nil {
		emit(AskStreamEvent{Type: StreamEventError, Error: err.Error()})
		return
	}
	now := time.Now()
	totalTokens := totalPrompt + totalCompletion
	if err := uc.sessionRepo.UpdateLastMessageAt(sess.ID, now, totalTokens); err != nil {
		log.Printf("[chat-stream] update session: %v", err)
	}
	if strings.TrimSpace(sess.Title) == "" || sess.Title == "새 대화" {
		title := truncateForTitle(in.Message)
		_ = uc.sessionRepo.UpdateTitle(sess.ID, title)
	}
	cost := llm.CostKRW(totalPrompt, totalCompletion, totalCache)
	if err := uc.usageRepo.AddUsage(in.UserID, now, 1, totalPrompt, totalCompletion, totalCache, cost); err != nil {
		log.Printf("[chat-stream] usage: %v", err)
	}
	emit(AskStreamEvent{
		Type:      StreamEventDone,
		MessageID: assistantMsg.ID,
		Tokens:    totalTokens,
	})
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

// absoluteImageURL — /uploads/xxx.png 같은 상대 path 를 absolute URL 로 변환.
// llama-server vision 은 외부 호출 가능한 http(s) URL 또는 data: URI 만 허용.
// PUBLIC_BASE_URL env (예: https://earnlearning.com) 가 있으면 그걸 prefix.
func absoluteImageURL(u string) string {
	if u == "" {
		return ""
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "data:") {
		return u
	}
	base := os.Getenv("PUBLIC_BASE_URL")
	if base == "" {
		base = "https://earnlearning.com"
	}
	if !strings.HasPrefix(u, "/") {
		u = "/" + u
	}
	return strings.TrimRight(base, "/") + u
}

func buildChatMessages(skill *chat.Skill, history []*chat.Message) []LLMChatMessage {
	out := make([]LLMChatMessage, 0, len(history)+2)
	// 시스템 프롬프트: skill 프롬프트 + 공통 보조 지침
	sys := strings.TrimSpace(skill.SystemPrompt)
	if sys == "" {
		sys = "너는 이화여대 창업 수업 LMS 의 친절한 조교야."
	}
	sys += "\n\n# 공통 지침 (반드시 준수)\n" +
		"- 한국어로 답변해.\n" +
		"- **환각 금지**: 도구 결과에 명시된 사실만 인용. 도구가 반환하지 않은 수치·규칙·조문은 절대 지어내지 마.\n" +
		"- **LMS 정책 질문 (지갑·회사·공시·정부과제·청산·주주총회 등)**:\n" +
		"  1) 먼저 search_wiki 도구로 관련 문서를 찾아.\n" +
		"  2) 답변은 **오직 wiki 내용만** 근거로 구성. 일반 상법·회사법·민법 등 외부 지식을 섞지 마. LMS 는 시뮬레이션이라 실제 법과 규칙이 다름.\n" +
		"  3) wiki 에 없는 내용이면 \"공식 가이드에 명시돼 있지 않습니다\" 라고 답하고 담당자(cycorld) 문의를 안내.\n" +
		"- 사용자의 개인 데이터(지갑 잔액, 거래내역) 가 필요한 질문엔 해당 도구를 호출해.\n" +
		"- **도구 호출은 최소**: 필요한 질문 하나만 던지고, 첫 결과로 답이 충분하면 즉시 답변 작성. 불필요한 추가 조회 금지.\n" +
		"- 확실하지 않으면 추측하지 말고 \"확실하지 않음\" 이라고 명시하고 추가 질문으로 명확화 요청.\n" +
		"- 학생이 개인정보를 입력하려 하면 정중히 만류해."
	out = append(out, LLMChatMessage{Role: "system", Content: sys})

	for _, m := range history {
		switch m.Role {
		case chat.RoleUser:
			msg := LLMChatMessage{Role: "user", Content: m.Content}
			// #106 vision: 첨부 이미지 있으면 OpenAI multimodal content array 로 변환
			if len(m.Attachments) > 0 {
				msg.ContentParts = []LLMContentBlock{{Type: "text", Text: m.Content}}
				for _, u := range m.Attachments {
					msg.ContentParts = append(msg.ContentParts, LLMContentBlock{
						Type:     "image_url",
						ImageURL: absoluteImageURL(u),
					})
				}
			}
			out = append(out, msg)
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
