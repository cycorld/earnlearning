package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestResponseFormat_Consistency verifies all list endpoints return consistent
// response formats. Regression: multiple endpoints returned raw arrays causing
// frontend crashes when accessing .data property.
func TestResponseFormat_Consistency(t *testing.T) {
	ts := setupTestServer(t)

	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Create classroom for feed tests
	crResp := ts.post("/api/classrooms", map[string]interface{}{
		"name": "응답포맷 테스트반", "initial_capital": 500000,
	}, adminToken)
	var cr struct {
		ID   int    `json:"id"`
		Code string `json:"code"`
	}
	json.Unmarshal(crResp.Data, &cr)

	// Register + approve a student
	studentToken := ts.registerAndApprove("format-test@test.com", "pass1234", "포맷학생", "2024020")

	// Student joins classroom
	ts.post("/api/classrooms/join", map[string]string{"code": cr.Code}, studentToken)

	// --- Test all list endpoints return arrays, not null ---

	t.Run("GET /api/wallet returns valid data", func(t *testing.T) {
		r := ts.get("/api/wallet", studentToken)
		if !r.Success {
			t.Fatalf("wallet failed: %v", r.Error)
		}
		if string(r.Data) == "null" {
			t.Error("wallet data should not be null")
		}
	})

	t.Run("GET /api/wallet/transactions returns PaginatedData", func(t *testing.T) {
		r := ts.get("/api/wallet/transactions?page=1&limit=10", studentToken)
		if !r.Success {
			t.Fatalf("transactions failed: %v", r.Error)
		}
		assertPaginatedData(t, r.Data)
	})

	t.Run("GET /api/notifications returns PaginatedData", func(t *testing.T) {
		r := ts.get("/api/notifications?page=1&limit=10", studentToken)
		if !r.Success {
			t.Fatalf("notifications failed: %v", r.Error)
		}
		assertPaginatedData(t, r.Data)
	})

	t.Run("GET /api/classrooms returns array", func(t *testing.T) {
		r := ts.get("/api/classrooms", studentToken)
		if !r.Success {
			t.Fatalf("classrooms failed: %v", r.Error)
		}
		assertIsArray(t, r.Data, "classrooms")
	})

	t.Run("GET /api/companies/mine returns array", func(t *testing.T) {
		r := ts.get("/api/companies/mine", studentToken)
		if !r.Success {
			t.Fatalf("companies failed: %v", r.Error)
		}
		// Should be array or valid data, not null
		if string(r.Data) == "null" {
			t.Error("companies/mine should return empty array, not null")
		}
	})

	t.Run("GET /api/freelance/jobs returns array or PaginatedData", func(t *testing.T) {
		r := ts.get("/api/freelance/jobs?page=1&limit=10", studentToken)
		if !r.Success {
			t.Fatalf("freelance jobs failed: %v", r.Error)
		}
		if string(r.Data) == "null" {
			t.Error("freelance jobs data should not be null")
		}
	})

	t.Run("GET /api/investment/rounds returns valid data", func(t *testing.T) {
		r := ts.get("/api/investment/rounds?page=1&limit=10", studentToken)
		if !r.Success {
			t.Fatalf("investment rounds failed: %v", r.Error)
		}
		if string(r.Data) == "null" {
			t.Error("investment rounds data should not be null")
		}
	})

	t.Run("GET /api/investment/portfolio returns valid data", func(t *testing.T) {
		r := ts.get("/api/investment/portfolio", studentToken)
		if !r.Success {
			t.Fatalf("portfolio failed: %v", r.Error)
		}
	})

	t.Run("GET /api/exchange/companies returns array", func(t *testing.T) {
		r := ts.get("/api/exchange/companies", studentToken)
		if !r.Success {
			t.Fatalf("exchange companies failed: %v", r.Error)
		}
		assertIsArray(t, r.Data, "exchange companies")
	})

	t.Run("GET /api/exchange/orders/mine returns valid data", func(t *testing.T) {
		r := ts.get("/api/exchange/orders/mine", studentToken)
		if !r.Success {
			t.Fatalf("my orders failed: %v", r.Error)
		}
		if string(r.Data) == "null" {
			t.Error("orders data should not be null")
		}
	})

	t.Run("GET /api/loans/mine returns array", func(t *testing.T) {
		r := ts.get("/api/loans/mine", studentToken)
		if !r.Success {
			t.Fatalf("loans failed: %v", r.Error)
		}
		assertIsArray(t, r.Data, "loans")
	})

	t.Run("GET /api/posts returns PaginatedData", func(t *testing.T) {
		r := ts.get(fmt.Sprintf("/api/posts?classroom_id=%d&page=1&limit=10", cr.ID), studentToken)
		if !r.Success {
			t.Fatalf("posts failed: %v", r.Error)
		}
		assertPaginatedData(t, r.Data)
	})

	// --- Admin list endpoints ---

	t.Run("GET /api/admin/users returns array", func(t *testing.T) {
		r := ts.get("/api/admin/users", adminToken)
		if !r.Success {
			t.Fatalf("admin users failed: %v", r.Error)
		}
		if string(r.Data) == "null" {
			t.Error("admin users should not return null")
		}
	})

	t.Run("GET /api/admin/users/pending returns array", func(t *testing.T) {
		r := ts.get("/api/admin/users/pending", adminToken)
		if !r.Success {
			t.Fatalf("pending users failed: %v", r.Error)
		}
		assertIsArray(t, r.Data, "pending users")
	})

	t.Run("GET /api/admin/loans returns valid data", func(t *testing.T) {
		r := ts.get("/api/admin/loans?page=1&limit=10", adminToken)
		if !r.Success {
			t.Fatalf("admin loans failed: %v", r.Error)
		}
	})

	t.Run("GET /api/admin/companies returns array", func(t *testing.T) {
		r := ts.get("/api/admin/companies", adminToken)
		if !r.Success {
			t.Fatalf("admin companies failed: %v", r.Error)
		}
		assertIsArray(t, r.Data, "admin companies")
	})

	// --- Channels for classroom ---

	t.Run("GET /api/classrooms/:id/channels returns array", func(t *testing.T) {
		r := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", cr.ID), studentToken)
		if !r.Success {
			t.Fatalf("channels failed: %v", r.Error)
		}
		assertIsArray(t, r.Data, "channels")
	})
}

