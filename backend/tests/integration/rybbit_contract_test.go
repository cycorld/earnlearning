package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/infrastructure/rybbit"
)

// #181: "실제 HTTP 클라이언트" 로 Rybbit 프로비저닝 계약을 검증하는 왕복 테스트.
// 유스케이스 → rybbit.New(실클라이언트) → 스텁 Rybbit(HMAC 재계산 검증) 전 구간을 태운다.

const contractSecret = "integration-shared-secret-0123456789abcd" // 40자 (≥32)

// wireProvisionPayload — 전송 계약 그대로의 형태 (스텁이 보는 최종 바이트).
type wireProvisionPayload struct {
	Company struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"company"`
	Site struct {
		Key    string `json:"key"`
		Name   string `json:"name"`
		Domain string `json:"domain"`
	} `json:"site"`
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	} `json:"user"`
	Grants struct {
		SiteKeys []string `json:"siteKeys"`
	} `json:"grants"`
}

// rybbitContractStub — 수신한 원시 바이트로 서명을 재계산해 검증하는 가짜 Rybbit.
type rybbitContractStub struct {
	mu         sync.Mutex
	t          *testing.T
	failStatus int // 0 이면 정상 200
	payloads   []wireProvisionPayload
}

func (st *rybbitContractStub) handler(w http.ResponseWriter, r *http.Request) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t := st.t

	if r.Method != http.MethodPost || r.URL.Path != "/api/earnlearning/provision" {
		t.Errorf("요청 = %s %s, want POST /api/earnlearning/provision", r.Method, r.URL.Path)
	}
	if _, hasOrigin := r.Header["Origin"]; hasOrigin {
		t.Errorf("Origin 헤더를 보내면 안 됨: %q", r.Header.Get("Origin"))
	}

	rawBody, _ := io.ReadAll(r.Body)
	timestamp := r.Header.Get("x-earnlearning-timestamp")
	signature := r.Header.Get("x-earnlearning-signature")
	mac := hmac.New(sha256.New, []byte(contractSecret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(rawBody)
	if want := hex.EncodeToString(mac.Sum(nil)); want != signature {
		t.Errorf("수신 바이트 기준 서명 불일치: got %q want %q", signature, want)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Unauthorized"}`))
		return
	}

	var payload wireProvisionPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		t.Errorf("본문 파싱 실패: %v (%s)", err, rawBody)
	}
	st.payloads = append(st.payloads, payload)

	if st.failStatus != 0 {
		w.WriteHeader(st.failStatus)
		w.Write([]byte(`{"error":"stub failure"}`))
		return
	}

	// site.key = service:<id> → siteId <id>, grants 키도 같은 규칙으로 응답.
	siteID := siteKeyToID(t, payload.Site.Key)
	granted := make([]string, 0, len(payload.Grants.SiteKeys))
	for _, k := range payload.Grants.SiteKeys {
		granted = append(granted, strconv.Itoa(siteKeyToID(t, k)))
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"organizationId":"elorg_` + payload.Company.ID +
		`","siteId":` + strconv.Itoa(siteID) +
		`,"userId":"u-` + payload.User.ID +
		`","memberId":"m-1","grantedSiteIds":[` + strings.Join(granted, ",") +
		`],"created":{"organization":true,"site":true,"user":true,"member":true}}`))
}

func siteKeyToID(t *testing.T, key string) int {
	t.Helper()
	id, err := strconv.Atoi(strings.TrimPrefix(key, "service:"))
	if err != nil || !strings.HasPrefix(key, "service:") {
		t.Errorf("사이트 키 형식 = %q, want service:<id>", key)
	}
	return id
}

func (st *rybbitContractStub) lastPayload() wireProvisionPayload {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.payloads) == 0 {
		return wireProvisionPayload{}
	}
	return st.payloads[len(st.payloads)-1]
}

func (st *rybbitContractStub) setFail(status int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.failStatus = status
}

