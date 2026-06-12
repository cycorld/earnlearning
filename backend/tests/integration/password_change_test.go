package integration

import "testing"

// #131 — 로그인 상태에서 프로필 비밀번호 변경
func TestPasswordChange(t *testing.T) {
	ts := setupTestServer(t)

	const (
		userEmail = "pwchange@test.com"
		oldPass   = "oldpass123"
		newPass   = "newpass456"
	)
	token := ts.registerAndApprove(userEmail, oldPass, "변경유저", "2024002")

	t.Run("wrong current password rejected", func(t *testing.T) {
		r := ts.put("/api/auth/password", map[string]string{
			"current_password": "wrongpass1", "new_password": newPass,
		}, token)
		if r.Success {
			t.Error("change with wrong current password should fail")
		}
		if r.Error == nil || r.Error.Code != "INVALID_CREDENTIALS" {
			t.Errorf("expected INVALID_CREDENTIALS, got: %v", r.Error)
		}
	})

	t.Run("weak new password rejected", func(t *testing.T) {
		r := ts.put("/api/auth/password", map[string]string{
			"current_password": oldPass, "new_password": "short",
		}, token)
		if r.Success {
			t.Error("change with <8 char new password should fail")
		}
		if r.Error == nil || r.Error.Code != "WEAK_PASSWORD" {
			t.Errorf("expected WEAK_PASSWORD, got: %v", r.Error)
		}
	})

	t.Run("unauthenticated rejected", func(t *testing.T) {
		r := ts.put("/api/auth/password", map[string]string{
			"current_password": oldPass, "new_password": newPass,
		}, "")
		if r.Success {
			t.Error("change without token should fail")
		}
	})

	t.Run("success: old password stops working, new password logs in", func(t *testing.T) {
		r := ts.put("/api/auth/password", map[string]string{
			"current_password": oldPass, "new_password": newPass,
		}, token)
		if !r.Success {
			t.Fatalf("password change should succeed, got: %v", r.Error)
		}

		login := ts.post("/api/auth/login", map[string]string{
			"email": userEmail, "password": newPass,
		}, "")
		if !login.Success {
			t.Errorf("login with new password should succeed, got: %v", login.Error)
		}

		login = ts.post("/api/auth/login", map[string]string{
			"email": userEmail, "password": oldPass,
		}, "")
		if login.Success {
			t.Error("login with old password should fail")
		}
	})
}
