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

// =============================================================================
// Early close (#030)
// =============================================================================

func TestInvestment_EarlyClose_PartialFill_Revalues(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "ec-own@test.com", "ecown", "20240400", "ec_co")
	_, aliceToken := createInvestor(t, ts, "ec-a@test.com", "eca", "20240401", 3_000_000)

	// Round: 1M target @ 20% → 2500 new shares, price 400
	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": cid, "target_amount": 1_000_000, "offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	// Alice buys 1000 shares (partial, 40% of round)
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest",
		map[string]int{"shares": 1000}, aliceToken)

	// Owner closes early
	cr := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/close", nil, ownerToken)
	if !cr.Success {
		t.Fatalf("close early: %v", cr.Error)
	}

	// Round should be funded with current_amount, not target_amount
	gr := ts.get("/api/investment/rounds/"+itoaUD(round.ID), aliceToken)
	var got struct {
		Status        string `json:"status"`
		CurrentAmount int    `json:"current_amount"`
	}
	_ = json.Unmarshal(gr.Data, &got)
	if got.Status != "funded" {
		t.Errorf("expected status=funded, got %q", got.Status)
	}
	if got.CurrentAmount != 400_000 {
		t.Errorf("expected current_amount=400000 (actual raised), got %d", got.CurrentAmount)
	}

	// Company valuation = price_per_share × total_shares = 400 × 11000 = 4,400,000
	gc := ts.get("/api/companies/"+itoaUD(cid), ownerToken)
	var co struct {
		TotalShares int `json:"total_shares"`
		Valuation   int `json:"valuation"`
	}
	_ = json.Unmarshal(gc.Data, &co)
	if co.TotalShares != 11000 {
		t.Errorf("total_shares: expected 11000 (founder 10000 + alice 1000), got %d", co.TotalShares)
	}
	if co.Valuation != 4_400_000 {
		t.Errorf("valuation after early close: expected 4.4M, got %d", co.Valuation)
	}
}

func TestInvestment_EarlyClose_ZeroInvestors_Rejected(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "ec0@test.com", "ec0", "20240410", "ec0co")
	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": cid, "target_amount": 1_000_000, "offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	cr := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/close", nil, ownerToken)
	if cr.Success {
		t.Fatal("early close with 0 investors should fail")
	}
}

func TestInvestment_EarlyClose_NonOwner_Forbidden(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "ec-no@test.com", "ecno", "20240420", "ecnoco")
	_, aliceToken := createInvestor(t, ts, "ec-a2@test.com", "eca2", "20240421", 3_000_000)
	strangerToken := ts.registerAndApprove("ec-str@test.com", "pass1234", "str", "20240422")

	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": cid, "target_amount": 1_000_000, "offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest",
		map[string]int{"shares": 500}, aliceToken)

	cr := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/close", nil, strangerToken)
	if cr.Success {
		t.Fatal("stranger early close should fail")
	}
}

// =============================================================================
// Cancel + refund (#030)
// =============================================================================

