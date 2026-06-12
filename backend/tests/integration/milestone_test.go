package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/earnlearning/backend/internal/application"
)

// =============================================================================
// 4대 평가지표 (#119) 통합 테스트
//
// - 회사 service_url / grant 응모 proposal 텍스트에서 MVP URL 자동 detect
// - vercel.app·자체도메인 OK, ai.studio·claude.ai·chatgpt.com·gemini 등 deny
// - 사업계획서·회고는 수동 제출
// - admin 이 승인/반려, 4개 승인 개수로 그룹(A/B/C/D) 분류
// =============================================================================

func updateCompanyServiceURL(t *testing.T, ts *testServer, token string, companyID int, serviceURL string) {
	t.Helper()
	r := ts.put("/api/companies/"+itoaUD(companyID), map[string]string{
		"service_url": serviceURL,
	}, token)
	if !r.Success {
		t.Fatalf("update service_url: %v", r.Error)
	}
}

type myMilestonesResp struct {
	Student struct {
		ID int `json:"id"`
	} `json:"student"`
	Milestones []struct {
		ID         int    `json:"id"`
		Type       string `json:"type"`
		URL        string `json:"url"`
		Content    string `json:"content"`
		Status     string `json:"status"`
		SourceType string `json:"source_type"`
		AdminNote  string `json:"admin_note"`
	} `json:"milestones"`
	ApprovedCount int    `json:"approved_count"`
	Group         string `json:"group"`
}

func getMyMilestones(t *testing.T, ts *testServer, token string) *myMilestonesResp {
	t.Helper()
	r := ts.get("/api/milestones/mine", token)
	if !r.Success {
		t.Fatalf("get my milestones: %v", r.Error)
	}
	var out myMilestonesResp
	if err := json.Unmarshal(r.Data, &out); err != nil {
		t.Fatalf("unmarshal milestones: %v\n%s", err, string(r.Data))
	}
	return &out
}

func findMilestone(t *testing.T, resp *myMilestonesResp, typ string) *struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	SourceType string `json:"source_type"`
	AdminNote  string `json:"admin_note"`
} {
	t.Helper()
	for i := range resp.Milestones {
		if resp.Milestones[i].Type == typ {
			return &resp.Milestones[i]
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// 1) 회사 service_url 의 첫 URL = MVP1 자동 detect
// -----------------------------------------------------------------------------
func TestMilestone_AutoDetectFromCompanyURL(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "ms-detect@test.com", "탐지", "20251001", "ms_detect_co")

	updateCompanyServiceURL(t, ts, token, cid, "https://my-mvp.vercel.app")

	got := getMyMilestones(t, ts, token)

	mvp1 := findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatalf("expected mvp1 milestone, got nothing")
	}
	if mvp1.URL != "https://my-mvp.vercel.app" {
		t.Errorf("mvp1.url = %q, want vercel URL", mvp1.URL)
	}
	if mvp1.SourceType != "company" {
		t.Errorf("mvp1.source_type = %q, want company", mvp1.SourceType)
	}
	if mvp1.Status != "pending" {
		t.Errorf("mvp1.status = %q, want pending", mvp1.Status)
	}
	if got.Group != "" {
		t.Errorf("group = %q, want empty (no approvals yet)", got.Group)
	}
}

// -----------------------------------------------------------------------------
// 2) AI Studio·Claude·ChatGPT·Gemini 등 deny list 는 자동 detect 제외
// -----------------------------------------------------------------------------
func TestMilestone_FilterOutPracticeDomains(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "ms-deny@test.com", "deny", "20251002", "ms_deny_co")

	updateCompanyServiceURL(t, ts, token, cid,
		"https://aistudio.google.com/prompts/123,https://claude.ai/chat/abc,https://chatgpt.com/c/x")

	got := getMyMilestones(t, ts, token)
	if mvp1 := findMilestone(t, got, "mvp1"); mvp1 != nil {
		t.Errorf("mvp1 should not be auto-detected (all URLs are practice), got %+v", mvp1)
	}
	if mvp2 := findMilestone(t, got, "mvp2"); mvp2 != nil {
		t.Errorf("mvp2 should not be auto-detected, got %+v", mvp2)
	}
}

