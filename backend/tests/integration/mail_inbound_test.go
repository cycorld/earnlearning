package integration

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// claimMailAddress — 유저 토큰으로 개인 주소를 발급(pending)한 뒤 관리자 승인까지 마치고,
// 사용 가능한(approved) 주소의 address_id 를 반환한다.
func (ts *testServer) claimMailAddress(t *testing.T, token, localPart string) int {
	t.Helper()
	status, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": localPart}, token)
	if status != http.StatusCreated {
		t.Fatalf("주소 %q 발급 실패: status=%d body=%s", localPart, status, string(r.Data))
	}
	id := ts.adminFindAddressID(t, localPart, "pending")
	ts.approveMailAddressID(t, id)
	return id
}

// adminFindAddressID — 관리자 승인 목록에서 local_part 로 address id 를 찾는다.
func (ts *testServer) adminFindAddressID(t *testing.T, localPart, status string) int {
	t.Helper()
	admin := ts.login(testAdminEmail, testAdminPass)
	_, r := ts.mailJSON("GET", "/api/admin/mail/addresses?status="+status, nil, admin)
	var items []struct {
		ID        int    `json:"id"`
		LocalPart string `json:"local_part"`
	}
	json.Unmarshal(r.Data, &items)
	for _, it := range items {
		if it.LocalPart == localPart {
			return it.ID
		}
	}
	t.Fatalf("관리자 목록(status=%s)에서 %q 를 찾지 못함: %s", status, localPart, string(r.Data))
	return 0
}

// approveMailAddressID — 관리자 권한으로 주소를 승인한다.
func (ts *testServer) approveMailAddressID(t *testing.T, addressID int) {
	t.Helper()
	admin := ts.login(testAdminEmail, testAdminPass)
	status, r := ts.mailJSON("POST", "/api/admin/mail/addresses/"+strconv.Itoa(addressID)+"/approve", nil, admin)
	if status != http.StatusOK {
		t.Fatalf("주소 %d 승인 실패: status=%d body=%s", addressID, status, string(r.Data))
	}
}

// inbound — webhook 호출. secret 이 "" 면 헤더를 붙이지 않는다.
func (ts *testServer) inbound(secret string, payload map[string]interface{}) (int, []byte) {
	headers := map[string]string{}
	if secret != "" {
		headers["X-Mail-Webhook-Secret"] = secret
	}
	return ts.mailReq("POST", "/api/mail/inbound", payload, "", headers)
}

// TestMailInboundAuth — webhook 인증: 시크릿 불일치 401, 비활성(빈 시크릿 서버) 503.
func TestMailInboundAuth(t *testing.T) {
	ts := setupTestServer(t)

	// 시크릿 헤더 없음 → 401
	if status, _ := ts.inbound("", map[string]interface{}{"to": "x@earnlearning.com"}); status != http.StatusUnauthorized {
		t.Fatalf("시크릿 없으면 401 이어야 함: got %d", status)
	}
	// 잘못된 시크릿 → 401
	if status, _ := ts.inbound("wrong-secret", map[string]interface{}{"to": "x@earnlearning.com"}); status != http.StatusUnauthorized {
		t.Fatalf("잘못된 시크릿은 401 이어야 함: got %d", status)
	}

	// 시크릿 미설정 서버 → 503
	tsDisabled := setupTestServer(t, func(c *testConfig) { c.mailWebhookSecret = "" })
	if status, _ := tsDisabled.inbound(testMailWebhookSecret, map[string]interface{}{"to": "x@earnlearning.com"}); status != http.StatusServiceUnavailable {
		t.Fatalf("webhook 미설정 서버는 503 이어야 함: got %d", status)
	}
}

// TestMailInboundUnknownRecipient — 매칭되는 주소가 없으면 404.
func TestMailInboundUnknownRecipient(t *testing.T) {
	ts := setupTestServer(t)
	status, body := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from":    "prof@univ.ac.kr",
		"to":      "nobody@earnlearning.com",
		"subject": "안녕",
		"text":    "테스트",
	})
	if status != http.StatusNotFound {
		t.Fatalf("미지의 수신자는 404 이어야 함: got %d body=%s", status, string(body))
	}
}

