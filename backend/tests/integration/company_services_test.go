package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/earnlearning/backend/internal/infrastructure/rybbit"
	"github.com/earnlearning/backend/internal/infrastructure/urlcheck"
)

// #181 회사별 등록 서비스 (URL 검증 + Rybbit 연동) 통합 테스트.

// serviceDTO — 서버가 내려주는 ServiceResponse 계약 (프론트엔드와 동결된 키).
type serviceDTO struct {
	ID                  int     `json:"id"`
	CompanyID           int     `json:"company_id"`
	Name                string  `json:"name"`
	URL                 string  `json:"url"`
	ValidationStatus    string  `json:"validation_status"`
	ValidationCheckedAt *string `json:"validation_checked_at"`
	ValidationDetail    string  `json:"validation_detail"`
	RybbitStatus        string  `json:"rybbit_status"`
	RybbitSiteID        string  `json:"rybbit_site_id"`
	RybbitConnectedAt   *string `json:"rybbit_connected_at"`
	ConnectReady        bool    `json:"connect_ready"`
}

// svcJSON — (status, envelope) 가 둘 다 필요한 서비스 API 용 얇은 래퍼.
func (ts *testServer) svcJSON(method, path string, body interface{}, token string) (int, *apiResponse) {
	ts.t.Helper()
	return ts.mailJSON(method, path, body, token)
}

// listServices — GET /companies/:id/services 결과를 파싱해 반환.
func (ts *testServer) listServices(t *testing.T, companyID int, token string) []serviceDTO {
	t.Helper()
	r := ts.get("/api/companies/"+itoaUD(companyID)+"/services", token)
	if !r.Success {
		t.Fatalf("list services: %v", r.Error)
	}
	var out []serviceDTO
	if err := json.Unmarshal(r.Data, &out); err != nil {
		t.Fatalf("parse services list: %v (%s)", err, string(r.Data))
	}
	return out
}

// createService — POST /companies/:id/services. 실패 시 t.Fatalf.
func (ts *testServer) createService(t *testing.T, companyID int, token, name, url string) serviceDTO {
	t.Helper()
	status, r := ts.svcJSON(http.MethodPost, "/api/companies/"+itoaUD(companyID)+"/services",
		map[string]string{"name": name, "url": url}, token)
	if !r.Success {
		t.Fatalf("create service (%d): %v", status, r.Error)
	}
	var out serviceDTO
	if err := json.Unmarshal(r.Data, &out); err != nil {
		t.Fatalf("parse created service: %v (%s)", err, string(r.Data))
	}
	return out
}

// assertErrCode — 에러 응답의 status/code 를 함께 검증.
func assertErrCode(t *testing.T, label string, status int, r *apiResponse, wantStatus int, wantCode string) {
	t.Helper()
	if r.Success {
		t.Fatalf("%s: 성공하면 안 됨 (data=%s)", label, string(r.Data))
	}
	if status != wantStatus {
		t.Errorf("%s: status %d, want %d (error=%+v)", label, status, wantStatus, r.Error)
	}
	if r.Error == nil || r.Error.Code != wantCode {
		t.Errorf("%s: error code %+v, want %s", label, r.Error, wantCode)
	}
}

// TestCompanyServices_CreateAndList_Multiple — 소유자가 2개 등록하면 id 오름차순으로
// 조회되고 초기 상태는 미검증/미연동. 비소유자도 목록은 읽을 수 있다.
func TestCompanyServices_CreateAndList_Multiple(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "svc-owner1@test.com", "서비스주인", "20261001", "svc_co_1")
	otherToken, _ := createUserWithCompany(t, ts, "svc-other1@test.com", "남", "20261002", "svc_co_2")

	first := ts.createService(t, cid, ownerToken, "첫서비스", "https://first.example.com")
	second := ts.createService(t, cid, ownerToken, "둘째서비스", "https://second.example.com/app")

	if first.ID == 0 || second.ID == 0 || first.ID >= second.ID {
		t.Fatalf("id 부여가 이상함: %d, %d", first.ID, second.ID)
	}
	if first.CompanyID != cid {
		t.Errorf("company_id %d, want %d", first.CompanyID, cid)
	}
	if second.URL != "https://second.example.com/app" {
		t.Errorf("정규화 URL %q", second.URL)
	}

	list := ts.listServices(t, cid, ownerToken)
	if len(list) != 2 {
		t.Fatalf("목록 %d개, want 2 (%+v)", len(list), list)
	}
	if list[0].ID != first.ID || list[1].ID != second.ID {
		t.Errorf("id ASC 정렬 아님: %d, %d", list[0].ID, list[1].ID)
	}
	for _, s := range list {
		if s.ValidationStatus != "unvalidated" {
			t.Errorf("초기 validation_status %q, want unvalidated", s.ValidationStatus)
		}
		if s.ValidationCheckedAt != nil {
			t.Errorf("초기 validation_checked_at %v, want null", *s.ValidationCheckedAt)
		}
		if s.RybbitStatus != "not_connected" {
			t.Errorf("초기 rybbit_status %q, want not_connected", s.RybbitStatus)
		}
		if s.RybbitSiteID != "" || s.RybbitConnectedAt != nil {
			t.Errorf("초기 rybbit 연동 정보가 비어야 함: %+v", s)
		}
		if s.ConnectReady {
			t.Errorf("초기 connect_ready 는 false 여야 함: %+v", s)
		}
	}

	// 비소유자도 목록 조회 가능 (공개 정보)
	readByOther := ts.listServices(t, cid, otherToken)
	if len(readByOther) != 2 {
		t.Errorf("비소유자 목록 %d개, want 2", len(readByOther))
	}

	// 없는 회사 → 404
	status, r := ts.svcJSON(http.MethodGet, "/api/companies/999999/services", nil, ownerToken)
	assertErrCode(t, "없는 회사 목록", status, r, http.StatusNotFound, "NOT_FOUND")
}

