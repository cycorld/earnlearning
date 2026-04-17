package integration

import (
	"encoding/json"
	"testing"
)

// #031 송금 주체 확장(개인↔법인) + 법인 Wallet 페이지

// companyWalletBalance returns the current balance of a company wallet.
func (ts *testServer) companyWalletBalance(t *testing.T, companyID int) int {
	t.Helper()
	cw, err := ts.companyRepo.FindCompanyWallet(companyID)
	if err != nil || cw == nil {
		t.Fatalf("find company wallet %d: %v", companyID, err)
	}
	return cw.Balance
}

// getMeID returns the caller's user id via /api/auth/me.
func getMeID(t *testing.T, ts *testServer, token string) int {
	t.Helper()
	r := ts.get("/api/auth/me", token)
	if !r.Success {
		t.Fatalf("auth/me: %v", r.Error)
	}
	var me struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(r.Data, &me)
	return me.ID
}

// =============================================================================
// Gap 1 — 개인 → 법인 송금
// =============================================================================

func TestTransfer_UserToCompany_Success(t *testing.T) {
	ts := setupTestServer(t)

	// Founder sets up company (captures initial capital into the company wallet).
	founderToken, cid := createUserWithCompany(t, ts, "mt-founder@test.com", "founder", "20250001", "mt_co1")
	_ = founderToken

	// Separate sender with funds.
	_, senderToken := createInvestor(t, ts, "mt-sender@test.com", "sender", "20250002", 500_000)

	beforeUser := walletBalance(t, ts, senderToken)
	beforeCompany := ts.companyWalletBalance(t, cid)

	r := ts.post("/api/wallet/transfer", map[string]interface{}{
		"target_user_id": cid,
		"target_type":    "company",
		"amount":         100_000,
		"description":    "스폰서십",
	}, senderToken)
	if !r.Success {
		t.Fatalf("user→company transfer failed: %v", r.Error)
	}

	afterUser := walletBalance(t, ts, senderToken)
	afterCompany := ts.companyWalletBalance(t, cid)

	if afterUser != beforeUser-100_000 {
		t.Errorf("sender balance: expected %d, got %d", beforeUser-100_000, afterUser)
	}
	if afterCompany != beforeCompany+100_000 {
		t.Errorf("company balance: expected %d, got %d", beforeCompany+100_000, afterCompany)
	}
}

func TestTransfer_UserToCompany_InsufficientFunds_RollsBack(t *testing.T) {
	ts := setupTestServer(t)

	_, cid := createUserWithCompany(t, ts, "mt-f2@test.com", "f2", "20250010", "mt_co2")
	_, senderToken := createInvestor(t, ts, "mt-poor@test.com", "poor", "20250011", 10_000)

	beforeUser := walletBalance(t, ts, senderToken)
	beforeCompany := ts.companyWalletBalance(t, cid)

	r := ts.post("/api/wallet/transfer", map[string]interface{}{
		"target_user_id": cid,
		"target_type":    "company",
		"amount":         100_000, // > 10_000 잔액
		"description":    "무리한 송금",
	}, senderToken)
	if r.Success {
		t.Fatal("transfer should fail with insufficient funds")
	}

	if walletBalance(t, ts, senderToken) != beforeUser {
		t.Errorf("sender balance changed on failed transfer")
	}
	if ts.companyWalletBalance(t, cid) != beforeCompany {
		t.Errorf("company balance changed on failed transfer")
	}
}

// =============================================================================
// Gap 2 — 법인 → 개인/법인 송금 (대표만 가능)
// =============================================================================

