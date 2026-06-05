package application

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/grant"
	"github.com/earnlearning/backend/internal/domain/milestone"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/user"
)

type MilestoneUseCase struct {
	repo        milestone.Repository
	userRepo    user.Repository
	companyRepo company.CompanyRepository
	grantRepo   grant.Repository
	notifUC     *NotificationUseCase
	llm         ChatLLMClient // #120 optional — nil 이면 LLM 평가 스킵, heuristic 만.
	llmModel    string        // 기본 "qwen-chat".
}

func NewMilestoneUseCase(
	repo milestone.Repository,
	userRepo user.Repository,
	companyRepo company.CompanyRepository,
	grantRepo grant.Repository,
	notifUC *NotificationUseCase,
) *MilestoneUseCase {
	return &MilestoneUseCase{
		repo:        repo,
		userRepo:    userRepo,
		companyRepo: companyRepo,
		grantRepo:   grantRepo,
		notifUC:     notifUC,
		llmModel:    "qwen-chat",
	}
}

// SetLLM — main.go 에서 chatLLM 어댑터 주입 (#120 회고 에세이 AI 평가).
// nil 이면 heuristic 만 사용.
func (uc *MilestoneUseCase) SetLLM(llm ChatLLMClient, model string) {
	uc.llm = llm
	if model != "" {
		uc.llmModel = model
	}
}

// SubmitManualInput — 학생의 수동 제출 입력.
type SubmitManualInput struct {
	Type    milestone.Type `json:"type"`
	URL     string         `json:"url"`
	Content string         `json:"content"`
}

// SubmitManual — 학생이 직접 (form으로) milestone을 제출/수정.
// 회고(retrospective) 의 경우 본문이 200자 이상이면 AI 작성 확률 자동 평가 후 저장.
// MVP 타입에 대해서도 학생이 직접 URL을 명시할 수 있게 허용함
// (자동 detect 가 실패할 때 fallback).
// 단, URL이 deny list 에 걸리면 거절.
func (uc *MilestoneUseCase) SubmitManual(studentID int, in SubmitManualInput) (*milestone.Milestone, error) {
	if !in.Type.Valid() {
		return nil, milestone.ErrInvalidType
	}
	url := strings.TrimSpace(in.URL)
	content := strings.TrimSpace(in.Content)

	// MVP 타입은 URL이 반드시 있어야 하고, deny list 통과해야 함.
	if in.Type == milestone.TypeMVP1 || in.Type == milestone.TypeMVP2 {
		if url == "" {
			return nil, fmt.Errorf("MVP 제출에는 배포 URL이 필요합니다")
		}
		if !milestone.IsValidMilestoneURL(url) {
			return nil, fmt.Errorf("vercel.app 또는 자체 도메인만 인정됩니다 (AI Studio·Claude·ChatGPT 등 연습용은 제외)")
		}
	} else {
		// business_plan / retrospective — content 또는 URL 둘 중 하나는 있어야 함.
		if url == "" && content == "" {
			return nil, fmt.Errorf("URL 또는 본문 중 하나는 입력해야 합니다")
		}
		// URL을 옵션으로 넣었으면 그것도 deny list 통과해야 함.
		if url != "" && !milestone.IsValidMilestoneURL(url) {
			return nil, fmt.Errorf("URL이 유효하지 않습니다 (연습용 도메인 제외)")
		}
	}

	m := &milestone.Milestone{
		StudentID:  studentID,
		Type:       in.Type,
		SourceType: milestone.SourceManual,
		URL:        url,
		Content:    content,
		Status:     milestone.StatusPending,
	}
	id, err := uc.repo.Upsert(m)
	if err != nil {
		return nil, err
	}

	// 회고 에세이는 자동 평가 (#120).
	if in.Type == milestone.TypeRetrospective && len([]rune(content)) >= 200 {
		// 평가 실패해도 제출은 성공으로 처리.
		_, _ = uc.ScoreAndStoreEssay(context.Background(), id, content)
	}

	return uc.repo.FindByID(id)
}

