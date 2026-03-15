package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// =============================================================================
// 정부과제 (Government Grant) 유저스토리 통합 테스트
//
// 관리자만 정부과제를 등록할 수 있고, 누구나 지원할 수 있으며,
// 관리자의 승인이 떨어지면 정해진 보상을 수령한다.
// =============================================================================

func createGrant(ts *testServer, token string, opts map[string]interface{}) (int, map[string]interface{}) {
	ts.t.Helper()
	defaults := map[string]interface{}{
		"title": "테스트 과제", "description": "설명",
		"reward": 1000, "max_applicants": 0,
	}
	for k, v := range opts {
		defaults[k] = v
	}
	resp := ts.post("/api/admin/grants", defaults, token)
	if !resp.Success {
		ts.t.Fatalf("createGrant failed: %v", resp.Error)
	}
	var grant map[string]interface{}
	json.Unmarshal(resp.Data, &grant)
	return int(grant["id"].(float64)), grant
}

func applyGrant(ts *testServer, token string, grantID int) (int, *apiResponse) {
	ts.t.Helper()
	resp := ts.post(fmt.Sprintf("/api/grants/%d/apply", grantID), map[string]interface{}{
		"proposal": "지원합니다",
	}, token)
	if !resp.Success {
		return 0, resp
	}
	var app map[string]interface{}
	json.Unmarshal(resp.Data, &app)
	return int(app["id"].(float64)), resp
}

func getGrant(ts *testServer, token string, grantID int) map[string]interface{} {
	ts.t.Helper()
	resp := ts.get(fmt.Sprintf("/api/grants/%d", grantID), token)
	var grant map[string]interface{}
	json.Unmarshal(resp.Data, &grant)
	return grant
}

func listGrants(ts *testServer, token string, query string) []map[string]interface{} {
	ts.t.Helper()
	resp := ts.get("/api/grants?"+query, token)
	var result map[string]interface{}
	json.Unmarshal(resp.Data, &result)
	data, ok := result["data"].([]interface{})
	if !ok || data == nil {
		return []map[string]interface{}{}
	}
	grants := make([]map[string]interface{}, len(data))
	for i, d := range data {
		grants[i] = d.(map[string]interface{})
	}
	return grants
}

// =============================================================================
// 유저스토리 1: 관리자가 정부과제를 등록한다
// =============================================================================
func TestGrantUserStory_AdminCreateGrant(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)
	studentToken := ts.registerAndApprove("grant-stu@test.com", "pass1234", "학생", "2025100")

	t.Run("관리자가 정부과제를 등록한다", func(t *testing.T) {
		grantID, grant := createGrant(ts, adminToken, map[string]interface{}{
			"title": "리액트 스터디", "description": "리액트 기초 학습 과제",
			"reward": 5000, "max_applicants": 10,
		})
		if grantID == 0 {
			t.Fatal("grant ID is 0")
		}
		if grant["status"] != "open" {
			t.Errorf("expected open, got %v", grant["status"])
		}
		if int(grant["reward"].(float64)) != 5000 {
			t.Errorf("expected reward=5000, got %v", grant["reward"])
		}
	})

	t.Run("일반 유저는 정부과제를 등록할 수 없다", func(t *testing.T) {
		resp := ts.post("/api/admin/grants", map[string]interface{}{
			"title": "무단등록", "description": "안됨",
			"reward": 1000,
		}, studentToken)
		if resp.Success {
			t.Error("일반 유저가 과제 등록에 성공하면 안됨")
		}
	})
}

// =============================================================================
// 유저스토리 2: 전체 흐름 — 과제 등록 → 학생 지원 → 관리자 승인 → 보상 지급
// =============================================================================
func TestGrantUserStory_FullFlow(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)
	stu1Token := ts.registerAndApprove("gf-stu1@test.com", "pass1234", "학생1", "2025101")
	stu2Token := ts.registerAndApprove("gf-stu2@test.com", "pass1234", "학생2", "2025102")

	// Give students initial balance
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 10000, "description": "초기자금",
	}, adminToken)

	t.Run("전체 흐름: 등록→지원→승인→보상지급", func(t *testing.T) {
		stu1Before := getBalance(ts, stu1Token)

		// 관리자가 과제 등록
		grantID, _ := createGrant(ts, adminToken, map[string]interface{}{
			"title": "파이썬 과제", "description": "파이썬 기초 실습",
			"reward": 3000,
		})

		// 학생1 지원
		app1ID, resp := applyGrant(ts, stu1Token, grantID)
		if !resp.Success {
			t.Fatalf("student1 apply failed: %v", resp.Error)
		}
		if app1ID == 0 {
			t.Fatal("app ID is 0")
		}

		// 학생2 지원
		_, resp = applyGrant(ts, stu2Token, grantID)
		if !resp.Success {
			t.Fatalf("student2 apply failed: %v", resp.Error)
		}

		// 관리자가 학생1 승인
		resp = ts.post(fmt.Sprintf("/api/admin/grants/%d/approve/%d", grantID, app1ID), nil, adminToken)
		if !resp.Success {
			t.Fatalf("approve failed: %v", resp.Error)
		}

		// 학생1 보상 수령 확인
		stu1After := getBalance(ts, stu1Token)
		if stu1After != stu1Before+3000 {
			t.Errorf("보상 미지급: before=%d, after=%d, expected=%d", stu1Before, stu1After, stu1Before+3000)
		}

		// 과제는 여전히 open (다른 학생도 지원 가능)
		grant := getGrant(ts, adminToken, grantID)
		if grant["status"] != "open" {
			t.Errorf("expected open, got %v", grant["status"])
		}
	})
}

