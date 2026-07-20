package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// ============================================================================
// #166 메일 주소 관리자 승인 플로우 + 멀티 메일함(개인/회사/공용)
// ============================================================================

// --- 공용 테스트 헬퍼 -------------------------------------------------------

type mbItem struct {
	AddressID int    `json:"address_id"`
	Kind      string `json:"kind"`
	CompanyID *int   `json:"company_id"`
	Name      string `json:"name"`
	LocalPart string `json:"local_part"`
	Email     string `json:"email"`
	Status    string `json:"status"`
}

// mailboxes — GET /api/mail/mailboxes 결과 목록.
func (ts *testServer) mailboxes(t *testing.T, token string) []mbItem {
	t.Helper()
	_, r := ts.mailJSON("GET", "/api/mail/mailboxes", nil, token)
	var d struct {
		Mailboxes []mbItem `json:"mailboxes"`
	}
	json.Unmarshal(r.Data, &d)
	return d.Mailboxes
}

// notifCount — 특정 notif_type 알림 개수.
func (ts *testServer) notifCount(t *testing.T, token, notifType string) int {
	t.Helper()
	_, nr := ts.mailJSON("GET", "/api/notifications?type="+notifType, nil, token)
	var n struct {
		Data []struct {
			NotifType string `json:"notif_type"`
		} `json:"data"`
	}
	json.Unmarshal(nr.Data, &n)
	return len(n.Data)
}

// findNotif — 특정 notif_type 첫 알림의 reference 필드를 반환 (없으면 ok=false).
func (ts *testServer) findNotif(t *testing.T, token, notifType string) (refType string, refID int, ok bool) {
	t.Helper()
	_, nr := ts.mailJSON("GET", "/api/notifications?type="+notifType, nil, token)
	var n struct {
		Data []struct {
			NotifType     string `json:"notif_type"`
			ReferenceType string `json:"reference_type"`
			ReferenceID   int    `json:"reference_id"`
		} `json:"data"`
	}
	json.Unmarshal(nr.Data, &n)
	for _, it := range n.Data {
		if it.NotifType == notifType {
			return it.ReferenceType, it.ReferenceID, true
		}
	}
	return "", 0, false
}

// rejectMailAddressID — 관리자 반려.
func (ts *testServer) rejectMailAddressID(t *testing.T, addressID int) (int, *apiResponse) {
	t.Helper()
	admin := ts.login(testAdminEmail, testAdminPass)
	return ts.mailJSON("POST", "/api/admin/mail/addresses/"+strconv.Itoa(addressID)+"/reject", nil, admin)
}

// setupCompanyOwner — 강의실+회사 소유주 학생을 만들고 (token, userID, companyID) 반환.
func (ts *testServer) setupCompanyOwner(t *testing.T, prefix, studentID string) (string, int, int) {
	t.Helper()
	admin := ts.login(testAdminEmail, testAdminPass)
	cr := ts.createClassroom(admin, "메일회사반-"+prefix, 60_000_000)
	uid, token := registerWithID(t, ts, prefix+"@student.com", "회사장"+prefix, studentID)
	if r := ts.joinClassroom(token, cr.Code); !r.Success {
		t.Fatalf("join classroom: %v", r.Error)
	}
	r := ts.post("/api/companies", map[string]interface{}{
		"name": "메일컴퍼니-" + prefix, "description": "x", "initial_capital": 50_000_000, "logo_url": "",
	}, token)
	if !r.Success {
		t.Fatalf("create company: %v", r.Error)
	}
	var c struct {
		ID int `json:"id"`
	}
	json.Unmarshal(r.Data, &c)
	return token, uid, c.ID
}

// --- 개인 주소 승인 플로우 ---------------------------------------------------

// TestMailAddressPendingUnusable — 발급 직후(pending)엔 발신 403 · 수신 404.
func TestMailAddressPendingUnusable(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	token := ts.registerAndApprove("pend@student.com", "pw12345678", "펜딩", "2024501")

	// 발급(pending)
	status, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "pendbox"}, token)
	if status != http.StatusCreated {
		t.Fatalf("발급 실패: status=%d body=%s", status, string(r.Data))
	}
	addrID := ts.adminFindAddressID(t, "pendbox", "pending")

	// 승인 전 발신 → 403
	st, sr := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "prof@univ.ac.kr", "subject": "안녕", "body_text": "본문",
	}, token)
	if st != http.StatusForbidden || sr.Success {
		t.Fatalf("승인 전 발신은 403 이어야 함: got %d body=%s", st, string(sr.Data))
	}

	// 승인 전 수신 → 404 (미지의 수신자)
	if ist, ib := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "prof@univ.ac.kr", "to": "pendbox@earnlearning.com", "subject": "x", "text": "y",
	}); ist != http.StatusNotFound {
		t.Fatalf("승인 전 수신은 404 이어야 함: got %d body=%s", ist, string(ib))
	}
}

