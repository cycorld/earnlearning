package rybbit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/earnlearning/backend/internal/domain/user"
)

// #181: Rybbit EarnLearning 프로비저닝 API 클라이언트 계약 테스트.
// 기준 문서: Rybbit EARNLEARNING_SSO.md (커밋 6cc9ee30).

const testSecret = "unit-test-shared-secret-0123456789abcdef" // 40자 (≥32)

func validRequest() ProvisionRequest {
	return ProvisionRequest{
		CompanyID:       7,
		CompanyName:     "Nextlab",
		ServiceID:       42,
		ServiceName:     "Main App",
		Domain:          "app.example.com",
		OwnerUserID:     9,
		OwnerEmail:      "owner@test.com",
		OwnerName:       "Kim Owner",
		GrantServiceIDs: []int{41, 42},
	}
}

// --- 설정 (fail-closed) ---

func TestNew_ConfigMatrix(t *testing.T) {
	cases := []struct {
		label    string
		base     string
		secret   string
		wantErr  bool
		wantNoop bool
	}{
		{"둘 다 빈값 = 의도적 미설정", "", "", false, true},
		{"공백만", "   ", "   ", false, true},
		{"시크릿 누락", "https://rybbit.example.com", "", true, true},
		{"베이스 URL 누락", "", testSecret, true, true},
		{"시크릿 31자 (너무 짧음)", "https://rybbit.example.com", strings.Repeat("a", 31), true, true},
		{"시크릿 32자 (경계)", "https://rybbit.example.com", strings.Repeat("a", 32), false, false},
		{"정상 설정", "https://rybbit.example.com", testSecret, false, false},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			p, err := New(c.base, c.secret)
			if (err != nil) != c.wantErr {
				t.Fatalf("New(%q, <secret len %d>) err = %v, wantErr %v", c.base, len(c.secret), err, c.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), strings.TrimSpace(c.secret)) && strings.TrimSpace(c.secret) != "" {
				t.Errorf("설정 에러에 시크릿 값이 노출됨: %s", err.Error())
			}
			_, isNoop := p.(*NoopProvisioner)
			if isNoop != c.wantNoop {
				t.Errorf("Noop 여부 = %v, want %v", isNoop, c.wantNoop)
			}
			if isNoop {
				if _, perr := p.Provision(context.Background(), validRequest()); !errors.Is(perr, ErrNotConfigured) {
					t.Errorf("Noop.Provision → %v, want ErrNotConfigured", perr)
				}
			}
		})
	}
}

// --- 페이로드 바이트 (골든) ---

// 서명은 전송 바이트에 대해 계산되므로, 본문 바이트 자체가 계약이다.
func TestBuildProvisionPayload_GoldenBytes(t *testing.T) {
	body, wantGrants, err := buildProvisionPayload(ProvisionRequest{
		CompanyID:   7,
		CompanyName: "Nextlab",
		ServiceID:   42,
		ServiceName: "Main App",
		Domain:      "App.Example.COM ", // 소문자 + 트림 정규화 확인
		OwnerUserID: 9,
		OwnerEmail:  " Owner@Test.com ", // 소문자 + 트림 정규화 확인
		OwnerName:   "Kim Owner",
		// 클릭한 42 미포함 + 중복 41 → 합집합/중복제거/오름차순 확인
		GrantServiceIDs: []int{41, 41},
	})
	if err != nil {
		t.Fatalf("buildProvisionPayload: %v", err)
	}
	want := `{"company":{"id":"7","name":"Nextlab"},"site":{"key":"service:42","name":"Main App","domain":"app.example.com"},"user":{"id":"9","email":"owner@test.com","name":"Kim Owner","role":"member"},"grants":{"siteKeys":["service:41","service:42"]}}`
	if string(body) != want {
		t.Errorf("페이로드 바이트 불일치:\n got %s\nwant %s", body, want)
	}
	if wantGrants != 2 {
		t.Errorf("grant 개수 = %d, want 2", wantGrants)
	}
}

func TestBuildProvisionPayload_NoOwnerName_OmitsField(t *testing.T) {
	req := validRequest()
	req.OwnerName = "   "
	req.GrantServiceIDs = nil // 클릭한 서비스만
	body, wantGrants, err := buildProvisionPayload(req)
	if err != nil {
		t.Fatalf("buildProvisionPayload: %v", err)
	}
	want := `{"company":{"id":"7","name":"Nextlab"},"site":{"key":"service:42","name":"Main App","domain":"app.example.com"},"user":{"id":"9","email":"owner@test.com","role":"member"},"grants":{"siteKeys":["service:42"]}}`
	if string(body) != want {
		t.Errorf("페이로드 바이트 불일치:\n got %s\nwant %s", body, want)
	}
	if wantGrants != 1 {
		t.Errorf("grant 개수 = %d, want 1", wantGrants)
	}
}