// TestMailInboundDelivery — 정상 수신: 저장 + 알림 + 첨부 다운로드 + 권한.
func TestMailInboundDelivery(t *testing.T) {
	ts := setupTestServer(t)
	alice := ts.registerAndApprove("alice@student.com", "pw12345678", "앨리스", "2024201")
	bob := ts.registerAndApprove("bob@student.com", "pw12345678", "밥", "2024202")
	aliceAddr := ts.claimMailAddress(t, alice, "alice")

	attachContent := "hello attachment 첨부내용"
	status, body := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from":       "prof@univ.ac.kr",
		"to":         "이교수 <alice@earnlearning.com>",
		"subject":    "과제 피드백입니다",
		"text":       "안녕하세요 앨리스, 과제 잘 봤어요. 아주 긴 본문을 넣어서 스니펫이 120자에서 잘리는지 확인합니다. " + strings.Repeat("가", 200),
		"message_id": "<orig-msg-123@univ>",
		"attachments": []map[string]string{
			{
				"filename":       "feedback.txt",
				"mime":           "text/plain",
				"content_base64": base64.StdEncoding.EncodeToString([]byte(attachContent)),
			},
		},
	})
	if status != http.StatusCreated {
		t.Fatalf("정상 수신은 201 이어야 함: got %d body=%s", status, string(body))
	}
	var created struct {
		ID int `json:"id"`
	}
	json.Unmarshal(body, &created)
	if created.ID == 0 {
		t.Fatalf("수신 응답에 id 없음: %s", string(body))
	}

	// 받은편지함에 표시
	_, r := ts.mailJSON("GET", "/api/mail?box=inbox&limit=20&offset=0&address_id="+strconv.Itoa(aliceAddr), nil, alice)
	var list struct {
		Emails []struct {
			ID             int    `json:"id"`
			Direction      string `json:"direction"`
			FromAddr       string `json:"from_addr"`
			Subject        string `json:"subject"`
			Snippet        string `json:"snippet"`
			Read           bool   `json:"read"`
			HasAttachments bool   `json:"has_attachments"`
		} `json:"emails"`
		Total int `json:"total"`
	}
	json.Unmarshal(r.Data, &list)
	if list.Total != 1 || len(list.Emails) != 1 {
		t.Fatalf("받은편지함 1건이어야 함: %s", string(r.Data))
	}
	it := list.Emails[0]
	if it.Direction != "in" || !it.HasAttachments {
		t.Fatalf("direction/has_attachments 불일치: %+v", it)
	}
	if len([]rune(it.Snippet)) > 120 {
		t.Fatalf("스니펫은 120자 이하여야 함: len=%d", len([]rune(it.Snippet)))
	}
	if it.Read {
		t.Fatalf("목록에서는 아직 안 읽음 상태여야 함")
	}

	emailID := it.ID

	// 상세 조회: 본문 + 첨부(경로 노출 금지) + 읽음 처리
	_, r = ts.mailJSON("GET", "/api/mail/"+strconv.Itoa(emailID), nil, alice)
	if !r.Success {
		t.Fatalf("상세 조회 실패: %v", r.Error)
	}
	var detail struct {
		ID          int    `json:"id"`
		BodyText    string `json:"body_text"`
		Read        bool   `json:"read"`
		Attachments []struct {
			ID         int    `json:"id"`
			Filename   string `json:"filename"`
			Mime       string `json:"mime"`
			Size       int    `json:"size"`
			StoredPath string `json:"stored_path"`
		} `json:"attachments"`
	}
	json.Unmarshal(r.Data, &detail)
	if len(detail.Attachments) != 1 {
		t.Fatalf("첨부 1건이어야 함: %s", string(r.Data))
	}
	if detail.Attachments[0].StoredPath != "" {
		t.Fatalf("stored_path 는 노출되면 안 됨: %s", string(r.Data))
	}
	if detail.Attachments[0].Filename != "feedback.txt" {
		t.Fatalf("첨부 파일명 불일치: %q", detail.Attachments[0].Filename)
	}
	attID := detail.Attachments[0].ID

	// 상세 조회 후 목록에서 읽음 처리 확인
	_, r = ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(aliceAddr), nil, alice)
	json.Unmarshal(r.Data, &list)
	if !list.Emails[0].Read {
		t.Fatalf("상세 조회 후 read=1 이어야 함")
	}

	// 알림 생성 확인
	_, nr := ts.mailJSON("GET", "/api/notifications?type=mail_received", nil, alice)
	var notifs struct {
		Data []struct {
			NotifType     string `json:"notif_type"`
			ReferenceType string `json:"reference_type"`
			ReferenceID   int    `json:"reference_id"`
		} `json:"data"`
	}
	json.Unmarshal(nr.Data, &notifs)
	found := false
	for _, n := range notifs.Data {
		if n.NotifType == "mail_received" && n.ReferenceType == "mail" && n.ReferenceID == emailID {
			found = true
		}
	}
	if !found {
		t.Fatalf("mail_received 알림(reference mail) 이 있어야 함: %s", string(nr.Data))
	}

	// 첨부 다운로드: 소유자 OK
	st, dl := ts.rawGet("/api/mail/attachments/"+strconv.Itoa(attID), alice)
	if st != http.StatusOK || string(dl) != attachContent {
		t.Fatalf("소유자 첨부 다운로드 실패: status=%d body=%q", st, string(dl))
	}
	// 첨부 다운로드: 타인 금지 403
	if st, _ := ts.rawGet("/api/mail/attachments/"+strconv.Itoa(attID), bob); st != http.StatusForbidden {
		t.Fatalf("타인 첨부 다운로드는 403 이어야 함: got %d", st)
	}

	// 상세 조회: 타인 403
	if st, r2 := ts.mailJSON("GET", "/api/mail/"+strconv.Itoa(emailID), nil, bob); st != http.StatusForbidden || r2.Success {
		t.Fatalf("타인 메일 상세는 403 이어야 함: got %d", st)
	}

	// 관리자: 열람 가능
	adminToken := ts.login(testAdminEmail, testAdminPass)
	if st, r2 := ts.mailJSON("GET", "/api/mail/"+strconv.Itoa(emailID), nil, adminToken); st != http.StatusOK || !r2.Success {
		t.Fatalf("관리자는 메일 상세 열람 가능해야 함: got %d body=%s", st, string(r2.Data))
	}

	// 관리자 메일 목록: owner 정보 포함
	_, ar := ts.mailJSON("GET", "/api/admin/mail?limit=50", nil, adminToken)
	if !ar.Success {
		t.Fatalf("관리자 메일 목록 실패: %v", ar.Error)
	}
	var adminList struct {
		Emails []struct {
			ID          int    `json:"id"`
			OwnerUserID int    `json:"owner_user_id"`
			OwnerName   string `json:"owner_name"`
		} `json:"emails"`
		Total int `json:"total"`
	}
	json.Unmarshal(ar.Data, &adminList)
	if adminList.Total < 1 || adminList.Emails[0].OwnerName == "" {
		t.Fatalf("관리자 목록에 owner_name 이 있어야 함: %s", string(ar.Data))
	}
}

