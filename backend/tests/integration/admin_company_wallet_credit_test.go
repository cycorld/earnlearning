package integration

import (
	"encoding/json"
	"testing"
)

type adminCompanyCreditResult struct {
	AdminTransactionID   int `json:"admin_transaction_id"`
	CompanyTransactionID int `json:"company_transaction_id"`
	AdminBalance         int `json:"admin_balance"`
	CompanyBalance       int `json:"company_balance"`
}

func setupAdminCompanyCredit(t *testing.T) (*testServer, string, string, int) {
	t.Helper()
	ts := setupTestServer(t)
	adminToken := ts.login(testAdminEmail, testAdminPass)
	_ = ts.createClassroom(adminToken, "archived", 0)
	classroom := ts.createClassroom(adminToken, "current", 0)
	if classroom.ID != 2 {
		t.Fatalf("classroom id = %d, want 2", classroom.ID)
	}
	if r := ts.joinClassroom(adminToken, classroom.Code); !r.Success {
		t.Fatalf("admin join classroom 2: %v", r.Error)
	}
	if _, err := ts.db.Exec(`UPDATE wallets SET classroom_id = 2, balance = 1000 WHERE user_id = (SELECT id FROM users WHERE email = ?)`, testAdminEmail); err != nil {
		t.Fatalf("fund admin wallet: %v", err)
	}

	owner := ts.registerAndApprove("credit-owner@test.com", "pass1234", "owner", "20000001")
	if r := ts.joinClassroom(owner, classroom.Code); !r.Success {
		t.Fatalf("owner join classroom 2: %v", r.Error)
	}
	company := ts.post("/api/companies", map[string]interface{}{
		"name": "credit-company", "description": "test", "initial_capital": 1_000_000,
	}, owner)
	if !company.Success {
		t.Fatalf("create company: %v", company.Error)
	}
	var created struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(company.Data, &created); err != nil || created.ID == 0 {
		t.Fatalf("decode company: %v data=%s", err, company.Data)
	}
	if _, err := ts.db.Exec(`UPDATE company_wallets SET balance = 0 WHERE company_id = ?`, created.ID); err != nil {
		t.Fatalf("reset company wallet: %v", err)
	}
	return ts, adminToken, owner, created.ID
}

func TestAdminCompanyWalletCredit_AtomicAndIdempotent(t *testing.T) {
	ts, adminToken, _, companyID := setupAdminCompanyCredit(t)
	body := map[string]interface{}{
		"company_id": companyID, "amount": 300, "description": "analytics reward", "idempotency_key": "reward-2026-08-06-1",
	}

	first := ts.post("/api/admin/company-wallet/credit", body, adminToken)
	if !first.Success {
		t.Fatalf("first credit failed: %v", first.Error)
	}
	var got adminCompanyCreditResult
	if err := json.Unmarshal(first.Data, &got); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if got.AdminTransactionID == 0 || got.CompanyTransactionID == 0 || got.AdminBalance != 700 || got.CompanyBalance != 300 {
		t.Fatalf("unexpected result: %+v", got)
	}

	second := ts.post("/api/admin/company-wallet/credit", body, adminToken)
	if !second.Success {
		t.Fatalf("idempotent retry failed: %v", second.Error)
	}
	var retry adminCompanyCreditResult
	_ = json.Unmarshal(second.Data, &retry)
	if retry != got {
		t.Fatalf("retry result = %+v, want %+v", retry, got)
	}

	var adminBalance, companyBalance, adminTxs, companyTxs int
	_ = ts.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = (SELECT id FROM users WHERE email = ?) AND classroom_id = 2`, testAdminEmail).Scan(&adminBalance)
	_ = ts.db.QueryRow(`SELECT balance FROM company_wallets WHERE company_id = ?`, companyID).Scan(&companyBalance)
	_ = ts.db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE id = ?`, got.AdminTransactionID).Scan(&adminTxs)
	_ = ts.db.QueryRow(`SELECT COUNT(*) FROM company_transactions WHERE id = ?`, got.CompanyTransactionID).Scan(&companyTxs)
	if adminBalance != 700 || companyBalance != 300 || adminTxs != 1 || companyTxs != 1 {
		t.Fatalf("persisted admin=%d company=%d adminTxs=%d companyTxs=%d", adminBalance, companyBalance, adminTxs, companyTxs)
	}
}

func TestAdminCompanyWalletCredit_ValidationAndScope(t *testing.T) {
	ts, adminToken, studentToken, companyID := setupAdminCompanyCredit(t)
	valid := map[string]interface{}{"company_id": companyID, "amount": 100, "description": "reward", "idempotency_key": "scope-key"}

	if r := ts.post("/api/admin/company-wallet/credit", valid, studentToken); r.Success {
		t.Fatal("non-admin credit must fail")
	}
	for name, body := range map[string]map[string]interface{}{
		"zero amount":  {"company_id": companyID, "amount": 0, "description": "reward", "idempotency_key": "zero"},
		"missing key":  {"company_id": companyID, "amount": 1, "description": "reward", "idempotency_key": ""},
		"insufficient": {"company_id": companyID, "amount": 1001, "description": "reward", "idempotency_key": "large"},
	} {
		t.Run(name, func(t *testing.T) {
			if r := ts.post("/api/admin/company-wallet/credit", body, adminToken); r.Success {
				t.Fatalf("request must fail: %s", r.Data)
			}
		})
	}

	if _, err := ts.db.Exec(`UPDATE companies SET status = 'dissolved' WHERE id = ?`, companyID); err != nil {
		t.Fatal(err)
	}
	if r := ts.post("/api/admin/company-wallet/credit", map[string]interface{}{"company_id": companyID, "amount": 1, "description": "reward", "idempotency_key": "inactive"}, adminToken); r.Success {
		t.Fatal("inactive company credit must fail")
	}

	if _, err := ts.db.Exec(`UPDATE companies SET status = 'active', classroom_id = 1 WHERE id = ?`, companyID); err != nil {
		t.Fatal(err)
	}
	if r := ts.post("/api/admin/company-wallet/credit", map[string]interface{}{"company_id": companyID, "amount": 1, "description": "reward", "idempotency_key": "wrong-classroom"}, adminToken); r.Success {
		t.Fatal("non-classroom-2 company credit must fail")
	}

	var balance int
	if err := ts.db.QueryRow(`SELECT balance FROM wallets WHERE user_id = (SELECT id FROM users WHERE email = ?) AND classroom_id = 2`, testAdminEmail).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 1000 {
		t.Fatalf("failed requests changed admin balance to %d", balance)
	}
}