func TestBuildProvisionPayload_UserIDEqualsOAuthSubject(t *testing.T) {
	body, _, err := buildProvisionPayload(validRequest())
	if err != nil {
		t.Fatalf("buildProvisionPayload: %v", err)
	}
	var decoded struct {
		User struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("본문 파싱: %v", err)
	}
	if decoded.User.ID != user.OAuthSubject(9) {
		t.Errorf("user.id = %q, want OAuthSubject(9) = %q", decoded.User.ID, user.OAuthSubject(9))
	}
	if decoded.User.Role != "member" {
		t.Errorf("user.role = %q, want member (고정)", decoded.User.Role)
	}
}

func TestBuildProvisionPayload_Rejections(t *testing.T) {
	mutate := func(f func(*ProvisionRequest)) ProvisionRequest {
		r := validRequest()
		f(&r)
		return r
	}
	tooMany := make([]int, 101)
	for i := range tooMany {
		tooMany[i] = i + 1
	}
	cases := []struct {
		label string
		req   ProvisionRequest
	}{
		{"빈 도메인", mutate(func(r *ProvisionRequest) { r.Domain = "  " })},
		{"빈 이메일", mutate(func(r *ProvisionRequest) { r.OwnerEmail = "" })},
		{"빈 회사 이름", mutate(func(r *ProvisionRequest) { r.CompanyName = " " })},
		{"회사 id 0", mutate(func(r *ProvisionRequest) { r.CompanyID = 0 })},
		{"소유자 id 0", mutate(func(r *ProvisionRequest) { r.OwnerUserID = 0 })},
		{"grant id 0 포함", mutate(func(r *ProvisionRequest) { r.GrantServiceIDs = []int{0} })},
		{"grant 100개 초과 (조용한 축소 금지)", mutate(func(r *ProvisionRequest) { r.GrantServiceIDs = tooMany })},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			if _, _, err := buildProvisionPayload(c.req); err == nil {
				t.Fatal("에러여야 함")
			}
		})
	}
}