// TestMailInboundHeaderFrom — #171: 헤더 From(주소+이름)을 저장하고 목록·상세에 노출한다.
// SES 발송 메일의 봉투 주소는 VERP 일회용 주소라 표시용으로는 헤더 From 이 필요하다.
func TestMailInboundHeaderFrom(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("hdrfrom@student.com", "pw12345678", "헤더프롬", "2024801")
	addrID := ts.claimMailAddress(t, token, "hdrfrombox")

	if st, b := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from":             "010cabc-verp-bounce@mail.earnlearning.com",
		"to":               "hdrfrombox@earnlearning.com",
		"subject":          "헤더 표시",
		"text":             "본문",
		"header_from":      "cyc@earnlearning.com",
		"header_from_name": "최용철",
	}); st != http.StatusCreated {
		t.Fatalf("inbound 실패: %d %s", st, string(b))
	}

	// 목록에 header_from 노출
	_, lr := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil, token)
	var lst struct {
		Emails []struct {
			ID             int    `json:"id"`
			FromAddr       string `json:"from_addr"`
			HeaderFrom     string `json:"header_from"`
			HeaderFromName string `json:"header_from_name"`
		} `json:"emails"`
	}
	json.Unmarshal(lr.Data, &lst)
	if len(lst.Emails) != 1 {
		t.Fatalf("inbox 1건이어야 함: %s", string(lr.Data))
	}
	e := lst.Emails[0]
	if e.HeaderFrom != "cyc@earnlearning.com" || e.HeaderFromName != "최용철" {
		t.Fatalf("목록 header_from 불일치: %+v", e)
	}
	if e.FromAddr != "010cabc-verp-bounce@mail.earnlearning.com" {
		t.Fatalf("봉투 from_addr 은 그대로 보존되어야 함: %+v", e)
	}

	// 상세에도 노출
	_, dr := ts.mailJSON("GET", "/api/mail/"+strconv.Itoa(e.ID), nil, token)
	var det struct {
		HeaderFrom     string `json:"header_from"`
		HeaderFromName string `json:"header_from_name"`
	}
	json.Unmarshal(dr.Data, &det)
	if det.HeaderFrom != "cyc@earnlearning.com" || det.HeaderFromName != "최용철" {
		t.Fatalf("상세 header_from 불일치: %s", string(dr.Data))
	}
}

// TestMailSendStoresHeaderFrom — #171: 발신 시 보낸편지함 기록에도 header_from(자기 주소·이름)을 채운다.
func TestMailSendStoresHeaderFrom(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	token := ts.registerAndApprove("hdrsend@student.com", "pw12345678", "헤더발신", "2024802")
	addrID := ts.claimMailAddress(t, token, "hdrsendbox")

	if st, b := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "x@y.com", "subject": "s", "body_text": "b",
	}, token); st != http.StatusCreated {
		t.Fatalf("send 실패: %d %s", st, string(b.Data))
	}
	_, lr := ts.mailJSON("GET", "/api/mail?box=sent&address_id="+strconv.Itoa(addrID), nil, token)
	var lst struct {
		Emails []struct {
			HeaderFrom     string `json:"header_from"`
			HeaderFromName string `json:"header_from_name"`
		} `json:"emails"`
	}
	json.Unmarshal(lr.Data, &lst)
	if len(lst.Emails) != 1 || lst.Emails[0].HeaderFrom != "hdrsendbox@earnlearning.com" || lst.Emails[0].HeaderFromName != "헤더발신" {
		t.Fatalf("보낸편지함 header_from 불일치: %s", string(lr.Data))
	}
}
