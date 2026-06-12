package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sync"
	"testing"
	"time"
)

// fakeEmailSender captures emails instead of sending them (#128).
type fakeEmailSender struct {
	mu     sync.Mutex
	emails []capturedEmail
}

type capturedEmail struct {
	to      string
	subject string
	html    string
	text    string
}

func (f *fakeEmailSender) SendEmail(to, subject, html, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.emails = append(f.emails, capturedEmail{to: to, subject: subject, html: html, text: text})
	return nil
}

func (f *fakeEmailSender) IsEnabled() bool { return true }

func (f *fakeEmailSender) last() *capturedEmail {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.emails) == 0 {
		return nil
	}
	return &f.emails[len(f.emails)-1]
}

func (f *fakeEmailSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.emails)
}

var resetTokenRegex = regexp.MustCompile(`/reset-password\?token=([0-9a-f]+)`)

func extractResetToken(t *testing.T, body string) string {
	t.Helper()
	m := resetTokenRegex.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("reset token not found in email body: %s", body)
	}
	return m[1]
}

func TestPasswordReset(t *testing.T) {
	ts := setupTestServer(t)
	fake := &fakeEmailSender{}
	ts.authUC.SetEmailService(fake, "http://test.local")

	const (
		userEmail = "reset-user@test.com"
		oldPass   = "oldpass123"
		newPass   = "newpass456"
	)
	ts.registerAndApprove(userEmail, oldPass, "리셋유저", "2024001")

	t.Run("full flow: forgot -> email -> reset -> login with new password", func(t *testing.T) {
		r := ts.post("/api/auth/forgot-password", map[string]string{"email": userEmail}, "")
		if !r.Success {
			t.Fatalf("forgot-password should succeed, got: %v", r.Error)
		}

		mail := fake.last()
		if mail == nil {
			t.Fatal("expected reset email to be sent")
		}
		if mail.to != userEmail {
			t.Errorf("email sent to %s, want %s", mail.to, userEmail)
		}
		token := extractResetToken(t, mail.text)

		r = ts.post("/api/auth/reset-password", map[string]string{
			"token": token, "password": newPass,
		}, "")
		if !r.Success {
			t.Fatalf("reset-password should succeed, got: %v", r.Error)
		}

		// New password works
		loginResp := ts.post("/api/auth/login", map[string]string{
			"email": userEmail, "password": newPass,
		}, "")
		if !loginResp.Success {
			t.Errorf("login with new password should succeed, got: %v", loginResp.Error)
		}

		// Old password rejected
		loginResp = ts.post("/api/auth/login", map[string]string{
			"email": userEmail, "password": oldPass,
		}, "")
		if loginResp.Success {
			t.Error("login with old password should fail")
		}
	})

	t.Run("token is single-use", func(t *testing.T) {
		ts.post("/api/auth/forgot-password", map[string]string{"email": userEmail}, "")
		token := extractResetToken(t, fake.last().text)

		r := ts.post("/api/auth/reset-password", map[string]string{
			"token": token, "password": "anotherpass1",
		}, "")
		if !r.Success {
			t.Fatalf("first reset should succeed, got: %v", r.Error)
		}

		r = ts.post("/api/auth/reset-password", map[string]string{
			"token": token, "password": "anotherpass2",
		}, "")
		if r.Success {
			t.Error("second use of same token should fail")
		}
	})

	t.Run("new request invalidates previous token", func(t *testing.T) {
		ts.post("/api/auth/forgot-password", map[string]string{"email": userEmail}, "")
		firstToken := extractResetToken(t, fake.last().text)

		ts.post("/api/auth/forgot-password", map[string]string{"email": userEmail}, "")

		r := ts.post("/api/auth/reset-password", map[string]string{
			"token": firstToken, "password": "anotherpass3",
		}, "")
		if r.Success {
			t.Error("old token should be invalidated by a new request")
		}
	})

	t.Run("unknown email: success response, no email sent", func(t *testing.T) {
		before := fake.count()
		r := ts.post("/api/auth/forgot-password", map[string]string{"email": "nobody@test.com"}, "")
		if !r.Success {
			t.Errorf("forgot-password with unknown email should not reveal existence, got: %v", r.Error)
		}
		if fake.count() != before {
			t.Error("no email should be sent for unknown address")
		}
	})

	t.Run("invalid token rejected", func(t *testing.T) {
		r := ts.post("/api/auth/reset-password", map[string]string{
			"token": "deadbeef", "password": "whatever123",
		}, "")
		if r.Success {
			t.Error("reset with bogus token should fail")
		}
	})

	t.Run("weak password rejected", func(t *testing.T) {
		ts.post("/api/auth/forgot-password", map[string]string{"email": userEmail}, "")
		token := extractResetToken(t, fake.last().text)

		r := ts.post("/api/auth/reset-password", map[string]string{
			"token": token, "password": "short",
		}, "")
		if r.Success {
			t.Error("reset with <8 char password should fail")
		}
		if r.Error == nil || r.Error.Code != "WEAK_PASSWORD" {
			t.Errorf("expected WEAK_PASSWORD, got: %v", r.Error)
		}
	})

	t.Run("expired token rejected", func(t *testing.T) {
		// Insert an already-expired token directly.
		rawToken := "00000000000000000000000000000000000000000000000000000000000000ff"
		hash := sha256.Sum256([]byte(rawToken))
		_, err := ts.db.Exec(
			`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
			1, hex.EncodeToString(hash[:]), time.Now().Add(-1*time.Minute).UTC(),
		)
		if err != nil {
			t.Fatalf("insert expired token: %v", err)
		}

		r := ts.post("/api/auth/reset-password", map[string]string{
			"token": rawToken, "password": "validpass123",
		}, "")
		if r.Success {
			t.Error("reset with expired token should fail")
		}
	})
}
