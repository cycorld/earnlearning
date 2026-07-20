package email

import (
	"errors"
	"testing"
)

// TestSESSendMailFromDisabledIsLoud — 비활성 발신기의 SendMailFrom 은 조용한 성공(nil) 이 아니라
// ErrSenderDisabled 를 반환해야 한다. (미설정 프로덕션에서 거짓 "발송 완료" + 보낸편지함 저장 방지)
func TestSESSendMailFromDisabledIsLoud(t *testing.T) {
	svc := NewSESService(Config{}) // FromEmail 비어있음 → disabled
	if svc.IsEnabled() {
		t.Fatalf("빈 Config 는 disabled 여야 함")
	}
	err := svc.SendMailFrom(OutgoingMail{To: "x@y.com", Subject: "s", TextBody: "b"})
	if !errors.Is(err, ErrSenderDisabled) {
		t.Fatalf("비활성 발신기는 ErrSenderDisabled 를 반환해야 함, got %v", err)
	}
}

// TestSESSendEmailDisabledStaysSilent — SendEmail(알림/비번재설정 경로) 은 기존대로 비활성 시 무시(nil).
func TestSESSendEmailDisabledStaysSilent(t *testing.T) {
	svc := NewSESService(Config{})
	if err := svc.SendEmail("x@y.com", "s", "", "b"); err != nil {
		t.Fatalf("비활성 SendEmail 은 조용히 nil 이어야 함(호환성), got %v", err)
	}
}

// TestCannotUseFromIdentity — From 신원 사용 불가 판정 (#168 회귀).
// prod IAM 이 특정 identity 만 허용하면 AccessDeniedException 으로 거부되는데,
// 이것도 미인증 신원과 동일하게 설정 From 폴백을 타야 한다.
func TestCannotUseFromIdentity(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"미인증 주소", errors.New("MessageRejected: Email address is not verified. The following identities failed the check"), true},
		{"IAM AccessDenied (prod 실제 메시지)", errors.New("operation error SESv2: SendEmail, https response error StatusCode: 403, AccessDeniedException: User `arn:aws:iam::0:user/earnlearning-ses' is not authorized to perform `ses:SendEmail' on resource `arn:aws:ses:ap-northeast-2:0:identity/earnlearning.com'"), true},
		{"무관한 에러", errors.New("connection refused"), false},
	}
	for _, c := range cases {
		if got := cannotUseFromIdentity(c.err); got != c.want {
			t.Errorf("%s: cannotUseFromIdentity=%v, want %v", c.name, got, c.want)
		}
	}
}
