package integration

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// Helpers shared with liquidation_test.go  (addCompanyWalletBalance,
// walletBalance, giveShares) already exist; add an investor helper here.
// =============================================================================

// createInvestor registers + approves + funds a fresh student.
// Returns the user's ID + token.
func createInvestor(t *testing.T, ts *testServer, email, name, studentID string, funding int) (int, string) {
	t.Helper()
	token := ts.registerAndApprove(email, "pass1234", name, studentID)
	// Look up user ID via /auth/me
	prof := ts.get("/api/auth/me", token)
	var me struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(prof.Data, &me)
	// Admin transfer funds
	adminToken := ts.login(testAdminEmail, testAdminPass)
	r := ts.post("/api/admin/wallet/transfer", map[string]interface{}{
		"target_user_ids": []int{me.ID},
		"amount":          funding,
		"description":     "테스트 자금",
	}, adminToken)
	if !r.Success {
		t.Fatalf("fund investor: %v", r.Error)
	}
	return me.ID, token
}

// =============================================================================
// Full round (single investor takes entire allocation)
// =============================================================================

func TestInvestment_FullRound_SingleInvestor_Funds(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "inv-own-1@test.com", "owner1", "20240300", "invco1")
	_ = ownerToken
	_, investorToken := createInvestor(t, ts, "inv-a@test.com", "invA", "20240301", 3_000_000)

	// Create round: 500k @ 20% → new_shares=2500, price=200
	r := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id":      cid,
		"target_amount":   500_000,
		"offered_percent": 0.2,
	}, ownerToken)
	if !r.Success {
		t.Fatalf("create round: %v", r.Error)
	}
	var round struct {
		ID            int     `json:"id"`
		NewShares     int     `json:"new_shares"`
		PricePerShare float64 `json:"price_per_share"`
		Status        string  `json:"status"`
	}
	_ = json.Unmarshal(r.Data, &round)
	if round.NewShares != 2500 || round.PricePerShare != 200 || round.Status != "open" {
		t.Fatalf("unexpected round state: %+v", round)
	}

	// Single investor takes full 2500 shares
	ir := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{
		"shares": 2500,
	}, investorToken)
	if !ir.Success {
		t.Fatalf("invest: %v", ir.Error)
	}

	// Verify round is fully funded
	gr := ts.get("/api/investment/rounds/"+itoaUD(round.ID), investorToken)
	if !gr.Success {
		t.Fatalf("get round: %v", gr.Error)
	}
	var post struct {
		Status        string `json:"status"`
		CurrentAmount int    `json:"current_amount"`
	}
	_ = json.Unmarshal(gr.Data, &post)
	if post.Status != "funded" {
		t.Errorf("expected status=funded, got %q", post.Status)
	}
	if post.CurrentAmount != 500_000 {
		t.Errorf("expected current_amount=500000, got %d", post.CurrentAmount)
	}
}

// =============================================================================
// Partial round (multiple investors split allocation)
// =============================================================================

