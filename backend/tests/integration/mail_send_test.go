package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/earnlearning/backend/internal/infrastructure/email"
)

// spyMailSender — 실제 SES 대신 주입하는 테스트용 발신기(DI seam).
// 발신한 메일을 기록해 스레딩 헤더/From 을 검증한다.
type spyMailSender struct {
	enabled  bool
	failNext bool
	sent     []email.OutgoingMail
}

func (s *spyMailSender) IsEnabled() bool { return s.enabled }

func (s *spyMailSender) SendMailFrom(m email.OutgoingMail) error {
	// 실제 SESService.SendMailFrom 과 동일하게, 비활성 발신기는 조용한 성공 대신 명시적 실패를 낸다.
	if !s.enabled {
		return email.ErrSenderDisabled
	}
	if s.failNext {
		s.failNext = false
		return fmt.Errorf("ses boom")
	}
	s.sent = append(s.sent, m)
	return nil
}

// TestMailSendNoAddress — 주소 미발급 상태에서 발신은 400.
func TestMailSendNoAddress(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("noaddr@student.com", "pw12345678", "무주소", "2024401")

	st, r := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"to": "prof@univ.ac.kr", "subject": "안녕", "body_text": "본문",
	}, token)
	if st != http.StatusBadRequest || r.Success {
		t.Fatalf("주소 없이 발신은 400 이어야 함: got %d body=%s", st, string(r.Data))
	}
}

// TestMailSendReply — 답장 시 원본 message_id 로 스레딩 헤더가 설정되고 보낸편지함에 저장.
func TestMailSendReply(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	alice := ts.registerAndApprove("alice2@student.com", "pw12345678", "앨리스", "2024402")
	aliceAddr := ts.claimMailAddress(t, alice, "alice2")

	// 원본 수신 (message_id 확보)
	ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "prof@univ.ac.kr", "to": "alice2@earnlearning.com",
		"subject": "질문", "text": "안녕하세요", "message_id": "<orig-msg-123@univ>",
	})
	_, r := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(aliceAddr), nil, alice)
	var inbox struct {
		Emails []struct {
			ID int `json:"id"`
		} `json:"emails"`
	}
	json.Unmarshal(r.Data, &inbox)
	origID := inbox.Emails[0].ID

	// 답장 발신
	before := len(ts.mailSpy.sent)
	st, sr := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": aliceAddr, "to": "prof@univ.ac.kr", "subject": "Re: 질문",
		"body_text": "답장 드립니다", "in_reply_to_id": origID,
	}, alice)
	if st != http.StatusCreated || !sr.Success {
		t.Fatalf("답장 발신은 201 이어야 함: got %d body=%s", st, string(sr.Data))
	}
	if len(ts.mailSpy.sent) != before+1 {
		t.Fatalf("SES 발신기가 호출되어야 함")
	}
	m := ts.mailSpy.sent[len(ts.mailSpy.sent)-1]
	if m.InReplyTo != "<orig-msg-123@univ>" {
		t.Fatalf("In-Reply-To 는 원본 message_id 여야 함: %q", m.InReplyTo)
	}
	if !strings.Contains(m.References, "<orig-msg-123@univ>") {
		t.Fatalf("References 에 원본 message_id 포함되어야 함: %q", m.References)
	}
	if m.To != "prof@univ.ac.kr" {
		t.Fatalf("수신자 불일치: %q", m.To)
	}
	if !strings.Contains(m.FromDisplay, "alice2@earnlearning.com") || !strings.Contains(m.FromDisplay, "앨리스") {
		t.Fatalf("From 은 '이름 <local@earnlearning.com>' 형식이어야 함: %q", m.FromDisplay)
	}

	// 보낸편지함에 표시
	_, br := ts.mailJSON("GET", "/api/mail?box=sent&address_id="+strconv.Itoa(aliceAddr), nil, alice)
	var sent struct {
		Emails []struct {
			Direction string `json:"direction"`
			ToAddr    string `json:"to_addr"`
			Subject   string `json:"subject"`
			Read      bool   `json:"read"`
		} `json:"emails"`
		Total int `json:"total"`
	}
	json.Unmarshal(br.Data, &sent)
	if sent.Total != 1 || sent.Emails[0].Direction != "out" || sent.Emails[0].Subject != "Re: 질문" {
		t.Fatalf("보낸편지함에 발신 메일이 있어야 함: %s", string(br.Data))
	}
	if !sent.Emails[0].Read {
		t.Fatalf("발신 메일은 read=1 이어야 함")
	}
}

// TestMailSendSESFailure — SES 실패 시 502, 저장하지 않음.
func TestMailSendSESFailure(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	ts.mailSpy.failNext = true
	alice := ts.registerAndApprove("alice3@student.com", "pw12345678", "앨리스", "2024403")
	aliceAddr := ts.claimMailAddress(t, alice, "alice3")

	st, r := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": aliceAddr, "to": "prof@univ.ac.kr", "subject": "안녕", "body_text": "본문",
	}, alice)
	if st != http.StatusBadGateway || r.Success {
		t.Fatalf("SES 실패는 502 이어야 함: got %d body=%s", st, string(r.Data))
	}

	// 저장되지 않았는지 확인
	_, br := ts.mailJSON("GET", "/api/mail?box=sent&address_id="+strconv.Itoa(aliceAddr), nil, alice)
	var sent struct {
		Total int `json:"total"`
	}
	json.Unmarshal(br.Data, &sent)
	if sent.Total != 0 {
		t.Fatalf("SES 실패 시 보낸편지함은 비어야 함: total=%d", sent.Total)
	}
}
