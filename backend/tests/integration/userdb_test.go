package integration

import (
	"encoding/json"
	"testing"
)

// UserDB 프로비저닝 API 회귀 테스트.
// NoopProvisioner 를 사용하므로 실제 PG 에 연결하지 않는다.

func TestUserDB_List_Empty(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.login(testAdminEmail, testAdminPass)

	r := ts.get("/api/users/me/databases", token)
	if !r.Success {
		t.Fatalf("list failed: %v", r.Error)
	}
	// 응답 data 는 빈 배열이어야 함
	var list []map[string]interface{}
	if err := json.Unmarshal(r.Data, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestUserDB_Create_Success(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.login(testAdminEmail, testAdminPass)

	r := ts.post("/api/users/me/databases", map[string]string{
		"project_name": "todoapp",
	}, token)
	if !r.Success {
		t.Fatalf("create failed: %v", r.Error)
	}
	var created map[string]interface{}
	if err := json.Unmarshal(r.Data, &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// 필수 필드 확인
	for _, k := range []string{"id", "db_name", "pg_username", "host", "port", "password", "url"} {
		if _, ok := created[k]; !ok {
			t.Errorf("missing field %q in response: %v", k, created)
		}
	}
	// db_name 은 admin 이메일 기반 (admin@test.com → admin_todoapp 기대)
	if got := created["db_name"].(string); got != "admin_todoapp" {
		t.Errorf("unexpected db_name: %s", got)
	}
}

func TestUserDB_Create_InvalidName(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.login(testAdminEmail, testAdminPass)

	cases := []string{
		"",           // 빈 문자열
		"a",          // 너무 짧음
		"With-Dash",  // 대시 불가
		"UPPERCASE",  // 대문자 불가
		"1starting",  // 숫자로 시작
		"has space",  // 공백
		"특수문자",   // 한글
	}
	for _, name := range cases {
		r := ts.post("/api/users/me/databases", map[string]string{
			"project_name": name,
		}, token)
		if r.Success {
			t.Errorf("expected failure for name=%q, but succeeded", name)
		}
	}
}

func TestUserDB_Create_Duplicate(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.login(testAdminEmail, testAdminPass)

	body := map[string]string{"project_name": "dupcheck"}
	if !ts.post("/api/users/me/databases", body, token).Success {
		t.Fatal("first create should succeed")
	}
	r := ts.post("/api/users/me/databases", body, token)
	if r.Success {
		t.Fatal("second create should fail (duplicate)")
	}
	if r.Error == nil || r.Error.Code != "DUPLICATE" {
		t.Errorf("expected DUPLICATE error, got %v", r.Error)
	}
}

func TestUserDB_QuotaExceeded(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.login(testAdminEmail, testAdminPass)

	// 기본 maxPerUser=3 이므로 4번째는 거부되어야 함
	for i, proj := range []string{"first", "second", "third"} {
		r := ts.post("/api/users/me/databases", map[string]string{"project_name": proj}, token)
		if !r.Success {
			t.Fatalf("create %d (%s) failed: %v", i+1, proj, r.Error)
		}
	}
	r := ts.post("/api/users/me/databases", map[string]string{"project_name": "fourth"}, token)
	if r.Success {
		t.Fatal("fourth create should have been rejected due to quota")
	}
	if r.Error == nil || r.Error.Code != "QUOTA_EXCEEDED" {
		t.Errorf("expected QUOTA_EXCEEDED, got %v", r.Error)
	}
}

func TestUserDB_Rotate(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.login(testAdminEmail, testAdminPass)

	// 생성
	cr := ts.post("/api/users/me/databases", map[string]string{"project_name": "rotateme"}, token)
	if !cr.Success {
		t.Fatalf("create: %v", cr.Error)
	}
	var created map[string]interface{}
	_ = json.Unmarshal(cr.Data, &created)
	id := int(created["id"].(float64))
	oldPw := created["password"].(string)

	// 재발급
	path := "/api/users/me/databases/" + itoaUD(id) + "/rotate"
	rr := ts.post(path, nil, token)
	if !rr.Success {
		t.Fatalf("rotate: %v", rr.Error)
	}
	var rotated map[string]interface{}
	_ = json.Unmarshal(rr.Data, &rotated)
	newPw := rotated["password"].(string)
	if newPw == oldPw {
		t.Error("rotated password should differ from original")
	}
}

func TestUserDB_Delete(t *testing.T) {
	ts := setupTestServer(t)
	token := ts.login(testAdminEmail, testAdminPass)

	cr := ts.post("/api/users/me/databases", map[string]string{"project_name": "byebye"}, token)
	if !cr.Success {
		t.Fatalf("create: %v", cr.Error)
	}
	var created map[string]interface{}
	_ = json.Unmarshal(cr.Data, &created)
	id := int(created["id"].(float64))

	dr := ts.delete("/api/users/me/databases/"+itoaUD(id), token)
	if !dr.Success {
		t.Fatalf("delete: %v", dr.Error)
	}

	// 목록이 다시 비어 있어야 함
	list := ts.get("/api/users/me/databases", token)
	var items []map[string]interface{}
	_ = json.Unmarshal(list.Data, &items)
	if len(items) != 0 {
		t.Errorf("after delete, list should be empty, got %d", len(items))
	}
}

func TestUserDB_Delete_NotOwner(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	// admin 이 만든 DB
	cr := ts.post("/api/users/me/databases", map[string]string{"project_name": "adminonly"}, adminToken)
	if !cr.Success {
		t.Fatalf("create: %v", cr.Error)
	}
	var created map[string]interface{}
	_ = json.Unmarshal(cr.Data, &created)
	id := int(created["id"].(float64))

	// 다른 유저가 삭제 시도
	_ = ts.register("other@test.com", "pass1234", "other", "2024999")
	// 가입 직후는 pending 상태라 approved 라우트에 접근 못 함.
	// 대신 직접 approved 상태로 만든 뒤 시도하려면 admin 승인이 필요.
	// 여기서는 단순히 "비승인 사용자는 접근 자체가 막힌다" 정도만 확인.
	otherToken := ts.login("other@test.com", "pass1234")
	dr := ts.delete("/api/users/me/databases/"+itoaUD(id), otherToken)
	if dr.Success {
		t.Fatal("non-approved (or non-owner) user should not delete admin's DB")
	}
}

// itoaUD helper (strconv 대신 간단한 int → string)
func itoaUD(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
