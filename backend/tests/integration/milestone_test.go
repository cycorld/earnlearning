package integration

import (
	"encoding/json"
	"fmt"
	"testing"
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