// TestCompanyServices_Create_DuplicateNormalizedURL — 대소문자/:443/꼬리슬래시만 다른
// URL 은 같은 서비스로 보고 409.
func TestCompanyServices_Create_DuplicateNormalizedURL(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "svc-dup@test.com", "중복", "20261003", "svc_dup_co")

	ts.createService(t, cid, token, "앱", "https://app.example.com")

	status, r := ts.svcJSON(http.MethodPost, "/api/companies/"+itoaUD(cid)+"/services",
		map[string]string{"name": "같은앱", "url": "https://App.EXAMPLE.com:443/"}, token)
	assertErrCode(t, "정규화 중복 등록", status, r, http.StatusConflict, "DUPLICATE_SERVICE_URL")

	if list := ts.listServices(t, cid, token); len(list) != 1 {
		t.Errorf("중복 거부 후 목록 %d개, want 1", len(list))
	}
}

// TestCompanyServices_Create_InvalidURL — https 전용 + 자격증명/프래그먼트 거부.
func TestCompanyServices_Create_InvalidURL(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "svc-bad@test.com", "잘못", "20261004", "svc_bad_co")

	cases := []struct {
		label string
		url   string
	}{
		{"http 평문", "http://plain.com"},
		{"자격증명 포함", "https://user:pw@h.com"},
		{"프래그먼트 포함", "https://h.com/#frag"},
		{"URL 아님", "not a url"},
		{"빈 문자열", ""},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			status, r := ts.svcJSON(http.MethodPost, "/api/companies/"+itoaUD(cid)+"/services",
				map[string]string{"name": "x", "url": tc.url}, token)
			assertErrCode(t, tc.label, status, r, http.StatusBadRequest, "INVALID_SERVICE_URL")
		})
	}

	if list := ts.listServices(t, cid, token); len(list) != 0 {
		t.Errorf("잘못된 URL 이 저장됨: %+v", list)
	}
}

// TestCompanyServices_NotOwner_Forbidden — 소유자 전용 정책 (admin 우회 없음) 회귀.
func TestCompanyServices_NotOwner_Forbidden(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cidA := createUserWithCompany(t, ts, "svc-a@test.com", "에이", "20261005", "svc_a_co")
	otherToken, cidB := createUserWithCompany(t, ts, "svc-b@test.com", "비", "20261006", "svc_b_co")

	svcA := ts.createService(t, cidA, ownerToken, "에이앱", "https://a.example.com")

	base := "/api/companies/" + itoaUD(cidA) + "/services"
	sid := itoaUD(svcA.ID)

	status, r := ts.svcJSON(http.MethodPost, base, map[string]string{"name": "침입", "url": "https://evil.example.com"}, otherToken)
	assertErrCode(t, "비소유자 생성", status, r, http.StatusForbidden, "NOT_OWNER")

	status, r = ts.svcJSON(http.MethodPut, base+"/"+sid, map[string]string{"name": "침입", "url": "https://evil.example.com"}, otherToken)
	assertErrCode(t, "비소유자 수정", status, r, http.StatusForbidden, "NOT_OWNER")

	status, r = ts.svcJSON(http.MethodDelete, base+"/"+sid, nil, otherToken)
	assertErrCode(t, "비소유자 삭제", status, r, http.StatusForbidden, "NOT_OWNER")

	status, r = ts.svcJSON(http.MethodPost, base+"/"+sid+"/validate", nil, otherToken)
	assertErrCode(t, "비소유자 검증", status, r, http.StatusForbidden, "NOT_OWNER")

	status, r = ts.svcJSON(http.MethodPost, base+"/"+sid+"/rybbit/connect", nil, otherToken)
	assertErrCode(t, "비소유자 연동", status, r, http.StatusForbidden, "NOT_OWNER")

	// A 회사의 serviceId 를 B 회사 경로로 접근 → 404 (존재 여부 탐색 방지)
	crossBase := "/api/companies/" + itoaUD(cidB) + "/services/" + sid
	status, r = ts.svcJSON(http.MethodPut, crossBase, map[string]string{"name": "x", "url": "https://x.example.com"}, otherToken)
	assertErrCode(t, "교차 접근 수정", status, r, http.StatusNotFound, "NOT_FOUND")

	status, r = ts.svcJSON(http.MethodDelete, crossBase, nil, otherToken)
	assertErrCode(t, "교차 접근 삭제", status, r, http.StatusNotFound, "NOT_FOUND")

	status, r = ts.svcJSON(http.MethodPost, crossBase+"/validate", nil, otherToken)
	assertErrCode(t, "교차 접근 검증", status, r, http.StatusNotFound, "NOT_FOUND")

	// admin 도 남의 회사 서비스는 못 만든다 (기존 회사 정책과 동일 — admin 우회 없음)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	status, r = ts.svcJSON(http.MethodPost, base, map[string]string{"name": "관리자", "url": "https://admin.example.com"}, adminToken)
	assertErrCode(t, "admin 생성", status, r, http.StatusForbidden, "NOT_OWNER")

	if list := ts.listServices(t, cidA, ownerToken); len(list) != 1 || list[0].Name != "에이앱" {
		t.Errorf("타인 요청으로 상태가 바뀜: %+v", list)
	}
}