// TestMailAddressRerequest — pending/rejected 상태에서 local_part 재요청 가능.
func TestMailAddressRerequest(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("rereq@student.com", "pw12345678", "재요청", "2024502")

	// 최초 요청
	if st, _ := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "firstpick"}, token); st != http.StatusCreated {
		t.Fatalf("최초 요청 실패: %d", st)
	}
	// pending 중 재요청 → local_part 변경, 여전히 pending
	st, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "secondpick"}, token)
	if st != http.StatusCreated || !r.Success {
		t.Fatalf("pending 재요청은 성공해야 함: got %d body=%s", st, string(r.Data))
	}
	var after struct {
		LocalPart string `json:"local_part"`
		Status    string `json:"status"`
	}
	json.Unmarshal(r.Data, &after)
	if after.LocalPart != "secondpick" || after.Status != "pending" {
		t.Fatalf("재요청으로 local_part 변경 + pending 유지되어야 함: %s", string(r.Data))
	}

	// 반려 후 재요청도 가능
	addrID := ts.adminFindAddressID(t, "secondpick", "pending")
	if rst, rr := ts.rejectMailAddressID(t, addrID); rst != http.StatusOK {
		t.Fatalf("반려 실패: %d body=%s", rst, string(rr.Data))
	}
	st, r = ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "thirdpick"}, token)
	if st != http.StatusCreated || !r.Success {
		t.Fatalf("반려 후 재요청은 성공해야 함: got %d body=%s", st, string(r.Data))
	}
	json.Unmarshal(r.Data, &after)
	if after.LocalPart != "thirdpick" || after.Status != "pending" {
		t.Fatalf("반려 후 재요청 결과 불일치: %s", string(r.Data))
	}
}

// TestMailAddressApprove — 승인 시 학생 알림 + 발신/수신 활성화.
func TestMailAddressApprove(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	token := ts.registerAndApprove("appr@student.com", "pw12345678", "승인됨", "2024503")
	addrID := ts.claimMailAddress(t, token, "apprbox") // 발급 + 승인

	// 학생에게 승인 알림 (reference_type=mail, reference_id=주소 id)
	refType, refID, ok := ts.findNotif(t, token, "mail_address_approved")
	if !ok {
		t.Fatalf("mail_address_approved 알림이 있어야 함")
	}
	if refType != "mail" || refID != addrID {
		t.Fatalf("승인 알림 reference 불일치: refType=%q refID=%d (기대 mail/%d)", refType, refID, addrID)
	}

	// 승인 후 발신 성공
	st, sr := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "prof@univ.ac.kr", "subject": "안녕", "body_text": "본문",
	}, token)
	if st != http.StatusCreated || !sr.Success {
		t.Fatalf("승인 후 발신은 201 이어야 함: got %d body=%s", st, string(sr.Data))
	}

	// 승인 후 수신 배달
	if ist, ib := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "prof@univ.ac.kr", "to": "apprbox@earnlearning.com", "subject": "수신", "text": "본문",
	}); ist != http.StatusCreated {
		t.Fatalf("승인 후 수신은 201 이어야 함: got %d body=%s", ist, string(ib))
	}
	if ts.notifCount(t, token, "mail_received") < 1 {
		t.Fatalf("수신 알림(mail_received)이 있어야 함")
	}
}

// TestMailAddressReject — 반려 시 알림 + 주소 사용 불가.
func TestMailAddressReject(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	token := ts.registerAndApprove("rej@student.com", "pw12345678", "반려됨", "2024504")

	if st, _ := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "rejbox"}, token); st != http.StatusCreated {
		t.Fatalf("발급 실패: %d", st)
	}
	addrID := ts.adminFindAddressID(t, "rejbox", "pending")
	if rst, rr := ts.rejectMailAddressID(t, addrID); rst != http.StatusOK {
		t.Fatalf("반려 실패: %d body=%s", rst, string(rr.Data))
	}

	refType, refID, ok := ts.findNotif(t, token, "mail_address_rejected")
	if !ok {
		t.Fatalf("mail_address_rejected 알림이 있어야 함")
	}
	if refType != "mail" || refID != addrID {
		t.Fatalf("반려 알림 reference 불일치: refType=%q refID=%d (기대 mail/%d)", refType, refID, addrID)
	}
	// 반려된 주소는 발신 불가(403)
	if st, sr := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "prof@univ.ac.kr", "subject": "x", "body_text": "y",
	}, token); st != http.StatusForbidden || sr.Success {
		t.Fatalf("반려 주소 발신은 403 이어야 함: got %d body=%s", st, string(sr.Data))
	}
	// 반려된 주소로는 수신도 안 됨(404)
	if ist, _ := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "p@u.kr", "to": "rejbox@earnlearning.com", "subject": "x", "text": "y",
	}); ist != http.StatusNotFound {
		t.Fatalf("반려 주소 수신은 404 이어야 함: got %d", ist)
	}
}