func TestTransfer_CompanyToUser_ByOwner_Success(t *testing.T) {
	ts := setupTestServer(t)

	ownerToken, cid := createUserWithCompany(t, ts, "ct-own@test.com", "own", "20250020", "ct_co1")
	// Company starts with initial_capital 1_000_000 from createUserWithCompany.
	recipientToken := ts.registerAndApprove("ct-recv@test.com", "pass1234", "recv", "20250021")
	recipientID := getMeID(t, ts, recipientToken)

	beforeRecv := walletBalance(t, ts, recipientToken)
	beforeCompany := ts.companyWalletBalance(t, cid)

	r := ts.post("/api/companies/"+itoaUD(cid)+"/transfer", map[string]interface{}{
		"target_id":   recipientID,
		"target_type": "user",
		"amount":      250_000,
		"description": "월급",
	}, ownerToken)
	if !r.Success {
		t.Fatalf("company→user transfer failed: %v", r.Error)
	}

	afterRecv := walletBalance(t, ts, recipientToken)
	afterCompany := ts.companyWalletBalance(t, cid)

	if afterRecv != beforeRecv+250_000 {
		t.Errorf("recipient balance: expected %d, got %d", beforeRecv+250_000, afterRecv)
	}
	if afterCompany != beforeCompany-250_000 {
		t.Errorf("company balance: expected %d, got %d", beforeCompany-250_000, afterCompany)
	}
}

func TestTransfer_CompanyToCompany_Success(t *testing.T) {
	ts := setupTestServer(t)

	ownerToken, cidA := createUserWithCompany(t, ts, "ct2-a@test.com", "ca", "20250030", "ct2_coA")
	_, cidB := createUserWithCompany(t, ts, "ct2-b@test.com", "cb", "20250031", "ct2_coB")

	beforeA := ts.companyWalletBalance(t, cidA)
	beforeB := ts.companyWalletBalance(t, cidB)

	r := ts.post("/api/companies/"+itoaUD(cidA)+"/transfer", map[string]interface{}{
		"target_id":   cidB,
		"target_type": "company",
		"amount":      300_000,
		"description": "B2B 결제",
	}, ownerToken)
	if !r.Success {
		t.Fatalf("company→company transfer failed: %v", r.Error)
	}

	if ts.companyWalletBalance(t, cidA) != beforeA-300_000 {
		t.Errorf("A balance mismatch")
	}
	if ts.companyWalletBalance(t, cidB) != beforeB+300_000 {
		t.Errorf("B balance mismatch")
	}
}

func TestTransfer_CompanyToUser_NonOwner_Forbidden(t *testing.T) {
	ts := setupTestServer(t)

	_, cid := createUserWithCompany(t, ts, "nof-own@test.com", "nofo", "20250040", "nof_co")
	_, strangerToken := createInvestor(t, ts, "nof-str@test.com", "str", "20250041", 100_000)
	strangerID := getMeID(t, ts, strangerToken)

	before := ts.companyWalletBalance(t, cid)

	r := ts.post("/api/companies/"+itoaUD(cid)+"/transfer", map[string]interface{}{
		"target_id":   strangerID,
		"target_type": "user",
		"amount":      50_000,
		"description": "몰래 빼가기",
	}, strangerToken)
	if r.Success {
		t.Fatal("non-owner should not be allowed to transfer from company")
	}
	if r.Error == nil || r.Error.Code != "NOT_OWNER" {
		t.Errorf("expected NOT_OWNER error, got %+v", r.Error)
	}

	if ts.companyWalletBalance(t, cid) != before {
		t.Error("company balance changed despite forbidden transfer")
	}
}

func TestTransfer_CompanyToUser_InsufficientFunds(t *testing.T) {
	ts := setupTestServer(t)

	ownerToken, cid := createUserWithCompany(t, ts, "ins-own@test.com", "insO", "20250050", "ins_co")
	recvToken := ts.registerAndApprove("ins-r@test.com", "pass1234", "insR", "20250051")
	recvID := getMeID(t, ts, recvToken)

	beforeCompany := ts.companyWalletBalance(t, cid) // 1_000_000 from createUserWithCompany
	beforeRecv := walletBalance(t, ts, recvToken)

	r := ts.post("/api/companies/"+itoaUD(cid)+"/transfer", map[string]interface{}{
		"target_id":   recvID,
		"target_type": "user",
		"amount":      99_000_000, // 잔액 훨씬 초과
		"description": "무리한 지출",
	}, ownerToken)
	if r.Success {
		t.Fatal("should fail with insufficient funds")
	}
	if ts.companyWalletBalance(t, cid) != beforeCompany {
		t.Error("company balance drifted on failed transfer")
	}
	if walletBalance(t, ts, recvToken) != beforeRecv {
		t.Error("recipient credited despite failed transfer")
	}
}

