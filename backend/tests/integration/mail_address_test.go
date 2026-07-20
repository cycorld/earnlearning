package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// ============================================================================
// #166 학생별 이메일 수신함 — 공용 테스트 헬퍼
// ============================================================================

// mailReq — 임의 메서드/헤더로 요청을 보내고 (status, raw body) 반환.
// 봉투(envelope)를 벗기지 않은 raw JSON 이 필요한 webhook 검증에 사용.
func (ts *testServer) mailReq(method, path string, body interface{}, token string, headers map[string]string) (int, []byte) {
	ts.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, ts.url(path), r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// mailJSON — 표준 봉투 응답 엔드포인트용. (status, apiResponse) 반환.
func (ts *testServer) mailJSON(method, path string, body interface{}, token string) (int, *apiResponse) {
	ts.t.Helper()
	status, raw := ts.mailReq(method, path, body, token, nil)
	var r apiResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		ts.t.Fatalf("parse %s %s: %v\nbody: %s", method, path, err, string(raw))
	}
	return status, &r
}

// TestMailAddressClaim — 주소 발급 happy path + 조회.
func TestMailAddressClaim(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("jane@student.com", "pw12345678", "제인", "2024001")

	// 발급 전: local_part / email 모두 null
	status, r := ts.mailJSON("GET", "/api/mail/address", nil, token)
	if status != http.StatusOK || !r.Success {
		t.Fatalf("발급 전 조회 실패: status=%d body=%s", status, string(r.Data))
	}
	var before struct {
		LocalPart *string `json:"local_part"`
		Email     *string `json:"email"`
		Status    *string `json:"status"`
	}
	json.Unmarshal(r.Data, &before)
	if before.LocalPart != nil || before.Email != nil || before.Status != nil {
		t.Fatalf("발급 전에는 null 이어야 함: %s", string(r.Data))
	}

	// 발급: 201, status=pending (아직 승인 안 됨)
	status, r = ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "jane99"}, token)
	if status != http.StatusCreated || !r.Success {
		t.Fatalf("주소 발급 실패: status=%d body=%s err=%v", status, string(r.Data), r.Error)
	}
	var after struct {
		LocalPart string `json:"local_part"`
		Email     string `json:"email"`
		Status    string `json:"status"`
	}
	json.Unmarshal(r.Data, &after)
	if after.LocalPart != "jane99" {
		t.Fatalf("local_part 불일치: %q", after.LocalPart)
	}
	if after.Email != "jane99@earnlearning.com" {
		t.Fatalf("email 불일치: %q", after.Email)
	}
	if after.Status != "pending" {
		t.Fatalf("발급 직후 status 는 pending 이어야 함: %q", after.Status)
	}

	// 발급 후 조회: 값 + pending 상태 반환
	status, r = ts.mailJSON("GET", "/api/mail/address", nil, token)
	json.Unmarshal(r.Data, &after)
	if status != http.StatusOK || after.LocalPart != "jane99" || after.Status != "pending" {
		t.Fatalf("발급 후 조회 실패: status=%d body=%s", status, string(r.Data))
	}
}

// TestMailAddressInvalid — 형식 검증 실패는 400.
func TestMailAddressInvalid(t *testing.T) {
	ts := setupTestServer(t)

	cases := []struct {
		name      string
		localPart string
		studentID string
	}{
		{"too_short", "ab", "2024010"},                              // 3자 미만
		{"uppercase", "Jane99", "2024011"},                          // 대문자
		{"consecutive_dot", "ja..ne", "2024012"},                    // 연속 점
		{"leading_symbol", ".jane", "2024013"},                      // 첫 글자 기호
		{"too_long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaX", "2024014"}, // 31자+
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := ts.registerAndApprove(tc.name+"@student.com", "pw12345678", tc.name, tc.studentID)
			status, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": tc.localPart}, token)
			if status != http.StatusBadRequest || r.Success {
				t.Fatalf("%q 는 400 이어야 함: status=%d body=%s", tc.localPart, status, string(r.Data))
			}
		})
	}
}

// TestMailAddressReserved — 예약어는 거부(400).
func TestMailAddressReserved(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("res@student.com", "pw12345678", "예약", "2024099")
	status, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "admin"}, token)
	if status != http.StatusBadRequest || r.Success {
		t.Fatalf("예약어 admin 은 거부되어야 함: status=%d body=%s", status, string(r.Data))
	}
}

// TestMailAddressDuplicate — 다른 유저가 이미 쓰는 local_part 는 409.
func TestMailAddressDuplicate(t *testing.T) {
	ts := setupTestServer(t)
	a := ts.registerAndApprove("dupa@student.com", "pw12345678", "에이", "2024101")
	b := ts.registerAndApprove("dupb@student.com", "pw12345678", "비", "2024102")

	if status, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "shared"}, a); status != http.StatusCreated {
		t.Fatalf("A 발급 실패: status=%d body=%s", status, string(r.Data))
	}
	status, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "shared"}, b)
	if status != http.StatusConflict || r.Success {
		t.Fatalf("중복 local_part 는 409 이어야 함: status=%d body=%s", status, string(r.Data))
	}
}

// TestMailAddressImmutable — 승인된 주소는 변경 불가(409). 승인 전에는 변경 가능하므로
// 먼저 관리자 승인까지 마친 뒤 재발급을 시도한다.
func TestMailAddressImmutable(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.registerAndApprove("immut@student.com", "pw12345678", "불변", "2024103")

	// 발급(pending) → 관리자 승인(approved)
	ts.claimMailAddress(t, token, "first")

	// 승인 후 재발급 시도 → 409 (불변)
	status, r := ts.mailJSON("POST", "/api/mail/address", map[string]string{"local_part": "second"}, token)
	if status != http.StatusConflict || r.Success {
		t.Fatalf("승인된 주소 재발급은 409 이어야 함: status=%d body=%s", status, string(r.Data))
	}
}
