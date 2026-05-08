package integration

import (
	"encoding/json"
	"testing"
)

// #017 회귀: 기업 정보 수정 + 전체 기업 목록

// helper: 학생을 만들고 회사를 하나 생성한 뒤 (token, companyID) 반환
// student_id 는 호출자가 7~10자리 숫자로 명시해야 함
func createUserWithCompany(t *testing.T, ts *testServer, email, name, studentID, projName string) (string, int) {
	t.Helper()
	token := ts.registerAndApprove(email, "pass1234", name, studentID)
	// 새 사용자 잔액은 0 이라 회사 설립을 위해 admin transfer 로 충전
	// target_all 이라 매번 호출하면 누적되지만 테스트에는 무해
	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all":  true,
		"amount":      1500000,
		"description": "테스트 초기자금",
	}, adminToken)

	r := ts.post("/api/companies", map[string]interface{}{
		"name":            projName,
		"description":     "테스트 회사 " + projName,
		"initial_capital": 1000000,
		"logo_url":        "",
	}, token)
	if !r.Success {
		t.Fatalf("create company %s: %v", projName, r.Error)
	}
	var c struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(r.Data, &c)
	return token, c.ID
}

func TestCompanyUpdate_NameChange_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "alice@test.com", "alice", "20240001", "alice_co")

	// 이름 변경
	r := ts.put("/api/companies/"+itoaUD(cid), map[string]string{
		"name":        "alice_renamed",
		"description": "이름 바꿈",
	}, token)
	if !r.Success {
		t.Fatalf("update name: %v", r.Error)
	}

	// 다시 조회 → 새 이름이어야 함
	g := ts.get("/api/companies/"+itoaUD(cid), token)
	if !g.Success {
		t.Fatalf("get after update: %v", g.Error)
	}
	var got struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	_ = json.Unmarshal(g.Data, &got)
	if got.Name != "alice_renamed" {
		t.Errorf("name not updated: got %q want alice_renamed", got.Name)
	}
	if got.Description != "이름 바꿈" {
		t.Errorf("description not updated: got %q", got.Description)
	}
}

func TestCompanyUpdate_LogoURLChange_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "logo@test.com", "logo", "20240002", "logo_co1")

	r := ts.put("/api/companies/"+itoaUD(cid), map[string]string{
		"name":        "logo_co1",
		"description": "로고 추가",
		"logo_url":    "https://example.com/logo.png",
	}, token)
	if !r.Success {
		t.Fatalf("update logo: %v", r.Error)
	}

	g := ts.get("/api/companies/"+itoaUD(cid), token)
	var got struct {
		LogoURL string `json:"logo_url"`
	}
	_ = json.Unmarshal(g.Data, &got)
	if got.LogoURL != "https://example.com/logo.png" {
		t.Errorf("logo_url not updated: got %q", got.LogoURL)
	}
}

func TestCompanyUpdate_DuplicateName_Conflict(t *testing.T) {
	ts := setupTestServer(t)
	// 두 학생, 각자 회사 만듦
	_, _ = createUserWithCompany(t, ts, "first@test.com", "first", "20240003", "first_co")
	tok2, cid2 := createUserWithCompany(t, ts, "second@test.com", "second", "20240004", "second_co")

	// second 가 first_co 로 이름 변경 시도 → 409
	r := ts.put("/api/companies/"+itoaUD(cid2), map[string]string{
		"name":        "first_co",
		"description": "남의 이름 훔치기",
	}, tok2)
	if r.Success {
		t.Fatal("should fail with duplicate name")
	}
	if r.Error == nil || r.Error.Code != "DUPLICATE_NAME" {
		t.Errorf("expected DUPLICATE_NAME, got %v", r.Error)
	}
}

func TestCompanyUpdate_NotOwner_Forbidden(t *testing.T) {
	ts := setupTestServer(t)
	_, cid := createUserWithCompany(t, ts, "owner@test.com", "owner", "20240005", "owner_co")
	otherToken, _ := createUserWithCompany(t, ts, "other@test.com", "other", "20240006", "other_co")

	r := ts.put("/api/companies/"+itoaUD(cid), map[string]string{
		"name":        "hijacked",
		"description": "남의 회사 수정 시도",
	}, otherToken)
	if r.Success {
		t.Fatal("should fail with NOT_OWNER")
	}
	if r.Error == nil || r.Error.Code != "NOT_OWNER" {
		t.Errorf("expected NOT_OWNER, got %v", r.Error)
	}
}