// TestMailAdminAddressAuth — 승인/반려/목록은 관리자 전용(학생 토큰 403).
func TestMailAdminAddressAuth(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("aauth@student.com", "pw12345678", "권한", "2024505")
	if st, _ := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "authbox"}, token); st != http.StatusCreated {
		t.Fatalf("발급 실패: %d", st)
	}
	addrID := ts.adminFindAddressID(t, "authbox", "pending")

	// 학생 토큰으로 관리자 엔드포인트 접근 → 403 (admin 미들웨어)
	if st, _ := ts.mailReq("GET", "/api/admin/mail/addresses?status=pending", nil, token, nil); st != http.StatusForbidden {
		t.Fatalf("학생의 관리자 목록 접근은 403 이어야 함: got %d", st)
	}
	if st, _ := ts.mailReq("POST", "/api/admin/mail/addresses/"+strconv.Itoa(addrID)+"/approve", nil, token, nil); st != http.StatusForbidden {
		t.Fatalf("학생의 승인은 403 이어야 함: got %d", st)
	}
	if st, _ := ts.mailReq("POST", "/api/admin/mail/addresses/"+strconv.Itoa(addrID)+"/reject", nil, token, nil); st != http.StatusForbidden {
		t.Fatalf("학생의 반려는 403 이어야 함: got %d", st)
	}
}

// TestMailApproveRejectConflicts — 승인된 주소 재승인/반려는 409.
func TestMailApproveRejectConflicts(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("conf@student.com", "pw12345678", "충돌", "2024506")
	addrID := ts.claimMailAddress(t, token, "confbox") // 승인 완료

	// 이미 승인 → 재승인 409
	admin := ts.login(testAdminEmail, testAdminPass)
	if st, r := ts.mailJSON("POST", "/api/admin/mail/addresses/"+strconv.Itoa(addrID)+"/approve", nil, admin); st != http.StatusConflict || r.Success {
		t.Fatalf("승인된 주소 재승인은 409 이어야 함: got %d body=%s", st, string(r.Data))
	}
	// 이미 승인 → 반려 409 (승인 취소 불가)
	if st, r := ts.mailJSON("POST", "/api/admin/mail/addresses/"+strconv.Itoa(addrID)+"/reject", nil, admin); st != http.StatusConflict || r.Success {
		t.Fatalf("승인된 주소 반려는 409 이어야 함: got %d body=%s", st, string(r.Data))
	}
}

// TestMailReservedWordExpanded — 확장된 예약어(billing)는 400.
func TestMailReservedWordExpanded(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("resv@student.com", "pw12345678", "예약", "2024507")
	if st, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "billing"}, token); st != http.StatusBadRequest || r.Success {
		t.Fatalf("예약어 billing 은 400 이어야 함: got %d body=%s", st, string(r.Data))
	}
}

// --- 회사 메일함 ------------------------------------------------------------

// TestMailCompanyAddress — 회사 주소 발급→승인→수신 배달 + 소유주 알림 + mailboxes 노출.
func TestMailCompanyAddress(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	ownerToken, _, companyID := ts.setupCompanyOwner(t, "acme", "2024601")

	// 회사 소유주가 회사 주소 발급
	st, r := ts.mailJSON("POST", "/api/companies/"+strconv.Itoa(companyID)+"/mail-address",
		map[string]string{"local_part": "acmebox"}, ownerToken)
	if st != http.StatusCreated || !r.Success {
		t.Fatalf("회사 주소 발급 실패: got %d body=%s", st, string(r.Data))
	}
	addrID := ts.adminFindAddressID(t, "acmebox", "pending")
	ts.approveMailAddressID(t, addrID)

	// mailboxes 에 회사 메일함 노출
	found := false
	for _, m := range ts.mailboxes(t, ownerToken) {
		if m.AddressID == addrID {
			found = true
			if m.Kind != "company" || m.CompanyID == nil || *m.CompanyID != companyID || m.Status != "approved" {
				t.Fatalf("회사 메일함 항목 불일치: %+v", m)
			}
		}
	}
	if !found {
		t.Fatalf("회사 메일함이 mailboxes 에 있어야 함")
	}

	// 회사 주소로 수신 → 회사 소유주 알림 + 소유주가 열람 가능
	if ist, ib := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "client@corp.com", "to": "acmebox@earnlearning.com", "subject": "제휴 문의", "text": "안녕하세요",
	}); ist != http.StatusCreated {
		t.Fatalf("회사 주소 수신은 201 이어야 함: got %d body=%s", ist, string(ib))
	}
	if ts.notifCount(t, ownerToken, "mail_received") < 1 {
		t.Fatalf("회사 수신 → 소유주에게 mail_received 알림이 있어야 함")
	}
	_, lr := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil, ownerToken)
	var list struct {
		Total int `json:"total"`
	}
	json.Unmarshal(lr.Data, &list)
	if list.Total != 1 {
		t.Fatalf("회사 받은편지함 1건이어야 함: %s", string(lr.Data))
	}
}

// TestMailCompanyAddressNonOwner — 비소유주의 회사 주소 발급/열람은 403.
func TestMailCompanyAddressNonOwner(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, _, companyID := ts.setupCompanyOwner(t, "beta", "2024602")

	other := ts.registerAndApprove("otherco@student.com", "pw12345678", "타인", "2024603")

	// 비소유주 발급 → 403
	if st, r := ts.mailJSON("POST", "/api/companies/"+strconv.Itoa(companyID)+"/mail-address",
		map[string]string{"local_part": "betabox"}, other); st != http.StatusForbidden || r.Success {
		t.Fatalf("비소유주 회사 주소 발급은 403 이어야 함: got %d body=%s", st, string(r.Data))
	}

	// 소유주가 정상 발급 + 승인
	if st, _ := ts.mailJSON("POST", "/api/companies/"+strconv.Itoa(companyID)+"/mail-address",
		map[string]string{"local_part": "betabox"}, ownerToken); st != http.StatusCreated {
		t.Fatalf("소유주 발급 실패: %d", st)
	}
	addrID := ts.adminFindAddressID(t, "betabox", "pending")
	ts.approveMailAddressID(t, addrID)

	// 타인이 회사 메일함 목록 조회 → 403
	if st, r := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil, other); st != http.StatusForbidden || r.Success {
		t.Fatalf("타인의 회사 메일함 조회는 403 이어야 함: got %d body=%s", st, string(r.Data))
	}
}

