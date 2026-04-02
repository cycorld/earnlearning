package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
)

// TestOAuthE2E mirrors the example app flow end-to-end:
// register client → authorize (PKCE) → exchange code → call APIs → refresh → revoke
func TestOAuthE2E(t *testing.T) {
	ts := setupTestServer(t)

	// Setup two users: one who registers the OAuth app, one who uses it
	appOwnerToken := ts.registerAndApprove("appowner@ewha.ac.kr", "password123", "앱개발자", "2024020")
	apiUserToken := ts.registerAndApprove("apiuser@ewha.ac.kr", "password123", "API사용자", "2024030")

	// ============================================================
	// Step 1: App owner registers an OAuth client
	// ============================================================
	regResp := ts.post("/api/oauth/clients", map[string]interface{}{
		"name":          "E2E테스트앱",
		"description":   "E2E 테스트용 앱",
		"redirect_uris": []string{"http://localhost:3000/callback", "http://localhost:8080/callback"},
		"scopes":        []string{"read:profile", "read:wallet", "write:posts", "read:posts"},
	}, appOwnerToken)
	if !regResp.Success {
		t.Fatalf("client registration failed: %v", regResp.Error)
	}
	var client struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	json.Unmarshal(regResp.Data, &client)

	// ============================================================
	// Step 2: API user authorizes the app (PKCE flow)
	// ============================================================
	// Generate PKCE pair
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk-test-e2e"
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	authResp := ts.post("/api/oauth/authorize", map[string]interface{}{
		"client_id":              client.ClientID,
		"redirect_uri":          "http://localhost:3000/callback",
		"scopes":                []string{"read:profile", "read:wallet", "write:posts", "read:posts"},
		"state":                 "e2e-state-abc",
		"code_challenge":        codeChallenge,
		"code_challenge_method": "S256",
	}, apiUserToken)
	if !authResp.Success {
		t.Fatalf("authorization failed: %v", authResp.Error)
	}
	var authData struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}
	json.Unmarshal(authResp.Data, &authData)

	if authData.State != "e2e-state-abc" {
		t.Errorf("state mismatch: %s", authData.State)
	}

	// ============================================================
	// Step 3: Exchange code for tokens (PKCE, no client_secret)
	// ============================================================
	tokenResp := ts.post("/api/oauth/token", map[string]interface{}{
		"grant_type":    "authorization_code",
		"code":          authData.Code,
		"client_id":     client.ClientID,
		"redirect_uri":  "http://localhost:3000/callback",
		"code_verifier": codeVerifier,
	}, "")
	if !tokenResp.Success {
		t.Fatalf("token exchange failed: %v", tokenResp.Error)
	}
	var tokens struct {
		AccessToken  string   `json:"access_token"`
		RefreshToken string   `json:"refresh_token"`
		TokenType    string   `json:"token_type"`
		ExpiresIn    int      `json:"expires_in"`
		Scopes       []string `json:"scopes"`
	}
	json.Unmarshal(tokenResp.Data, &tokens)

	if tokens.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", tokens.TokenType)
	}
	if len(tokens.Scopes) != 4 {
		t.Errorf("expected 4 scopes, got %d", len(tokens.Scopes))
	}

	// ============================================================
	// Step 4: Call APIs with OAuth token
	// ============================================================

	// 4a. read:profile — Get user info
	t.Run("read:profile - userinfo", func(t *testing.T) {
		resp := ts.get("/api/oauth/userinfo", tokens.AccessToken)
		if !resp.Success {
			t.Fatalf("userinfo failed: %v", resp.Error)
		}
		var info struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		json.Unmarshal(resp.Data, &info)
		if info.Name != "API사용자" {
			t.Errorf("expected 'API사용자', got '%s'", info.Name)
		}
		if info.Email != "apiuser@ewha.ac.kr" {
			t.Errorf("expected 'apiuser@ewha.ac.kr', got '%s'", info.Email)
		}
	})

	// 4b. OAuth token으로 기존 API 접근 (JWT fallback)
	t.Run("OAuth 토큰으로 wallet API 접근", func(t *testing.T) {
		resp := ts.get("/api/wallet", tokens.AccessToken)
		if !resp.Success {
			t.Fatalf("wallet API with OAuth token should succeed: %v", resp.Error)
		}
	})

	t.Run("OAuth 토큰으로 posts API 접근", func(t *testing.T) {
		resp := ts.get("/api/posts?classroom_id=0&page=1&limit=1", tokens.AccessToken)
		if !resp.Success {
			t.Fatalf("posts API with OAuth token should succeed: %v", resp.Error)
		}
	})

	t.Run("OAuth 토큰으로 dm conversations API 접근", func(t *testing.T) {
		resp := ts.get("/api/dm/conversations", tokens.AccessToken)
		if !resp.Success {
			t.Fatalf("dm API with OAuth token should succeed: %v", resp.Error)
		}
	})

	// 4c. 스코프 없는 API는 접근 가능하지만, 데이터는 정상 반환되어야 함
	t.Run("OAuth 토큰으로 자기 프로필 조회", func(t *testing.T) {
		resp := ts.get("/api/auth/me", tokens.AccessToken)
		if !resp.Success {
			t.Fatalf("auth/me with OAuth token should succeed: %v", resp.Error)
		}
	})

	// 4d. 스코프 제한 검증 — read:profile만으로 제한된 토큰 생성
	t.Run("스코프 제한: read:profile 토큰으로 wallet 접근 불가", func(t *testing.T) {
		// 새 인가 (read:profile만)
		v2 := "limited-scope-verifier-1234567890123456789012"
		h2 := sha256.Sum256([]byte(v2))
		ch2 := base64.RawURLEncoding.EncodeToString(h2[:])

		authR := ts.post("/api/oauth/authorize", map[string]interface{}{
			"client_id":              client.ClientID,
			"redirect_uri":          "http://localhost:3000/callback",
			"scopes":                []string{"read:profile"},
			"state":                 "limited",
			"code_challenge":        ch2,
			"code_challenge_method": "S256",
		}, apiUserToken)
		if !authR.Success {
			t.Fatalf("limited auth failed: %v", authR.Error)
		}
		var ad struct{ Code string `json:"code"` }
		json.Unmarshal(authR.Data, &ad)

		tokR := ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type": "authorization_code", "code": ad.Code,
			"client_id": client.ClientID, "redirect_uri": "http://localhost:3000/callback",
			"code_verifier": v2,
		}, "")
		if !tokR.Success {
			t.Fatalf("limited token failed: %v", tokR.Error)
		}
		var lt struct{ AccessToken string `json:"access_token"` }
		json.Unmarshal(tokR.Data, &lt)

		// read:profile → wallet 접근 불가 (403)
		walletR := ts.get("/api/wallet", lt.AccessToken)
		if walletR.Success {
			t.Error("wallet should be forbidden with read:profile only scope")
		}

		// read:profile → posts 접근 불가 (403)
		postsR := ts.get("/api/posts?classroom_id=0&page=1&limit=1", lt.AccessToken)
		if postsR.Success {
			t.Error("posts should be forbidden with read:profile only scope")
		}

		// read:profile → userinfo 접근 가능
		infoR := ts.get("/api/oauth/userinfo", lt.AccessToken)
		if !infoR.Success {
			t.Error("userinfo should work with read:profile scope")
		}
	})

	// ============================================================
	// Step 5: Refresh token
	// ============================================================
	t.Run("토큰 갱신", func(t *testing.T) {
		refreshResp := ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type":    "refresh_token",
			"refresh_token": tokens.RefreshToken,
			"client_id":     client.ClientID,
			"client_secret": client.ClientSecret,
		}, "")
		if !refreshResp.Success {
			t.Fatalf("refresh failed: %v", refreshResp.Error)
		}
		var newTokens struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		json.Unmarshal(refreshResp.Data, &newTokens)

		if newTokens.AccessToken == "" {
			t.Error("new access token is empty")
		}

		// Old access token should be revoked
		oldResp := ts.get("/api/oauth/userinfo", tokens.AccessToken)
		if oldResp.Success {
			t.Error("old access token should be revoked after refresh")
		}

		// New token should work
		newResp := ts.get("/api/oauth/userinfo", newTokens.AccessToken)
		if !newResp.Success {
			t.Error("new access token should work")
		}

		// Update for next steps
		tokens.AccessToken = newTokens.AccessToken
		tokens.RefreshToken = newTokens.RefreshToken
	})

	// ============================================================
	// Step 6: Revoke token
	// ============================================================
	t.Run("토큰 폐기", func(t *testing.T) {
		revokeResp := ts.post("/api/oauth/revoke", map[string]interface{}{
			"token": tokens.AccessToken,
		}, apiUserToken)
		if !revokeResp.Success {
			t.Fatalf("revoke failed: %v", revokeResp.Error)
		}

		// Revoked token should not work
		failResp := ts.get("/api/oauth/userinfo", tokens.AccessToken)
		if failResp.Success {
			t.Error("revoked token should not work")
		}
	})

	// ============================================================
	// Verify: App owner can list and delete their client
	// ============================================================
	t.Run("앱 관리", func(t *testing.T) {
		listResp := ts.get("/api/oauth/clients", appOwnerToken)
		if !listResp.Success {
			t.Fatalf("list clients failed: %v", listResp.Error)
		}

		delResp := ts.delete("/api/oauth/clients/"+client.ClientID, appOwnerToken)
		if !delResp.Success {
			t.Fatalf("delete client failed: %v", delResp.Error)
		}
	})
}
