package integration

import (
	"encoding/json"
	"fmt"
	"testing"
)

// #159 Phase 2 — 금융 도메인 강의실 격리.
// 회사·거래소·투자·외주·지원금은 생성된 강의실 안에서만 보이고 거래 가능해야 하며,
// 비동기 자금 흐름(배당 등)은 수령자의 활성 강의실이 바뀌어도 원 강의실 지갑으로 간다.

type isoEnv struct {
	ts         *testServer
	adminToken string
	crA, crB   classroomData
	a1, b1     string // a1: A만, b1: B만
	a1ID, b1ID int
}

// registerWithID — 가입+승인 후 (userID, token). 자금은 강의실 조인 초기자본으로 충당.
func registerWithID(t *testing.T, ts *testServer, email, name, studentID string) (int, string) {
	t.Helper()
	token := ts.registerAndApprove(email, "pass1234", name, studentID)
	prof := ts.get("/api/auth/me", token)
	var me struct {
		ID int `json:"id"`
	}
	json.Unmarshal(prof.Data, &me)
	return me.ID, token
}

func setupIsolation(t *testing.T) *isoEnv {
	t.Helper()
	ts := setupTestServer(t)
	admin := ts.login(testAdminEmail, testAdminPass)
	env := &isoEnv{ts: ts, adminToken: admin}
	env.crA = ts.createClassroom(admin, "격리강의A", 60_000_000)
	env.crB = ts.createClassroom(admin, "격리강의B", 60_000_000)

	env.a1ID, env.a1 = registerWithID(t, ts, "iso-a1@test.com", "가나", "20270001")
	env.b1ID, env.b1 = registerWithID(t, ts, "iso-b1@test.com", "다라", "20270002")
	if r := ts.joinClassroom(env.a1, env.crA.Code); !r.Success {
		t.Fatalf("a1 join A: %v", r.Error)
	}
	if r := ts.joinClassroom(env.b1, env.crB.Code); !r.Success {
		t.Fatalf("b1 join B: %v", r.Error)
	}
	return env
}

func (e *isoEnv) createCompany(t *testing.T, token, name string) int {
	t.Helper()
	r := e.ts.post("/api/companies", map[string]interface{}{
		"name": name, "description": "x", "initial_capital": 50_000_000, "logo_url": "",
	}, token)
	if !r.Success {
		t.Fatalf("create company %s: %v", name, r.Error)
	}
	var c struct {
		ID int `json:"id"`
	}
	json.Unmarshal(r.Data, &c)
	return c.ID
}

// activate — 유저의 활성 강의실을 전환한다 (멤버여야 성공).
func (e *isoEnv) activate(t *testing.T, token string, classroomID int) {
	t.Helper()
	if r := e.ts.post(fmt.Sprintf("/api/classrooms/%d/activate", classroomID), nil, token); !r.Success {
		t.Fatalf("activate %d: %v", classroomID, r.Error)
	}
}

type assetView struct {
	Cash          int `json:"cash"`
	StockValue    int `json:"stock_value"`
	CompanyEquity int `json:"company_equity"`
	TotalDebt     int `json:"total_debt"`
}

// assetBreakdown — GET /api/wallet 의 assets 블록만 뽑아온다.
func assetBreakdown(t *testing.T, ts *testServer, token string) assetView {
	t.Helper()
	r := ts.get("/api/wallet", token)
	if !r.Success {
		t.Fatalf("get wallet: %v", r.Error)
	}
	var w struct {
		Assets assetView `json:"assets"`
	}
	if err := json.Unmarshal(r.Data, &w); err != nil {
		t.Fatalf("parse wallet: %v", err)
	}
	return w.Assets
}

func containsID(data json.RawMessage, path string, id int) bool {
	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		// {items: [...]} 형태 지원
		var wrap map[string]json.RawMessage
		if json.Unmarshal(data, &wrap) == nil {
			if inner, ok := wrap[path]; ok {
				json.Unmarshal(inner, &items)
			}
		}
	}
	for _, it := range items {
		if v, ok := it["id"].(float64); ok && int(v) == id {
			return true
		}
	}
	return false
}