// TestMailPersonalAndCompanyCoexist — 개인 + 회사 메일함 공존 + address_id 필터 정확성.
func TestMailPersonalAndCompanyCoexist(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, _, companyID := ts.setupCompanyOwner(t, "coex", "2024604")

	personalID := ts.claimMailAddress(t, ownerToken, "coexme")
	if st, _ := ts.mailJSON("POST", "/api/companies/"+strconv.Itoa(companyID)+"/mail-address",
		map[string]string{"local_part": "coexco"}, ownerToken); st != http.StatusCreated {
		t.Fatalf("회사 주소 발급 실패: %d", st)
	}
	companyAddrID := ts.adminFindAddressID(t, "coexco", "pending")
	ts.approveMailAddressID(t, companyAddrID)

	// mailboxes 에 개인 + 회사 둘 다
	var kinds = map[string]bool{}
	for _, m := range ts.mailboxes(t, ownerToken) {
		kinds[m.Kind] = true
	}
	if !kinds["user"] || !kinds["company"] {
		t.Fatalf("개인+회사 메일함이 모두 보여야 함: %+v", kinds)
	}

	// 개인·회사 각각에 1통씩 수신
	ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "x@u.kr", "to": "coexme@earnlearning.com", "subject": "개인메일", "text": "p",
	})
	ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "y@u.kr", "to": "coexco@earnlearning.com", "subject": "회사메일", "text": "c",
	})

	// address_id 필터: 개인함엔 개인메일만
	_, pr := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(personalID), nil, ownerToken)
	var pl struct {
		Emails []struct {
			Subject string `json:"subject"`
		} `json:"emails"`
		Total int `json:"total"`
	}
	json.Unmarshal(pr.Data, &pl)
	if pl.Total != 1 || pl.Emails[0].Subject != "개인메일" {
		t.Fatalf("개인함은 개인메일 1건이어야 함: %s", string(pr.Data))
	}
	// 회사함엔 회사메일만
	_, cr := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(companyAddrID), nil, ownerToken)
	var cl struct {
		Emails []struct {
			Subject string `json:"subject"`
		} `json:"emails"`
		Total int `json:"total"`
	}
	json.Unmarshal(cr.Data, &cl)
	if cl.Total != 1 || cl.Emails[0].Subject != "회사메일" {
		t.Fatalf("회사함은 회사메일 1건이어야 함: %s", string(cr.Data))
	}
}

// TestMailCompanySendDisplayName — 회사 주소 발신 시 From 표시명이 회사명.
func TestMailCompanySendDisplayName(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	ownerToken, _, companyID := ts.setupCompanyOwner(t, "disp", "2024605")

	if st, _ := ts.mailJSON("POST", "/api/companies/"+strconv.Itoa(companyID)+"/mail-address",
		map[string]string{"local_part": "dispbox"}, ownerToken); st != http.StatusCreated {
		t.Fatalf("회사 주소 발급 실패: %d", st)
	}
	addrID := ts.adminFindAddressID(t, "dispbox", "pending")
	ts.approveMailAddressID(t, addrID)

	before := len(ts.mailSpy.sent)
	if st, sr := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "client@corp.com", "subject": "회신", "body_text": "감사합니다",
	}, ownerToken); st != http.StatusCreated || !sr.Success {
		t.Fatalf("회사 발신은 201 이어야 함: got %d body=%s", st, string(sr.Data))
	}
	if len(ts.mailSpy.sent) != before+1 {
		t.Fatalf("발신기가 호출되어야 함")
	}
	m := ts.mailSpy.sent[len(ts.mailSpy.sent)-1]
	if !strings.Contains(m.FromDisplay, "메일컴퍼니-disp") || !strings.Contains(m.FromDisplay, "dispbox@earnlearning.com") {
		t.Fatalf("From 표시명은 회사명이어야 함: %q", m.FromDisplay)
	}
}

// --- 공용(shared) 메일함 ----------------------------------------------------

// createShared — 관리자 공용 주소 생성 후 address_id 반환.
func (ts *testServer) createShared(t *testing.T, localPart, displayName string) int {
	t.Helper()
	admin := ts.login(testAdminEmail, testAdminPass)
	st, r := ts.mailJSON("POST", "/api/admin/mail/shared",
		map[string]string{"local_part": localPart, "display_name": displayName}, admin)
	if st != http.StatusCreated || !r.Success {
		t.Fatalf("공용 주소 생성 실패: got %d body=%s", st, string(r.Data))
	}
	var d struct {
		AddressID int    `json:"address_id"`
		Status    string `json:"status"`
	}
	json.Unmarshal(r.Data, &d)
	if d.Status != "approved" {
		t.Fatalf("공용 주소는 생성 즉시 approved 여야 함: %s", string(r.Data))
	}
	return d.AddressID
}

