package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"
)

func TestOAuth(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: register and approve a user
	userToken := ts.registerAndApprove("oauth-user@ewha.ac.kr", "password123", "OAuth테스트", "2024010")

	t.Run("클라이언트 등록", func(t *testing.T) {
		resp := ts.post("/api/oauth/clients", map[string]interface{}{
			"name":          "테스트앱",
			"description":   "테스트용 앱입니다",
			"redirect_uris": []string{"http://localhost:3000/callback"},
			"scopes":        []string{"read:profile", "read:wallet"},
		}, userToken)

		if !resp.Success {
			t.Fatalf("register client failed: %v", resp.Error)
		}

		var data struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			Name         string `json:"name"`
		}
		json.Unmarshal(resp.Data, &data)

		if data.ClientID == "" || data.ClientSecret == "" {
			t.Fatal("client_id or client_secret is empty")
		}
		if data.Name != "테스트앱" {
			t.Errorf("expected name '테스트앱', got '%s'", data.Name)
		}
	})

	t.Run("클라이언트 목록 조회", func(t *testing.T) {
		resp := ts.get("/api/oauth/clients", userToken)
		if !resp.Success {
			t.Fatalf("list clients failed: %v", resp.Error)
		}
		var clients []interface{}
		json.Unmarshal(resp.Data, &clients)
		if len(clients) == 0 {
			t.Error("expected at least 1 client")
		}
	})

	t.Run("Authorization Code 전체 플로우", func(t *testing.T) {
		// 1. Register client
		regResp := ts.post("/api/oauth/clients", map[string]interface{}{
			"name":          "플로우테스트앱",
			"redirect_uris": []string{"http://localhost:3000/callback"},
			"scopes":        []string{"read:profile", "read:wallet"},
		}, userToken)
		var regData struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		json.Unmarshal(regResp.Data, &regData)

		// 2. Get authorize info
		infoResp := ts.get("/api/oauth/authorize?client_id="+regData.ClientID+"&redirect_uri="+url.QueryEscape("http://localhost:3000/callback")+"&scope="+url.QueryEscape("read:profile read:wallet"), userToken)
		if !infoResp.Success {
			t.Fatalf("authorize info failed: %v", infoResp.Error)
		}

		// 3. Authorize (grant consent)
		authResp := ts.post("/api/oauth/authorize", map[string]interface{}{
			"client_id":    regData.ClientID,
			"redirect_uri": "http://localhost:3000/callback",
			"scopes":       []string{"read:profile", "read:wallet"},
			"state":        "test-state-123",
		}, userToken)
		if !authResp.Success {
			t.Fatalf("authorize failed: %v", authResp.Error)
		}
		var authData struct {
			Code  string `json:"code"`
			State string `json:"state"`
		}
		json.Unmarshal(authResp.Data, &authData)

		if authData.Code == "" {
			t.Fatal("authorization code is empty")
		}
		if authData.State != "test-state-123" {
			t.Errorf("state mismatch: got '%s'", authData.State)
		}

		// 4. Exchange code for tokens
		tokenResp := ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          authData.Code,
			"client_id":     regData.ClientID,
			"client_secret": regData.ClientSecret,
			"redirect_uri":  "http://localhost:3000/callback",
		}, "")
		if !tokenResp.Success {
			t.Fatalf("token exchange failed: %v", tokenResp.Error)
		}
		var tokenData struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    int    `json:"expires_in"`
		}
		json.Unmarshal(tokenResp.Data, &tokenData)

		if tokenData.AccessToken == "" || tokenData.RefreshToken == "" {
			t.Fatal("access_token or refresh_token is empty")
		}
		if tokenData.TokenType != "Bearer" {
			t.Errorf("expected Bearer, got '%s'", tokenData.TokenType)
		}
		if tokenData.ExpiresIn != 3600 {
			t.Errorf("expected 3600, got %d", tokenData.ExpiresIn)
		}

		// 5. Use access token to get user info
		userinfoResp := ts.get("/api/oauth/userinfo", tokenData.AccessToken)
		if !userinfoResp.Success {
			t.Fatalf("userinfo failed: %v", userinfoResp.Error)
		}
		var userInfo struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		json.Unmarshal(userinfoResp.Data, &userInfo)
		if userInfo.Name != "OAuth테스트" {
			t.Errorf("expected name 'OAuth테스트', got '%s'", userInfo.Name)
		}

		// 6. Refresh token
		refreshResp := ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type":    "refresh_token",
			"refresh_token": tokenData.RefreshToken,
			"client_id":     regData.ClientID,
			"client_secret": regData.ClientSecret,
		}, "")
		if !refreshResp.Success {
			t.Fatalf("refresh failed: %v", refreshResp.Error)
		}
		var newTokenData struct {
			AccessToken string `json:"access_token"`
		}
		json.Unmarshal(refreshResp.Data, &newTokenData)
		if newTokenData.AccessToken == "" {
			t.Fatal("new access_token is empty after refresh")
		}

		// 7. Revoke token
		revokeResp := ts.post("/api/oauth/revoke", map[string]interface{}{
			"token": newTokenData.AccessToken,
		}, userToken)
		if !revokeResp.Success {
			t.Fatalf("revoke failed: %v", revokeResp.Error)
		}

		// 8. Revoked token should fail
		failResp := ts.get("/api/oauth/userinfo", newTokenData.AccessToken)
		if failResp.Success {
			t.Error("expected revoked token to fail")
		}
	})

	t.Run("PKCE 플로우", func(t *testing.T) {
		// Register client
		regResp := ts.post("/api/oauth/clients", map[string]interface{}{
			"name":          "PKCE앱",
			"redirect_uris": []string{"http://localhost:3000/callback"},
			"scopes":        []string{"read:profile"},
		}, userToken)
		var regData struct {
			ClientID string `json:"client_id"`
		}
		json.Unmarshal(regResp.Data, &regData)

		// Generate PKCE code verifier and challenge
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		hash := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

		// Authorize with PKCE
		authResp := ts.post("/api/oauth/authorize", map[string]interface{}{
			"client_id":              regData.ClientID,
			"redirect_uri":          "http://localhost:3000/callback",
			"scopes":                []string{"read:profile"},
			"code_challenge":        codeChallenge,
			"code_challenge_method": "S256",
		}, userToken)
		if !authResp.Success {
			t.Fatalf("pkce authorize failed: %v", authResp.Error)
		}
		var authData struct {
			Code string `json:"code"`
		}
		json.Unmarshal(authResp.Data, &authData)

		// Exchange with code_verifier (no client_secret needed)
		tokenResp := ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          authData.Code,
			"client_id":     regData.ClientID,
			"redirect_uri":  "http://localhost:3000/callback",
			"code_verifier": codeVerifier,
		}, "")
		if !tokenResp.Success {
			t.Fatalf("pkce token exchange failed: %v", tokenResp.Error)
		}

		// Wrong verifier should fail
		authResp2 := ts.post("/api/oauth/authorize", map[string]interface{}{
			"client_id":              regData.ClientID,
			"redirect_uri":          "http://localhost:3000/callback",
			"scopes":                []string{"read:profile"},
			"code_challenge":        codeChallenge,
			"code_challenge_method": "S256",
		}, userToken)
		var authData2 struct {
			Code string `json:"code"`
		}
		json.Unmarshal(authResp2.Data, &authData2)

		badResp := ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          authData2.Code,
			"client_id":     regData.ClientID,
			"redirect_uri":  "http://localhost:3000/callback",
			"code_verifier": "wrong-verifier",
		}, "")
		if badResp.Success {
			t.Error("expected PKCE verification to fail with wrong verifier")
		}
	})

	t.Run("에러 케이스: 잘못된 redirect_uri", func(t *testing.T) {
		regResp := ts.post("/api/oauth/clients", map[string]interface{}{
			"name":          "에러테스트앱",
			"redirect_uris": []string{"http://localhost:3000/callback"},
			"scopes":        []string{"read:profile"},
		}, userToken)
		var regData struct {
			ClientID string `json:"client_id"`
		}
		json.Unmarshal(regResp.Data, &regData)

		resp := ts.post("/api/oauth/authorize", map[string]interface{}{
			"client_id":    regData.ClientID,
			"redirect_uri": "http://evil.com/callback",
			"scopes":       []string{"read:profile"},
		}, userToken)
		if resp.Success {
			t.Error("expected redirect_uri mismatch to fail")
		}
	})

	t.Run("에러 케이스: 만료/재사용 코드", func(t *testing.T) {
		regResp := ts.post("/api/oauth/clients", map[string]interface{}{
			"name":          "재사용테스트",
			"redirect_uris": []string{"http://localhost:3000/callback"},
			"scopes":        []string{"read:profile"},
		}, userToken)
		var regData struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		json.Unmarshal(regResp.Data, &regData)

		authResp := ts.post("/api/oauth/authorize", map[string]interface{}{
			"client_id":    regData.ClientID,
			"redirect_uri": "http://localhost:3000/callback",
			"scopes":       []string{"read:profile"},
		}, userToken)
		var authData struct {
			Code string `json:"code"`
		}
		json.Unmarshal(authResp.Data, &authData)

		// First exchange should succeed
		ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          authData.Code,
			"client_id":     regData.ClientID,
			"client_secret": regData.ClientSecret,
			"redirect_uri":  "http://localhost:3000/callback",
		}, "")

		// Second use should fail (code already used)
		reuse := ts.post("/api/oauth/token", map[string]interface{}{
			"grant_type":    "authorization_code",
			"code":          authData.Code,
			"client_id":     regData.ClientID,
			"client_secret": regData.ClientSecret,
			"redirect_uri":  "http://localhost:3000/callback",
		}, "")
		if reuse.Success {
			t.Error("expected reused code to fail")
		}
	})

	t.Run("클라이언트 삭제", func(t *testing.T) {
		// Register a client to delete
		regResp := ts.post("/api/oauth/clients", map[string]interface{}{
			"name":          "삭제테스트",
			"redirect_uris": []string{"http://localhost:3000/callback"},
			"scopes":        []string{"read:profile"},
		}, userToken)
		var regData struct {
			ClientID string `json:"client_id"`
		}
		json.Unmarshal(regResp.Data, &regData)

		delResp := ts.delete("/api/oauth/clients/"+regData.ClientID, userToken)
		if !delResp.Success {
			t.Fatalf("delete client failed: %v", delResp.Error)
		}

		// Verify it's gone - authorize should fail
		authResp := ts.post("/api/oauth/authorize", map[string]interface{}{
			"client_id":    regData.ClientID,
			"redirect_uri": "http://localhost:3000/callback",
			"scopes":       []string{"read:profile"},
		}, userToken)
		if authResp.Success {
			t.Error("expected deleted client to fail authorization")
		}
	})

	t.Run("유효하지 않은 스코프 거부", func(t *testing.T) {
		resp := ts.post("/api/oauth/clients", map[string]interface{}{
			"name":          "잘못된스코프앱",
			"redirect_uris": []string{"http://localhost:3000/callback"},
			"scopes":        []string{"invalid:scope"},
		}, userToken)
		if resp.Success {
			t.Error("expected invalid scope to be rejected")
		}
	})
}
