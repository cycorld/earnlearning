package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// #159 멀티 강의(코호트) 지원 — 강의별 지갑 분리 회귀 테스트.
//
// 한 유저가 여러 강의실에 속할 때 지갑(잔액)은 강의실마다 독립적이어야 한다.
// 과거 구조: wallets.user_id UNIQUE (유저당 전역 지갑 1개) → 두 번째 강의 조인 시
// 같은 지갑에 초기자본이 중복 입금되고 잔액이 강의 간에 섞였다.

type classroomData struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	InitialCapital int    `json:"initial_capital"`
}

func (ts *testServer) createClassroom(adminToken, name string, initialCapital int) classroomData {
	ts.t.Helper()
	r := ts.post("/api/classrooms", map[string]interface{}{
		"name": name, "initial_capital": initialCapital,
	}, adminToken)
	if !r.Success {
		ts.t.Fatalf("create classroom %q failed: %v", name, r.Error)
	}
	var c classroomData
	json.Unmarshal(r.Data, &c)
	if c.ID == 0 || c.Code == "" {
		ts.t.Fatalf("create classroom %q: bad response %s", name, string(r.Data))
	}
	return c
}

func (ts *testServer) joinClassroom(token, code string) *apiResponse {
	ts.t.Helper()
	return ts.post("/api/classrooms/join", map[string]string{"code": code}, token)
}

func (ts *testServer) walletBalance(token string) int {
	ts.t.Helper()
	r := ts.get("/api/wallet", token)
	if !r.Success {
		ts.t.Fatalf("get wallet failed: %v", r.Error)
	}
	var w struct {
		Wallet struct {
			Balance int `json:"balance"`
		} `json:"wallet"`
	}
	json.Unmarshal(r.Data, &w)
	return w.Wallet.Balance
}

// TestClassroomWallets_SeparateBalancesPerClassroom
// 강의실 A/B 각각 조인하면 지갑이 강의실별로 따로 생기고,
// 활성 강의실 전환에 따라 해당 강의실 잔액이 보여야 한다.
func TestClassroomWallets_SeparateBalancesPerClassroom(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	crA := ts.createClassroom(adminToken, "강의A", 1_000_000)
	crB := ts.createClassroom(adminToken, "강의B", 2_000_000)

	student := ts.registerAndApprove("cw1@test.com", "pass1234", "학생1", "20260001")

	// A 조인 → A 잔액
	if r := ts.joinClassroom(student, crA.Code); !r.Success {
		t.Fatalf("join A failed: %v", r.Error)
	}
	if got := ts.walletBalance(student); got != 1_000_000 {
		t.Errorf("after join A: balance=%d, want 1000000", got)
	}

	// B 조인 → 활성 강의실이 B로 전환, B 지갑 잔액만 보여야 함 (합산 3_000_000 이면 회귀)
	if r := ts.joinClassroom(student, crB.Code); !r.Success {
		t.Fatalf("join B failed: %v", r.Error)
	}
	if got := ts.walletBalance(student); got != 2_000_000 {
		t.Errorf("after join B: balance=%d, want 2000000 (B wallet only)", got)
	}

	// A로 활성 전환 → A 잔액
	if r := ts.post(fmt.Sprintf("/api/classrooms/%d/activate", crA.ID), nil, student); !r.Success {
		t.Fatalf("activate A failed: %v", r.Error)
	}
	if got := ts.walletBalance(student); got != 1_000_000 {
		t.Errorf("after activate A: balance=%d, want 1000000", got)
	}

	// DB 검증: (user, classroom) 별 지갑 2개
	var cnt int
	if err := ts.db.QueryRow(
		`SELECT COUNT(*) FROM wallets w
		 JOIN users u ON u.id = w.user_id
		 WHERE u.email = 'cw1@test.com'`,
	).Scan(&cnt); err != nil {
		t.Fatalf("count wallets: %v", err)
	}
	if cnt != 2 {
		t.Errorf("wallet rows=%d, want 2 (one per classroom)", cnt)
	}
}

// TestJoinClassroomTwice_NoDoubleCredit
// 같은 강의실 재조인은 멱등 — 초기자본 중복 지급 금지.
func TestJoinClassroomTwice_NoDoubleCredit(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	cr := ts.createClassroom(adminToken, "강의C", 1_000_000)
	student := ts.registerAndApprove("cw2@test.com", "pass1234", "학생2", "20260002")

	for i := 0; i < 2; i++ {
		if r := ts.joinClassroom(student, cr.Code); !r.Success {
			t.Fatalf("join #%d failed: %v", i+1, r.Error)
		}
	}
	if got := ts.walletBalance(student); got != 1_000_000 {
		t.Errorf("balance=%d, want 1000000 (no double credit)", got)
	}
}

// TestActivateClassroom_NonMemberForbidden
// 멤버가 아닌 강의실로는 활성 전환 불가.
func TestActivateClassroom_NonMemberForbidden(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	crA := ts.createClassroom(adminToken, "강의D", 1_000_000)
	crB := ts.createClassroom(adminToken, "강의E", 1_000_000)

	student := ts.registerAndApprove("cw3@test.com", "pass1234", "학생3", "20260003")
	if r := ts.joinClassroom(student, crA.Code); !r.Success {
		t.Fatalf("join A failed: %v", r.Error)
	}

	if r := ts.post(fmt.Sprintf("/api/classrooms/%d/activate", crB.ID), nil, student); r.Success {
		t.Errorf("activate non-member classroom must fail, got success")
	}
}

// TestRanking_ScopedToActiveClassroom
// 랭킹은 활성 강의실 멤버의 해당 강의실 지갑만 집계해야 한다.
func TestRanking_ScopedToActiveClassroom(t *testing.T) {
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)

	crA := ts.createClassroom(adminToken, "강의F", 1_000_000)
	crB := ts.createClassroom(adminToken, "강의G", 2_000_000)

	s1 := ts.registerAndApprove("cw4@test.com", "pass1234", "학생4", "20260004")
	s2 := ts.registerAndApprove("cw5@test.com", "pass1234", "학생5", "20260005")

	ts.joinClassroom(s1, crA.Code) // s1: A만
	ts.joinClassroom(s2, crA.Code)
	ts.joinClassroom(s2, crB.Code) // s2: A+B, 활성=B

	parse := func(r *apiResponse) []int {
		t.Helper()
		if !r.Success {
			t.Fatalf("ranking failed: %v", r.Error)
		}
		var entries []struct {
			UserID int `json:"user_id"`
		}
		json.Unmarshal(r.Data, &entries)
		ids := []int{}
		for _, e := range entries {
			ids = append(ids, e.UserID)
		}
		return ids
	}

	// s1 (활성 A): A 멤버 s1, s2 모두 보임
	idsA := parse(ts.get("/api/wallet/ranking", s1))
	if len(idsA) != 2 {
		t.Errorf("ranking in A: got %d entries (%v), want 2", len(idsA), idsA)
	}

	// s2 (활성 B): B 멤버는 s2 뿐
	idsB := parse(ts.get("/api/wallet/ranking", s2))
	if len(idsB) != 1 {
		t.Errorf("ranking in B: got %d entries (%v), want 1", len(idsB), idsB)
	}
}