// 회사 목록·거래소·주문이 강의실 밖에서 보이거나 실행되면 안 된다.
func TestIsolation_CompanyAndExchange(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	caID := env.createCompany(t, env.a1, "A강의회사")

	// 회사 공개 목록: a1엔 보이고 b1엔 안 보임
	if !containsID(ts.get("/api/companies", env.a1).Data, "companies", caID) {
		t.Errorf("company must be visible in classroom A list")
	}
	if containsID(ts.get("/api/companies", env.b1).Data, "companies", caID) {
		t.Errorf("company must NOT be visible in classroom B list")
	}

	// 거래소 상장 목록
	if !containsID(ts.get("/api/exchange/companies", env.a1).Data, "companies", caID) {
		t.Errorf("exchange listing must be visible in A")
	}
	if containsID(ts.get("/api/exchange/companies", env.b1).Data, "companies", caID) {
		t.Errorf("exchange listing must NOT be visible in B")
	}

	// 타 강의실 주문 차단
	if r := ts.post("/api/exchange/orders", map[string]interface{}{
		"company_id": caID, "order_type": "buy", "shares": 1, "price": 5000,
	}, env.b1); r.Success {
		t.Errorf("cross-classroom exchange order must fail")
	}
}

// 투자 라운드: 목록 격리 + 타 강의실 투자 차단.
func TestIsolation_InvestmentRound(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	caID := env.createCompany(t, env.a1, "A투자회사")
	r := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": caID, "target_amount": 500_000, "offered_percent": 0.2,
	}, env.a1)
	if !r.Success {
		t.Fatalf("create round: %v", r.Error)
	}
	var round struct {
		ID int `json:"id"`
	}
	json.Unmarshal(r.Data, &round)

	if containsID(ts.get("/api/investment/rounds", env.b1).Data, "rounds", round.ID) {
		t.Errorf("round must NOT be listed in classroom B")
	}
	if rr := ts.post(fmt.Sprintf("/api/investment/rounds/%d/invest", round.ID),
		map[string]interface{}{"shares": 1}, env.b1); rr.Success {
		t.Errorf("cross-classroom invest must fail")
	}
}

// 외주 마켓 격리.
func TestIsolation_Freelance(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	r := ts.post("/api/freelance/jobs", map[string]interface{}{
		"title": "A강의외주", "description": "x", "budget": 100_000,
		"skills": "go", "price_type": "fixed",
	}, env.a1)
	if !r.Success {
		t.Fatalf("create job: %v", r.Error)
	}
	var job struct {
		ID int `json:"id"`
	}
	json.Unmarshal(r.Data, &job)

	if containsID(ts.get("/api/freelance/jobs", env.b1).Data, "data", job.ID) {
		t.Errorf("job must NOT be listed in classroom B")
	}
	if rr := ts.post(fmt.Sprintf("/api/freelance/jobs/%d/apply", job.ID),
		map[string]interface{}{"message": "지원"}, env.b1); rr.Success {
		t.Errorf("cross-classroom job apply must fail")
	}
}

// 지원금: 관리자가 A 강의실 컨텍스트에서 만들면 A에서만 보이고 지원 가능.
func TestIsolation_Grant(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	// 관리자는 멤버가 아니어도 활성 강의실 전환 가능해야 함
	if r := ts.post(fmt.Sprintf("/api/classrooms/%d/activate", env.crA.ID), nil, env.adminToken); !r.Success {
		t.Fatalf("admin activate A must succeed: %v", r.Error)
	}

	r := ts.post("/api/admin/grants", map[string]interface{}{
		"title": "A강의지원금", "description": "x", "reward": 500_000, "max_applicants": 5,
	}, env.adminToken)
	if !r.Success {
		t.Fatalf("create grant: %v", r.Error)
	}
	var g struct {
		ID int `json:"id"`
	}
	json.Unmarshal(r.Data, &g)

	if !containsID(ts.get("/api/grants", env.a1).Data, "data", g.ID) {
		t.Errorf("grant must be visible in A")
	}
	if containsID(ts.get("/api/grants", env.b1).Data, "data", g.ID) {
		t.Errorf("grant must NOT be visible in B")
	}
	if rr := ts.post(fmt.Sprintf("/api/grants/%d/apply", g.ID),
		map[string]interface{}{"proposal": "지원합니다"}, env.b1); rr.Success {
		t.Errorf("cross-classroom grant apply must fail")
	}
}