func TestInvestment_Cancel_FullRefund(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "cx-own@test.com", "cxown", "20240430", "cxco")
	aliceID, aliceToken := createInvestor(t, ts, "cx-a@test.com", "cxa", "20240431", 3_000_000)
	bobID, bobToken := createInvestor(t, ts, "cx-b@test.com", "cxb", "20240432", 3_000_000)
	_ = aliceID
	_ = bobID

	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": cid, "target_amount": 1_000_000, "offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	// Alice 500 + Bob 800 = 1300 shares, 520,000원 raised
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest",
		map[string]int{"shares": 500}, aliceToken)
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest",
		map[string]int{"shares": 800}, bobToken)

	// Wallets before cancel
	aliceBefore := walletBalance(t, ts, aliceToken)
	bobBefore := walletBalance(t, ts, bobToken)

	// Cancel
	cancel := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/cancel", nil, ownerToken)
	if !cancel.Success {
		t.Fatalf("cancel: %v", cancel.Error)
	}

	// Alice refund: 500 shares × 400 = 200,000
	aliceAfter := walletBalance(t, ts, aliceToken)
	if aliceAfter-aliceBefore != 200_000 {
		t.Errorf("alice refund: expected +200k, got %d", aliceAfter-aliceBefore)
	}
	// Bob refund: 800 × 400 = 320,000
	bobAfter := walletBalance(t, ts, bobToken)
	if bobAfter-bobBefore != 320_000 {
		t.Errorf("bob refund: expected +320k, got %d", bobAfter-bobBefore)
	}

	// Round is cancelled
	gr := ts.get("/api/investment/rounds/"+itoaUD(round.ID), aliceToken)
	var state struct{ Status string }
	_ = json.Unmarshal(gr.Data, &state)
	if state.Status != "cancelled" {
		t.Errorf("expected status=cancelled, got %q", state.Status)
	}

	// Company rolled back to 10000 shares, 1M capital, original valuation
	gc := ts.get("/api/companies/"+itoaUD(cid), ownerToken)
	var co struct {
		TotalShares  int `json:"total_shares"`
		TotalCapital int `json:"total_capital"`
		Shareholders []struct {
			Name   string `json:"name"`
			Shares int    `json:"shares"`
		} `json:"shareholders"`
	}
	_ = json.Unmarshal(gc.Data, &co)
	if co.TotalShares != 10_000 {
		t.Errorf("total_shares rolled back: expected 10000, got %d", co.TotalShares)
	}
	if co.TotalCapital != 1_000_000 {
		t.Errorf("total_capital rolled back: expected 1M, got %d", co.TotalCapital)
	}
	// Only founder should remain — alice/bob removed
	if len(co.Shareholders) != 1 || co.Shareholders[0].Shares != 10_000 {
		t.Errorf("expected only founder, got %+v", co.Shareholders)
	}
}

func TestInvestment_Cancel_InsufficientCompanyFunds_Rejected(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "ci-own@test.com", "ciown", "20240440", "cico")
	_, aliceToken := createInvestor(t, ts, "ci-a@test.com", "cia", "20240441", 3_000_000)

	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": cid, "target_amount": 1_000_000, "offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	// Alice invests 500 shares = 200k → company wallet now holds 1,200,000
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest",
		map[string]int{"shares": 500}, aliceToken)

	// Simulate company spending its money: admin drains the wallet via dividend
	ts.post("/api/investment/dividends", map[string]interface{}{
		"company_id": cid, "total_amount": 1_100_000,
	}, ownerToken)

	// Now cancel should fail — wallet below the refund amount
	cancel := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/cancel", nil, ownerToken)
	if cancel.Success {
		t.Fatal("cancel should fail when company wallet can't cover refund")
	}
}

func TestInvestment_Cancel_NonOwner_Forbidden(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "cn-own@test.com", "cnown", "20240450", "cnco")
	_, aliceToken := createInvestor(t, ts, "cn-a@test.com", "cna", "20240451", 3_000_000)
	strangerToken := ts.registerAndApprove("cn-str@test.com", "pass1234", "cnstr", "20240452")

	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": cid, "target_amount": 1_000_000, "offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest",
		map[string]int{"shares": 500}, aliceToken)

	cancel := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/cancel", nil, strangerToken)
	if cancel.Success {
		t.Fatal("stranger cancel should fail")
	}
}

func TestInvestment_CloseOrCancel_AlreadyFundedRound_Rejected(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "rf-own@test.com", "rfown", "20240460", "rfco")
	_, aliceToken := createInvestor(t, ts, "rf-a@test.com", "rfa", "20240461", 3_000_000)

	rr := ts.post("/api/investment/rounds", map[string]interface{}{
		"company_id": cid, "target_amount": 500_000, "offered_percent": 0.2,
	}, ownerToken)
	var round struct{ ID int }
	_ = json.Unmarshal(rr.Data, &round)

	// Fully fund → status=funded
	ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/invest",
		map[string]int{"shares": 2500}, aliceToken)

	// Both close and cancel should fail for a non-open round
	close1 := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/close", nil, ownerToken)
	if close1.Success {
		t.Fatal("close on funded round should fail")
	}
	cancel := ts.post("/api/investment/rounds/"+itoaUD(round.ID)+"/cancel", nil, ownerToken)
	if cancel.Success {
		t.Fatal("cancel on funded round should fail")
	}
}

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