// TestCompanyServices_Delete — 소유자 삭제 후 목록에서 사라지고 재삭제는 404.
func TestCompanyServices_Delete(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "svc-del@test.com", "삭제", "20261007", "svc_del_co")

	svc := ts.createService(t, cid, token, "지울앱", "https://delete.example.com")
	path := "/api/companies/" + itoaUD(cid) + "/services/" + itoaUD(svc.ID)

	status, r := ts.svcJSON(http.MethodDelete, path, nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("삭제 실패 (%d): %v", status, r.Error)
	}
	var msg struct {
		Message string `json:"message"`
	}
	json.Unmarshal(r.Data, &msg)
	if msg.Message == "" {
		t.Errorf("삭제 응답 메시지 없음: %s", string(r.Data))
	}

	if list := ts.listServices(t, cid, token); len(list) != 0 {
		t.Errorf("삭제 후에도 목록에 남음: %+v", list)
	}

	status, r = ts.svcJSON(http.MethodDelete, path, nil, token)
	assertErrCode(t, "재삭제", status, r, http.StatusNotFound, "NOT_FOUND")
}

// --- 검증 / 연동 테스트용 헬퍼 (실제 인터넷에 나가지 않는다) ---

// createClassroomOwnerCompany — 강의실에 소속된 학생 + 그 강의실 회사.
// Rybbit 연동은 "활성 강의실 == 회사 강의실" 을 요구하므로 순서가 중요하다:
// 강의실 생성 → 가입(초기 자본 지급) → 회사 설립(설립 시점의 활성 강의실이 회사에 각인됨).
func createClassroomOwnerCompany(t *testing.T, ts *testServer, email, name, studentID, projName string) (string, int, int) {
	t.Helper()
	adminToken := ts.login(testAdminEmail, testAdminPass)
	cr := ts.createClassroom(adminToken, projName+"반", 2_000_000)
	token := ts.registerAndApprove(email, "pass1234", name, studentID)
	if r := ts.joinClassroom(token, cr.Code); !r.Success {
		t.Fatalf("join classroom: %v", r.Error)
	}
	r := ts.post("/api/companies", map[string]interface{}{
		"name":            projName,
		"description":     "테스트 회사 " + projName,
		"initial_capital": 1000000,
		"logo_url":        "",
	}, token)
	if !r.Success {
		t.Fatalf("create company %s: %v (balance=%d)", projName, r.Error, ts.walletBalance(token))
	}
	var c struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(r.Data, &c)
	return token, c.ID, cr.ID
}

// okChecker — 항상 통과하는 체커 (연동 게이트 테스트에서 검증 상태만 만들 때 사용).
type okChecker struct{}

func (okChecker) Check(ctx context.Context, rawURL string) urlcheck.Result {
	return urlcheck.Result{OK: true}
}

// fakeProvisioner — 호출 횟수 / 요청 이력을 기록하는 가짜 Rybbit 프로비저너.
// siteID 가 비어 있으면 서비스별로 "site-<serviceID>" 를 돌려준다 (다중 서비스 테스트용).
type fakeProvisioner struct {
	mu     sync.Mutex
	calls  int
	siteID string
	err    error
	reqs   []rybbit.ProvisionRequest
}

func (f *fakeProvisioner) Provision(ctx context.Context, req rybbit.ProvisionRequest) (*rybbit.ProvisionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	cp := req
	cp.GrantServiceIDs = append([]int{}, req.GrantServiceIDs...)
	f.reqs = append(f.reqs, cp)
	if f.err != nil {
		return nil, f.err
	}
	id := f.siteID
	if id == "" {
		id = "site-" + strconv.Itoa(req.ServiceID)
	}
	granted := make([]int64, 0, len(req.GrantServiceIDs))
	for _, sid := range req.GrantServiceIDs {
		granted = append(granted, int64(sid))
	}
	return &rybbit.ProvisionResult{
		SiteID:         id,
		OrganizationID: "org-" + strconv.Itoa(req.CompanyID),
		GrantedSiteIDs: granted,
	}, nil
}

func (f *fakeProvisioner) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeProvisioner) request() rybbit.ProvisionRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.reqs) == 0 {
		return rybbit.ProvisionRequest{}
	}
	return f.reqs[len(f.reqs)-1]
}

func (f *fakeProvisioner) requests() []rybbit.ProvisionRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]rybbit.ProvisionRequest{}, f.reqs...)
}

// grantIDsEqual — 페이로드의 GrantServiceIDs 가 기대 집합(순서 포함)과 같은지.
func grantIDsEqual(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// fakeDNSChecker — 가짜 DNS + 허용 IP 규칙 + 테스트 인증서를 쓰는 체커.
// hosts 에 없는 호스트는 에러(dns_error) → 어떤 테스트도 실제 DNS 를 타지 않는다.
func fakeDNSChecker(hosts map[string][]net.IP, allow func(net.IP) bool, srv *httptest.Server, timeout time.Duration) *urlcheck.Checker {
	var tlsCfg *tls.Config
	if srv != nil {
		pool := x509.NewCertPool()
		pool.AddCert(srv.Certificate())
		// httptest 자체 서명 인증서는 example.com 용 → 호스트 검증을 그 이름으로 고정.
		tlsCfg = &tls.Config{RootCAs: pool, ServerName: "example.com"}
	}
	return &urlcheck.Checker{
		Timeout:      timeout,
		MaxRedirects: 3,
		LookupIP: func(ctx context.Context, host string) ([]net.IP, error) {
			if ips, ok := hosts[host]; ok {
				return ips, nil
			}
			return nil, errors.New("test dns: unknown host " + host)
		},
		AllowIP:         allow,
		TLSClientConfig: tlsCfg,
	}
}

// testServiceURL — 테스트 TLS 서버를 가리키되 호스트는 가짜 이름(example.com) 인 URL.
// 등록 시점은 네트워크를 타지 않으므로 이 주소로 저장할 수 있고,
// 검증 시점에는 주입된 가짜 DNS 가 루프백으로 해석한다.
func testServiceURL(t *testing.T, srv *httptest.Server, path string) string {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}
	return "https://example.com:" + u.Port() + path
}