// -----------------------------------------------------------------------------
// 3) 다중 URL: 첫 = MVP1, 둘 = MVP2 (deny URL 은 카운트 안함)
// -----------------------------------------------------------------------------
func TestMilestone_MultipleURLsBecomeMVP1andMVP2(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "ms-multi@test.com", "multi", "20251003", "ms_multi_co")

	updateCompanyServiceURL(t, ts, token, cid,
		"https://first.vercel.app,https://ai.studio/practice,https://second.vercel.app")

	got := getMyMilestones(t, ts, token)
	mvp1 := findMilestone(t, got, "mvp1")
	mvp2 := findMilestone(t, got, "mvp2")

	if mvp1 == nil || mvp1.URL != "https://first.vercel.app" {
		t.Errorf("mvp1 = %+v, want first.vercel.app", mvp1)
	}
	if mvp2 == nil || mvp2.URL != "https://second.vercel.app" {
		t.Errorf("mvp2 = %+v, want second.vercel.app", mvp2)
	}
}

// -----------------------------------------------------------------------------
// 4) grant 응모 proposal 텍스트 안의 URL 도 자동 detect
// -----------------------------------------------------------------------------
func TestMilestone_ExtractFromGrantProposal(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	studentToken := ts.registerAndApprove("ms-grant@test.com", "pass1234", "grant학생", "20251004")
	// admin 보상 지급 위해 충전.
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 1500000, "description": "init",
	}, adminToken)

	grantID, _ := createGrant(ts, adminToken, nil)

	// proposal 본문 안에 URL 박혀있음 — extract 가 잡아내야 함.
	resp := ts.post(fmt.Sprintf("/api/grants/%d/apply", grantID), map[string]interface{}{
		"proposal": "저희 MVP 는 https://grant-mvp.vercel.app 입니다. ai.studio 는 연습용입니다 https://aistudio.google.com/x .",
	}, studentToken)
	if !resp.Success {
		t.Fatalf("apply grant: %v", resp.Error)
	}

	got := getMyMilestones(t, ts, studentToken)
	mvp1 := findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatalf("expected mvp1 from grant proposal, got none")
	}
	if mvp1.URL != "https://grant-mvp.vercel.app" {
		t.Errorf("mvp1.url = %q, want grant-mvp.vercel.app", mvp1.URL)
	}
	if mvp1.SourceType != "grant" {
		t.Errorf("mvp1.source_type = %q, want grant", mvp1.SourceType)
	}
}

// -----------------------------------------------------------------------------
// 5) 학생이 사업계획서·회고 발표 를 수동 제출
// -----------------------------------------------------------------------------
func TestMilestone_SubmitBusinessPlanAndRetrospective(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("ms-plan@test.com", "pass1234", "plan학생", "20251005")

	r1 := ts.post("/api/milestones", map[string]string{
		"type": "business_plan", "content": "저희는 ~~을 합니다 (10장 분량 첨부)",
	}, token)
	if !r1.Success {
		t.Fatalf("submit business_plan: %v", r1.Error)
	}

	r2 := ts.post("/api/milestones", map[string]string{
		"type": "retrospective", "content": "한 학기 배운 점 정리",
	}, token)
	if !r2.Success {
		t.Fatalf("submit retrospective: %v", r2.Error)
	}

	got := getMyMilestones(t, ts, token)
	if bp := findMilestone(t, got, "business_plan"); bp == nil || bp.Status != "pending" {
		t.Errorf("business_plan not present or status wrong: %+v", bp)
	}
	if rs := findMilestone(t, got, "retrospective"); rs == nil || rs.Status != "pending" {
		t.Errorf("retrospective not present or status wrong: %+v", rs)
	}
}

