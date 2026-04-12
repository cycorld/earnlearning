package integration

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// Helpers
// =============================================================================

// addCompanyWalletBalance credits the company wallet directly via the repo
// for test setup (simulates a company that has earned revenue).
func (ts *testServer) addCompanyWalletBalance(t *testing.T, companyID, amount int) {
	t.Helper()
	cw, err := ts.companyRepo.FindCompanyWallet(companyID)
	if err != nil || cw == nil {
		t.Fatalf("find company wallet: %v", err)
	}
	if err := ts.companyRepo.CreditCompanyWallet(
		cw.ID, amount, "test_seed", "테스트 자금", "test", 0,
	); err != nil {
		t.Fatalf("credit company wallet: %v", err)
	}
}

// walletBalance returns a user's current wallet balance via /api/wallet.
func walletBalance(t *testing.T, ts *testServer, token string) int {
	t.Helper()
	r := ts.get("/api/wallet", token)
	if !r.Success {
		t.Fatalf("get wallet: %v", r.Error)
	}
	var resp struct {
		Wallet struct {
			Balance int `json:"balance"`
		} `json:"wallet"`
	}
	_ = json.Unmarshal(r.Data, &resp)
	return resp.Wallet.Balance
}

// =============================================================================
// Liquidation execution
// =============================================================================

func TestLiquidation_FullFlow_SingleShareholder(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq1@test.com", "liq1", "20240200", "liq1_co")

	// Company starts with 1M initial capital. Add 9M so wallet has exactly 10M.
	ts.addCompanyWalletBalance(t, cid, 9_000_000)

	// Capture owner's cash before liquidation (after initial capital spent on company)
	balanceBefore := walletBalance(t, ts, token)

	// Owner creates + votes on liquidation proposal (100% shares → auto-passes)
	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type":  "liquidation",
		"title":          "회사 청산",
		"pass_threshold": 70,
	}, token)
	if !pr.Success {
		t.Fatalf("create proposal: %v", pr.Error)
	}
	var proposal struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(pr.Data, &proposal)

	vr := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)
	if !vr.Success {
		t.Fatalf("vote: %v", vr.Error)
	}

	// Verify proposal is now passed
	gr := ts.get("/api/proposals/"+itoaUD(proposal.ID), token)
	var detail struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(gr.Data, &detail)
	if detail.Status != "passed" {
		t.Fatalf("expected passed, got %q", detail.Status)
	}

	// Execute liquidation
	er := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)
	if !er.Success {
		t.Fatalf("execute liquidation: %v", er.Error)
	}
	var result struct {
		TotalBalance  int `json:"total_balance"`
		Tax           int `json:"tax"`
		Distributable int `json:"distributable"`
		Payouts       []struct {
			UserID int `json:"user_id"`
			Amount int `json:"amount"`
		} `json:"payouts"`
	}
	_ = json.Unmarshal(er.Data, &result)

	// Validate: total 10M, tax 20% = 2M, distributable 8M
	if result.TotalBalance != 10_000_000 {
		t.Errorf("total balance: expected 10M, got %d", result.TotalBalance)
	}
	if result.Tax != 2_000_000 {
		t.Errorf("tax: expected 2M, got %d", result.Tax)
	}
	if result.Distributable != 8_000_000 {
		t.Errorf("distributable: expected 8M, got %d", result.Distributable)
	}
	if len(result.Payouts) != 1 {
		t.Fatalf("expected 1 payout, got %d", len(result.Payouts))
	}
	if result.Payouts[0].Amount != 8_000_000 {
		t.Errorf("sole shareholder payout: expected 8M, got %d", result.Payouts[0].Amount)
	}

	// Owner's personal wallet increased by 8M
	balanceAfter := walletBalance(t, ts, token)
	if balanceAfter-balanceBefore != 8_000_000 {
		t.Errorf("owner wallet delta: expected +8M, got %d", balanceAfter-balanceBefore)
	}

	// Company is dissolved
	cr := ts.get("/api/companies/"+itoaUD(cid), token)
	var company struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(cr.Data, &company)
	if company.Status != "dissolved" {
		t.Errorf("company status: expected dissolved, got %q", company.Status)
	}

	// Proposal is marked as executed
	gr2 := ts.get("/api/proposals/"+itoaUD(proposal.ID), token)
	var detail2 struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(gr2.Data, &detail2)
	if detail2.Status != "executed" {
		t.Errorf("proposal status: expected executed, got %q", detail2.Status)
	}
}