// =============================================================================
// Gap 1 bis — SearchRecipients includes companies
// =============================================================================

func TestSearchRecipients_IncludesCompanies(t *testing.T) {
	ts := setupTestServer(t)

	_, cid := createUserWithCompany(t, ts, "sr-own@test.com", "srOwn", "20250060", "search_target_co")
	_, senderToken := createInvestor(t, ts, "sr-s@test.com", "srS", "20250061", 100_000)

	r := ts.get("/api/wallet/recipients?q=search_target", senderToken)
	if !r.Success {
		t.Fatalf("search recipients: %v", r.Error)
	}
	var recs []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	_ = json.Unmarshal(r.Data, &recs)

	foundCompany := false
	for _, rc := range recs {
		if rc.Type == "company" && rc.ID == cid {
			foundCompany = true
			// 컨벤션: "회사명(대표자명)"
			if rc.Name != "search_target_co(srOwn)" {
				t.Errorf("company account name convention mismatch, got %q", rc.Name)
			}
			break
		}
	}
	if !foundCompany {
		t.Errorf("expected company (id=%d) in recipients, got %+v", cid, recs)
	}
}

// 대표자명으로도 법인 검색이 되는지 검증
func TestSearchRecipients_FindsCompanyByOwnerName(t *testing.T) {
	ts := setupTestServer(t)

	_, cid := createUserWithCompany(t, ts, "on-own@test.com", "uniqueOwnerXyz", "20250080", "ownername_co")
	_, searcherToken := createInvestor(t, ts, "on-s@test.com", "onS", "20250081", 100_000)

	r := ts.get("/api/wallet/recipients?q=uniqueOwnerXyz", searcherToken)
	if !r.Success {
		t.Fatalf("search: %v", r.Error)
	}
	var recs []struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
	}
	_ = json.Unmarshal(r.Data, &recs)
	hit := false
	for _, rc := range recs {
		if rc.Type == "company" && rc.ID == cid {
			hit = true
			break
		}
	}
	if !hit {
		t.Errorf("expected to find company by owner name, got %+v", recs)
	}
}

// =============================================================================
// Gap 3 — Company wallet read endpoints
// =============================================================================

func TestGetCompanyWallet_ReturnsBalanceAndTransactions(t *testing.T) {
	ts := setupTestServer(t)

	ownerToken, cid := createUserWithCompany(t, ts, "cw-own@test.com", "cwO", "20250070", "cw_co")

	// Trigger a transaction (user→company 송금) so 거래내역이 존재하도록
	_, senderToken := createInvestor(t, ts, "cw-s@test.com", "cwS", "20250071", 500_000)
	ts.post("/api/wallet/transfer", map[string]interface{}{
		"target_user_id": cid,
		"target_type":    "company",
		"amount":         200_000,
		"description":    "seed",
	}, senderToken)

	r := ts.get("/api/companies/"+itoaUD(cid)+"/wallet", ownerToken)
	if !r.Success {
		t.Fatalf("get company wallet: %v", r.Error)
	}
	var resp struct {
		Wallet struct {
			Balance int `json:"balance"`
		} `json:"wallet"`
	}
	_ = json.Unmarshal(r.Data, &resp)
	if resp.Wallet.Balance != 1_200_000 { // 1_000_000 (initial) + 200_000 (seed)
		t.Errorf("expected balance 1200000, got %d", resp.Wallet.Balance)
	}

	// Transactions endpoint
	tr := ts.get("/api/companies/"+itoaUD(cid)+"/transactions?page=1&limit=20", ownerToken)
	if !tr.Success {
		t.Fatalf("get company txs: %v", tr.Error)
	}
	var txResp struct {
		Data []struct {
			Amount      int    `json:"amount"`
			Description string `json:"description"`
		} `json:"data"`
	}
	_ = json.Unmarshal(tr.Data, &txResp)
	if len(txResp.Data) == 0 {
		t.Error("expected at least one transaction, got none")
	}
}