// 게시글 보상은 활성 강의실이 아니라 "채널이 속한 강의실" 지갑으로 지급.
func TestIsolation_PostRewardChannelClassroom(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	// s2: A+B 모두 가입, 활성=B
	_, s2 := registerWithID(t, ts, "iso-s2@test.com", "마바", "20270003")
	if r := ts.joinClassroom(s2, env.crA.Code); !r.Success {
		t.Fatalf("s2 join A: %v", r.Error)
	}
	if r := ts.joinClassroom(s2, env.crB.Code); !r.Success {
		t.Fatalf("s2 join B: %v", r.Error)
	}
	// 활성=B 상태에서 A 강의실 자유채널에 게시글 작성
	chResp := ts.get(fmt.Sprintf("/api/classrooms/%d/channels", env.crA.ID), s2)
	var channels []struct {
		ID          int    `json:"id"`
		ChannelType string `json:"channel_type"`
	}
	json.Unmarshal(chResp.Data, &channels)
	channelID := 0
	for _, ch := range channels {
		if ch.ChannelType == "free" {
			channelID = ch.ID
		}
	}
	if channelID == 0 {
		t.Fatalf("no free channel in A")
	}

	balBBefore := ts.walletBalance(s2) // 활성 B 지갑
	if r := ts.post(fmt.Sprintf("/api/channels/%d/posts", channelID),
		map[string]interface{}{"content": "크로스 강의실 게시글"}, s2); !r.Success {
		t.Fatalf("create post: %v", r.Error)
	}

	// B 지갑 잔액 불변, A 지갑에 보상 지급
	if got := ts.walletBalance(s2); got != balBBefore {
		t.Errorf("active(B) wallet changed by post in A: %d -> %d", balBBefore, got)
	}
	var balA int
	if err := ts.db.QueryRow(
		`SELECT w.balance FROM wallets w JOIN users u ON u.id = w.user_id
		 WHERE u.email = 'iso-s2@test.com' AND w.classroom_id = ?`, env.crA.ID,
	).Scan(&balA); err != nil {
		t.Fatalf("read A wallet: %v", err)
	}
	if balA != 60_000_000+10_000 { // 초기자본 + 게시글 보상
		t.Errorf("A wallet=%d, want %d (initial + post reward)", balA, 60_000_000+10_000)
	}
}

// 송금: 수신자 지갑은 송신자의 활성 강의실 기준으로 귀속.
func TestIsolation_TransferSameClassroomWallet(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	// s3: A+B 가입, 활성=B. a1(활성 A)이 s3에게 송금 → s3의 A 지갑에 입금돼야 함
	s3ID, s3 := registerWithID(t, ts, "iso-s3@test.com", "사아", "20270004")
	if r := ts.joinClassroom(s3, env.crA.Code); !r.Success {
		t.Fatalf("s3 join A: %v", r.Error)
	}
	if r := ts.joinClassroom(s3, env.crB.Code); !r.Success {
		t.Fatalf("s3 join B: %v", r.Error)
	}

	if r := ts.post("/api/wallet/transfer", map[string]interface{}{
		"target_user_id": s3ID, "target_type": "user", "amount": 100_000, "description": "테스트",
	}, env.a1); !r.Success {
		t.Fatalf("transfer: %v", r.Error)
	}

	var balA, balB int
	ts.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = ? AND classroom_id = ?`, s3ID, env.crA.ID).Scan(&balA)
	ts.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = ? AND classroom_id = ?`, s3ID, env.crB.ID).Scan(&balB)
	if balA != 60_000_000+100_000 {
		t.Errorf("recipient A wallet=%d, want %d", balA, 60_000_000+100_000)
	}
	if balB != 60_000_000 {
		t.Errorf("recipient B wallet=%d, want %d (untouched)", balB, 60_000_000)
	}
}