// =============================================================================
// 유저스토리 3: 지원 제한
// =============================================================================
func TestGrantUserStory_ApplyRestrictions(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)
	stu1Token := ts.registerAndApprove("gr-stu1@test.com", "pass1234", "학생1", "2025110")

	grantID, _ := createGrant(ts, adminToken, map[string]interface{}{
		"title": "중복테스트", "reward": 1000,
	})

	t.Run("중복 지원 불가", func(t *testing.T) {
		_, resp := applyGrant(ts, stu1Token, grantID)
		if !resp.Success {
			t.Fatalf("첫 지원 실패: %v", resp.Error)
		}

		_, resp = applyGrant(ts, stu1Token, grantID)
		if resp.Success {
			t.Error("중복 지원이 성공하면 안됨")
		}
	})

	t.Run("관리자도 자기 과제에 지원 불가", func(t *testing.T) {
		_, resp := applyGrant(ts, adminToken, grantID)
		if resp.Success {
			t.Error("관리자가 자기 과제에 지원 성공하면 안됨")
		}
	})
}

// =============================================================================
// 유저스토리 4: 과제 닫기
// =============================================================================
func TestGrantUserStory_CloseGrant(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)
	stuToken := ts.registerAndApprove("gc-stu@test.com", "pass1234", "학생", "2025120")

	grantID, _ := createGrant(ts, adminToken, map[string]interface{}{
		"title": "종료테스트", "reward": 1000,
	})

	// 학생 지원
	applyGrant(ts, stuToken, grantID)

	t.Run("관리자가 과제를 종료한다", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/admin/grants/%d/close", grantID), nil, adminToken)
		if !resp.Success {
			t.Fatalf("close failed: %v", resp.Error)
		}

		grant := getGrant(ts, adminToken, grantID)
		if grant["status"] != "closed" {
			t.Errorf("expected closed, got %v", grant["status"])
		}
	})

	t.Run("종료된 과제에 지원 불가", func(t *testing.T) {
		stu2Token := ts.registerAndApprove("gc-stu2@test.com", "pass1234", "학생2", "2025121")
		_, resp := applyGrant(ts, stu2Token, grantID)
		if resp.Success {
			t.Error("종료된 과제에 지원 성공하면 안됨")
		}
	})

	t.Run("일반 유저는 과제를 종료할 수 없다", func(t *testing.T) {
		grantID2, _ := createGrant(ts, adminToken, map[string]interface{}{
			"title": "종료권한테스트", "reward": 500,
		})
		resp := ts.post(fmt.Sprintf("/api/grants/%d/close", grantID2), nil, stuToken)
		if resp.Success {
			t.Error("일반 유저가 과제 종료에 성공하면 안됨")
		}
	})
}

// =============================================================================
// 유저스토리 5: 목록 조회
// =============================================================================
func TestGrantUserStory_ListGrants(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)
	stuToken := ts.registerAndApprove("gl-stu@test.com", "pass1234", "학생", "2025130")

	createGrant(ts, adminToken, map[string]interface{}{"title": "과제1", "reward": 1000})
	createGrant(ts, adminToken, map[string]interface{}{"title": "과제2", "reward": 2000})
	createGrant(ts, adminToken, map[string]interface{}{"title": "과제3", "reward": 3000})

	t.Run("학생도 과제 목록을 볼 수 있다", func(t *testing.T) {
		grants := listGrants(ts, stuToken, "page=1&limit=10")
		if len(grants) != 3 {
			t.Errorf("expected 3 grants, got %d", len(grants))
		}
	})

	t.Run("상태 필터가 동작한다", func(t *testing.T) {
		openGrants := listGrants(ts, stuToken, "status=open&page=1&limit=10")
		if len(openGrants) != 3 {
			t.Errorf("expected 3 open grants, got %d", len(openGrants))
		}
	})
}