// SyncAuto — 학생의 회사 service_url + grant_applications.proposal 에서
// MVP1 / MVP2 를 자동 detect 해서 upsert.
// 이미 admin이 승인한 row 는 건드리지 않음 (URL 변경에도 승인 유지).
//
// 규칙:
// - 학생이 owner인 모든 회사의 service_url 를 모두 모아 파싱 → deny list 통과한 URL 만 추출
// - + grant_applications 의 proposal 텍스트에서 URL 추출 → 같은 필터 적용
// - 회사 등록 순서 + grant 등록 순서 (시간순)로 1번 = MVP1, 2번 = MVP2
func (uc *MilestoneUseCase) SyncAuto(studentID int) ([]*milestone.Milestone, error) {
	candidates, err := uc.collectCandidates(studentID)
	if err != nil {
		return nil, err
	}

	mvpTypes := []milestone.Type{milestone.TypeMVP1, milestone.TypeMVP2}
	for i, t := range mvpTypes {
		if i >= len(candidates) {
			break
		}
		cand := candidates[i]

		// 이미 approved 상태면 자동 갱신 스킵 (admin이 다시 검토하지 않도록).
		existing, err := uc.repo.FindByStudentAndType(studentID, t)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.Status == milestone.StatusApproved {
			// 같은 URL이면 그대로, 다른 URL이면 그래도 유지 (승인 보호).
			continue
		}

		m := &milestone.Milestone{
			StudentID:   studentID,
			Type:        t,
			SourceType:  cand.SourceType,
			SourceRefID: cand.SourceRefID,
			URL:         cand.URL,
		}
		if _, err := uc.repo.Upsert(m); err != nil {
			return nil, err
		}
	}

	return uc.repo.ListByStudent(studentID)
}

type autoCandidate struct {
	URL         string
	SourceType  milestone.SourceType
	SourceRefID *int
}

// collectCandidates — 회사/grant 에서 유효 URL 후보를 모아 시간순 정렬.
func (uc *MilestoneUseCase) collectCandidates(studentID int) ([]autoCandidate, error) {
	var out []autoCandidate

	// 1) 회사 service_url — created_at ASC 순으로 (FindByOwnerID 는 DESC라서 reverse)
	companies, err := uc.companyRepo.FindByOwnerID(studentID)
	if err != nil {
		return nil, err
	}
	// reverse to ASC
	for i := len(companies) - 1; i >= 0; i-- {
		c := companies[i]
		urls := milestone.ParseCommaSeparated(c.ServiceURL)
		valid := milestone.FilterValidURLs(urls)
		for _, u := range valid {
			cid := c.ID
			out = append(out, autoCandidate{
				URL:         u,
				SourceType:  milestone.SourceCompany,
				SourceRefID: &cid,
			})
		}
	}

	// 2) grant_applications.proposal 에서 URL 추출
	apps, err := uc.grantRepo.ListApplicationsByUserID(studentID)
	if err != nil {
		return nil, err
	}
	// ListApplicationsByUserID returns DESC; reverse to ASC.
	for i := len(apps) - 1; i >= 0; i-- {
		a := apps[i]
		extracted := milestone.ExtractURLsFromText(a.Proposal)
		valid := milestone.FilterValidURLs(extracted)
		for _, u := range valid {
			aid := a.ID
			out = append(out, autoCandidate{
				URL:         u,
				SourceType:  milestone.SourceGrant,
				SourceRefID: &aid,
			})
		}
	}

	return dedupCandidates(out), nil
}

// dedupCandidates — same URL 두 번 (회사+grant 양쪽 등록) 인 경우 최초 등장만.
func dedupCandidates(in []autoCandidate) []autoCandidate {
	seen := map[string]bool{}
	out := make([]autoCandidate, 0, len(in))
	for _, c := range in {
		if seen[c.URL] {
			continue
		}
		seen[c.URL] = true
		out = append(out, c)
	}
	return out
}

// ListForStudent — 학생 본인용 대시보드. 4개 type 순서대로 (없는 자리는 nil).
func (uc *MilestoneUseCase) ListForStudent(studentID int) (*milestone.StudentProgress, error) {
	u, err := uc.userRepo.FindByID(studentID)
	if err != nil {
		return nil, err
	}
	all, err := uc.repo.ListByStudent(studentID)
	if err != nil {
		return nil, err
	}
	return buildProgress(u, all), nil
}