func (ts *testServer) grantShared(t *testing.T, addressID, userID int) {
	t.Helper()
	admin := ts.login(testAdminEmail, testAdminPass)
	if st, r := ts.mailJSON("POST", "/api/admin/mail/shared/"+strconv.Itoa(addressID)+"/grants",
		map[string]int{"user_id": userID}, admin); st != http.StatusOK || !r.Success {
		t.Fatalf("권한 부여 실패: got %d body=%s", st, string(r.Data))
	}
}

// TestMailSharedReservedWordAdminOK — 관리자는 예약어(hello)로 공용 주소 생성 가능, 학생은 400.
func TestMailSharedReservedWordAdminOK(t *testing.T) {
	ts := setupTestServer(t)

	// 관리자: 예약어 hello 로 공용 주소 생성 OK
	ts.createShared(t, "hello", "언러닝 지원팀")

	// 학생: hello 개인 발급은 예약어라 400
	token := ts.registerAndApprove("shres@student.com", "pw12345678", "예약공용", "2024701")
	if st, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "hello"}, token); st != http.StatusBadRequest || r.Success {
		t.Fatalf("학생의 예약어 hello 발급은 400 이어야 함: got %d body=%s", st, string(r.Data))
	}
}

// TestMailSharedGrantRevoke — 권한 부여 시 mailboxes 노출, 회수 시 사라짐 + 접근 403.
func TestMailSharedGrantRevoke(t *testing.T) {
	ts := setupTestServer(t)
	addrID := ts.createShared(t, "supportdesk", "지원데스크")
	uid, token := registerWithID(t, ts, "shgr@student.com", "권한자", "2024702")

	// 부여 전: mailboxes 에 없음, 접근 403
	if hasSharedMailbox(ts.mailboxes(t, token), addrID) {
		t.Fatalf("권한 부여 전에는 공용 메일함이 없어야 함")
	}
	if st, _ := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil, token); st != http.StatusForbidden {
		t.Fatalf("권한 없으면 접근 403 이어야 함: got %d", st)
	}

	// 부여
	ts.grantShared(t, addrID, uid)
	boxes := ts.mailboxes(t, token)
	if !hasSharedMailbox(boxes, addrID) {
		t.Fatalf("권한 부여 후 공용 메일함이 보여야 함: %+v", boxes)
	}
	// 접근 OK
	if st, _ := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil, token); st != http.StatusOK {
		t.Fatalf("권한자 접근은 200 이어야 함: got %d", st)
	}

	// 회수
	admin := ts.login(testAdminEmail, testAdminPass)
	if st, r := ts.mailJSON("POST", "/api/admin/mail/shared/"+strconv.Itoa(addrID)+"/grants/"+strconv.Itoa(uid)+"/revoke", nil, admin); st != http.StatusOK || !r.Success {
		t.Fatalf("권한 회수 실패: got %d body=%s", st, string(r.Data))
	}
	if hasSharedMailbox(ts.mailboxes(t, token), addrID) {
		t.Fatalf("회수 후 공용 메일함이 사라져야 함")
	}
	if st, _ := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil, token); st != http.StatusForbidden {
		t.Fatalf("회수 후 접근은 403 이어야 함: got %d", st)
	}
}

// TestMailSharedInboundFanout — 공용 수신 시 활성 권한자 전원 알림 + 열람, 비권한자 403.
func TestMailSharedInboundFanout(t *testing.T) {
	ts := setupTestServer(t)
	addrID := ts.createShared(t, "teaminbox", "팀 인박스")
	u1, t1 := registerWithID(t, ts, "shf1@student.com", "권한자1", "2024703")
	u2, t2 := registerWithID(t, ts, "shf2@student.com", "권한자2", "2024704")
	_, t3 := registerWithID(t, ts, "shf3@student.com", "비권한자", "2024705")
	ts.grantShared(t, addrID, u1)
	ts.grantShared(t, addrID, u2)

	// 공용 주소로 수신
	ist, ib := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "vendor@corp.com", "to": "teaminbox@earnlearning.com", "subject": "공용수신", "text": "본문",
	})
	if ist != http.StatusCreated {
		t.Fatalf("공용 주소 수신은 201 이어야 함: got %d body=%s", ist, string(ib))
	}

	// 권한자 둘 다 알림 + 열람 가능
	if ts.notifCount(t, t1, "mail_received") < 1 || ts.notifCount(t, t2, "mail_received") < 1 {
		t.Fatalf("권한자 전원에게 mail_received 알림이 있어야 함")
	}
	_, lr := ts.mailJSON("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil, t1)
	var list struct {
		Emails []struct {
			ID int `json:"id"`
		} `json:"emails"`
		Total int `json:"total"`
	}
	json.Unmarshal(lr.Data, &list)
	if list.Total != 1 {
		t.Fatalf("공용 받은편지함 1건이어야 함: %s", string(lr.Data))
	}
	emailID := list.Emails[0].ID
	// u2 도 상세 열람 가능
	if st, r := ts.mailJSON("GET", "/api/mail/"+strconv.Itoa(emailID), nil, t2); st != http.StatusOK || !r.Success {
		t.Fatalf("권한자2 도 열람 가능해야 함: got %d body=%s", st, string(r.Data))
	}
	// 비권한자 열람 403
	if st, _ := ts.mailJSON("GET", "/api/mail/"+strconv.Itoa(emailID), nil, t3); st != http.StatusForbidden {
		t.Fatalf("비권한자 열람은 403 이어야 함: got %d", st)
	}
	// 비권한자 알림 없음
	if ts.notifCount(t, t3, "mail_received") != 0 {
		t.Fatalf("비권한자에게는 알림이 없어야 함")
	}
}

