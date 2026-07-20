package integration

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

// TestMailScoping — 메일함은 소유자 스코프. A 는 B 의 메일을 볼 수 없다.
func TestMailScoping(t *testing.T) {
	ts := setupTestServer(t)
	a := ts.registerAndApprove("scopea@student.com", "pw12345678", "스코프에이", "2024301")
	b := ts.registerAndApprove("scopeb@student.com", "pw12345678", "스코프비", "2024302")
	ts.claimMailAddress(t, a, "scopea")
	ts.claimMailAddress(t, b, "scopeb")

	// 각자 1통씩 수신
	ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "x@univ.ac.kr", "to": "scopea@earnlearning.com", "subject": "A메일", "text": "for A",
	})
	ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "y@univ.ac.kr", "to": "scopeb@earnlearning.com", "subject": "B메일", "text": "for B",
	})

	// A 의 받은편지함: 1건 (자기 것만)
	_, ra := ts.mailJSON("GET", "/api/mail?box=inbox", nil, a)
	var listA struct {
		Emails []struct {
			ID      int    `json:"id"`
			Subject string `json:"subject"`
		} `json:"emails"`
		Total int `json:"total"`
	}
	json.Unmarshal(ra.Data, &listA)
	if listA.Total != 1 || listA.Emails[0].Subject != "A메일" {
		t.Fatalf("A 목록은 자기 메일 1건만 보여야 함: %s", string(ra.Data))
	}

	// B 의 받은편지함에서 B 메일 id 확보
	_, rb := ts.mailJSON("GET", "/api/mail?box=inbox", nil, b)
	var listB struct {
		Emails []struct {
			ID int `json:"id"`
		} `json:"emails"`
	}
	json.Unmarshal(rb.Data, &listB)
	bEmailID := listB.Emails[0].ID

	// A 가 B 의 메일 id 로 상세 조회 → 403
	st, r := ts.mailJSON("GET", "/api/mail/"+strconv.Itoa(bEmailID), nil, a)
	if st != http.StatusForbidden || r.Success {
		t.Fatalf("A 가 B 메일 조회 시 403 이어야 함: got %d body=%s", st, string(r.Data))
	}
}