// -----------------------------------------------------------------------------
// 6) MVP 수동 제출 — deny list URL 은 400 거절
// -----------------------------------------------------------------------------
func TestMilestone_SubmitManualMVP_DenyListRejected(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("ms-manual@test.com", "pass1234", "manual학생", "20251006")

	r := ts.post("/api/milestones", map[string]string{
		"type": "mvp1", "url": "https://aistudio.google.com/x",
	}, token)
	if r.Success {
		t.Fatalf("expected deny-list URL to be rejected, got success: %s", string(r.Data))
	}
}

// -----------------------------------------------------------------------------
// 7) admin 승인/반려 흐름 + 그룹 분류
// -----------------------------------------------------------------------------
func TestMilestone_AdminApproveAndGroupClassification(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	token, cid := createUserWithCompany(t, ts, "ms-approve@test.com", "승인학생", "20251007", "ms_approve_co")

	// MVP1, MVP2 자동 detect 되게 두 URL 등록.
	updateCompanyServiceURL(t, ts, token, cid,
		"https://app1.vercel.app,https://app2.vercel.app")
	// 사업계획서 수동 제출.
	ts.post("/api/milestones", map[string]string{"type": "business_plan", "content": "사업계획서"}, token)
	// 회고는 미제출.

	got := getMyMilestones(t, ts, token)
	if got.ApprovedCount != 0 || got.Group != "" {
		t.Errorf("initial: approved_count=%d group=%q, want 0/empty", got.ApprovedCount, got.Group)
	}

	// admin 이 모든 평가지표 매트릭스 조회.
	listResp := ts.get("/api/admin/milestones", adminToken)
	if !listResp.Success {
		t.Fatalf("admin list: %v", listResp.Error)
	}

	// 학생의 3개 (mvp1, mvp2, business_plan) 승인.
	mvp1 := findMilestone(t, got, "mvp1")
	mvp2 := findMilestone(t, got, "mvp2")
	bp := findMilestone(t, got, "business_plan")
	if mvp1 == nil || mvp2 == nil || bp == nil {
		t.Fatalf("missing milestones: mvp1=%v mvp2=%v bp=%v", mvp1, mvp2, bp)
	}

	for _, id := range []int{mvp1.ID, mvp2.ID, bp.ID} {
		r := ts.post(fmt.Sprintf("/api/admin/milestones/%d/approve", id),
			map[string]string{"admin_note": "OK"}, adminToken)
		if !r.Success {
			t.Fatalf("approve %d: %v", id, r.Error)
		}
	}

	// 다시 조회 — 3개 approved → B 그룹.
	got = getMyMilestones(t, ts, token)
	if got.ApprovedCount != 3 {
		t.Errorf("approved_count = %d, want 3", got.ApprovedCount)
	}
	if got.Group != "B" {
		t.Errorf("group = %q, want B (3 approved)", got.Group)
	}

	// 회고도 제출 + 승인 → 4 approved → A 그룹.
	rsResp := ts.post("/api/milestones", map[string]string{
		"type": "retrospective", "content": "회고",
	}, token)
	var rsData struct {
		ID int `json:"id"`
	}
	json.Unmarshal(rsResp.Data, &rsData)
	ts.post(fmt.Sprintf("/api/admin/milestones/%d/approve", rsData.ID),
		map[string]string{"admin_note": "OK"}, adminToken)

	got = getMyMilestones(t, ts, token)
	if got.ApprovedCount != 4 {
		t.Errorf("approved_count = %d, want 4", got.ApprovedCount)
	}
	if got.Group != "A" {
		t.Errorf("group = %q, want A", got.Group)
	}

	// admin 이 mvp1 반려 → 3 approved → B 그룹 복귀.
	rejResp := ts.post(fmt.Sprintf("/api/admin/milestones/%d/reject", mvp1.ID),
		map[string]string{"admin_note": "다시 작업해주세요"}, adminToken)
	if !rejResp.Success {
		t.Fatalf("reject mvp1: %v", rejResp.Error)
	}

	got = getMyMilestones(t, ts, token)
	if got.ApprovedCount != 3 {
		t.Errorf("after reject: approved_count = %d, want 3", got.ApprovedCount)
	}
	if got.Group != "B" {
		t.Errorf("after reject: group = %q, want B", got.Group)
	}
}

