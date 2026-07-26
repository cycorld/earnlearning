package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestAdminApproveCreatesNotification — 관리자가 가입을 승인하면
// 해당 사용자에게 user_approved 알림이 생성되어야 한다 (#167).
func TestAdminApproveCreatesNotification(t *testing.T) {
	ts := setupTestServer(t)

	// 신규(pending) 사용자 등록
	regResp := ts.register("approve-notif@ewha.ac.kr", "password123", "승인테스트", "2024600")
	if !regResp.Success {
		t.Fatalf("register failed: %v", regResp.Error)
	}
	var regData struct {
		User struct {
			ID int `json:"id"`
		} `json:"user"`
	}
	json.Unmarshal(regResp.Data, &regData)
	userID := regData.User.ID

	// 관리자 승인
	adminToken := ts.login(testAdminEmail, testAdminPass)
	if appr := ts.approveUser(adminToken, userID); !appr.Success {
		t.Fatalf("approve failed: %v", appr.Error)
	}

	// 승인된 사용자로 로그인 후 알림 조회
	userToken := ts.login("approve-notif@ewha.ac.kr", "password123")
	notifsResp := ts.get("/api/notifications?limit=10", userToken)
	if !notifsResp.Success {
		t.Fatalf("get notifications failed: %v", notifsResp.Error)
	}

	var notifsData struct {
		Data []struct {
			NotifType     string `json:"notif_type"`
			Title         string `json:"title"`
			ReferenceType string `json:"reference_type"`
			ReferenceID   int    `json:"reference_id"`
		} `json:"data"`
	}
	json.Unmarshal(notifsResp.Data, &notifsData)

	found := false
	for _, n := range notifsData.Data {
		if n.NotifType == "user_approved" {
			found = true
			if n.ReferenceType != "user" {
				t.Errorf("expected reference_type 'user', got '%s'", n.ReferenceType)
			}
			if n.ReferenceID != userID {
				t.Errorf("expected reference_id %d, got %d", userID, n.ReferenceID)
			}
			if n.Title == "" {
				t.Error("expected non-empty title")
			}
		}
	}
	if !found {
		t.Error("가입 승인 알림(user_approved)이 생성되지 않았습니다")
	}

	// 이미 승인된 사용자를 다시 승인해도 크래시 없이 동작해야 한다
	// (기존 idempotency 동작은 바꾸지 않는다 — 알림이 하나 더 생길 수 있음).
	if appr2 := ts.approveUser(adminToken, userID); !appr2.Success {
		t.Errorf("re-approve should not crash: %v", appr2.Error)
	}
}

// TestRefreshReflectsApproval — /auth/refresh 가 최신 DB 상태를 반영해
// 오래된 pending 토큰으로도 승인 후 approved 토큰을 발급받아
// approved 전용 엔드포인트를 사용할 수 있어야 한다 (#167 핵심 프리미티브 회귀 고정).
func TestRefreshReflectsApproval(t *testing.T) {
	ts := setupTestServer(t)

	// 등록 — 응답에 pending 토큰이 담겨온다
	regResp := ts.register("refresh-approval@ewha.ac.kr", "password123", "리프레시", "2024601")
	if !regResp.Success {
		t.Fatalf("register failed: %v", regResp.Error)
	}
	var regData struct {
		Token string `json:"token"`
		User  struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"user"`
	}
	json.Unmarshal(regResp.Data, &regData)
	pendingToken := regData.Token
	userID := regData.User.ID

	if regData.User.Status != "pending" {
		t.Fatalf("expected pending after register, got %s", regData.User.Status)
	}

	// 승인 전 refresh → 여전히 pending
	r := ts.post("/api/auth/refresh", nil, pendingToken)
	if !r.Success {
		t.Fatalf("refresh with pending token should succeed: %v", r.Error)
	}
	var refreshBefore struct {
		User struct {
			Status string `json:"status"`
		} `json:"user"`
	}
	json.Unmarshal(r.Data, &refreshBefore)
	if refreshBefore.User.Status != "pending" {
		t.Errorf("expected still pending before approval, got %s", refreshBefore.User.Status)
	}

	// pending 토큰은 approved 전용 엔드포인트에서 403 으로 막혀야 한다
	if st, _ := ts.rawGet("/api/wallet", pendingToken); st != http.StatusForbidden {
		t.Fatalf("pending token should be 403 on /wallet, got %d", st)
	}

	// 관리자 승인
	adminToken := ts.login(testAdminEmail, testAdminPass)
	if appr := ts.approveUser(adminToken, userID); !appr.Success {
		t.Fatalf("approve failed: %v", appr.Error)
	}

	// 오래된 pending 토큰으로 refresh → approved 상태 + 새 토큰
	r2 := ts.post("/api/auth/refresh", nil, pendingToken)
	if !r2.Success {
		t.Fatalf("refresh after approval should succeed: %v", r2.Error)
	}
	var refreshAfter struct {
		Token string `json:"token"`
		User  struct {
			Status string `json:"status"`
		} `json:"user"`
	}
	json.Unmarshal(r2.Data, &refreshAfter)
	if refreshAfter.User.Status != "approved" {
		t.Errorf("expected approved after refresh, got %s", refreshAfter.User.Status)
	}
	if refreshAfter.Token == "" {
		t.Fatal("expected a fresh token after approval")
	}

	// 새 토큰은 approved 전용 엔드포인트에서 403 이 아니어야 한다 (상태 게이트 통과)
	if st, body := ts.rawGet("/api/wallet", refreshAfter.Token); st == http.StatusForbidden {
		t.Fatalf("refreshed token should NOT be 403 on /wallet, got %d body=%s", st, string(body))
	}
}
