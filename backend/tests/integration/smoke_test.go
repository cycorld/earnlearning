package integration

import (
	"testing"
)

// TestSmoke verifies all critical endpoints are reachable and return valid JSON.
// This must pass before any commit or further testing.
func TestSmoke(t *testing.T) {
	ts := setupTestServer(t)

	// --- Public endpoints ---

	t.Run("GET /api/health returns success", func(t *testing.T) {
		r := ts.get("/api/health", "")
		if !r.Success {
			t.Fatalf("health check failed: %v", r.Error)
		}
	})

	t.Run("POST /api/auth/login returns proper error for bad creds", func(t *testing.T) {
		r := ts.post("/api/auth/login", map[string]string{
			"email": "nobody@test.com", "password": "wrong",
		}, "")
		if r.Success {
			t.Fatal("login should fail with bad creds")
		}
		if r.Error == nil {
			t.Fatal("error should be present")
		}
	})

	// --- Auth flow ---

	t.Run("POST /api/auth/register works", func(t *testing.T) {
		r := ts.register("smoke@test.com", "pass1234", "스모크", "2024001")
		if !r.Success {
			t.Fatalf("register failed: %v", r.Error)
		}
	})

	t.Run("Admin login works", func(t *testing.T) {
		token := ts.login(testAdminEmail, testAdminPass)
		if token == "" {
			t.Fatal("admin login returned empty token")
		}
	})

	// --- Auth-required endpoints return 401 without token ---

	t.Run("GET /api/wallet requires auth", func(t *testing.T) {
		r := ts.get("/api/wallet", "")
		if r.Success {
			t.Fatal("should require auth")
		}
	})

	t.Run("GET /api/classrooms requires auth", func(t *testing.T) {
		r := ts.get("/api/classrooms", "")
		if r.Success {
			t.Fatal("should require auth")
		}
	})

	t.Run("GET /api/notifications requires auth", func(t *testing.T) {
		r := ts.get("/api/notifications", "")
		if r.Success {
			t.Fatal("should require auth")
		}
	})

	// --- Approved user endpoints work with valid token ---

	adminToken := ts.login(testAdminEmail, testAdminPass)

	approvedEndpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/wallet"},
		{"GET", "/api/wallet/transactions"},
		{"GET", "/api/classrooms"},
		{"GET", "/api/companies/mine"},
		{"GET", "/api/freelance/jobs"},
		{"GET", "/api/investment/rounds"},
		{"GET", "/api/investment/portfolio"},
		{"GET", "/api/exchange/companies"},
		{"GET", "/api/exchange/orders/mine"},
		{"GET", "/api/loans/mine"},
		{"GET", "/api/notifications"},
		{"GET", "/api/posts?classroom_id=0"},
	}

	for _, ep := range approvedEndpoints {
		t.Run(ep.method+" "+ep.path+" responds", func(t *testing.T) {
			r := ts.get(ep.path, adminToken)
			if !r.Success {
				t.Fatalf("%s %s failed: %v", ep.method, ep.path, r.Error)
			}
		})
	}

	// --- Admin endpoints ---

	adminEndpoints := []string{
		"/api/admin/users/pending",
		"/api/admin/users",
		"/api/admin/loans",
		"/api/admin/companies",
	}

	for _, path := range adminEndpoints {
		t.Run("GET "+path+" responds", func(t *testing.T) {
			r := ts.get(path, adminToken)
			if !r.Success {
				t.Fatalf("GET %s failed: %v", path, r.Error)
			}
		})
	}
}