func TestInvestment_PartialRound_MultipleInvestors(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "inv-own-2@test.com", "owner2", "20240310", "invco2")
	_, aliceToken := createInvestor(t, ts, "alice-inv@test.com", "alice", "20240311", 3_000_000)
	_, bobToken := createInvestor(t, ts, "bob-inv@test.com", "bob", "20240312", 3_000_000)

	// Round: 1,000,000 @ 20% → new_shares = round(10000*0.2/0.8) = 2500, price = 400
	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id":      cid,
		"target_amount":   1_000_000,
		"offered_percent": 0.2,
	}, ownerToken)
	if !rr.Success {
		t.Fatalf("create round: %v", rr.Error)
	}
	var round struct {
		ID            int     `json:"id"`
		NewShares     int     `json:"new_shares"`
		PricePerShare float64 `json:"price_per_share"`
	}
	_ = json.Unmarshal(rr.Data, &round)
	if round.NewShares != 2500 || round.PricePerShare != 400 {
		t.Fatalf("unexpected round: %+v", round)
	}

	// Alice buys 1000 shares → 400,000원
	ar := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": 1000}, aliceToken)
	if !ar.Success {
		t.Fatalf("alice invest: %v", ar.Error)
	}
	var aliceInv struct {
		Shares int `json:"shares"`
		Amount int `json:"amount"`
	}
	_ = json.Unmarshal(ar.Data, &aliceInv)
	if aliceInv.Shares != 1000 || aliceInv.Amount != 400_000 {
		t.Errorf("alice investment mismatch: %+v", aliceInv)
	}

	// After alice: round should still be open
	g1 := ts.get("/api/investment/rounds/"+itoaUD(round.ID), aliceToken)
	var mid struct {
		Status        string `json:"status"`
		CurrentAmount int    `json:"current_amount"`
	}
	_ = json.Unmarshal(g1.Data, &mid)
	if mid.Status != "open" {
		t.Errorf("after partial: expected status=open, got %q", mid.Status)
	}
	if mid.CurrentAmount != 400_000 {
		t.Errorf("after partial: expected current_amount=400000, got %d", mid.CurrentAmount)
	}

	// Bob tries to buy 2000 shares — only 1500 remaining
	br := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": 2000}, bobToken)
	if br.Success {
		t.Fatal("expected overbuy to fail")
	}

	// Bob buys exactly 1500 (remaining) → should close the round
	br2 := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": 1500}, bobToken)
	if !br2.Success {
		t.Fatalf("bob final invest: %v", br2.Error)
	}
	var bobInv struct {
		Shares int `json:"shares"`
		Amount int `json:"amount"`
	}
	_ = json.Unmarshal(br2.Data, &bobInv)
	// Bob's amount = target_amount - current_amount (600k) after rounding fix
	if bobInv.Amount != 600_000 {
		t.Errorf("bob final amount: expected 600000, got %d", bobInv.Amount)
	}

	// Round now funded
	g2 := ts.get("/api/investment/rounds/"+itoaUD(round.ID), aliceToken)
	var final struct {
		Status        string `json:"status"`
		CurrentAmount int    `json:"current_amount"`
	}
	_ = json.Unmarshal(g2.Data, &final)
	if final.Status != "funded" {
		t.Errorf("expected funded, got %q", final.Status)
	}
	if final.CurrentAmount != 1_000_000 {
		t.Errorf("expected current_amount=1M, got %d", final.CurrentAmount)
	}

	// Company should now have: 10000 founder + 1000 alice + 1500 bob = 12500 shares
	// Valuation = 1M / 0.2 = 5M
	c := ts.get("/api/companies/"+itoaUD(cid), aliceToken)
	var company struct {
		TotalShares int `json:"total_shares"`
		Valuation   int `json:"valuation"`
	}
	_ = json.Unmarshal(c.Data, &company)
	if company.TotalShares != 12500 {
		t.Errorf("expected total_shares=12500, got %d", company.TotalShares)
	}
	if company.Valuation != 5_000_000 {
		t.Errorf("expected valuation=5M, got %d", company.Valuation)
	}
}

// =============================================================================
// Portfolio response shape (Bug C regression)
// =============================================================================