// validateService — POST .../validate 후 응답 파싱 (검사 완료면 결과가 invalid 여도 200).
func (ts *testServer) validateService(t *testing.T, companyID, serviceID int, token string) serviceDTO {
	t.Helper()
	status, r := ts.svcJSON(http.MethodPost,
		"/api/companies/"+itoaUD(companyID)+"/services/"+itoaUD(serviceID)+"/validate", nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("validate service (%d): %v", status, r.Error)
	}
	var out serviceDTO
	if err := json.Unmarshal(r.Data, &out); err != nil {
		t.Fatalf("parse validated service: %v (%s)", err, string(r.Data))
	}
	return out
}

// connectPath — Rybbit 연동 엔드포인트 경로.
func connectPath(companyID, serviceID int) string {
	return "/api/companies/" + itoaUD(companyID) + "/services/" + itoaUD(serviceID) + "/rybbit/connect"
}

// TestCompanyServices_Validate_PrivateAndSSRF — 사설 IP 로 해석되는 주소는 invalid/private_ip,
// DNS 해석 실패는 invalid/dns_error 로 기록된다 (서버가 내부망에 요청하지 않는다).
func TestCompanyServices_Validate_PrivateAndSSRF(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "svc-ssrf@test.com", "에스에스", "20261008", "svc_ssrf_co")

	ts.companyServiceUC.SetChecker(fakeDNSChecker(map[string][]net.IP{
		"private.example.com": {net.ParseIP("10.0.0.1")},
	}, urlcheck.PublicIPOnly, nil, 2*time.Second))

	privateSvc := ts.createService(t, cid, token, "내부망", "https://private.example.com")
	got := ts.validateService(t, cid, privateSvc.ID, token)
	if got.ValidationStatus != "invalid" || got.ValidationDetail != "private_ip" {
		t.Errorf("사설 IP: status=%q detail=%q, want invalid/private_ip", got.ValidationStatus, got.ValidationDetail)
	}
	if got.ValidationCheckedAt == nil {
		t.Error("검사 시각이 기록되지 않음")
	}
	if got.ConnectReady {
		t.Error("검증 실패인데 connect_ready 가 true")
	}

	unknownSvc := ts.createService(t, cid, token, "없는도메인", "https://nowhere.example.com")
	got = ts.validateService(t, cid, unknownSvc.ID, token)
	if got.ValidationStatus != "invalid" || got.ValidationDetail != "dns_error" {
		t.Errorf("DNS 실패: status=%q detail=%q, want invalid/dns_error", got.ValidationStatus, got.ValidationDetail)
	}
	if got.ValidationCheckedAt == nil {
		t.Error("검사 시각이 기록되지 않음 (dns_error)")
	}
}

// TestCompanyServices_Validate_RedirectSSRF — 리다이렉트로 우회하는 SSRF 를 막는다.
func TestCompanyServices_Validate_RedirectSSRF(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "svc-redir@test.com", "리다", "20261009", "svc_redir_co")

	mux := http.NewServeMux()
	// https → http 다운그레이드
	mux.HandleFunc("/to-http", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/plain", http.StatusFound)
	})
	// 사설 IP 로 해석되는 호스트로 이동
	mux.HandleFunc("/to-private", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.test/", http.StatusFound)
	})
	// 끝없는 리다이렉트 체인
	mux.HandleFunc("/r", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		http.Redirect(w, r, "/r?n="+strconv.Itoa(n+1), http.StatusFound)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	ts.companyServiceUC.SetChecker(fakeDNSChecker(map[string][]net.IP{
		"example.com": {net.ParseIP("127.0.0.1")},
		"evil.test":   {net.ParseIP("10.0.0.1")},
	}, func(ip net.IP) bool { return ip.IsLoopback() }, srv, 3*time.Second))

	cases := []struct {
		label string
		path  string
		want  []string
	}{
		{"https→http 다운그레이드", "/to-http", []string{"redirect_not_https"}},
		{"사설 IP 로 리다이렉트", "/to-private", []string{"private_ip", "redirect_blocked"}},
		{"리다이렉트 과다", "/r?n=0", []string{"too_many_redirects"}},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			svc := ts.createService(t, cid, token, tc.label, testServiceURL(t, srv, tc.path))
			got := ts.validateService(t, cid, svc.ID, token)
			if got.ValidationStatus != "invalid" {
				t.Fatalf("%s: status=%q detail=%q, want invalid", tc.label, got.ValidationStatus, got.ValidationDetail)
			}
			ok := false
			for _, w := range tc.want {
				if got.ValidationDetail == w {
					ok = true
				}
			}
			if !ok {
				t.Errorf("%s: detail=%q, want one of %v", tc.label, got.ValidationDetail, tc.want)
			}
			if got.ValidationCheckedAt == nil {
				t.Error("검사 시각이 기록되지 않음")
			}
		})
	}
}

