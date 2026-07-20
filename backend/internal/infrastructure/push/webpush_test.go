package push

import (
	"testing"

	"github.com/earnlearning/backend/internal/domain/notification"
)

// TestFormatPayloadMailDeepLink — #173: 메일 알림 푸시 URL 딥링크.
// 일반식 /<reference_type>/<id> 는 /mail/123 같은 없는 라우트를 만들어 홈으로 떨어졌다.
func TestFormatPayloadMailDeepLink(t *testing.T) {
	s := &WebPushService{}

	// 수신 알림 → 해당 메일 바로 열기
	got := s.FormatPayload(&notification.Notification{
		NotifType: notification.NotifMailReceived, ReferenceType: "mail", ReferenceID: 5,
	})
	if got.URL != "/mail?open=5" {
		t.Fatalf("mail_received URL = %q, want /mail?open=5", got.URL)
	}

	// 주소 승인/반려 알림(reference_id=주소 id)은 메일함 홈으로
	got = s.FormatPayload(&notification.Notification{
		NotifType: notification.NotifMailAddressApproved, ReferenceType: "mail", ReferenceID: 3,
	})
	if got.URL != "/mail" {
		t.Fatalf("mail_address_approved URL = %q, want /mail", got.URL)
	}

	// 다른 타입은 기존 일반식 유지
	got = s.FormatPayload(&notification.Notification{
		NotifType: "new_comment", ReferenceType: "post", ReferenceID: 7,
	})
	if got.URL != "/post/7" {
		t.Fatalf("일반 타입 URL = %q, want /post/7", got.URL)
	}
}