// ListAll — 관리자용. 모든 학생 + 각자 4개 milestone 매트릭스.
func (uc *MilestoneUseCase) ListAll() ([]*milestone.StudentProgress, error) {
	// 학생은 보통 40명 이하 — 한번에 가져옴.
	students, _, err := uc.userRepo.ListAll(1, 1000)
	if err != nil {
		return nil, err
	}
	out := make([]*milestone.StudentProgress, 0, len(students))
	for _, u := range students {
		if u.Role != user.RoleStudent {
			continue
		}
		ms, err := uc.repo.ListByStudent(u.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, buildProgress(u, ms))
	}
	return out, nil
}

func buildProgress(u *user.User, ms []*milestone.Milestone) *milestone.StudentProgress {
	byType := map[milestone.Type]*milestone.Milestone{}
	approved := 0
	for _, m := range ms {
		byType[m.Type] = m
		if m.Status == milestone.StatusApproved {
			approved++
		}
	}
	ordered := make([]*milestone.Milestone, 0, len(milestone.AllTypes))
	for _, t := range milestone.AllTypes {
		ordered = append(ordered, byType[t]) // may be nil
	}
	return &milestone.StudentProgress{
		Student: milestone.StudentRef{
			ID:         u.ID,
			Name:       u.Name,
			StudentID:  u.StudentID,
			Department: u.Department,
		},
		Milestones:    ordered,
		ApprovedCount: approved,
		Group:         milestone.ClassifyGroup(approved),
	}
}

// Approve — admin 승인.
func (uc *MilestoneUseCase) Approve(milestoneID, adminID int, adminNote string) error {
	m, err := uc.repo.FindByID(milestoneID)
	if err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(milestoneID, milestone.StatusApproved, adminNote, adminID); err != nil {
		return err
	}
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(
			m.StudentID,
			notification.NotifType("milestone_approved"),
			"평가지표 승인",
			fmt.Sprintf("'%s' 평가지표가 승인되었습니다.", milestoneTitle(m.Type)),
			"milestone", milestoneID,
		)
	}
	return nil
}

// Reject — admin 반려.
func (uc *MilestoneUseCase) Reject(milestoneID, adminID int, adminNote string) error {
	m, err := uc.repo.FindByID(milestoneID)
	if err != nil {
		return err
	}
	if err := uc.repo.UpdateStatus(milestoneID, milestone.StatusRejected, adminNote, adminID); err != nil {
		return err
	}
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(
			m.StudentID,
			notification.NotifType("milestone_rejected"),
			"평가지표 반려",
			fmt.Sprintf("'%s' 평가지표가 반려되었습니다.", milestoneTitle(m.Type)),
			"milestone", milestoneID,
		)
	}
	return nil
}

// =============================================================================
// #120 — 회고 에세이 AI 작성 확률 평가
// =============================================================================

// EssayScoreResult — heuristic + LLM 통합 결과.
type EssayScoreResult struct {
	HeuristicScore int                `json:"heuristic_score"` // 0~100
	LLMScore       int                `json:"llm_score"`       // 0~100, LLM 없으면 -1
	CombinedScore  int                `json:"combined_score"`  // 둘의 가중평균
	LLMReasoning   string             `json:"llm_reasoning"`
	Signals        []milestone.Signal `json:"signals"`
}

// EvaluateEssay — 텍스트 → AI 작성 확률 평가.
// LLM 가 nil 이거나 실패하면 heuristic 만 사용.
func (uc *MilestoneUseCase) EvaluateEssay(ctx context.Context, text string) EssayScoreResult {
	h := milestone.ScoreHeuristic(text)
	result := EssayScoreResult{
		HeuristicScore: h.Score,
		LLMScore:       -1,
		CombinedScore:  h.Score,
		Signals:        h.Signals,
	}

	if uc.llm == nil || len(strings.TrimSpace(text)) < 200 {
		return result
	}

	// 4000자 초과는 잘라서 호출 (LLM 입력 토큰 절약, 의미 보존엔 충분).
	snippet := text
	if r := []rune(snippet); len(r) > 4000 {
		snippet = string(r[:4000])
	}

	llmResult, err := uc.callLLMScorer(ctx, snippet)
	if err != nil {
		// LLM 실패는 silent — heuristic 만으로 진행.
		result.LLMReasoning = fmt.Sprintf("(LLM 평가 실패: %v)", err)
		return result
	}

	result.LLMScore = llmResult.score
	result.LLMReasoning = llmResult.reasoning
	// 가중평균: LLM 60% + heuristic 40%.
	result.CombinedScore = (llmResult.score*60 + h.Score*40) / 100
	if result.CombinedScore > 100 {
		result.CombinedScore = 100
	}
	if result.CombinedScore < 0 {
		result.CombinedScore = 0
	}
	return result
}