// TestCompanyServices_Validate_TimeoutAndStatus — 타임아웃 / 정상 / HTTP 오류 상태 기록.
func TestCompanyServices_Validate_TimeoutAndStatus(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "svc-status@test.com", "상태", "20261010", "svc_status_co")

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		// 서버 종료가 막히지 않도록 요청 취소도 함께 기다린다.
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})
	mux.HandleFunc("/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	ts.companyServiceUC.SetChecker(fakeDNSChecker(map[string][]net.IP{
		"example.com": {net.ParseIP("127.0.0.1")},
	}, func(ip net.IP) bool { return ip.IsLoopback() }, srv, 500*time.Millisecond))

	slow := ts.createService(t, cid, token, "느림", testServiceURL(t, srv, "/slow"))
	if got := ts.validateService(t, cid, slow.ID, token); got.ValidationDetail != "timeout" {
		t.Errorf("느린 서버: status=%q detail=%q, want timeout", got.ValidationStatus, got.ValidationDetail)
	}

	missing := ts.createService(t, cid, token, "404", testServiceURL(t, srv, "/missing"))
	if got := ts.validateService(t, cid, missing.ID, token); got.ValidationDetail != "http_status_404" {
		t.Errorf("404 서버: status=%q detail=%q, want http_status_404", got.ValidationStatus, got.ValidationDetail)
	}

	good := ts.createService(t, cid, token, "정상", testServiceURL(t, srv, "/ok"))
	got := ts.validateService(t, cid, good.ID, token)
	if got.ValidationStatus != "valid" || got.ValidationDetail != "" {
		t.Fatalf("정상 서버: status=%q detail=%q, want valid/빈 문자열", got.ValidationStatus, got.ValidationDetail)
	}
	if !got.ConnectReady {
		t.Error("검증 직후 connect_ready 가 false")
	}

	// 목록 조회에서도 connect_ready 가 유지돼야 한다 (저장값이 아니라 매번 계산).
	for _, s := range ts.listServices(t, cid, token) {
		if s.ID == good.ID && !s.ConnectReady {
			t.Error("목록의 connect_ready 가 false")
		}
		if s.ID == slow.ID && s.ConnectReady {
			t.Error("타임아웃 서비스의 connect_ready 가 true")
		}
	}
}

// TestCompanyServices_Connect_RequiresValidation — 검증 안 된 서비스는 연동 불가.
func TestCompanyServices_Connect_RequiresValidation(t *testing.T) {
	ts := setupTestServer(t)
	token, cid, _ := createClassroomOwnerCompany(t, ts, "svc-req@test.com", "검증필요", "20261011", "svc_req_co")

	fake := &fakeProvisioner{siteID: "site-req"}
	ts.companyServiceUC.SetProvisioner(fake)

	svc := ts.createService(t, cid, token, "미검증앱", "https://req.example.com")

	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "미검증 연동", status, r, http.StatusConflict, "VALIDATION_REQUIRED")

	// 검증 실패(invalid) 여도 여전히 연동 불가
	ts.companyServiceUC.SetChecker(fakeDNSChecker(map[string][]net.IP{
		"req.example.com": {net.ParseIP("10.0.0.1")},
	}, urlcheck.PublicIPOnly, nil, 2*time.Second))
	if got := ts.validateService(t, cid, svc.ID, token); got.ValidationStatus != "invalid" {
		t.Fatalf("검증 실패 상태를 만들지 못함: %+v", got)
	}

	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "검증실패 연동", status, r, http.StatusConflict, "VALIDATION_REQUIRED")

	if fake.callCount() != 0 {
		t.Errorf("프로비저너가 호출됨: %d회", fake.callCount())
	}
}

// TestCompanyServices_Connect_StaleValidation — 검증한 지 24시간이 지나면 재검증 요구.
func TestCompanyServices_Connect_StaleValidation(t *testing.T) {
	ts := setupTestServer(t)
	token, cid, _ := createClassroomOwnerCompany(t, ts, "svc-stale@test.com", "만료", "20261012", "svc_stale_co")

	fake := &fakeProvisioner{siteID: "site-stale"}
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	svc := ts.createService(t, cid, token, "만료앱", "https://stale.example.com")
	if got := ts.validateService(t, cid, svc.ID, token); got.ValidationStatus != "valid" {
		t.Fatalf("검증 통과 상태를 만들지 못함: %+v", got)
	}

	if _, err := ts.db.Exec(
		"UPDATE company_services SET validation_checked_at = datetime('now','-25 hours') WHERE id = ?",
		svc.ID); err != nil {
		t.Fatalf("검증 시각 조작: %v", err)
	}

	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "만료된 검증", status, r, http.StatusConflict, "VALIDATION_STALE")

	for _, s := range ts.listServices(t, cid, token) {
		if s.ID == svc.ID && s.ConnectReady {
			t.Error("만료된 검증인데 connect_ready 가 true")
		}
	}
	if fake.callCount() != 0 {
		t.Errorf("프로비저너가 호출됨: %d회", fake.callCount())
	}
}

// TestCompanyServices_Connect_IdempotentSuccess — 연동 성공 후 재호출은 외부 호출 없이 200.
func TestCompanyServices_Connect_IdempotentSuccess(t *testing.T) {
	ts := setupTestServer(t)
	token, cid, _ := createClassroomOwnerCompany(t, ts, "svc-idem@test.com", "멱등", "20261013", "svc_idem_co")

	fake := &fakeProvisioner{siteID: "site-1"}
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	svc := ts.createService(t, cid, token, "멱등앱", "https://idem.example.com/app")
	ts.validateService(t, cid, svc.ID, token)

	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("연동 실패 (%d): %v", status, r.Error)
	}
	var got serviceDTO
	json.Unmarshal(r.Data, &got)
	if got.RybbitStatus != "connected" || got.RybbitSiteID != "site-1" || got.RybbitConnectedAt == nil {
		t.Fatalf("연동 결과가 반영되지 않음: %+v", got)
	}

	req := fake.request()
	if req.CompanyID != cid || req.ServiceID != svc.ID || req.OwnerUserID == 0 {
		t.Errorf("프로비저너 요청 식별자: %+v (companyID=%d serviceID=%d)", req, cid, svc.ID)
	}
	if req.Domain != "idem.example.com" {
		t.Errorf("프로비저너 요청 도메인: %+v", req)
	}
	if req.OwnerEmail != "svc-idem@test.com" || req.OwnerName != "멱등" || req.CompanyName == "" {
		t.Errorf("프로비저너 소유자/회사 매핑: %+v", req)
	}
	if !grantIDsEqual(req.GrantServiceIDs, []int{svc.ID}) {
		t.Errorf("첫 연동 grants = %v, want [%d] (클릭한 서비스 포함)", req.GrantServiceIDs, svc.ID)
	}

	// 재호출 → 외부 호출 없이 같은 상태
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("재연동 실패 (%d): %v", status, r.Error)
	}
	var again serviceDTO
	json.Unmarshal(r.Data, &again)
	if again.RybbitSiteID != "site-1" || again.RybbitStatus != "connected" {
		t.Errorf("재연동 후 상태가 바뀜: %+v", again)
	}
	if fake.callCount() != 1 {
		t.Errorf("프로비저너 호출 %d회, want 1 (멱등)", fake.callCount())
	}
}