// TestMailAdminListShared — 관리자 공용함 목록: 주소 필드 + grants(활성/회수 모두 포함) 형태 검증.
func TestMailAdminListShared(t *testing.T) {
	ts := setupTestServer(t)
	addrID := ts.createShared(t, "listshared", "목록팀")
	u1, _ := registerWithID(t, ts, "shls1@student.com", "활성권한자", "2024707")
	u2, _ := registerWithID(t, ts, "shls2@student.com", "회수권한자", "2024708")
	ts.grantShared(t, addrID, u1)
	ts.grantShared(t, addrID, u2)

	admin := ts.login(testAdminEmail, testAdminPass)
	// u2 회수 → revoked=true 로 목록에 남아야 함 (삭제 아님)
	if st, _ := ts.mailJSON("POST", "/api/admin/mail/shared/"+strconv.Itoa(addrID)+"/grants/"+strconv.Itoa(u2)+"/revoke", nil, admin); st != http.StatusOK {
		t.Fatalf("회수 실패: %d", st)
	}

	_, r := ts.mailJSON("GET", "/api/admin/mail/shared", nil, admin)
	var items []struct {
		AddressID   int    `json:"address_id"`
		LocalPart   string `json:"local_part"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Grants      []struct {
			UserID   int    `json:"user_id"`
			UserName string `json:"user_name"`
			Revoked  bool   `json:"revoked"`
		} `json:"grants"`
	}
	json.Unmarshal(r.Data, &items)

	var found bool
	for _, it := range items {
		if it.AddressID != addrID {
			continue
		}
		found = true
		if it.LocalPart != "listshared" || it.DisplayName != "목록팀" || it.Email != "listshared@earnlearning.com" {
			t.Fatalf("공용함 주소 필드 불일치: %+v", it)
		}
		var sawActive, sawRevoked bool
		for _, g := range it.Grants {
			if g.UserID == u1 && !g.Revoked && g.UserName == "활성권한자" {
				sawActive = true
			}
			if g.UserID == u2 && g.Revoked {
				sawRevoked = true
			}
		}
		if !sawActive {
			t.Fatalf("활성 grant(u1)가 목록에 있어야 함: %+v", it.Grants)
		}
		if !sawRevoked {
			t.Fatalf("회수된 grant(u2, revoked=true)가 목록에 남아야 함: %+v", it.Grants)
		}
	}
	if !found {
		t.Fatalf("생성한 공용함이 목록에 있어야 함: %s", string(r.Data))
	}
}

// TestMailSharedSendDisplayName — 권한자가 공용 주소로 발신 시 From 표시명이 display_name.
func TestMailSharedSendDisplayName(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	addrID := ts.createShared(t, "outbox", "언러닝 공식")
	uid, token := registerWithID(t, ts, "shsend@student.com", "발신자", "2024706")
	ts.grantShared(t, addrID, uid)

	before := len(ts.mailSpy.sent)
	if st, sr := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "someone@corp.com", "subject": "공지", "body_text": "안내드립니다",
	}, token); st != http.StatusCreated || !sr.Success {
		t.Fatalf("공용 발신은 201 이어야 함: got %d body=%s", st, string(sr.Data))
	}
	if len(ts.mailSpy.sent) != before+1 {
		t.Fatalf("발신기가 호출되어야 함")
	}
	m := ts.mailSpy.sent[len(ts.mailSpy.sent)-1]
	if !strings.Contains(m.FromDisplay, "언러닝 공식") || !strings.Contains(m.FromDisplay, "outbox@earnlearning.com") {
		t.Fatalf("From 표시명은 display_name 이어야 함: %q", m.FromDisplay)
	}
}

// hasSharedMailbox — mailboxes 결과에 특정 공용 address_id 가 있는지.
func hasSharedMailbox(boxes []mbItem, addressID int) bool {
	for _, m := range boxes {
		if m.AddressID == addressID && m.Kind == "shared" {
			return true
		}
	}
	return false
}

// --- 보안 회귀 테스트 -------------------------------------------------------

// mintOAuthToken — userToken 유저가 appOwnerToken 소유 OAuth 앱을 인가하고 access_token 을 발급받는다.
func (ts *testServer) mintOAuthToken(t *testing.T, name, appOwnerToken, userToken string, scopes []string) string {
	t.Helper()
	reg := ts.post("/api/oauth/clients", map[string]interface{}{
		"name": name, "description": "보안테스트앱",
		"redirect_uris": []string{"http://localhost:3000/callback"},
		"scopes":        scopes,
	}, appOwnerToken)
	if !reg.Success {
		t.Fatalf("oauth client 등록 실패: %v", reg.Error)
	}
	var client struct {
		ClientID string `json:"client_id"`
	}
	json.Unmarshal(reg.Data, &client)

	verifier := "mail-security-code-verifier-0123456789abcdefghij"
	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])
	authR := ts.post("/api/oauth/authorize", map[string]interface{}{
		"client_id":             client.ClientID,
		"redirect_uri":          "http://localhost:3000/callback",
		"scopes":                scopes,
		"state":                 "sec",
		"code_challenge":        challenge,
		"code_challenge_method": "S256",
	}, userToken)
	if !authR.Success {
		t.Fatalf("oauth authorize 실패: %v", authR.Error)
	}
	var ad struct {
		Code string `json:"code"`
	}
	json.Unmarshal(authR.Data, &ad)
	tokR := ts.post("/api/oauth/token", map[string]interface{}{
		"grant_type":    "authorization_code",
		"code":          ad.Code,
		"client_id":     client.ClientID,
		"redirect_uri":  "http://localhost:3000/callback",
		"code_verifier": verifier,
	}, "")
	if !tokR.Success {
		t.Fatalf("oauth token 교환 실패: %v", tokR.Error)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(tokR.Data, &tok)
	if tok.AccessToken == "" {
		t.Fatalf("빈 access_token")
	}
	return tok.AccessToken
}

// TestMailRejectsOAuthTokens — 메일함은 내부 전용: OAuth 토큰으로 접근 시 403(OAUTH_FORBIDDEN),
// 단 동일 토큰은 인가된 다른 API(wallet)에서는 정상 동작해야 한다.
func TestMailRejectsOAuthTokens(t *testing.T) {
	ts := setupTestServer(t)
	appOwner := ts.registerAndApprove("mailoauthdev@ewha.ac.kr", "password123", "앱개발자", "2024801")
	user := ts.registerAndApprove("mailoauthuser@ewha.ac.kr", "password123", "메일유저", "2024802")
	addrID := ts.claimMailAddress(t, user, "oauthmailbox") // 1st-party 세션 JWT 로는 정상 발급/승인

	oauthTok := ts.mintOAuthToken(t, "메일보안앱-user", appOwner, user, []string{"read:wallet", "read:profile"})

	// sanity: 인가된 wallet API 는 OAuth 토큰으로 정상 동작
	if r := ts.get("/api/wallet", oauthTok); !r.Success {
		t.Fatalf("sanity: OAuth 토큰으로 wallet 접근은 성공해야 함: %v", r.Error)
	}

	// 메일함 엔드포인트는 OAuth 토큰 거부(403, OAUTH_FORBIDDEN)
	assertOAuthForbidden := func(method, path string, body interface{}) {
		st, raw := ts.mailReq(method, path, body, oauthTok, nil)
		if st != http.StatusForbidden {
			t.Fatalf("%s %s: OAuth 토큰은 403 이어야 함: got %d body=%s", method, path, st, string(raw))
		}
		var r apiResponse
		json.Unmarshal(raw, &r)
		if r.Error == nil || r.Error.Code != "OAUTH_FORBIDDEN" {
			t.Fatalf("%s %s: 403 은 OAUTH_FORBIDDEN 이어야 함: body=%s", method, path, string(raw))
		}
	}
	assertOAuthForbidden("GET", "/api/mail/mailboxes", nil)
	assertOAuthForbidden("GET", "/api/mail/address", nil)
	assertOAuthForbidden("GET", "/api/mail?box=inbox&address_id="+strconv.Itoa(addrID), nil)
	assertOAuthForbidden("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "x@y.com", "subject": "s", "body_text": "b",
	})
}

// TestMailAdminRejectsOAuthTokens — 관리자 메일 API 도 OAuth 토큰 거부.
// 관리자 OAuth 토큰이면 AdminOnly 는 통과하므로 403 이 나오면 그것은 RejectOAuth 때문(코드로 확인).
func TestMailAdminRejectsOAuthTokens(t *testing.T) {
	ts := setupTestServer(t)
	appOwner := ts.registerAndApprove("mailoauthdev2@ewha.ac.kr", "password123", "앱개발자2", "2024803")
	adminSession := ts.login(testAdminEmail, testAdminPass)

	adminOAuth := ts.mintOAuthToken(t, "메일보안앱-admin", appOwner, adminSession, []string{"read:profile"})

	st, raw := ts.mailReq("GET", "/api/admin/mail/addresses?status=pending", nil, adminOAuth, nil)
	if st != http.StatusForbidden {
		t.Fatalf("관리자 OAuth 토큰의 admin 메일 접근은 403 이어야 함: got %d body=%s", st, string(raw))
	}
	var r apiResponse
	json.Unmarshal(raw, &r)
	if r.Error == nil || r.Error.Code != "OAUTH_FORBIDDEN" {
		t.Fatalf("admin 메일 403 은 OAUTH_FORBIDDEN(RejectOAuth) 이어야 함: body=%s", string(raw))
	}
}

// TestMailSendQuotesDisplayName — From 표시명은 RFC 5322 quoted-string 으로 감싸고 CR/LF 제거.
func TestMailSendQuotesDisplayName(t *testing.T) {
	ts := setupTestServer(t)
	ts.mailSpy.enabled = true
	// 공용 주소 display_name 에 따옴표 + CRLF + 각괄호 인젝션 시도를 넣는다.
	addrID := ts.createShared(t, "quotebox", "Evil\r\n\"Ops\", <ceo@earnlearning.com>")
	uid, token := registerWithID(t, ts, "quoteuser@student.com", "발신자", "2024804")
	ts.grantShared(t, addrID, uid)

	before := len(ts.mailSpy.sent)
	if st, sr := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "someone@corp.com", "subject": "공지", "body_text": "본문",
	}, token); st != http.StatusCreated || !sr.Success {
		t.Fatalf("발신은 201 이어야 함: got %d body=%s", st, string(sr.Data))
	}
	if len(ts.mailSpy.sent) != before+1 {
		t.Fatalf("발신기가 호출되어야 함")
	}
	from := ts.mailSpy.sent[len(ts.mailSpy.sent)-1].FromDisplay

	if strings.ContainsAny(from, "\r\n") {
		t.Fatalf("From 에 CR/LF 가 남으면 안 됨: %q", from)
	}
	if !strings.HasPrefix(from, `"`) {
		t.Fatalf("From 표시명은 큰따옴표로 시작해야 함: %q", from)
	}
	if !strings.Contains(from, `\"Ops\"`) {
		t.Fatalf("내부 따옴표는 이스케이프되어야 함: %q", from)
	}
	// 실제 발신 주소는 인용부호 밖 맨 끝의 angle-addr 여야 한다(인젝션된 ceo@ 가 아님).
	if !strings.HasSuffix(from, `<quotebox@earnlearning.com>`) {
		t.Fatalf("실제 발신 주소가 맨 끝 angle-addr 여야 함: %q", from)
	}
}

