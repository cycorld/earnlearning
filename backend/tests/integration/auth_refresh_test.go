package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestRefreshToken(t *testing.T) {
	ts := setupTestServer(t)

	t.Run("valid token refresh", func(t *testing.T) {
		token := ts.login(testAdminEmail, testAdminPass)

		r := ts.post("/api/auth/refresh", nil, token)
		if !r.Success {
			t.Fatalf("refresh should succeed, got error: %v", r.Error)
		}

		var data struct {
			Token string `json:"token"`
			User  struct {
				Email string `json:"email"`
			} `json:"user"`
		}
		json.Unmarshal(r.Data, &data)

		if data.Token == "" {
			t.Error("expected new token, got empty")
		}
		if data.User.Email != testAdminEmail {
			t.Errorf("expected email %s, got %s", testAdminEmail, data.User.Email)
		}

		// New token should also work
		me := ts.get("/api/auth/me", data.Token)
		if !me.Success {
			t.Error("new token should be valid for /auth/me")
		}
	})

	t.Run("expired token within grace period", func(t *testing.T) {
		// Create a token that expired 1 hour ago (within 7-day grace period)
		claims := jwt.MapClaims{
			"user_id": float64(1),
			"email":   testAdminEmail,
			"role":    "admin",
			"status":  "approved",
			"exp":     float64(time.Now().Add(-1 * time.Hour).Unix()),
			"iat":     float64(time.Now().Add(-25 * time.Hour).Unix()),
		}
		expiredToken := createTestToken(t, claims)

		r := ts.post("/api/auth/refresh", nil, expiredToken)
		if !r.Success {
			t.Fatalf("refresh with recently expired token should succeed, got: %v", r.Error)
		}

		var data struct {
			Token string `json:"token"`
		}
		json.Unmarshal(r.Data, &data)

		if data.Token == "" {
			t.Error("expected new token")
		}

		// New token should work
		me := ts.get("/api/auth/me", data.Token)
		if !me.Success {
			t.Error("refreshed token should be valid")
		}
	})

	t.Run("expired token beyond grace period", func(t *testing.T) {
		// Create a token that expired 8 days ago (beyond 7-day grace period)
		claims := jwt.MapClaims{
			"user_id": float64(1),
			"email":   testAdminEmail,
			"role":    "admin",
			"status":  "approved",
			"exp":     float64(time.Now().Add(-8 * 24 * time.Hour).Unix()),
			"iat":     float64(time.Now().Add(-9 * 24 * time.Hour).Unix()),
		}
		oldToken := createTestToken(t, claims)

		r := ts.post("/api/auth/refresh", nil, oldToken)
		if r.Success {
			t.Error("refresh with very old token should fail")
		}
	})

	t.Run("no token", func(t *testing.T) {
		r := ts.post("/api/auth/refresh", nil, "")
		if r.Success {
			t.Error("refresh without token should fail")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		r := ts.post("/api/auth/refresh", nil, "invalid.token.here")
		if r.Success {
			t.Error("refresh with invalid token should fail")
		}
	})
}

func createTestToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("create test token: %v", err)
	}
	return signed
}
