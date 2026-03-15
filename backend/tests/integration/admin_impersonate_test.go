package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestAdminImpersonate tests admin ability to impersonate (login as) any user.
func TestAdminImpersonate(t *testing.T) {
	ts := setupTestServer(t)

	// Create a student
	studentToken := ts.registerAndApprove("student@test.com", "pass1234", "테스트학생", "2024500")
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// Get student ID from their profile
	studentResp := ts.get("/api/auth/me", studentToken)
	var studentData map[string]interface{}
	json.Unmarshal(studentResp.Data, &studentData)
	studentID := int(studentData["id"].(float64))

	t.Run("admin can impersonate a student", func(t *testing.T) {
		resp := ts.post(fmt.Sprintf("/api/admin/users/%d/impersonate", studentID), nil, adminToken)

		if !resp.Success {
			t.Fatalf("expected success, got error: %v", resp.Error)
		}

		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)

		// Should return a token
		token, ok := data["token"].(string)
		if !ok || token == "" {
			t.Fatal("expected a non-empty token in response")
		}

		// Should return user info
		userData, ok := data["user"].(map[string]interface{})
		if !ok {
			t.Fatal("expected user object in response")
		}
		if userData["name"] != "테스트학생" {
			t.Errorf("expected user name=테스트학생, got %v", userData["name"])
		}
		if userData["email"] != "student@test.com" {
			t.Errorf("expected user email=student@test.com, got %v", userData["email"])
		}
	})

	t.Run("impersonated token works for API calls", func(t *testing.T) {
		// Get impersonation token
		resp := ts.post(fmt.Sprintf("/api/admin/users/%d/impersonate", studentID), nil, adminToken)
		var data map[string]interface{}
		json.Unmarshal(resp.Data, &data)
		impToken := data["token"].(string)

		// Use impersonated token to access the student's profile
		meResp := ts.get("/api/auth/me", impToken)
		if !meResp.Success {
			t.Fatalf("expected success, got error: %v", meResp.Error)
		}

		var meData map[string]interface{}
		json.Unmarshal(meResp.Data, &meData)
		if meData["email"] != "student@test.com" {
			t.Errorf("expected email=student@test.com, got %v", meData["email"])
		}
	})

	t.Run("non-admin cannot impersonate", func(t *testing.T) {
		// Student tries to impersonate (should use admin route which requires admin role)
		resp := ts.post(fmt.Sprintf("/api/admin/users/%d/impersonate", studentID), nil, studentToken)

		if resp.Success {
			t.Errorf("expected failure for non-admin impersonation, but got success")
		}
	})

	t.Run("impersonate non-existent user fails", func(t *testing.T) {
		resp := ts.post("/api/admin/users/99999/impersonate", nil, adminToken)

		if resp.Success {
			t.Errorf("expected failure for non-existent user, but got success")
		}
	})
}
