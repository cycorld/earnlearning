package integration

import (
	"encoding/json"
	"testing"

	"github.com/earnlearning/backend/internal/domain/company"
)

// Helper: transfer shares from owner to another user via direct DB writes.
// Used to create multi-shareholder scenarios without running a full investment round.
// Writes raw SET (not upsert's add) so we can split the founder's pool precisely.
func (ts *testServer) giveShares(t *testing.T, companyID, ownerUserID, userID, shares int) {
	t.Helper()
	owner, err := ts.companyRepo.FindShareholder(companyID, ownerUserID)
	if err != nil || owner == nil {
		t.Fatalf("find owner shareholder: %v", err)
	}
	if owner.Shares < shares {
		t.Fatalf("owner has only %d shares, need %d", owner.Shares, shares)
	}
	// Debit owner (direct SET)
	if _, err := ts.db.Exec(
		`UPDATE shareholders SET shares = ? WHERE company_id = ? AND user_id = ?`,
		owner.Shares-shares, companyID, ownerUserID,
	); err != nil {
		t.Fatalf("debit owner shares: %v", err)
	}
	// Insert or update recipient
	if _, err := ts.db.Exec(
		`INSERT INTO shareholders (company_id, user_id, shares, acquisition_type) VALUES (?, ?, ?, 'trade')
		 ON CONFLICT(company_id, user_id) DO UPDATE SET shares = shares + ?`,
		companyID, userID, shares, shares,
	); err != nil {
		t.Fatalf("credit recipient shares: %v", err)
	}
}

// =============================================================================
// Basic proposal creation & voting
// =============================================================================

func TestProposal_CreateAndVote_SingleShareholder_Passes(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "prop1@test.com", "prop1", "20240100", "prop1_co")

	// Owner creates a proposal (owner holds 100% of shares)
	r := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type": "general",
		"title":         "사무실 이전 안건",
		"description":   "강남으로 이전합니다",
		"duration_days": 7,
	}, token)
	if !r.Success {
		t.Fatalf("create proposal: %v", r.Error)
	}
	var created struct {
		ID            int    `json:"id"`
		Status        string `json:"status"`
		PassThreshold int    `json:"pass_threshold"`
	}
	_ = json.Unmarshal(r.Data, &created)
	if created.Status != company.ProposalStatusActive {
		t.Errorf("expected active, got %q", created.Status)
	}
	if created.PassThreshold != 50 {
		t.Errorf("expected default threshold 50, got %d", created.PassThreshold)
	}

	// Owner votes yes → since they have 100% shares, proposal auto-passes
	vr := ts.post("/api/proposals/"+itoaUD(created.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)
	if !vr.Success {
		t.Fatalf("cast vote: %v", vr.Error)
	}

	// Fetch and verify auto-close
	gr := ts.get("/api/proposals/"+itoaUD(created.ID), token)
	if !gr.Success {
		t.Fatalf("get proposal: %v", gr.Error)
	}
	var detail struct {
		Status string  `json:"status"`
		Tally  struct {
			YesShares  int     `json:"yes_shares"`
			YesPercent float64 `json:"yes_percent"`
		} `json:"tally"`
	}
	_ = json.Unmarshal(gr.Data, &detail)
	if detail.Status != company.ProposalStatusPassed {
		t.Errorf("expected passed after 100%% yes, got %q", detail.Status)
	}
	if detail.Tally.YesShares != 10000 {
		t.Errorf("expected 10000 yes shares, got %d", detail.Tally.YesShares)
	}
	if detail.Tally.YesPercent < 99 {
		t.Errorf("expected ~100%% yes, got %.1f", detail.Tally.YesPercent)
	}
}

func TestProposal_LiquidationThreshold_DefaultsTo70(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "prop2@test.com", "prop2", "20240101", "prop2_co")

	r := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"proposal_type": "liquidation",
		"title":         "회사 청산",
	}, token)
	if !r.Success {
		t.Fatalf("create liquidation: %v", r.Error)
	}
	var created struct {
		PassThreshold int `json:"pass_threshold"`
	}
	_ = json.Unmarshal(r.Data, &created)
	if created.PassThreshold != 70 {
		t.Errorf("expected default threshold 70 for liquidation, got %d", created.PassThreshold)
	}
}

