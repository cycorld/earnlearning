package persistence

import (
	"fmt"
	"testing"
)

// setupExchangeTestDB spins up an in-memory DB with a single owner user and
// returns the ExchangeRepo plus the owner id for FK references.
func setupExchangeTestDB(t *testing.T) (*ExchangeRepo, int) {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	res, err := db.Exec(`INSERT INTO users (email, password, name, department, student_id, role, status)
		VALUES ('owner@test', 'x', 'Owner', 'CS', '0001', 'student', 'approved')`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	ownerID, _ := res.LastInsertId()
	return NewExchangeRepo(db), int(ownerID)
}

func insertListedCompany(t *testing.T, repo *ExchangeRepo, ownerID int, name string, totalShares int) int {
	t.Helper()
	res, err := repo.db.Exec(`INSERT INTO companies (owner_id, name, initial_capital, total_shares, listed, status)
		VALUES (?, ?, 1000000, ?, 1, 'active')`, ownerID, name, totalShares)
	if err != nil {
		t.Fatalf("insert company %s: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func insertOrder(t *testing.T, repo *ExchangeRepo, companyID, userID int, orderType string, price int) int {
	t.Helper()
	res, err := repo.db.Exec(`INSERT INTO stock_orders
		(company_id, user_id, order_type, shares, remaining_shares, price_per_share, status)
		VALUES (?, ?, ?, 1, 0, ?, 'filled')`, companyID, userID, orderType, price)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// GetCompanyTrades returns a company's executed trades newest-first, capped by limit.
func TestGetCompanyTrades(t *testing.T) {
	repo, ownerID := setupExchangeTestDB(t)
	co := insertListedCompany(t, repo, ownerID, "Alpha", 10000)
	other := insertListedCompany(t, repo, ownerID, "Bravo", 10000)

	// 3 trades for `co` at increasing time, plus 1 for another company (must be excluded).
	insertTrade := func(companyID, price, shares, minutesAgo int) {
		t.Helper()
		buy := insertOrder(t, repo, companyID, ownerID, "buy", price)
		sell := insertOrder(t, repo, companyID, ownerID, "sell", price)
		_, err := repo.db.Exec(`INSERT INTO stock_trades
			(company_id, buy_order_id, sell_order_id, buyer_id, seller_id, shares, price_per_share, total_amount, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now', ?))`,
			companyID, buy, sell, ownerID, ownerID, shares, price, price*shares,
			fmt.Sprintf("-%d minutes", minutesAgo))
		if err != nil {
			t.Fatalf("insert trade: %v", err)
		}
	}
	insertTrade(co, 5000, 2, 30) // oldest
	insertTrade(co, 6000, 1, 20)
	insertTrade(co, 5500, 3, 10) // newest
	insertTrade(other, 9999, 1, 5)

	trades, err := repo.GetCompanyTrades(co, 50)
	if err != nil {
		t.Fatalf("GetCompanyTrades: %v", err)
	}
	if len(trades) != 3 {
		t.Fatalf("len = %d, want 3 (only Alpha trades)", len(trades))
	}
	// newest-first
	if trades[0].PricePerShare != 5500 || trades[2].PricePerShare != 5000 {
		t.Errorf("order wrong: got [%d ... %d], want [5500 ... 5000]",
			trades[0].PricePerShare, trades[2].PricePerShare)
	}
	if trades[0].Shares != 3 || trades[0].TotalAmount != 5500*3 {
		t.Errorf("newest trade fields wrong: shares=%d total=%d", trades[0].Shares, trades[0].TotalAmount)
	}

	// limit caps the result
	limited, err := repo.GetCompanyTrades(co, 2)
	if err != nil {
		t.Fatalf("GetCompanyTrades limit: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("limited len = %d, want 2", len(limited))
	}
}

// GetListedCompanies must report the company's "시가" (market price):
//   - last trade price when trades exist
//   - otherwise the last funded investment round's price_per_share
//   - 0 only when neither exists
func TestGetListedCompanies_LastPriceFallback(t *testing.T) {
	repo, ownerID := setupExchangeTestDB(t)

	// Company A: has a funded round (price 5000) AND a later trade (price 7000) → trade wins.
	a := insertListedCompany(t, repo, ownerID, "Alpha", 10000)
	if _, err := repo.db.Exec(`INSERT INTO investment_rounds
		(company_id, target_amount, offered_percent, price_per_share, new_shares, status, funded_at)
		VALUES (?, 1000000, 0.1, 5000, 200, 'funded', datetime('now','-2 hours'))`, a); err != nil {
		t.Fatalf("round A: %v", err)
	}
	buyOrder := insertOrder(t, repo, a, ownerID, "buy", 7000)
	sellOrder := insertOrder(t, repo, a, ownerID, "sell", 7000)
	if _, err := repo.db.Exec(`INSERT INTO stock_trades
		(company_id, buy_order_id, sell_order_id, buyer_id, seller_id, shares, price_per_share, total_amount, created_at)
		VALUES (?, ?, ?, ?, ?, 1, 7000, 7000, datetime('now','-1 hours'))`,
		a, buyOrder, sellOrder, ownerID, ownerID); err != nil {
		t.Fatalf("trade A: %v", err)
	}

	// Company B: only a funded round (price 5000), no trades → falls back to round price.
	b := insertListedCompany(t, repo, ownerID, "Bravo", 10000)
	if _, err := repo.db.Exec(`INSERT INTO investment_rounds
		(company_id, target_amount, offered_percent, price_per_share, new_shares, status, funded_at)
		VALUES (?, 1000000, 0.1, 5000, 200, 'funded', datetime('now','-3 hours'))`, b); err != nil {
		t.Fatalf("round B: %v", err)
	}

	// Company C: no trades, no funded round, valuation 30,000,000 / 10,000 shares
	// → falls back to company valuation per share (3000).
	c := insertListedCompany(t, repo, ownerID, "Charlie", 10000)
	if _, err := repo.db.Exec(`UPDATE companies SET valuation = 30000000 WHERE id = ?`, c); err != nil {
		t.Fatalf("set valuation C: %v", err)
	}

	// Company D: no trades, no round, valuation 0 → 0 (true empty state).
	d := insertListedCompany(t, repo, ownerID, "Delta", 10000)

	companies, err := repo.GetListedCompanies()
	if err != nil {
		t.Fatalf("GetListedCompanies: %v", err)
	}

	prices := map[int]int{}
	for _, lc := range companies {
		prices[lc.ID] = lc.LastPrice
	}
	if prices[a] != 7000 {
		t.Errorf("Alpha last_price = %d, want 7000 (last trade)", prices[a])
	}
	if prices[b] != 5000 {
		t.Errorf("Bravo last_price = %d, want 5000 (last funded round fallback)", prices[b])
	}
	if prices[c] != 3000 {
		t.Errorf("Charlie last_price = %d, want 3000 (valuation/total_shares fallback)", prices[c])
	}
	if prices[d] != 0 {
		t.Errorf("Delta last_price = %d, want 0 (no trade, no round, no valuation)", prices[d])
	}
}