func TestListAllCompanies_ReturnsAll(t *testing.T) {
	ts := setupTestServer(t)
	_, _ = createUserWithCompany(t, ts, "x1@test.com", "x1", "20240007", "x1_co")
	_, _ = createUserWithCompany(t, ts, "x2@test.com", "x2", "20240008", "x2_co")
	tok3, _ := createUserWithCompany(t, ts, "x3@test.com", "x3", "20240009", "x3_co")

	// x3 가 전체 목록 조회 — 본인 + 타인 회사 모두 보여야 함
	r := ts.get("/api/companies", tok3)
	if !r.Success {
		t.Fatalf("list all: %v", r.Error)
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(r.Data, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 companies, got %d", len(list))
	}

	// 응답에 owner_name 포함 확인
	for _, c := range list {
		if _, ok := c["owner_name"]; !ok {
			t.Errorf("response missing owner_name: %v", c)
		}
	}
}

func TestCompanyUpdate_ServiceURL_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "svcurl@test.com", "svcurl", "20240020", "svcurl_co")

	// service_url 설정
	r := ts.put("/api/companies/"+itoaUD(cid), map[string]string{
		"name":        "svcurl_co",
		"description": "서비스 URL 테스트",
		"service_url": "https://my-app.example.com",
	}, token)
	if !r.Success {
		t.Fatalf("update service_url: %v", r.Error)
	}

	// 다시 조회 → service_url 반영 확인
	g := ts.get("/api/companies/"+itoaUD(cid), token)
	if !g.Success {
		t.Fatalf("get after update: %v", g.Error)
	}
	var got struct {
		ServiceURL string `json:"service_url"`
	}
	_ = json.Unmarshal(g.Data, &got)
	if got.ServiceURL != "https://my-app.example.com" {
		t.Errorf("service_url not updated: got %q want %q", got.ServiceURL, "https://my-app.example.com")
	}
}

func TestListAllCompanies_NoAuth(t *testing.T) {
	ts := setupTestServer(t)
	r := ts.get("/api/companies", "")
	if r.Success {
		t.Fatal("should require auth")
	}
}

func TestCompanyCreate_DuplicateName_Conflict(t *testing.T) {
	ts := setupTestServer(t)
	// 첫 학생이 회사 만듦
	_, _ = createUserWithCompany(t, ts, "u1@test.com", "u1", "20240010", "samename")

	// 두 번째 학생이 같은 이름으로 만들려 함
	tok2 := ts.registerAndApprove("u2@test.com", "pass1234", "u2", "20240011")
	adminToken := ts.login(testAdminEmail, testAdminPass)
	ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_all": true, "amount": 1500000, "description": "초기자금",
	}, adminToken)

	r := ts.post("/api/companies", map[string]interface{}{
		"name":            "samename",
		"description":     "이름 중복 시도",
		"initial_capital": 1000000,
	}, tok2)
	if r.Success {
		t.Fatal("should fail with duplicate name")
	}
	if r.Error == nil || r.Error.Code != "DUPLICATE_NAME" {
		t.Errorf("expected DUPLICATE_NAME, got %v", r.Error)
	}
}

// #115 회귀: service_url 다중 URL 지원 (쉼표 구분)
func TestCompanyUpdate_ServiceURL_Multi_Success(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "url1@test.com", "url1", "20240091", "url1_co")

	r := ts.put("/api/companies/"+itoaUD(cid), map[string]string{
		"name":         "url1_co",
		"description":  "set urls",
		"service_url":  "https://a.example,  https://b.example , https://c.example",
	}, token)
	if !r.Success {
		t.Fatalf("update with multi service_url failed: %v", r.Error)
	}

	g := ts.get("/api/companies/"+itoaUD(cid), token)
	if !g.Success {
		t.Fatalf("get failed: %v", g.Error)
	}
	var got struct {
		ServiceURL string `json:"service_url"`
	}
	_ = json.Unmarshal(g.Data, &got)
	// 정규화 — 공백 제거 + 쉼표 구분 그대로
	if got.ServiceURL != "https://a.example,https://b.example,https://c.example" {
		t.Errorf("expected normalized comma-joined urls, got %q", got.ServiceURL)
	}
}

func TestCompanyUpdate_ServiceURL_Empty_OK(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "url2@test.com", "url2", "20240092", "url2_co")

	// 빈 문자열은 허용 (URL 0개)
	r := ts.put("/api/companies/"+itoaUD(cid), map[string]string{
		"name":        "url2_co",
		"description": "no urls",
		"service_url": "",
	}, token)
	if !r.Success {
		t.Fatalf("update with empty service_url should succeed: %v", r.Error)
	}
}

func TestCompanyUpdate_ServiceURL_Invalid_Reject(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "url3@test.com", "url3", "20240093", "url3_co")

	cases := []struct {
		name string
		url  string
	}{
		{"protocol 없음", "example.com"},
		{"ftp 거부", "ftp://example.com"},
		{"javascript 거부", "javascript:alert(1)"},
		{"한 piece 라도 invalid 면 전부 거부", "https://a.example,not-a-url"},
		{"호스트 없음", "https://"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := ts.put("/api/companies/"+itoaUD(cid), map[string]string{
				"name":        "url3_co",
				"description": tc.name,
				"service_url": tc.url,
			}, token)
			if r.Success {
				t.Errorf("should reject invalid url %q", tc.url)
			}
		})
	}
}