// TestCompanyServices_Connect_FailClosed — 외부 실패/미설정 시 상태를 절대 바꾸지 않는다.
func TestCompanyServices_Connect_FailClosed(t *testing.T) {
	ts := setupTestServer(t)
	token, cid, _ := createClassroomOwnerCompany(t, ts, "svc-fail@test.com", "실패", "20261014", "svc_fail_co")

	ts.companyServiceUC.SetChecker(okChecker{})
	failing := &fakeProvisioner{err: errors.New("rybbit: create site failed with status 500")}
	ts.companyServiceUC.SetProvisioner(failing)

	svc := ts.createService(t, cid, token, "실패앱", "https://fail.example.com")
	ts.validateService(t, cid, svc.ID, token)

	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "프로비저너 실패", status, r, http.StatusBadGateway, "RYBBIT_CONNECT_FAILED")

	assertNotConnected := func(label string) {
		t.Helper()
		for _, s := range ts.listServices(t, cid, token) {
			if s.ID != svc.ID {
				continue
			}
			if s.RybbitStatus != "not_connected" || s.RybbitSiteID != "" || s.RybbitConnectedAt != nil {
				t.Errorf("%s: 상태가 바뀜 %+v", label, s)
			}
		}
	}
	assertNotConnected("프로비저너 실패 후")

	// 미설정(Noop) → 503, 상태 변화 없음
	ts.companyServiceUC.SetProvisioner(rybbit.NewNoop())
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "미설정 연동", status, r, http.StatusServiceUnavailable, "RYBBIT_NOT_CONFIGURED")
	assertNotConnected("미설정 응답 후")
}

// TestCompanyServices_Edit_InvalidatesAndDrifts — URL 수정 → 검증 초기화 + 재연동 필요.
func TestCompanyServices_Edit_InvalidatesAndDrifts(t *testing.T) {
	ts := setupTestServer(t)
	token, cid, _ := createClassroomOwnerCompany(t, ts, "svc-drift@test.com", "드리프트", "20261015", "svc_drift_co")

	fake := &fakeProvisioner{siteID: "site-drift"}
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	svc := ts.createService(t, cid, token, "드리프트앱", "https://drift.example.com")
	ts.validateService(t, cid, svc.ID, token)

	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	if !r.Success {
		t.Fatalf("최초 연동 실패 (%d): %v", status, r.Error)
	}

	// URL 변경 → 검증 초기화 + connected → needs_reconnect
	path := "/api/companies/" + itoaUD(cid) + "/services/" + itoaUD(svc.ID)
	status, r = ts.svcJSON(http.MethodPut, path,
		map[string]string{"name": "드리프트앱", "url": "https://drift2.example.com"}, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("URL 수정 실패 (%d): %v", status, r.Error)
	}
	var edited serviceDTO
	json.Unmarshal(r.Data, &edited)
	if edited.URL != "https://drift2.example.com" {
		t.Errorf("URL 이 반영되지 않음: %q", edited.URL)
	}
	if edited.ValidationStatus != "unvalidated" || edited.ValidationCheckedAt != nil || edited.ValidationDetail != "" {
		t.Errorf("검증 상태가 초기화되지 않음: %+v", edited)
	}
	if edited.RybbitStatus != "needs_reconnect" {
		t.Errorf("rybbit_status %q, want needs_reconnect", edited.RybbitStatus)
	}
	if edited.ConnectReady {
		t.Error("URL 수정 직후 connect_ready 가 true")
	}

	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "URL 수정 후 연동", status, r, http.StatusConflict, "VALIDATION_REQUIRED")
	if fake.callCount() != 1 {
		t.Fatalf("프로비저너 호출 %d회, want 1", fake.callCount())
	}

	// 재검증 → 재연동 성공
	ts.validateService(t, cid, svc.ID, token)
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("재연동 실패 (%d): %v", status, r.Error)
	}
	var reconnected serviceDTO
	json.Unmarshal(r.Data, &reconnected)
	if reconnected.RybbitStatus != "connected" || reconnected.RybbitSiteID != "site-drift" {
		t.Errorf("재연동 결과: %+v", reconnected)
	}
	if fake.callCount() != 2 {
		t.Errorf("프로비저너 호출 %d회, want 2", fake.callCount())
	}

	// 이름만 바꾸면 검증/연동 상태는 그대로
	status, r = ts.svcJSON(http.MethodPut, path,
		map[string]string{"name": "이름만변경", "url": "https://drift2.example.com"}, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("이름 수정 실패 (%d): %v", status, r.Error)
	}
	var renamed serviceDTO
	json.Unmarshal(r.Data, &renamed)
	if renamed.Name != "이름만변경" {
		t.Errorf("이름이 반영되지 않음: %q", renamed.Name)
	}
	if renamed.ValidationStatus != "valid" || renamed.ValidationCheckedAt == nil {
		t.Errorf("이름 변경이 검증 상태를 건드림: %+v", renamed)
	}
	if renamed.RybbitStatus != "connected" || renamed.RybbitSiteID != "site-drift" {
		t.Errorf("이름 변경이 연동 상태를 건드림: %+v", renamed)
	}
}