// 배당: 투자자가 활성 강의실을 바꿔도 배당은 회사가 속한 강의실 지갑으로.
func TestIsolation_DividendRoutesToCompanyClassroom(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	caID := env.createCompany(t, env.a1, "A배당회사")
	r := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": caID, "target_amount": 500_000, "offered_percent": 0.2,
	}, env.a1)
	if !r.Success {
		t.Fatalf("create round: %v", r.Error)
	}
	var round struct {
		ID int `json:"id"`
	}
	json.Unmarshal(r.Data, &round)

	// s4: A+B 가입, 활성 A에서 투자 후 활성을 B로 전환
	s4ID, s4 := registerWithID(t, ts, "iso-s4@test.com", "자차", "20270005")
	if r := ts.joinClassroom(s4, env.crA.Code); !r.Success {
		t.Fatalf("s4 join A: %v", r.Error)
	}
	if rr := ts.post(fmt.Sprintf("/api/investment/rounds/%d/invest", round.ID),
		map[string]interface{}{"shares": 100}, s4); !rr.Success {
		t.Fatalf("invest: %v", rr.Error)
	}
	if r := ts.joinClassroom(s4, env.crB.Code); !r.Success {
		t.Fatalf("s4 join B: %v", r.Error)
	}

	var balABefore int
	ts.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = ? AND classroom_id = ?`, s4ID, env.crA.ID).Scan(&balABefore)

	// 배당 실행 (회사 소유주 a1)
	if r := ts.post("/api/investment/dividends", map[string]interface{}{
		"company_id": caID, "total_amount": 100_000,
	}, env.a1); !r.Success {
		t.Fatalf("dividend: %v", r.Error)
	}

	var balAAfter, balB int
	ts.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = ? AND classroom_id = ?`, s4ID, env.crA.ID).Scan(&balAAfter)
	ts.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = ? AND classroom_id = ?`, s4ID, env.crB.ID).Scan(&balB)
	if balAAfter <= balABefore {
		t.Errorf("dividend must credit A wallet: before=%d after=%d", balABefore, balAAfter)
	}
	if balB != 60_000_000 {
		t.Errorf("B wallet=%d, want 60000000 (untouched by dividend)", balB)
	}
}

// #159 자산 분석(GetAssetBreakdown)은 활성 강의실 지갑 기준으로 주식·지분·부채를 집계해야 한다.
// A 강의실에서 창업/대출한 유저가 활성 강의실을 B로 바꾸면 stock_value/company_equity/total_debt = 0.
func TestIsolation_AssetBreakdownScoping(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	// ab: A+B 모두 가입한 유저 (멤버여야 활성 전환 가능)
	abID, ab := registerWithID(t, ts, "iso-ab@test.com", "카타", "20270010")
	if r := ts.joinClassroom(ab, env.crA.Code); !r.Success {
		t.Fatalf("ab join A: %v", r.Error)
	}
	if r := ts.joinClassroom(ab, env.crB.Code); !r.Success {
		t.Fatalf("ab join B: %v", r.Error)
	}

	// 활성 A 에서 창업 → company.classroom_id = A, 소유주 지분 + 회사지갑 발생
	env.activate(t, ab, env.crA.ID)
	env.createCompany(t, ab, "AB자산회사")

	// 활성 A 에서 대출 1건 (직접 삽입 — 승인 절차 생략, 기존 직접-DB 스타일)
	if _, err := ts.db.Exec(
		`INSERT INTO loans (borrower_id, amount, remaining, interest_rate, penalty_rate, purpose, status, classroom_id)
		 VALUES (?, ?, ?, ?, ?, ?, 'active', ?)`,
		abID, 1_000_000, 1_000_000, 0.05, 0.1, "격리테스트", env.crA.ID,
	); err != nil {
		t.Fatalf("insert loan: %v", err)
	}

	// 활성 A: 주식·지분·부채 모두 > 0 이어야 함
	env.activate(t, ab, env.crA.ID)
	aAssets := assetBreakdown(t, ts, ab)
	if aAssets.StockValue == 0 || aAssets.CompanyEquity == 0 || aAssets.TotalDebt == 0 {
		t.Fatalf("active A must show non-zero assets: %+v", aAssets)
	}

	// 활성 B: A 강의실 자산이므로 모두 0 이어야 함 (누수 방지)
	env.activate(t, ab, env.crB.ID)
	bAssets := assetBreakdown(t, ts, ab)
	if bAssets.StockValue != 0 {
		t.Errorf("active B stock_value=%d, want 0 (company belongs to A)", bAssets.StockValue)
	}
	if bAssets.CompanyEquity != 0 {
		t.Errorf("active B company_equity=%d, want 0 (company belongs to A)", bAssets.CompanyEquity)
	}
	if bAssets.TotalDebt != 0 {
		t.Errorf("active B total_debt=%d, want 0 (loan belongs to A)", bAssets.TotalDebt)
	}
}

// #159 내 회사 목록(/companies/mine)도 활성 강의실 기준이어야 한다.
// A 에서 창업한 회사는 활성 B 에서는 내 회사 목록에 안 보이고, 활성 A 에서는 보인다.
func TestIsolation_MyCompaniesScoping(t *testing.T) {
	env := setupIsolation(t)
	ts := env.ts

	_, ab := registerWithID(t, ts, "iso-mine@test.com", "파하", "20270011")
	if r := ts.joinClassroom(ab, env.crA.Code); !r.Success {
		t.Fatalf("ab join A: %v", r.Error)
	}
	if r := ts.joinClassroom(ab, env.crB.Code); !r.Success {
		t.Fatalf("ab join B: %v", r.Error)
	}

	env.activate(t, ab, env.crA.ID)
	caID := env.createCompany(t, ab, "내회사스코프A")

	// 활성 B: 내 회사 목록에 A 회사가 안 보여야 함
	env.activate(t, ab, env.crB.ID)
	if containsID(ts.get("/api/companies/mine", ab).Data, "", caID) {
		t.Errorf("company from A must NOT appear in /companies/mine when active B")
	}

	// 활성 A: 다시 보여야 함
	env.activate(t, ab, env.crA.ID)
	if !containsID(ts.get("/api/companies/mine", ab).Data, "", caID) {
		t.Errorf("company from A must appear in /companies/mine when active A")
	}
}