func TestLiquidation_MultipleShareholders_DistributedByPercentage(t *testing.T) {
	ts := setupTestServer(t)
	ownerToken, cid := createUserWithCompany(t, ts, "liq2a@test.com", "liq2a", "20240201", "liq2_co")

	// Make a second shareholder with 30% (3000 shares)
	other1Token := ts.registerAndApprove("liq2b@test.com", "pass1234", "liq2b", "20240202")
	// Third shareholder with 20% (2000 shares)
	other2Token := ts.registerAndApprove("liq2c@test.com", "pass1234", "liq2c", "20240203")

	var owner struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", ownerToken).Data, &owner)
	var other1 struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", other1Token).Data, &other1)
	var other2 struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", other2Token).Data, &other2)

	ts.giveShares(t, cid, owner.ID, other1.ID, 3000)
	ts.giveShares(t, cid, owner.ID, other2.ID, 2000)
	// Owner: 5000 (50%), other1: 3000 (30%), other2: 2000 (20%)

	// Company has 10M total (1M initial + 9M added)
	ts.addCompanyWalletBalance(t, cid, 9_000_000)

	// Capture balances before
	ownerBefore := walletBalance(t, ts, ownerToken)
	other1Before := walletBalance(t, ts, other1Token)
	other2Before := walletBalance(t, ts, other2Token)

	// Owner proposes liquidation
	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type":  "liquidation",
		"title":          "청산",
		"pass_threshold": 70,
	}, ownerToken)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)

	// Owner (50%) + other1 (30%) vote yes = 80% → passes
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, ownerToken)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, other1Token)

	// Verify passed
	var detail struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(ts.get("/api/proposals/"+itoaUD(proposal.ID), ownerToken).Data, &detail)
	if detail.Status != "passed" {
		t.Fatalf("expected passed, got %q", detail.Status)
	}

	// Execute
	er := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, ownerToken)
	if !er.Success {
		t.Fatalf("execute: %v", er.Error)
	}

	// Validate distribution
	// Tax: 2M. Distributable: 8M.
	// Owner (50%): 4M. Other1 (30%): 2.4M. Other2 (20%): 1.6M.
	ownerAfter := walletBalance(t, ts, ownerToken)
	other1After := walletBalance(t, ts, other1Token)
	other2After := walletBalance(t, ts, other2Token)

	if ownerAfter-ownerBefore != 4_000_000 {
		t.Errorf("owner payout: expected 4M, got %d", ownerAfter-ownerBefore)
	}
	if other1After-other1Before != 2_400_000 {
		t.Errorf("other1 payout: expected 2.4M, got %d", other1After-other1Before)
	}
	if other2After-other2Before != 1_600_000 {
		t.Errorf("other2 payout: expected 1.6M, got %d", other2After-other2Before)
	}
}

func TestLiquidation_NotPassed_Rejected(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq3@test.com", "liq3", "20240204", "liq3_co")
	ts.addCompanyWalletBalance(t, cid, 5_000_000)

	// Create a liquidation proposal but don't vote yet (active, not passed)
	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type": "liquidation",
		"title":         "청산",
	}, token)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)

	// Try to execute without passing
	er := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)
	if er.Success {
		t.Fatal("expected execution of non-passed proposal to fail")
	}
}

func TestLiquidation_GeneralProposal_CannotExecute(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq4@test.com", "liq4", "20240205", "liq4_co")

	// Create a general proposal (not liquidation)
	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type": "general",
		"title":         "사무실 이전",
	}, token)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)

	// Owner votes yes → passes
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)

	// Try to execute (should fail since it's not liquidation type)
	er := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)
	if er.Success {
		t.Fatal("expected general proposal execution to fail")
	}
}

func TestLiquidation_NonShareholder_CannotExecute(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq5@test.com", "liq5", "20240206", "liq5_co")
	stranger := ts.registerAndApprove("stranger_liq@test.com", "pass1234", "stranger", "20240207")
	ts.addCompanyWalletBalance(t, cid, 1_000_000)

	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type":  "liquidation",
		"title":          "청산",
		"pass_threshold": 70,
	}, token)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)

	er := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, stranger)
	if er.Success {
		t.Fatal("expected stranger execute to fail")
	}
	if er.Error == nil || er.Error.Code != "NOT_SHAREHOLDER" {
		t.Errorf("expected NOT_SHAREHOLDER, got %v", er.Error)
	}
}

func TestLiquidation_DissolvedCompany_BlocksNewDisclosures(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq6@test.com", "liq6", "20240208", "liq6_co")
	ts.addCompanyWalletBalance(t, cid, 1_000_000)

	// Liquidate
	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type":  "liquidation",
		"title":          "청산",
		"pass_threshold": 70,
	}, token)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)

	// Try to create a disclosure on the dissolved company
	dr := ts.post("/api/companies/"+itoaUD(cid)+"/disclosures", map[string]string{
		"content":     "공시 시도",
		"period_from": "2026-04-12",
		"period_to":   "2026-04-12",
	}, token)
	if dr.Success {
		t.Fatal("expected disclosure creation on dissolved company to fail")
	}
}

func TestLiquidation_DissolvedCompany_BlocksNewProposals(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq7@test.com", "liq7", "20240209", "liq7_co")
	ts.addCompanyWalletBalance(t, cid, 1_000_000)

	// Liquidate
	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type":  "liquidation",
		"title":          "청산",
		"pass_threshold": 70,
	}, token)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)

	// Try to create a new proposal
	np := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title": "더 안건",
	}, token)
	if np.Success {
		t.Fatal("expected new proposal on dissolved company to fail")
	}
}

func TestLiquidation_DoubleExecute_Rejected(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq8@test.com", "liq8", "20240210", "liq8_co")
	ts.addCompanyWalletBalance(t, cid, 1_000_000)

	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type":  "liquidation",
		"title":          "청산",
		"pass_threshold": 70,
	}, token)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)

	// First execute succeeds
	er1 := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)
	if !er1.Success {
		t.Fatalf("first execute: %v", er1.Error)
	}

	// Second execute should fail (status=executed now)
	er2 := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)
	if er2.Success {
		t.Fatal("expected double-execute to fail")
	}
}

func TestLiquidation_ZeroBalance_StillDissolves(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "liq9@test.com", "liq9", "20240211", "liq9_co")
	// No balance added — company wallet has 0

	pr := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type":  "liquidation",
		"title":          "청산",
		"pass_threshold": 70,
	}, token)
	var proposal struct{ ID int }
	_ = json.Unmarshal(pr.Data, &proposal)
	ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)

	er := ts.post("/api/proposals/"+itoaUD(proposal.ID)+"/execute", nil, token)
	if !er.Success {
		t.Fatalf("zero-balance liquidation: %v", er.Error)
	}

	// Company should be dissolved even with 0 balance
	cr := ts.get("/api/companies/"+itoaUD(cid), token)
	var company struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(cr.Data, &company)
	if company.Status != "dissolved" {
		t.Errorf("expected dissolved, got %q", company.Status)
	}
}