func TestCompanyServices_Connect_RealClientContract(t *testing.T) {
	ts := setupTestServer(t)
	email := "svc-wire@test.com"
	token, cid, _ := createClassroomOwnerCompany(t, ts, email, "와이어", "20261023", "svc_wire_co")

	stub := &rybbitContractStub{t: t}
	srv := httptest.NewServer(http.HandlerFunc(stub.handler))
	defer srv.Close()

	p, err := rybbit.New(srv.URL, contractSecret)
	if err != nil {
		t.Fatalf("rybbit.New: %v", err)
	}
	ts.companyServiceUC.SetProvisioner(p)
	ts.companyServiceUC.SetChecker(okChecker{})

	svc := ts.createService(t, cid, token, "와이어앱", "https://wire.example.com/app")
	ts.validateService(t, cid, svc.ID, token)

	status, r := ts.svcJSON(http.MethodPost, connectPath(cid, svc.ID), nil, token)
	if !r.Success || status != http.StatusOK {
		t.Fatalf("실클라이언트 연동 실패 (%d): %v", status, r.Error)
	}
	var got serviceDTO
	json.Unmarshal(r.Data, &got)
	if got.RybbitStatus != "connected" || got.RybbitSiteID != strconv.Itoa(svc.ID) {
		t.Errorf("연동 결과: %+v, want connected / site id %d (응답 siteId 저장)", got, svc.ID)
	}

	// 스텁이 본 최종 전송 페이로드 == 계약 매핑.
	payload := stub.lastPayload()
	if payload.Company.ID != strconv.Itoa(cid) || payload.Company.Name == "" {
		t.Errorf("company 매핑: %+v (cid=%d)", payload.Company, cid)
	}
	if payload.Site.Key != "service:"+strconv.Itoa(svc.ID) || payload.Site.Domain != "wire.example.com" {
		t.Errorf("site 매핑: %+v", payload.Site)
	}
	if payload.User.Email != email || payload.User.Role != "member" {
		t.Errorf("user 매핑: %+v", payload.User)
	}
	if len(payload.Grants.SiteKeys) != 1 || payload.Grants.SiteKeys[0] != payload.Site.Key {
		t.Errorf("grants 에 클릭한 사이트 키가 없음: %+v", payload.Grants)
	}

	// 프로비저닝 user.id == OAuth userinfo sub (계정 연결 키의 종단간 일치).
	accessToken := oauthAccessTokenFor(t, ts, token, "와이어계약앱",
		"dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk-wire-181")
	info := fetchUserInfo(t, ts, accessToken)
	if payload.User.ID != info.Sub {
		t.Errorf("provision user.id %q != userinfo sub %q", payload.User.ID, info.Sub)
	}
	if payload.User.ID != user.OAuthSubject(info.ID) {
		t.Errorf("provision user.id %q != OAuthSubject(%d)", payload.User.ID, info.ID)
	}

	// Rybbit 이 401/409/500 을 돌려주면 → 502 + 로컬 상태 불변 (fail-closed).
	for i, failStatus := range []int{401, 409, 500} {
		stub.setFail(failStatus)
		u := "https://wire-fail" + strconv.Itoa(i) + ".example.com"
		failSvc := ts.createService(t, cid, token, "실패앱", u)
		ts.validateService(t, cid, failSvc.ID, token)

		status, r = ts.svcJSON(http.MethodPost, connectPath(cid, failSvc.ID), nil, token)
		assertErrCode(t, "Rybbit "+strconv.Itoa(failStatus)+" 응답",
			status, r, http.StatusBadGateway, "RYBBIT_CONNECT_FAILED")

		for _, s := range ts.listServices(t, cid, token) {
			if s.ID == failSvc.ID && (s.RybbitStatus != "not_connected" || s.RybbitSiteID != "") {
				t.Errorf("Rybbit %d 실패인데 상태가 바뀜: %+v", failStatus, s)
			}
		}
	}
}
