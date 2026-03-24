package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestLoginRememberMe(t *testing.T) {
	ts := setupTestServer(t)

	t.Run("remember_me=true gives long-lived token (180 days)", func(t *testing.T) {
		r := ts.post("/api/auth/login", map[string]interface{}{
			"email":       testAdminEmail,
			"password":    testAdminPass,
			"remember_me": true,
		}, "")

		if !r.Success {
			t.Fatalf("login should succeed, got error: %v", r.Error)
		}

		var data struct {
			Token string `json:"token"`
		}
		json.Unmarshal(r.Data, &data)

		if data.Token == "" {
			t.Fatal("expected token, got empty")
		}

		// Parse token and check expiry
		claims := parseJWTClaims(t, data.Token)
		expiry := time.Unix(int64(claims["exp"].(float64)), 0)
		issuedAt := time.Unix(int64(claims["iat"].(float64)), 0)
		duration := expiry.Sub(issuedAt)

		// Should be ~180 days (allow 1 minute tolerance for test execution time)
		expectedDuration := 180 * 24 * time.Hour
		if duration < expectedDuration-time.Minute || duration > expectedDuration+time.Minute {
			t.Errorf("expected token duration ~180 days, got %v", duration)
		}
	})

	t.Run("remember_me=false gives short-lived token (24 hours)", func(t *testing.T) {
		r := ts.post("/api/auth/login", map[string]interface{}{
			"email":       testAdminEmail,
			"password":    testAdminPass,
			"remember_me": false,
		}, "")

		if !r.Success {
			t.Fatalf("login should succeed, got error: %v", r.Error)
		}

		var data struct {
			Token string `json:"token"`
		}
		json.Unmarshal(r.Data, &data)

		claims := parseJWTClaims(t, data.Token)
		expiry := time.Unix(int64(claims["exp"].(float64)), 0)
		issuedAt := time.Unix(int64(claims["iat"].(float64)), 0)
		duration := expiry.Sub(issuedAt)

		expectedDuration := 24 * time.Hour
		if duration < expectedDuration-time.Minute || duration > expectedDuration+time.Minute {
			t.Errorf("expected token duration ~24 hours, got %v", duration)
		}
	})

	t.Run("remember_me omitted defaults to short-lived token", func(t *testing.T) {
		r := ts.post("/api/auth/login", map[string]string{
			"email":    testAdminEmail,
			"password": testAdminPass,
		}, "")

		if !r.Success {
			t.Fatalf("login should succeed, got error: %v", r.Error)
		}

		var data struct {
			Token string `json:"token"`
		}
		json.Unmarshal(r.Data, &data)

		claims := parseJWTClaims(t, data.Token)
		expiry := time.Unix(int64(claims["exp"].(float64)), 0)
		issuedAt := time.Unix(int64(claims["iat"].(float64)), 0)
		duration := expiry.Sub(issuedAt)

		expectedDuration := 24 * time.Hour
		if duration < expectedDuration-time.Minute || duration > expectedDuration+time.Minute {
			t.Errorf("expected token duration ~24 hours when remember_me omitted, got %v", duration)
		}
	})

	t.Run("remember_me token is usable for API calls", func(t *testing.T) {
		r := ts.post("/api/auth/login", map[string]interface{}{
			"email":       testAdminEmail,
			"password":    testAdminPass,
			"remember_me": true,
		}, "")

		var data struct {
			Token string `json:"token"`
		}
		json.Unmarshal(r.Data, &data)

		// Token should work for authenticated API calls
		me := ts.get("/api/auth/me", data.Token)
		if !me.Success {
			t.Error("remember_me token should be valid for /auth/me")
		}

		var user struct {
			Email string `json:"email"`
		}
		json.Unmarshal(me.Data, &user)
		if user.Email != testAdminEmail {
			t.Errorf("expected email %s, got %s", testAdminEmail, user.Email)
		}
	})

	t.Run("remember_me token can be refreshed", func(t *testing.T) {
		r := ts.post("/api/auth/login", map[string]interface{}{
			"email":       testAdminEmail,
			"password":    testAdminPass,
			"remember_me": true,
		}, "")

		var data struct {
			Token string `json:"token"`
		}
		json.Unmarshal(r.Data, &data)

		// Refresh should work
		refreshR := ts.post("/api/auth/refresh", nil, data.Token)
		if !refreshR.Success {
			t.Fatalf("refresh of remember_me token should succeed, got: %v", refreshR.Error)
		}

		var refreshData struct {
			Token string `json:"token"`
		}
		json.Unmarshal(refreshR.Data, &refreshData)

		if refreshData.Token == "" {
			t.Error("expected refreshed token, got empty")
		}
	})

	t.Run("wrong password with remember_me still fails", func(t *testing.T) {
		r := ts.post("/api/auth/login", map[string]interface{}{
			"email":       testAdminEmail,
			"password":    "wrongpassword",
			"remember_me": true,
		}, "")

		if r.Success {
			t.Error("login with wrong password should fail even with remember_me=true")
		}
	})
}

func parseJWTClaims(t *testing.T, tokenStr string) jwt.MapClaims {
	t.Helper()
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected MapClaims")
	}
	return claims
}
