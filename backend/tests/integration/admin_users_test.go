package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestAdminUsers_PaginationRegression verifies that the admin users list
// correctly returns ALL users even when the count exceeds the default limit (20).
// This is a regression test for the bug where only 20 users were shown.
func TestAdminUsers_PaginationRegression(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Register 60 pending users
	const totalUsers = 60
	for i := 0; i < totalUsers; i++ {
		email := fmt.Sprintf("pending%d@test.com", i)
		name := fmt.Sprintf("대기유저%d", i)
		studentID := fmt.Sprintf("20260%04d", i)
		r := ts.register(email, "pass1234", name, studentID)
		if !r.Success {
			t.Fatalf("register user %d failed: %v", i, r.Error)
		}
	}

	t.Run("GET /admin/users with limit=1000 returns all users", func(t *testing.T) {
		r := ts.get("/api/admin/users?page=1&limit=1000", adminToken)
		if !r.Success {
			t.Fatalf("list users failed: %v", r.Error)
		}

		var data struct {
			Users []struct {
				ID     int    `json:"id"`
				Email  string `json:"email"`
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"users"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		}
		json.Unmarshal(r.Data, &data)

		// 60 registered + 1 admin = 61 total
		expectedTotal := totalUsers + 1
		if data.Total != expectedTotal {
			t.Errorf("total = %d, want %d", data.Total, expectedTotal)
		}
		if len(data.Users) != expectedTotal {
			t.Errorf("returned %d users, want %d", len(data.Users), expectedTotal)
		}
	})

	t.Run("GET /admin/users/pending returns all 60 pending users", func(t *testing.T) {
		r := ts.get("/api/admin/users/pending", adminToken)
		if !r.Success {
			t.Fatalf("pending users failed: %v", r.Error)
		}

		var pending []struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		}
		json.Unmarshal(r.Data, &pending)

		if len(pending) != totalUsers {
			t.Errorf("pending count = %d, want %d", len(pending), totalUsers)
		}

		// All should be pending status
		for _, u := range pending {
			if u.Status != "pending" {
				t.Errorf("user %d status = %q, want %q", u.ID, u.Status, "pending")
			}
		}
	})

	t.Run("default limit=20 only returns 20 users", func(t *testing.T) {
		r := ts.get("/api/admin/users?page=1&limit=20", adminToken)
		if !r.Success {
			t.Fatalf("list users failed: %v", r.Error)
		}

		var data struct {
			Users      json.RawMessage `json:"users"`
			Total      int             `json:"total"`
			TotalPages int             `json:"total_pages"`
		}
		json.Unmarshal(r.Data, &data)

		var users []json.RawMessage
		json.Unmarshal(data.Users, &users)

		if len(users) != 20 {
			t.Errorf("returned %d users with limit=20, want 20", len(users))
		}

		// total should still be 61
		expectedTotal := totalUsers + 1
		if data.Total != expectedTotal {
			t.Errorf("total = %d, want %d", data.Total, expectedTotal)
		}
		if data.TotalPages != 4 { // ceil(61/20) = 4
			t.Errorf("total_pages = %d, want 4", data.TotalPages)
		}
	})

	t.Run("page 2 returns next batch of users", func(t *testing.T) {
		r := ts.get("/api/admin/users?page=2&limit=20", adminToken)
		if !r.Success {
			t.Fatalf("list users page 2 failed: %v", r.Error)
		}

		var data struct {
			Users []json.RawMessage `json:"users"`
			Total int               `json:"total"`
		}
		json.Unmarshal(r.Data, &data)

		if len(data.Users) != 20 {
			t.Errorf("page 2 returned %d users, want 20", len(data.Users))
		}
	})

	t.Run("approve a pending user changes their status", func(t *testing.T) {
		// Get pending users to find one to approve
		r := ts.get("/api/admin/users/pending", adminToken)
		var pending []struct {
			ID int `json:"id"`
		}
		json.Unmarshal(r.Data, &pending)

		if len(pending) == 0 {
			t.Fatal("no pending users to approve")
		}

		// Approve first pending user
		approveResp := ts.approveUser(adminToken, pending[0].ID)
		if !approveResp.Success {
			t.Fatalf("approve failed: %v", approveResp.Error)
		}

		// Verify pending count decreased
		r2 := ts.get("/api/admin/users/pending", adminToken)
		var pending2 []json.RawMessage
		json.Unmarshal(r2.Data, &pending2)

		if len(pending2) != totalUsers-1 {
			t.Errorf("pending after approve = %d, want %d", len(pending2), totalUsers-1)
		}
	})
}