// TestCompanyServices_Connect_ClassroomGate — 강의실 소속이 아니거나 다른 강의실 회사면 연동 불가.
func TestCompanyServices_Connect_ClassroomGate(t *testing.T) {
	ts := setupTestServer(t)

	// (1) 강의실 무소속 학생 → NO_CLASSROOM
	noClassToken, noClassCID := createUserWithCompany(t, ts, "svc-nocls@test.com", "무소속", "20261016", "svc_nocls_co")
	noClassSvc := ts.createService(t, noClassCID, noClassToken, "무소속앱", "https://nocls.example.com")
	status, r := ts.svcJSON(http.MethodPost, connectPath(noClassCID, noClassSvc.ID), nil, noClassToken)
	assertErrCode(t, "강의실 무소속", status, r, http.StatusForbidden, "NO_CLASSROOM")

	// (2) 회사가 다른 강의실에 속한 경우 → WRONG_CLASSROOM
	token, cid, classroomID := createClassroomOwnerCompany(t, ts, "svc-wrongcls@test.com", "다른반", "20261017", "svc_wrongcls_co")
	svc := ts.createService(t, cid, token, "다른반앱", "https://wrongcls.example.com")
	if _, err := ts.db.Exec("UPDATE companies SET classroom_id = ? WHERE id = ?", classroomID+999, cid); err != nil {
		t.Fatalf("회사 강의실 조작: %v", err)
	}
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "다른 강의실 회사", status, r, http.StatusForbidden, "WRONG_CLASSROOM")
}

// --- #181 HMAC 프로비저닝 계약: 누적 grants / 교차 회사 배제 / 자격 게이트 ---

// TestCompanyServices_Connect_CumulativeGrants — Rybbit 은 grants 를 누적이 아니라
// "재조정(reconcile)" 하므로, 두 번째 서비스 연동 때 첫 서비스 키를 함께 보내야
// 기존 접근이 유지된다. 클릭한 서비스는 항상 포함, 집합은 id 오름차순.
func TestCompanyServices_Connect_CumulativeGrants(t *testing.T) {
	ts := setupTestServer(t)
	token, cid, _ := createClassroomOwnerCompany(t, ts, "svc-cum@test.com", "누적", "20261018", "svc_cum_co")

	fake := &fakeProvisioner{} // siteID 비움 → 서비스별 site-<id> 응답
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	a := ts.createService(t, cid, token, "에이", "https://cum-a.example.com")
	b := ts.createService(t, cid, token, "비", "https://cum-b.example.com")
	ts.validateService(t, cid, a.ID, token)
	ts.validateService(t, cid, b.ID, token)

	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, a.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("A 연동 실패 (%d): %v", status, r.Error)
	}
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, b.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("B 연동 실패 (%d): %v", status, r.Error)
	}

	reqs := fake.requests()
	if len(reqs) != 2 {
		t.Fatalf("프로비저너 호출 %d회, want 2", len(reqs))
	}
	if !grantIDsEqual(reqs[0].GrantServiceIDs, []int{a.ID}) {
		t.Errorf("1차 grants = %v, want [%d]", reqs[0].GrantServiceIDs, a.ID)
	}
	if !grantIDsEqual(reqs[1].GrantServiceIDs, []int{a.ID, b.ID}) {
		t.Errorf("2차 grants = %v, want [%d %d] (이전 연동 포함 전체 집합)", reqs[1].GrantServiceIDs, a.ID, b.ID)
	}

	// 두 번째 클릭 후에도 첫 서비스 연동이 보존된다.
	for _, s := range ts.listServices(t, cid, token) {
		wantSite := "site-" + itoaUD(s.ID)
		if s.RybbitStatus != "connected" || s.RybbitSiteID != wantSite {
			t.Errorf("서비스 %d 상태 %q/%q, want connected/%s", s.ID, s.RybbitStatus, s.RybbitSiteID, wantSite)
		}
	}

	// 이미 연동된 B 재클릭 → 외부 재조정 호출 없음 (접근 집합 그대로).
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, b.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("B 재클릭 실패 (%d): %v", status, r.Error)
	}
	if fake.callCount() != 2 {
		t.Errorf("재클릭 후 호출 %d회, want 2 (멱등)", fake.callCount())
	}
}

// TestCompanyServices_Connect_DriftKeepsPriorGrant — needs_reconnect(드리프트) 서비스도
// Rybbit 사이트 매핑(키)이 살아 있으므로, 다른 서비스 연동 시 grants 에 포함되어야
// 기존 접근이 회수되지 않는다.
func TestCompanyServices_Connect_DriftKeepsPriorGrant(t *testing.T) {
	ts := setupTestServer(t)
	token, cid, _ := createClassroomOwnerCompany(t, ts, "svc-dkeep@test.com", "드보존", "20261019", "svc_dkeep_co")

	fake := &fakeProvisioner{}
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	a := ts.createService(t, cid, token, "에이", "https://dkeep-a.example.com")
	ts.validateService(t, cid, a.ID, token)
	if status, r := ts.svcJSON(http.MethodPost, connectPath(cid, a.ID), nil, token); !r.Success {
		t.Fatalf("A 연동 실패 (%d): %v", status, r.Error)
	}

	// A URL 수정 → needs_reconnect (RybbitSiteID 는 유지됨)
	path := "/api/companies/" + itoaUD(cid) + "/services/" + itoaUD(a.ID)
	if status, r := ts.svcJSON(http.MethodPut, path,
		map[string]string{"name": "에이", "url": "https://dkeep-a2.example.com"}, token); !r.Success {
		t.Fatalf("A URL 수정 실패 (%d): %v", status, r.Error)
	}

	b := ts.createService(t, cid, token, "비", "https://dkeep-b.example.com")
	ts.validateService(t, cid, b.ID, token)
	if status, r := ts.svcJSON(http.MethodPost, connectPath(cid, b.ID), nil, token); !r.Success {
		t.Fatalf("B 연동 실패 (%d): %v", status, r.Error)
	}

	last := fake.request()
	if !grantIDsEqual(last.GrantServiceIDs, []int{a.ID, b.ID}) {
		t.Errorf("드리프트 A 포함 grants = %v, want [%d %d]", last.GrantServiceIDs, a.ID, b.ID)
	}
}

