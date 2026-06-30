package integration

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/wallet"
	"github.com/earnlearning/backend/internal/infrastructure/persistence"
)

// failingWalletRepo wraps a real wallet.Repository but always fails Credit, to
// simulate a write failing partway through matching (after CreateTrade + buyer
// Debit have already run). Everything else delegates to the real repo so the
// earlier steps genuinely write — proving they get rolled back.
type failingWalletRepo struct {
	wallet.Repository
}

func (f *failingWalletRepo) Credit(int, int, wallet.TxType, string, string, int) error {
	return fmt.Errorf("injected credit failure")
}

func (f *failingWalletRepo) WithTx(tx *sql.Tx) wallet.Repository {
	return &failingWalletRepo{Repository: f.Repository.WithTx(tx)}
}

// Regression (#142): if any step of runMatching fails, the whole settlement must
// roll back — no trade row, no wallet movement, no shareholder transfer. Before the
// transaction wrap, CreateTrade and the buyer debit were already committed when a
// later step failed, leaving "money moved but shares didn't" half-state (#143).
func TestExchange_Matching_RollsBackOnMidFailure(t *testing.T) {
	ts := setupTestServer(t)

	sellerID, sellerToken := createInvestor(t, ts, "atomic-seller@test.com", "seller", "20240801", 60_000_000)
	buyerID, _ := createInvestor(t, ts, "atomic-buyer@test.com", "buyer", "20240802", 60_000_000)

	// Seller creates a listed company (50M capital → auto-list, 10000 founding shares).
	r := ts.post("/api/companies", map[string]interface{}{
		"name": "원자성테스트사", "description": "롤백 회귀", "initial_capital": 50_000_000, "logo_url": "",
	}, sellerToken)
	if !r.Success {
		t.Fatalf("create company: %v", r.Error)
	}
	var c struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(r.Data, &c)

	// Seller rests a sell order the buyer will cross.
	if sr := ts.post("/api/exchange/orders", map[string]interface{}{
		"company_id": c.ID, "order_type": "sell", "shares": 5, "price": 5000,
	}, sellerToken); !sr.Success {
		t.Fatalf("seller sell order: %v", sr.Error)
	}

	// Build an exchange use case on the SAME db whose wallet Credit always fails,
	// then drive the crossing buy through it directly.
	walletRepo := persistence.NewWalletRepo(ts.db)
	companyRepo := persistence.NewCompanyRepo(ts.db)
	exchangeRepo := persistence.NewExchangeRepo(ts.db)
	uc := application.NewExchangeUseCase(exchangeRepo, companyRepo, &failingWalletRepo{Repository: walletRepo})
	uc.SetDB(ts.db)

	// Pre-state.
	buyerWalletBefore, err := walletRepo.FindByUserID(buyerID)
	if err != nil {
		t.Fatalf("buyer wallet: %v", err)
	}
	sellerSHBefore, err := companyRepo.FindShareholder(c.ID, sellerID)
	if err != nil || sellerSHBefore == nil {
		t.Fatalf("seller shareholder pre-state: sh=%v err=%v", sellerSHBefore, err)
	}
	tradesBefore, _ := exchangeRepo.GetCompanyTrades(c.ID, 50)

	// Crossing buy — matching must fail at the seller Credit step.
	_, err = uc.PlaceOrder(application.PlaceOrderInput{
		CompanyID: c.ID, OrderType: "buy", Shares: 5, Price: 5000,
	}, buyerID)
	if err == nil {
		t.Fatal("expected PlaceOrder to fail (injected credit failure), got nil")
	}

	// Everything the matching tx wrote must be rolled back.
	tradesAfter, _ := exchangeRepo.GetCompanyTrades(c.ID, 50)
	if len(tradesAfter) != len(tradesBefore) {
		t.Errorf("trade rows = %d, want %d (no trade should persist)", len(tradesAfter), len(tradesBefore))
	}

	buyerWalletAfter, err := walletRepo.FindByUserID(buyerID)
	if err != nil {
		t.Fatalf("buyer wallet after: %v", err)
	}
	if buyerWalletAfter.Balance != buyerWalletBefore.Balance {
		t.Errorf("buyer balance = %d, want %d (debit must roll back)", buyerWalletAfter.Balance, buyerWalletBefore.Balance)
	}

	buyerSH, _ := companyRepo.FindShareholder(c.ID, buyerID)
	if buyerSH != nil {
		t.Errorf("buyer shareholder = %+v, want nil (no share transfer should persist)", buyerSH)
	}

	sellerSHAfter, err := companyRepo.FindShareholder(c.ID, sellerID)
	if err != nil || sellerSHAfter == nil {
		t.Fatalf("seller shareholder after: sh=%v err=%v", sellerSHAfter, err)
	}
	if sellerSHAfter.Shares != sellerSHBefore.Shares {
		t.Errorf("seller shares = %d, want %d (no decrement should persist)", sellerSHAfter.Shares, sellerSHBefore.Shares)
	}
}