func TestInvestment_Portfolio_ResponseShape(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "inv-own-3@test.com", "owner3", "20240320", "invco3")
	_, investorToken := createInvestor(t, ts, "port-inv@test.com", "port", "20240321", 2_000_000)

	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id":      cid,
		"target_amount":   500_000,
		"offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	// Fully invest
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": 2500}, investorToken)

	// Fetch portfolio
	pr := ts.get("/api/investment/portfolio", investorToken)
	if !pr.Success {
		t.Fatalf("portfolio: %v", pr.Error)
	}
	var portfolio []struct {
		Company struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Valuation int    `json:"valuation"`
		} `json:"company"`
		Shares            int     `json:"shares"`
		InvestedAmount    int     `json:"invested_amount"`
		CurrentValue      int     `json:"current_value"`
		Profit            int     `json:"profit"`
		DividendsReceived int     `json:"dividends_received"`
		Percentage        float64 `json:"percentage"`
	}
	if err := json.Unmarshal(pr.Data, &portfolio); err != nil {
		t.Fatalf("portfolio parse: %v", err)
	}
	if len(portfolio) != 1 {
		t.Fatalf("expected 1 portfolio item, got %d", len(portfolio))
	}
	p := portfolio[0]
	if p.Company.ID != cid {
		t.Errorf("company.id: expected %d, got %d", cid, p.Company.ID)
	}
	if p.Company.Name != "invco3" {
		t.Errorf("company.name: expected invco3, got %q", p.Company.Name)
	}
	if p.Shares != 2500 {
		t.Errorf("shares: expected 2500, got %d", p.Shares)
	}
	if p.InvestedAmount != 500_000 {
		t.Errorf("invested_amount: expected 500k, got %d", p.InvestedAmount)
	}
	if p.Percentage < 19 || p.Percentage > 21 {
		t.Errorf("percentage: expected ~20%%, got %.2f", p.Percentage)
	}
	// Current value after full invest = 2,500,000 × 2500/12500 = 500,000
	if p.CurrentValue != 500_000 {
		t.Errorf("current_value: expected 500k, got %d", p.CurrentValue)
	}
	if p.Profit != 0 {
		t.Errorf("profit at funded round: expected 0, got %d", p.Profit)
	}
	// No dividends yet
	if p.DividendsReceived != 0 {
		t.Errorf("dividends_received: expected 0, got %d", p.DividendsReceived)
	}
}

// =============================================================================
// Authorization / validation
// =============================================================================

func TestInvestment_OwnerCannotInvest(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "own-self@test.com", "self", "20240330", "selfco")
	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id":      cid,
		"target_amount":   500_000,
		"offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	ir := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": 100}, ownerToken)
	if ir.Success {
		t.Fatal("owner invest should fail")
	}
}

func TestInvestment_InvalidShares(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "own-v@test.com", "ownv", "20240340", "vco")
	_, invToken := createInvestor(t, ts, "inv-v@test.com", "invv", "20240341", 3_000_000)
	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id":      cid,
		"target_amount":   500_000,
		"offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	// 0 shares
	r := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": 0}, invToken)
	if r.Success {
		t.Fatal("0 shares should fail")
	}
	// negative
	r = ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": -1}, invToken)
	if r.Success {
		t.Fatal("negative shares should fail")
	}
}

// =============================================================================
// KPI Rule owner check (Bug E)
// =============================================================================

func TestInvestment_KpiRule_OwnerCheck(t *testing.T) {
	ts := setupTestServer(t)
	_, cid := createUserWithCompany(t, ts, "kpi-own@test.com", "kpiown", "20240350", "kpico")
	strangerToken := ts.registerAndApprove("kpi-str@test.com", "pass1234", "str", "20240351")

	r := ts.post("/api/investment/kpi-rules", map[string]interface{}{
		"company_id":       cid,
		"rule_description": "Monthly revenue > 5M",
	}, strangerToken)
	if r.Success {
		t.Fatal("stranger KPI rule should fail")
	}
}

// =============================================================================
// Dividend → DividendsReceived in portfolio rolls up
// =============================================================================

func TestInvestment_DividendsRolledIntoPortfolio(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "div-own@test.com", "divown", "20240360", "divco")
	_, invToken := createInvestor(t, ts, "div-inv@test.com", "divinv", "20240361", 2_000_000)

	// Create and fully fund round
	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id":      cid,
		"target_amount":   500_000,
		"offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest", map[string]int{"shares": 2500}, invToken)

	// Execute 100k dividend. Investor has 20% → 20000
	dr := ts.post("/api/investment/dividends", map[string]interface{}{
		"company_id":   cid,
		"total_amount": 100_000,
	}, ownerToken)
	if !dr.Success {
		t.Fatalf("dividend: %v", dr.Error)
	}

	// Portfolio should show dividends_received = 20000
	pr := ts.get("/api/investment/portfolio", invToken)
	var portfolio []struct {
		DividendsReceived int `json:"dividends_received"`
	}
	_ = json.Unmarshal(pr.Data, &portfolio)
	if len(portfolio) != 1 || portfolio[0].DividendsReceived != 20_000 {
		t.Errorf("expected dividends_received=20000, got %v", portfolio)
	}
}