type llmEssayScore struct {
	score     int
	reasoning string
}

var llmJSONExtract = regexp.MustCompile(`(?s)\{.*?\}`)

func (uc *MilestoneUseCase) callLLMScorer(ctx context.Context, essay string) (*llmEssayScore, error) {
	systemPrompt := `당신은 한국어 학생 에세이가 AI(ChatGPT, Claude 등) 로 작성됐는지 판별하는 평가관입니다.

평가 기준:
- 1인칭 + 구체적 본인 경험 (시간·장소·인물) 이 있는가
- 문장 길이/구조가 다양한가 (AI는 균질)
- "~을 통해", "결론적으로", "다음과 같다" 같은 GPT 특유 구문 빈도
- 감정 표현·이모지·반말 (사람 글의 시그널) 이 있는가
- 추상적 일반론(부정적) vs 구체적 일화(긍정적)

반드시 다음 JSON 형식으로만 응답하세요. 다른 텍스트 금지:
{"score": <0~100 정수, 높을수록 AI 가능성>, "reasoning": "<한 줄 한국어 평가 근거 (50자 이내)>"}`

	userPrompt := "다음 에세이를 평가하세요:\n\n" + essay

	req := &LLMChatRequest{
		Model: uc.llmModel,
		Messages: []LLMChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: 200,
	}

	// 타임아웃 — LLM 평가는 빠르게.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := uc.llm.ChatComplete(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 응답이 비어있습니다")
	}

	content := resp.Choices[0].Message.Content
	// JSON 추출 — LLM 이 ```json ... ``` 같은 코드블록을 붙일 수 있으니 첫 {...} 만 잡음.
	match := llmJSONExtract.FindString(content)
	if match == "" {
		return nil, fmt.Errorf("LLM 응답에 JSON 없음: %s", truncate(content, 100))
	}

	var parsed struct {
		Score     int    `json:"score"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(match), &parsed); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %w", err)
	}
	if parsed.Score < 0 {
		parsed.Score = 0
	}
	if parsed.Score > 100 {
		parsed.Score = 100
	}
	return &llmEssayScore{score: parsed.Score, reasoning: parsed.Reasoning}, nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ScoreAndStoreEssay — 회고 milestone 의 essay 를 평가하고 결과를 DB 에 저장.
// 회고가 아니거나 milestone 이 없으면 무시.
func (uc *MilestoneUseCase) ScoreAndStoreEssay(ctx context.Context, milestoneID int, text string) (EssayScoreResult, error) {
	m, err := uc.repo.FindByID(milestoneID)
	if err != nil {
		return EssayScoreResult{}, err
	}
	if m.Type != milestone.TypeRetrospective {
		return EssayScoreResult{}, fmt.Errorf("회고(retrospective) 만 평가 가능합니다")
	}
	result := uc.EvaluateEssay(ctx, text)

	signalsJSON, _ := json.Marshal(result.Signals)
	if err := uc.repo.UpdateAIScore(milestoneID, result.CombinedScore, result.LLMReasoning, string(signalsJSON)); err != nil {
		return result, err
	}
	return result, nil
}

func milestoneTitle(t milestone.Type) string {
	switch t {
	case milestone.TypeMVP1:
		return "1차 MVP 배포"
	case milestone.TypeMVP2:
		return "2차 MVP 배포"
	case milestone.TypeBusinessPlan:
		return "사업계획서 제출"
	case milestone.TypeRetrospective:
		return "한 학기 회고 발표"
	}
	return string(t)
}