// =============================================================================
// Authorization
// =============================================================================

func TestProposal_NonShareholder_CannotCreate(t *testing.T) {
	ts := setupTestServer(t)
	_, cid := createUserWithCompany(t, ts, "own3@test.com", "own3", "20240102", "own3_co")
	otherToken := ts.registerAndApprove("stranger@test.com", "pass1234", "stranger", "20240103")

	r := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]string{
		"title": "남의 회사 안건",
	}, otherToken)
	if r.Success {
		t.Fatal("expected non-shareholder to fail")
	}
	if r.Error == nil || r.Error.Code != "NOT_SHAREHOLDER" {
		t.Errorf("expected NOT_SHAREHOLDER, got %v", r.Error)
	}
}

func TestProposal_NonShareholder_CannotVote(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "own4@test.com", "own4", "20240104", "own4_co")
	otherToken := ts.registerAndApprove("stranger2@test.com", "pass1234", "stranger2", "20240105")

	// Owner creates proposal
	r := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title":          "테스트",
		"pass_threshold": 70,
	}, token)
	if !r.Success {
		t.Fatalf("create: %v", r.Error)
	}
	var created struct{ ID int }
	_ = json.Unmarshal(r.Data, &created)

	// Non-shareholder tries to vote
	vr := ts.post("/api/proposals/"+itoaUD(created.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, otherToken)
	if vr.Success {
		t.Fatal("expected non-shareholder vote to fail")
	}
	if vr.Error == nil || vr.Error.Code != "NOT_SHAREHOLDER" {
		t.Errorf("expected NOT_SHAREHOLDER, got %v", vr.Error)
	}
}

func TestProposal_DoubleVote_Rejected(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "own5@test.com", "own5", "20240106", "own5_co")

	// Give some shares away so owner doesn't own 100% and auto-close doesn't trigger
	otherToken := ts.registerAndApprove("sh2@test.com", "pass1234", "sh2", "20240107")
	// Look up other user's ID via profile
	prof := ts.get("/api/auth/me", otherToken)
	var otherUser struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(prof.Data, &otherUser)
	// Look up owner's ID
	prof2 := ts.get("/api/auth/me", token)
	var owner struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(prof2.Data, &owner)
	ts.giveShares(t, cid, owner.ID, otherUser.ID, 3500) // owner 6500 / other 3500

	r := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title":          "테스트",
		"pass_threshold": 70, // owner 65% alone can't pass
	}, token)
	if !r.Success {
		t.Fatalf("create: %v", r.Error)
	}
	var created struct{ ID int }
	_ = json.Unmarshal(r.Data, &created)

	// First vote succeeds
	vr1 := ts.post("/api/proposals/"+itoaUD(created.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)
	if !vr1.Success {
		t.Fatalf("first vote failed: %v", vr1.Error)
	}

	// Second vote from same user should fail
	vr2 := ts.post("/api/proposals/"+itoaUD(created.ID)+"/vote", map[string]string{
		"choice": "no",
	}, token)
	if vr2.Success {
		t.Fatal("expected second vote to fail")
	}
	if vr2.Error == nil || vr2.Error.Code != "ALREADY_VOTED" {
		t.Errorf("expected ALREADY_VOTED, got %v", vr2.Error)
	}
}

// =============================================================================
// Tally & threshold logic
// =============================================================================