// TestMailCompanyOwnerChangeNotifiesCurrentOwner — 회사 소유권이 바뀌면 수신 알림은 현재 소유주에게 간다.
func TestMailCompanyOwnerChangeNotifiesCurrentOwner(t *testing.T) {
	ts := setupTestServer(t)
	oldOwnerToken, _, companyID := ts.setupCompanyOwner(t, "handover", "2024805")

	if st, _ := ts.mailJSON("POST", "/api/companies/"+strconv.Itoa(companyID)+"/mail-address",
		map[string]string{"local_part": "handoverbox"}, oldOwnerToken); st != http.StatusCreated {
		t.Fatalf("회사 주소 발급 실패: %d", st)
	}
	addrID := ts.adminFindAddressID(t, "handoverbox", "pending")
	ts.approveMailAddressID(t, addrID)

	newOwnerID, newOwnerToken := registerWithID(t, ts, "newowner@student.com", "새소유주", "2024806")
	// 소유권을 DB에서 직접 이전 (테스트 목적).
	if _, err := ts.db.Exec(`UPDATE companies SET owner_id = ? WHERE id = ?`, newOwnerID, companyID); err != nil {
		t.Fatalf("소유권 이전 실패: %v", err)
	}

	// 회사 주소로 수신
	if ist, ib := ts.inbound(testMailWebhookSecret, map[string]interface{}{
		"from": "client@corp.com", "to": "handoverbox@earnlearning.com", "subject": "이전후수신", "text": "본문",
	}); ist != http.StatusCreated {
		t.Fatalf("수신은 201 이어야 함: got %d body=%s", ist, string(ib))
	}

	// 현재(새) 소유주에게 알림, 이전 소유주에겐 없음
	if ts.notifCount(t, newOwnerToken, "mail_received") < 1 {
		t.Fatalf("새 소유주에게 mail_received 알림이 있어야 함")
	}
	if ts.notifCount(t, oldOwnerToken, "mail_received") != 0 {
		t.Fatalf("이전 소유주에게는 알림이 없어야 함")
	}
}

