package integration

import (
	"encoding/json"
	"testing"
)

func TestDisclosure_CreateAndApprove_Flow(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "disc@test.com", "disc", "20240030", "disc_co")

	// 1. 공시 작성
	r := ts.post("/api/companies/"+itoaUD(cid)+"/disclosures", map[string]string{
		"content":     "이번 주 MVP 개발 완료. 사용자 10명 확보.",
		"period_from": "2026-04-07",
		"period_to":   "2026-04-11",
	}, token)
	if !r.Success {
		t.Fatalf("create disclosure: %v", r.Error)
	}
	var created struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}
	_ = json.Unmarshal(r.Data, &created)
	if created.Status != "pending" {
		t.Errorf("expected pending status, got %q", created.Status)
	}

	// 2. 공시 목록 조회
	g := ts.get("/api/companies/"+itoaUD(cid)+"/disclosures", token)
	if !g.Success {
		t.Fatalf("get disclosures: %v", g.Error)
	}
	var discList []map[string]interface{}
	_ = json.Unmarshal(g.Data, &discList)
	if len(discList) != 1 {
		t.Fatalf("expected 1 disclosure, got %d", len(discList))
	}

	// 3. 관리자가 승인 + 수익금 입금
	adminToken := ts.login(testAdminEmail, testAdminPass)
	ar := ts.post("/api/admin/disclosures/"+itoaUD(created.ID)+"/approve", map[string]interface{}{
		"reward":     500000,
		"admin_note": "좋은 성과입니다!",
	}, adminToken)
	if !ar.Success {
		t.Fatalf("approve disclosure: %v", ar.Error)
	}

	// 4. 다시 조회 → approved, reward 확인
	g2 := ts.get("/api/companies/"+itoaUD(cid)+"/disclosures", token)
	var discList2 []struct {
		Status string `json:"status"`
		Reward int    `json:"reward"`
	}
	_ = json.Unmarshal(g2.Data, &discList2)
	if discList2[0].Status != "approved" {
		t.Errorf("expected approved, got %q", discList2[0].Status)
	}
	if discList2[0].Reward != 500000 {
		t.Errorf("expected reward 500000, got %d", discList2[0].Reward)
	}
}

func TestDisclosure_Reject_Flow(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "rej@test.com", "rej", "20240031", "rej_co")

	// 공시 작성
	r := ts.post("/api/companies/"+itoaUD(cid)+"/disclosures", map[string]string{
		"content":     "이번 주 별 성과 없음",
		"period_from": "2026-04-07",
		"period_to":   "2026-04-11",
	}, token)
	if !r.Success {
		t.Fatalf("create disclosure: %v", r.Error)
	}
	var created struct{ ID int `json:"id"` }
	_ = json.Unmarshal(r.Data, &created)

	// 관리자 거절
	adminToken := ts.login(testAdminEmail, testAdminPass)
	rr := ts.post("/api/admin/disclosures/"+itoaUD(created.ID)+"/reject", map[string]string{
		"admin_note": "내용이 부족합니다. 구체적인 지표를 포함해주세요.",
	}, adminToken)
	if !rr.Success {
		t.Fatalf("reject disclosure: %v", rr.Error)
	}

	// 다시 조회 → rejected
	g := ts.get("/api/companies/"+itoaUD(cid)+"/disclosures", token)
	var discList []struct {
		Status    string `json:"status"`
		AdminNote string `json:"admin_note"`
	}
	_ = json.Unmarshal(g.Data, &discList)
	if discList[0].Status != "rejected" {
		t.Errorf("expected rejected, got %q", discList[0].Status)
	}
	if discList[0].AdminNote != "내용이 부족합니다. 구체적인 지표를 포함해주세요." {
		t.Errorf("admin_note mismatch: got %q", discList[0].AdminNote)
	}
}

func TestDisclosure_NotOwner_Forbidden(t *testing.T) {
	ts := setupTestServer(t)
	_, cid := createUserWithCompany(t, ts, "own@test.com", "own", "20240032", "own_co")
	otherToken, _ := createUserWithCompany(t, ts, "oth@test.com", "oth", "20240033", "oth_co")

	// 타인이 다른 회사에 공시 작성 시도
	r := ts.post("/api/companies/"+itoaUD(cid)+"/disclosures", map[string]string{
		"content":     "남의 회사 공시",
		"period_from": "2026-04-07",
		"period_to":   "2026-04-11",
	}, otherToken)
	if r.Success {
		t.Fatal("should fail with NOT_OWNER")
	}
	if r.Error == nil || r.Error.Code != "NOT_OWNER" {
		t.Errorf("expected NOT_OWNER, got %v", r.Error)
	}
}

func TestDisclosure_AdminListAll(t *testing.T) {
	ts := setupTestServer(t)
	token1, cid1 := createUserWithCompany(t, ts, "d1@test.com", "d1", "20240034", "d1_co")
	token2, cid2 := createUserWithCompany(t, ts, "d2@test.com", "d2", "20240035", "d2_co")

	// 각 회사에서 공시 1개씩 작성
	ts.post("/api/companies/"+itoaUD(cid1)+"/disclosures", map[string]string{
		"content": "d1 공시", "period_from": "2026-04-07", "period_to": "2026-04-11",
	}, token1)
	ts.post("/api/companies/"+itoaUD(cid2)+"/disclosures", map[string]string{
		"content": "d2 공시", "period_from": "2026-04-07", "period_to": "2026-04-11",
	}, token2)

	// 관리자 전체 조회
	adminToken := ts.login(testAdminEmail, testAdminPass)
	r := ts.get("/api/admin/disclosures", adminToken)
	if !r.Success {
		t.Fatalf("admin list disclosures: %v", r.Error)
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(r.Data, &list)
	if len(list) < 2 {
		t.Errorf("expected at least 2 disclosures, got %d", len(list))
	}
}