func TestProposal_ThresholdReached_Passes(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "own6@test.com", "own6", "20240108", "own6_co")

	// Owner keeps 7500 / other 2500 (owner 75%)
	otherToken := ts.registerAndApprove("sh3@test.com", "pass1234", "sh3", "20240109")
	var owner struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", token).Data, &owner)
	var other struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", otherToken).Data, &other)
	ts.giveShares(t, cid, owner.ID, other.ID, 2500)

	// 70% threshold — use a general proposal so threshold logic is tested
	// independent of liquidation's #033 auto-execute flow.
	r := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title":          "일반 안건",
		"proposal_type":  "general",
		"pass_threshold": 70,
	}, token)
	var created struct{ ID int }
	_ = json.Unmarshal(r.Data, &created)

	// Owner's 75% yes should auto-pass
	ts.post("/api/proposals/"+itoaUD(created.ID)+"/vote", map[string]string{
		"choice": "yes",
	}, token)

	gr := ts.get("/api/proposals/"+itoaUD(created.ID), token)
	var detail struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(gr.Data, &detail)
	if detail.Status != company.ProposalStatusPassed {
		t.Errorf("expected passed (75%% >= 70%% threshold), got %q", detail.Status)
	}
}

func TestProposal_RejectedWhenMathematicallyImpossible(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "own7@test.com", "own7", "20240110", "own7_co")

	// Owner 4000 / other1 4000 / other2 2000 (10000 total)
	other1Token := ts.registerAndApprove("sh4a@test.com", "pass1234", "sh4a", "20240111")
	other2Token := ts.registerAndApprove("sh4b@test.com", "pass1234", "sh4b", "20240112")

	var owner struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", token).Data, &owner)
	var other1 struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", other1Token).Data, &other1)
	var other2 struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", other2Token).Data, &other2)

	ts.giveShares(t, cid, owner.ID, other1.ID, 4000)
	ts.giveShares(t, cid, owner.ID, other2.ID, 2000)
	// Now: owner 4000, other1 4000, other2 2000

	// Threshold 70% → need 7000 yes shares
	r := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title":          "청산",
		"proposal_type":  "liquidation",
		"pass_threshold": 70,
	}, token)
	var created struct{ ID int }
	_ = json.Unmarshal(r.Data, &created)

	// Other1 votes NO (40% no) → remaining 60% cannot reach 70%
	ts.post("/api/proposals/"+itoaUD(created.ID)+"/vote", map[string]string{
		"choice": "no",
	}, other1Token)

	gr := ts.get("/api/proposals/"+itoaUD(created.ID), token)
	var detail struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(gr.Data, &detail)
	if detail.Status != company.ProposalStatusRejected {
		t.Errorf("expected rejected (40%% no > 30%% max no allowed), got %q", detail.Status)
	}
}

func TestProposal_ListByCompany(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "own8@test.com", "own8", "20240113", "own8_co")

	ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title": "안건 1",
	}, token)
	ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title":         "안건 2 (청산)",
		"proposal_type": "liquidation",
	}, token)

	r := ts.get("/api/companies/"+itoaUD(cid)+"/proposals", token)
	if !r.Success {
		t.Fatalf("list: %v", r.Error)
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(r.Data, &list)
	if len(list) != 2 {
		t.Errorf("expected 2 proposals, got %d", len(list))
	}
}

func TestProposal_DuplicateActiveType_Rejected(t *testing.T) {
	ts := setupTestServer(t)
	token, cid := createUserWithCompany(t, ts, "own9@test.com", "own9", "20240114", "own9_co")

	// Give shares so owner 75% (70% threshold would pass, we want to keep this active)
	otherToken := ts.registerAndApprove("sh9@test.com", "pass1234", "sh9", "20240115")
	var owner struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", token).Data, &owner)
	var other struct{ ID int }
	_ = json.Unmarshal(ts.get("/api/auth/me", otherToken).Data, &other)
	ts.giveShares(t, cid, owner.ID, other.ID, 4000) // 6000 / 4000 — owner can't pass 70% alone

	r1 := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title":          "청산 1",
		"proposal_type":  "liquidation",
		"pass_threshold": 70,
	}, token)
	if !r1.Success {
		t.Fatalf("first liquidation proposal failed: %v", r1.Error)
	}

	r2 := ts.post("/api/companies/"+itoaUD(cid)+"/proposals", map[string]interface{}{
		"title":          "청산 2",
		"proposal_type":  "liquidation",
		"pass_threshold": 70,
	}, token)
	if r2.Success {
		t.Fatal("expected duplicate liquidation proposal to fail")
	}
}