// TestMailSendDisabledSenderNoStore — 발신기 비활성 시 5xx + 보낸편지함에 아무것도 저장하지 않는다.
func TestMailSendDisabledSenderNoStore(t *testing.T) {
	ts := setupTestServer(t)
	// mailSpy.enabled 기본 false → 비활성 발신기 시뮬레이션
	token := ts.registerAndApprove("disabled@student.com", "pw12345678", "비활성", "2024807")
	addrID := ts.claimMailAddress(t, token, "disabledbox")

	st, r := ts.mailJSON("POST", "/api/mail/send", map[string]interface{}{
		"address_id": addrID, "to": "prof@univ.ac.kr", "subject": "안녕", "body_text": "본문",
	}, token)
	if st != http.StatusServiceUnavailable || r.Success {
		t.Fatalf("비활성 발신기는 503 이어야 함: got %d body=%s", st, string(r.Data))
	}

	// 보낸편지함에 저장되지 않아야 함
	_, br := ts.mailJSON("GET", "/api/mail?box=sent&address_id="+strconv.Itoa(addrID), nil, token)
	var sent struct {
		Total int `json:"total"`
	}
	json.Unmarshal(br.Data, &sent)
	if sent.Total != 0 {
		t.Fatalf("비활성 발신 시 보낸편지함은 비어야 함: total=%d", sent.Total)
	}
}