// TestCompanyServices_Connect_CrossCompanyExclusion — 다른 회사의 서비스 키는
// 절대 grants 에 섞이지 않는다 (회사 범위 조회만 사용).
func TestCompanyServices_Connect_CrossCompanyExclusion(t *testing.T) {
	ts := setupTestServer(t)
	tokenA, cidA, _ := createClassroomOwnerCompany(t, ts, "svc-xa@test.com", "엑스에이", "20261020", "svc_xa_co")
	tokenB, cidB, _ := createClassroomOwnerCompany(t, ts, "svc-xb@test.com", "엑스비", "20261021", "svc_xb_co")

	fake := &fakeProvisioner{}
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	connectOK := func(token string, cid, sid int) {
		t.Helper()
		if status, r := ts.svcJSON(http.MethodPost, connectPath(cid, sid), nil, token); !r.Success {
			t.Fatalf("연동 실패 (%d): %v", status, r.Error)
		}
	}

	a1 := ts.createService(t, cidA, tokenA, "에이1", "https://xa-1.example.com")
	ts.validateService(t, cidA, a1.ID, tokenA)
	connectOK(tokenA, cidA, a1.ID)

	// B 회사 서비스 id 가 A 회사 서비스 id 사이에 끼도록 순서 배치 → 섞이면 바로 드러난다.
	b1 := ts.createService(t, cidB, tokenB, "비1", "https://xb-1.example.com")
	ts.validateService(t, cidB, b1.ID, tokenB)
	connectOK(tokenB, cidB, b1.ID)

	a2 := ts.createService(t, cidA, tokenA, "에이2", "https://xa-2.example.com")
	ts.validateService(t, cidA, a2.ID, tokenA)
	connectOK(tokenA, cidA, a2.ID)

	reqs := fake.requests()
	if len(reqs) != 3 {
		t.Fatalf("프로비저너 호출 %d회, want 3", len(reqs))
	}
	if reqs[1].CompanyID != cidB || !grantIDsEqual(reqs[1].GrantServiceIDs, []int{b1.ID}) {
		t.Errorf("B 회사 요청에 다른 회사 키가 섞임: company=%d grants=%v, want company=%d grants=[%d]",
			reqs[1].CompanyID, reqs[1].GrantServiceIDs, cidB, b1.ID)
	}
	if reqs[2].CompanyID != cidA || !grantIDsEqual(reqs[2].GrantServiceIDs, []int{a1.ID, a2.ID}) {
		t.Errorf("A 회사 요청에 다른 회사 키가 섞임: company=%d grants=%v, want company=%d grants=[%d %d]",
			reqs[2].CompanyID, reqs[2].GrantServiceIDs, cidA, a1.ID, a2.ID)
	}
}

// TestCompanyServices_Connect_RequiresApprovedCampRole — 캠프 연동은 승인된 학생이나
// 수업 관리자 소유자만 가능하다 (JWT 클레임이 아니라 DB 기준).
func TestCompanyServices_Connect_RequiresApprovedCampRole(t *testing.T) {
	ts := setupTestServer(t)
	email := "svc-elig@test.com"
	token, cid, _ := createClassroomOwnerCompany(t, ts, email, "자격", "20261022", "svc_elig_co")

	fake := &fakeProvisioner{}
	ts.companyServiceUC.SetProvisioner(fake)
	ts.companyServiceUC.SetChecker(okChecker{})

	svc := ts.createService(t, cid, token, "자격앱", "https://elig.example.com")
	ts.validateService(t, cid, svc.ID, token)

	// (1) 소유자 role 이 admin 으로 바뀐 경우에도 자기 회사 연동 가능
	if _, err := ts.db.Exec("UPDATE users SET role = 'admin' WHERE email = ?", email); err != nil {
		t.Fatalf("role 변경: %v", err)
	}
	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	if status != http.StatusOK || !r.Success {
		t.Fatalf("admin 소유자 연동: status=%d error=%+v", status, r.Error)
	}

	// (2) 학생이지만 승인 상태가 아닌 경우 → NOT_APPROVED (비활성 계정 fail-closed)
	if _, err := ts.db.Exec("UPDATE users SET role = 'student', status = 'pending' WHERE email = ?", email); err != nil {
		t.Fatalf("status 변경: %v", err)
	}
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	assertErrCode(t, "미승인 소유자 연동", status, r, http.StatusForbidden, "NOT_APPROVED")

	if fake.callCount() != 1 {
		t.Fatalf("미승인 요청이 프로비저너를 추가 호출함: 총 %d회", fake.callCount())
	}
	for _, s := range ts.listServices(t, cid, token) {
		if s.ID == svc.ID && s.RybbitStatus != "connected" {
			t.Errorf("자격 미달 요청이 상태를 바꿈: %+v", s)
		}
	}

	// (3) 승인된 학생으로 복구 → 정상 연동
	if _, err := ts.db.Exec("UPDATE users SET status = 'approved' WHERE email = ?", email); err != nil {
		t.Fatalf("status 복구: %v", err)
	}
	status, r = ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("복구 후 연동 실패 (%d): %v", status, r.Error)
	}
	if fake.callCount() != 1 {
		t.Errorf("복구 후 호출 %d회, want 1", fake.callCount())
	}
}