// -----------------------------------------------------------------------------
// #120 — 회고 에세이 AI 작성 확률 평가
// -----------------------------------------------------------------------------

// fakeChatLLM — milestone usecase 에 주입할 fake ChatLLMClient.
// "AI" 가 들어간 텍스트는 score 90, 그 외는 10 으로 응답.
type fakeChatLLM struct {
	lastUser string
}

func (f *fakeChatLLM) Stats() application.LLMStats { return application.LLMStats{} }

func (f *fakeChatLLM) ChatComplete(_ context.Context, req *application.LLMChatRequest) (*application.LLMChatResponse, error) {
	user := ""
	for _, m := range req.Messages {
		if m.Role == "user" {
			user += m.Content
		}
	}
	f.lastUser = user
	score := 10
	reason := "1인칭과 구체적 경험이 풍부함"
	if strings.Contains(user, "FAKE_AI_MARK") {
		score = 90
		reason = "AI 특유 구문 다수, 추상적 일반론 위주"
	}
	js := fmt.Sprintf(`{"score": %d, "reasoning": "%s"}`, score, reason)
	return &application.LLMChatResponse{
		Choices: []application.LLMChatChoice{{
			Message: application.LLMChatMessage{Role: "assistant", Content: js},
		}},
	}, nil
}

func (f *fakeChatLLM) ChatCompleteStream(_ context.Context, _ *application.LLMChatRequest) (<-chan application.LLMStreamEvent, error) {
	ch := make(chan application.LLMStreamEvent)
	close(ch)
	return ch, nil
}

func TestMilestone_EssayScore_SelfCheck(t *testing.T) {
	ts := setupTestServer(t)
	// LLM 주입 — fake 가 deterministic 응답
	ts.injectMilestoneFakeLLM()

	token := ts.registerAndApprove("ms-essay@test.com", "pass1234", "에세이", "20251010")

	// 사람 풍 글 — fake LLM 은 score 10 반환 (FAKE_AI_MARK 없음)
	humanEssay := strings.Repeat("내가 이번 학기에 정말 힘들었던 건 8주차 즈음이었다. 팀원과 다투고 새벽까지 카톡으로 말다툼했다. ", 5)
	r := ts.post("/api/milestones/essay/score", map[string]string{"text": humanEssay}, token)
	if !r.Success {
		t.Fatalf("score essay: %v", r.Error)
	}
	var got struct {
		HeuristicScore int    `json:"heuristic_score"`
		LLMScore       int    `json:"llm_score"`
		CombinedScore  int    `json:"combined_score"`
		LLMReasoning   string `json:"llm_reasoning"`
	}
	json.Unmarshal(r.Data, &got)
	if got.LLMScore != 10 {
		t.Errorf("human essay LLM score = %d, want 10 (fake)", got.LLMScore)
	}
	if got.CombinedScore > 50 {
		t.Errorf("human essay combined score = %d, want low", got.CombinedScore)
	}

	// AI 마크 포함 — fake 가 score 90 반환
	aiEssay := strings.Repeat("이번 학기를 통해 다양한 측면에서 발전시킬 수 있었다. 결론적으로 매우 의미 있는 시간이었다. FAKE_AI_MARK ", 5)
	r2 := ts.post("/api/milestones/essay/score", map[string]string{"text": aiEssay}, token)
	if !r2.Success {
		t.Fatalf("score AI essay: %v", r2.Error)
	}
	var got2 struct {
		LLMScore      int `json:"llm_score"`
		CombinedScore int `json:"combined_score"`
	}
	json.Unmarshal(r2.Data, &got2)
	if got2.LLMScore != 90 {
		t.Errorf("AI essay LLM score = %d, want 90 (fake)", got2.LLMScore)
	}
	if got2.CombinedScore < 50 {
		t.Errorf("AI essay combined score = %d, want >= 50", got2.CombinedScore)
	}
}

