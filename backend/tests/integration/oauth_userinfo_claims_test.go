package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/earnlearning/backend/internal/domain/user"
)

// #181: /api/oauth/userinfo 는 Rybbit SSO 가 캠프 자격을 fail-closed 로 판정할 근거를 함께 내려준다.
// (승인 상태 + 활성 강의실 소속) 둘 다 만족해야 camp_eligible = true.

type userInfoClaims struct {
	ID                int    `json:"id"`
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	Department        string `json:"department"`
	Bio               string `json:"bio"`
	AvatarURL         string `json:"avatar_url"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	Active            bool   `json:"active"`
	Approved          bool   `json:"approved"`
	ActiveClassroomID int    `json:"active_classroom_id"`
	CampEligible      bool   `json:"camp_eligible"`
}

// oauthAccessTokenFor — PKCE 전체 흐름으로 사용자의 access token 을 얻는다.
func oauthAccessTokenFor(t *testing.T, ts *testServer, userToken, appName, verifier string) string {
	t.Helper()

	regResp := ts.post("/api/oauth/clients", map[string]interface{}{
		"name":          appName,
		"description":   "#181 캠프 자격 클레임 테스트",
		"redirect_uris": []string{"http://localhost:3000/callback"},
		"scopes":        []string{"read:profile"},
	}, userToken)
	if !regResp.Success {
		t.Fatalf("client registration failed: %v", regResp.Error)
	}
	var client struct {
		ClientID string `json:"client_id"`
	}
	json.Unmarshal(regResp.Data, &client)

	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	authResp := ts.post("/api/oauth/authorize", map[string]interface{}{
		"client_id":             client.ClientID,
		"redirect_uri":          "http://localhost:3000/callback",
		"scopes":                []string{"read:profile"},
		"state":                 "camp-eligibility",
		"code_challenge":        challenge,
		"code_challenge_method": "S256",
	}, userToken)
	if !authResp.Success {
		t.Fatalf("authorize failed: %v", authResp.Error)
	}
	var authData struct {
		Code string `json:"code"`
	}
	json.Unmarshal(authResp.Data, &authData)

	tokenResp := ts.post("/api/oauth/token", map[string]interface{}{
		"grant_type":    "authorization_code",
		"code":          authData.Code,
		"client_id":     client.ClientID,
		"redirect_uri":  "http://localhost:3000/callback",
		"code_verifier": verifier,
	}, "")
	if !tokenResp.Success {
		t.Fatalf("token exchange failed: %v", tokenResp.Error)
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(tokenResp.Data, &tokens)
	if tokens.AccessToken == "" {
		t.Fatalf("access token 이 비었음: %s", string(tokenResp.Data))
	}
	return tokens.AccessToken
}

func fetchUserInfo(t *testing.T, ts *testServer, accessToken string) userInfoClaims {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.server.URL+"/api/oauth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	raw, err := ts.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Body.Close()
	body, _ := io.ReadAll(raw.Body)
	var info userInfoClaims
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("userinfo decode: %v", err)
	}
	return info
}

func TestOAuthUserInfo_CampEligibilityClaims(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	cr := ts.createClassroom(adminToken, "캠프자격반", 1_000_000)
	email := "camp-user@test.com"
	userToken := ts.registerAndApprove(email, "pass1234", "캠프학생", "20261101")
	if r := ts.joinClassroom(userToken, cr.Code); !r.Success {
		t.Fatalf("join classroom: %v", r.Error)
	}

	var userID int
	if err := ts.db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID); err != nil {
		t.Fatalf("find user id: %v", err)
	}

	accessToken := oauthAccessTokenFor(t, ts, userToken, "캠프자격테스트앱",
		"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk-camp-181")

	// 승인 + 활성 강의실 → 캠프 자격 있음
	info := fetchUserInfo(t, ts, accessToken)
	if info.ID != userID || info.Email != email || info.Name != "캠프학생" {
		t.Errorf("기존 필드가 바뀜: %+v", info)
	}
	// #181 Rybbit SSO 필수 클레임: sub 는 안정 문자열(user.OAuthSubject), active 는 승인 여부.
	if info.Sub != user.OAuthSubject(userID) || info.Sub == "" {
		t.Errorf("sub=%q, want %q (OAuthSubject)", info.Sub, user.OAuthSubject(userID))
	}
	if !info.Active {
		t.Errorf("승인 상태인데 active=false: %+v", info)
	}
	if info.Role == "" {
		t.Errorf("role 이 비었음: %+v", info)
	}
	if info.Status != "approved" || !info.Approved {
		t.Errorf("status=%q approved=%v, want approved/true", info.Status, info.Approved)
	}
	if info.ActiveClassroomID != cr.ID {
		t.Errorf("active_classroom_id %d, want %d", info.ActiveClassroomID, cr.ID)
	}
	if !info.CampEligible {
		t.Errorf("camp_eligible 가 false: %+v", info)
	}

	// 강의실에서 빠지면 자격 상실 (fail-closed)
	if _, err := ts.db.Exec("UPDATE users SET active_classroom_id = 0 WHERE id = ?", userID); err != nil {
		t.Fatalf("활성 강의실 해제: %v", err)
	}
	info = fetchUserInfo(t, ts, accessToken)
	if info.ActiveClassroomID != 0 {
		t.Errorf("active_classroom_id %d, want 0", info.ActiveClassroomID)
	}
	if !info.Approved {
		t.Errorf("강의실만 빠졌는데 approved 가 false: %+v", info)
	}
	if info.CampEligible {
		t.Errorf("강의실 무소속인데 camp_eligible 가 true: %+v", info)
	}

	// 승인 상태가 내려가도 자격 상실
	if _, err := ts.db.Exec("UPDATE users SET status = 'pending', active_classroom_id = ? WHERE id = ?", cr.ID, userID); err != nil {
		t.Fatalf("승인 상태 변경: %v", err)
	}
	info = fetchUserInfo(t, ts, accessToken)
	if info.Status != "pending" || info.Approved {
		t.Errorf("status=%q approved=%v, want pending/false", info.Status, info.Approved)
	}
	if info.Active {
		t.Errorf("미승인(비활성)인데 active=true: %+v", info)
	}
	if info.CampEligible {
		t.Errorf("미승인인데 camp_eligible 가 true: %+v", info)
	}
	// sub 는 상태와 무관하게 안정적이어야 한다 (Rybbit 계정 연결 키).
	if info.Sub != user.OAuthSubject(userID) {
		t.Errorf("상태 변경 후 sub 가 바뀜: %q", info.Sub)
	}
}

// TestOAuthUserInfo_AdminCampEligible — 승인된 수업 관리자는 활성 강의실이 있으면
// Rybbit에 로그인해 자기 회사 사이트를 운영할 수 있다.
func TestOAuthUserInfo_AdminCampEligible(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	cr := ts.createClassroom(adminToken, "관리자자격반", 1_000_000)
	var adminID int
	if err := ts.db.QueryRow("SELECT id FROM users WHERE email = ?", testAdminEmail).Scan(&adminID); err != nil {
		t.Fatalf("find admin id: %v", err)
	}
	// 강의실 소속까지 만들어 역할 게이트만 남긴다.
	if _, err := ts.db.Exec("UPDATE users SET active_classroom_id = ? WHERE id = ?", cr.ID, adminID); err != nil {
		t.Fatalf("admin 강의실 부여: %v", err)
	}

	accessToken := oauthAccessTokenFor(t, ts, adminToken, "관리자자격테스트앱",
		"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk-admin-181")
	info := fetchUserInfo(t, ts, accessToken)

	if info.Role != "admin" || !info.Approved || info.ActiveClassroomID != cr.ID {
		t.Fatalf("전제 조건이 다름 (role/approved/classroom): %+v", info)
	}
	if !info.Active {
		t.Errorf("승인된 admin 계정인데 active=false: %+v", info)
	}
	if info.Sub != user.OAuthSubject(adminID) {
		t.Errorf("admin sub=%q, want %q", info.Sub, user.OAuthSubject(adminID))
	}
	if !info.CampEligible {
		t.Errorf("승인·강의실 소속 admin 인데 camp_eligible=false: %+v", info)
	}
}