func TestBuildProvisionPayload_TruncatesLongNames(t *testing.T) {
	req := validRequest()
	req.OwnerName = strings.Repeat("가", 300)
	req.ServiceName = strings.Repeat("나", 300)
	req.CompanyName = strings.Repeat("다", 300)
	body, _, err := buildProvisionPayload(req)
	if err != nil {
		t.Fatalf("buildProvisionPayload: %v", err)
	}
	var decoded struct {
		Company struct {
			Name string `json:"name"`
		} `json:"company"`
		Site struct {
			Name string `json:"name"`
		} `json:"site"`
		User struct {
			Name string `json:"name"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("본문 파싱: %v", err)
	}
	for label, got := range map[string]string{
		"company.name": decoded.Company.Name,
		"site.name":    decoded.Site.Name,
		"user.name":    decoded.User.Name,
	} {
		if n := len([]rune(got)); n != 256 {
			t.Errorf("%s 길이 %d rune, want 256 (계약 상한)", label, n)
		}
	}
}

// --- 서명 (계약 문서의 재현 벡터) ---

// EARNLEARNING_SSO.md "Worked example" 그대로 — 이 벡터가 깨지면
// 서명 레시피(ts + "." + raw body, HMAC-SHA256, 소문자 hex)가 계약과 다른 것이다.
func TestSignProvision_DocumentedVector(t *testing.T) {
	secret := "dummy-provisioning-secret-change-me-32"
	timestamp := "1767225600"
	body := `{"company":{"id":"camp-2026","name":"Sunrise Camp"},"site":{"key":"camp-2026-main","name":"Sunrise Camp Main","domain":"camp.example.com"},"user":{"id":"el-user-9f2a","email":"camper@example.com","name":"Ada Camper","role":"member"},"grants":{"siteKeys":["camp-2026-main"]}}`
	want := "d1c78307f5712f83e9828c23ce136c7a819c102cb2a3dbeed021a006494bb8e5"
	if got := signProvision(secret, timestamp, []byte(body)); got != want {
		t.Errorf("signProvision = %s, want %s", got, want)
	}
}

// --- 왕복 계약 (httptest 로 Rybbit 검증 로직 재현) ---

var (
	tsPattern  = regexp.MustCompile(`^\d+$`)
	sigPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// rybbitStub 은 Rybbit 서버의 검증 순서를 재현한다: 수신한 "원시 바이트" 로
// 서명을 재계산해 비교하고, Origin 부재와 헤더 형식을 검사한다.
func rybbitStub(t *testing.T, secret string, respond func(w http.ResponseWriter, payload provisionPayload, rawBody []byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/earnlearning/provision" {
			t.Errorf("요청 = %s %s, want POST /api/earnlearning/provision", r.Method, r.URL.Path)
		}
		if _, hasOrigin := r.Header["Origin"]; hasOrigin {
			t.Errorf("Origin 헤더를 보내면 안 됨 (Rybbit 이 403 으로 거른다): %q", r.Header.Get("Origin"))
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type = %q", ct)
		}

		rawBody, _ := io.ReadAll(r.Body)
		timestamp := r.Header.Get("x-earnlearning-timestamp")
		signature := r.Header.Get("x-earnlearning-signature")
		if !tsPattern.MatchString(timestamp) {
			t.Errorf("timestamp 헤더 형식 = %q, want 숫자만", timestamp)
		}
		if !sigPattern.MatchString(signature) {
			t.Errorf("signature 헤더 형식 = %q, want 소문자 hex 64자", signature)
		}
		if skew := math.Abs(float64(time.Now().Unix()) - parseUnixSeconds(t, timestamp)); skew > 300 {
			t.Errorf("timestamp 스큐 %.0f초 > 300초", skew)
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
		mac.Write(rawBody)
		if want := hex.EncodeToString(mac.Sum(nil)); want != signature {
			t.Errorf("수신 바이트로 재계산한 서명 불일치: got %s want %s (body=%s)", signature, want, rawBody)
		}

		var payload provisionPayload
		if err := json.Unmarshal(rawBody, &payload); err != nil {
			t.Errorf("수신 본문 파싱 실패: %v (%s)", err, rawBody)
		}
		w.Header().Set("Content-Type", "application/json")
		respond(w, payload, rawBody)
	}))
}

func parseUnixSeconds(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Errorf("timestamp 파싱 실패: %q", s)
	}
	return v
}

func TestProvision_ContractRoundtrip(t *testing.T) {
	srv := rybbitStub(t, testSecret, func(w http.ResponseWriter, payload provisionPayload, _ []byte) {
		// 필드 매핑 검증 (서버가 보는 최종 형태)
		if payload.Company.ID != "7" || payload.Company.Name != "Nextlab" {
			t.Errorf("company = %+v", payload.Company)
		}
		if payload.Site.Key != "service:42" || payload.Site.Domain != "app.example.com" || payload.Site.Name != "Main App" {
			t.Errorf("site = %+v", payload.Site)
		}
		if payload.User.ID != user.OAuthSubject(9) || payload.User.Email != "owner@test.com" || payload.User.Role != "member" {
			t.Errorf("user = %+v", payload.User)
		}
		if len(payload.Grants.SiteKeys) != 2 || payload.Grants.SiteKeys[0] != "service:41" || payload.Grants.SiteKeys[1] != "service:42" {
			t.Errorf("grants = %+v", payload.Grants)
		}
		w.Write([]byte(`{"organizationId":"elorg_7-abc","siteId":42,"userId":"u-1","memberId":"m-1","grantedSiteIds":[41,42],"created":{"organization":false,"site":true,"user":false,"member":false}}`))
	})
	defer srv.Close()

	p, err := New(srv.URL, testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res, err := p.Provision(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.SiteID != "42" {
		t.Errorf("SiteID = %q, want \"42\"", res.SiteID)
	}
	if res.OrganizationID != "elorg_7-abc" {
		t.Errorf("OrganizationID = %q", res.OrganizationID)
	}
	if len(res.GrantedSiteIDs) != 2 || res.GrantedSiteIDs[0] != 41 || res.GrantedSiteIDs[1] != 42 {
		t.Errorf("GrantedSiteIDs = %v", res.GrantedSiteIDs)
	}
}

// 같은 요청을 두 번 보내도 (Rybbit 멱등 재호출) 클라이언트는 같은 결과를 돌려준다.
func TestProvision_RepeatCall_SameResult(t *testing.T) {
	srv := rybbitStub(t, testSecret, func(w http.ResponseWriter, payload provisionPayload, _ []byte) {
		w.Write([]byte(`{"organizationId":"elorg_7-abc","siteId":42,"userId":"u-1","memberId":"m-1","grantedSiteIds":[41,42],"created":{"organization":false,"site":false,"user":false,"member":false}}`))
	})
	defer srv.Close()

	p, err := New(srv.URL, testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	first, err := p.Provision(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("1차 Provision: %v", err)
	}
	second, err := p.Provision(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("2차 Provision: %v", err)
	}
	if first.SiteID != second.SiteID || first.OrganizationID != second.OrganizationID {
		t.Errorf("멱등 재호출 결과 불일치: %+v vs %+v", first, second)
	}
}

// --- 응답 strict 검증 ---

func TestParseProvisionResponse_Strict(t *testing.T) {
	valid := `{"organizationId":"o","siteId":42,"userId":"u","memberId":"m","grantedSiteIds":[41,42],"created":{"organization":true,"site":true,"user":true,"member":true}}`
	cases := []struct {
		label      string
		body       string
		wantGrants int
		wantErr    bool
	}{
		{"정상", valid, 2, false},
		{"모르는 필드 (strict 위반)", `{"organizationId":"o","siteId":42,"userId":"u","memberId":"m","grantedSiteIds":[42],"created":{"organization":true,"site":true,"user":true,"member":true},"extra":1}`, 1, true},
		{"created 누락", `{"organizationId":"o","siteId":42,"userId":"u","memberId":"m","grantedSiteIds":[42]}`, 1, true},
		{"siteId 누락", `{"organizationId":"o","userId":"u","memberId":"m","grantedSiteIds":[42],"created":{"organization":true,"site":true,"user":true,"member":true}}`, 1, true},
		{"siteId 가 grantedSiteIds 에 없음", `{"organizationId":"o","siteId":42,"userId":"u","memberId":"m","grantedSiteIds":[41],"created":{"organization":true,"site":true,"user":true,"member":true}}`, 1, true},
		{"granted 개수 불일치 (요청 2, 응답 1)", `{"organizationId":"o","siteId":42,"userId":"u","memberId":"m","grantedSiteIds":[42],"created":{"organization":true,"site":true,"user":true,"member":true}}`, 2, true},
		{"granted 에 0 이하 id", `{"organizationId":"o","siteId":42,"userId":"u","memberId":"m","grantedSiteIds":[0,42],"created":{"organization":true,"site":true,"user":true,"member":true}}`, 2, true},
		{"organizationId 빈 문자열", `{"organizationId":"","siteId":42,"userId":"u","memberId":"m","grantedSiteIds":[42],"created":{"organization":true,"site":true,"user":true,"member":true}}`, 1, true},
		{"JSON 아님", `not-json`, 1, true},
		{"뒤에 잡동사니", valid + `garbage`, 2, true},
		{"siteId 소수", `{"organizationId":"o","siteId":42.5,"userId":"u","memberId":"m","grantedSiteIds":[42],"created":{"organization":true,"site":true,"user":true,"member":true}}`, 1, true},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			res, err := parseProvisionResponse([]byte(c.body), c.wantGrants)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if !c.wantErr && (res == nil || res.SiteID != "42") {
				t.Errorf("res = %+v", res)
			}
		})
	}
}

// --- 상태 코드 매핑 + 비유출 ---

func TestProvision_ErrorStatuses_NoLeak(t *testing.T) {
	leakMarker := "super-internal-detail-should-not-leak"
	for _, status := range []int{400, 401, 403, 404, 409, 500, 502} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				w.Write([]byte(`{"error":"` + leakMarker + ` ` + testSecret + `"}`))
			}))
			defer srv.Close()

			p, err := New(srv.URL, testSecret)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			_, perr := p.Provision(context.Background(), validRequest())
			if perr == nil {
				t.Fatalf("status %d 는 에러여야 함", status)
			}
			if errors.Is(perr, ErrNotConfigured) {
				t.Fatal("설정된 프로비저너의 장애를 ErrNotConfigured 로 보고하면 안 됨 (503 vs 502 구분)")
			}
			msg := perr.Error()
			if !strings.Contains(msg, strconv.Itoa(status)) {
				t.Errorf("에러에 상태 코드가 있어야 함: %s", msg)
			}
			if strings.Contains(msg, leakMarker) {
				t.Errorf("에러에 응답 본문이 노출됨: %s", msg)
			}
			if strings.Contains(msg, testSecret) {
				t.Errorf("에러에 시크릿이 노출됨: %s", msg)
			}
		})
	}
}

// 2xx 라도 200 이 아니면 계약 위반 (Rybbit 은 성공을 200 으로만 응답).
func TestProvision_Non200Success_IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"organizationId":"o","siteId":1,"userId":"u","memberId":"m","grantedSiteIds":[1],"created":{"organization":true,"site":true,"user":true,"member":true}}`))
	}))
	defer srv.Close()

	p, err := New(srv.URL, testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, perr := p.Provision(context.Background(), validRequest()); perr == nil {
		t.Fatal("201 은 계약 밖이므로 에러여야 함")
	}
}

// 페이로드 검증 실패 시 외부 호출 자체가 없어야 한다.
func TestProvision_InvalidRequest_NoHTTPCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("잘못된 요청인데 외부 호출이 발생함")
	}))
	defer srv.Close()

	p, err := New(srv.URL, testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	req := validRequest()
	req.Domain = "   "
	if _, perr := p.Provision(context.Background(), req); perr == nil {
		t.Fatal("빈 도메인은 에러여야 함")
	}
}