func TestMilestone_EssayScore_TooShort(t *testing.T) {
	ts := setupTestServer(t)
	ts.injectMilestoneFakeLLM()
	token := ts.registerAndApprove("ms-essay-short@test.com", "pass1234", "짧", "20251011")

	r := ts.post("/api/milestones/essay/score", map[string]string{"text": "너무 짧은 글"}, token)
	if r.Success {
		t.Errorf("expected error for too-short essay, got success")
	}
}

func TestMilestone_SubmitRetrospective_AutoScored(t *testing.T) {
	ts := setupTestServer(t)
	ts.injectMilestoneFakeLLM()
	token := ts.registerAndApprove("ms-retro@test.com", "pass1234", "회고", "20251012")

	// 회고 제출 (AI 마크 포함 — fake 가 high score 반환)
	essay := strings.Repeat("이번 학기를 통해 다양한 측면에서 발전시킬 수 있었다. FAKE_AI_MARK 결론적으로 ", 8)
	r := ts.post("/api/milestones", map[string]string{
		"type": "retrospective", "content": essay,
	}, token)
	if !r.Success {
		t.Fatalf("submit retrospective: %v", r.Error)
	}

	// /milestones/mine 으로 다시 가져왔을 때 ai_score 가 저장돼있어야 함
	got := getMyMilestones(t, ts, token)
	retro := findMilestone(t, got, "retrospective")
	if retro == nil {
		t.Fatal("retrospective milestone not found")
	}
	// 응답 struct 에는 ai_score 가 없으니 raw 응답 한번 더 확인
	r2 := ts.get("/api/milestones/mine", token)
	var raw map[string]any
	json.Unmarshal(r2.Data, &raw)
	milestones := raw["milestones"].([]any)
	var found bool
	for _, m := range milestones {
		if m == nil {
			continue
		}
		mm := m.(map[string]any)
		if mm["type"] == "retrospective" {
			ai, ok := mm["ai_score"].(float64)
			if !ok {
				t.Fatalf("ai_score not present or wrong type: %v", mm["ai_score"])
			}
			if int(ai) < 50 {
				t.Errorf("ai_score = %d, expected >= 50 for AI-marked essay", int(ai))
			}
			found = true
		}
	}
	if !found {
		t.Fatal("retrospective not in mine response")
	}
}

// -----------------------------------------------------------------------------
// 8) approved milestone 은 회사 service_url 이 바뀌어도 그대로 (admin 결정 보호)
// -----------------------------------------------------------------------------
func TestMilestone_ApprovedNotOverwrittenOnResync(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	token, cid := createUserWithCompany(t, ts, "ms-protect@test.com", "보호", "20251008", "ms_protect_co")

	updateCompanyServiceURL(t, ts, token, cid, "https://protect-v1.vercel.app")
	got := getMyMilestones(t, ts, token)
	mvp1 := findMilestone(t, got, "mvp1")
	if mvp1 == nil {
		t.Fatal("mvp1 not detected")
	}

	// admin 승인.
	ts.post(fmt.Sprintf("/api/admin/milestones/%d/approve", mvp1.ID),
		map[string]string{"admin_note": "v1 OK"}, adminToken)

	// 학생이 회사 URL 을 다른 걸로 교체.
	updateCompanyServiceURL(t, ts, token, cid, "https://protect-v2.vercel.app")

	got = getMyMilestones(t, ts, token)
	mvp1After := findMilestone(t, got, "mvp1")
	if mvp1After == nil {
		t.Fatal("mvp1 disappeared")
	}
	if mvp1After.Status != "approved" {
		t.Errorf("mvp1 status = %q, want approved (must not be reset)", mvp1After.Status)
	}
	if mvp1After.URL != "https://protect-v1.vercel.app" {
		t.Errorf("mvp1 url = %q, want original protect-v1 (approved URL must not be overwritten)", mvp1After.URL)
	}
}