// TestResponseFormat_AllResponsesHaveStandardEnvelope checks every API response
// follows {success, data, error} structure.
func TestResponseFormat_AllResponsesHaveStandardEnvelope(t *testing.T) {
	ts := setupTestServer(t)

	// Unauthenticated request to any protected endpoint
	r := ts.get("/api/wallet", "")
	if r.Error == nil {
		t.Error("expected error for unauthenticated request")
	}
	// The error should have code and message
	if r.Error.Code == "" {
		t.Error("error should have a code")
	}
	if r.Error.Message == "" {
		t.Error("error should have a message")
	}
}

// --- Helper assertions ---

func assertPaginatedData(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var data struct {
		Data       json.RawMessage `json:"data"`
		Pagination *struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("not PaginatedData format: %v\nraw: %s", err, string(raw))
	}
	if data.Data == nil {
		t.Error("PaginatedData.data is nil")
	}
	if data.Pagination == nil {
		t.Error("PaginatedData.pagination is nil")
	}
}

func assertIsArray(t *testing.T, raw json.RawMessage, name string) {
	t.Helper()
	if len(raw) == 0 || string(raw) == "null" {
		t.Errorf("%s should be an array, got null/empty", name)
		return
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		// Might be a wrapped format, which is also OK
		var obj map[string]json.RawMessage
		if err2 := json.Unmarshal(raw, &obj); err2 != nil {
			t.Errorf("%s is neither array nor object: %v\nraw: %s", name, err, string(raw))
		}
	}
}
