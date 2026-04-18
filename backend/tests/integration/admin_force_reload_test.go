package integration

import (
	"testing"
)

// #027: 관리자가 POST /api/admin/force-reload 로 전체 클라이언트 강제 새로고침을 트리거한다.
// 스모크 수준 테스트 — WS 브로드캐스트 자체 확인은 통합하기 어려우므로 HTTP API 계약만 검증.

func TestAdminForceReload_NonAdminIsForbidden(t *testing.T) {
	ts := setupTestServer(t)
	userToken := ts.registerAndApprove("frstu@test.com", "pass1234", "student", "20240500")

	r := ts.post("/api/admin/force-reload", map[string]string{"reason": "test"}, userToken)
	if r.Success {
		t.Fatal("non-admin should be forbidden from force-reload")
	}
}

func TestAdminForceReload_AdminSucceeds(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	r := ts.post("/api/admin/force-reload", map[string]string{"reason": "청산 기능 롤아웃"}, adminToken)
	if !r.Success {
		t.Fatalf("admin force-reload should succeed: %v", r.Error)
	}
}

// #027 회귀: 1분 간격 rate limit. 연속 호출 시 429 로 막혀야 한다.
func TestAdminForceReload_RateLimitedOnSecondCall(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// 1회차 성공
	r1 := ts.post("/api/admin/force-reload", map[string]string{"reason": "first"}, adminToken)
	if !r1.Success {
		t.Fatalf("first call should succeed: %v", r1.Error)
	}

	// 2회차 즉시 → rate limited
	r2 := ts.post("/api/admin/force-reload", map[string]string{"reason": "second"}, adminToken)
	if r2.Success {
		t.Fatal("second call within 1 minute should be rate-limited")
	}
	if r2.Error == nil || r2.Error.Code != "RATE_LIMITED" {
		t.Errorf("expected RATE_LIMITED error, got %+v", r2.Error)
	}
}
