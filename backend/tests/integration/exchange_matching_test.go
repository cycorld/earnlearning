package integration

import (
	"encoding/json"
	"testing"
)

// #144: GET /exchange/position reports tradeable limits — held/available shares
// (minus pending sells) and balance/available cash (minus pending buys) — the same
// numbers PlaceOrder validates against, so the frontend can constrain inputs.
func TestExchange_Position(t *testing.T) {
	ts := setupTestServer(t)
	_, token := createInvestor(t, ts, "pos-user@test.com", "pos", "20240711", 60_000_000)

	r := ts.post("/api/companies", map[string]interface{}{
		"name": "포지션테스트사", "description": "x", "initial_capital": 50_000_000, "logo_url": "",
	}, token)
	if !r.Success {
		t.Fatalf("create company: %v", r.Error)
	}
	var c struct {
		ID int `json:"id"`
	}
	_ = json.Unmarshal(r.Data, &c)

	type position struct {
		Shares          int `json:"shares"`
		AvailableShares int `json:"available_shares"`
		Balance         int `json:"balance"`
		AvailableCash   int `json:"available_cash"`
	}
	getPos := func() position {
		pr := ts.get("/api/exchange/position/"+itoaUD(c.ID), token)
		if !pr.Success {
			t.Fatalf("get position: %v", pr.Error)
		}
		var p position
		_ = json.Unmarshal(pr.Data, &p)
		return p
	}

	// Fresh founder: 10000 shares, no pending → available == held, cash == balance.
	p := getPos()
	if p.Shares != 10000 || p.AvailableShares != 10000 {
		t.Errorf("initial shares=%d avail=%d, want 10000/10000", p.Shares, p.AvailableShares)
	}
	if p.Balance != p.AvailableCash {
		t.Errorf("initial cash %d != balance %d (no pending expected)", p.AvailableCash, p.Balance)
	}
	balance := p.Balance

	// Pending sell 5 → available_shares drops by 5, held unchanged.
	if sr := ts.post("/api/exchange/orders", map[string]interface{}{
		"company_id": c.ID, "order_type": "sell", "shares": 5, "price": 9000,
	}, token); !sr.Success {
		t.Fatalf("sell: %v", sr.Error)
	}
	// Pending buy 3@8000 (below own sell, no cross) → available_cash drops by 24000.
	if br := ts.post("/api/exchange/orders", map[string]interface{}{
		"company_id": c.ID, "order_type": "buy", "shares": 3, "price": 8000,
	}, token); !br.Success {
		t.Fatalf("buy: %v", br.Error)
	}

	p = getPos()
	if p.Shares != 10000 || p.AvailableShares != 9995 {
		t.Errorf("after sell: shares=%d avail=%d, want 10000/9995", p.Shares, p.AvailableShares)
	}
	if p.AvailableCash != balance-24000 {
		t.Errorf("after buy: available_cash=%d, want %d", p.AvailableCash, balance-24000)
	}
}

// Regression (#140): a crossing trade where the BUYER has no prior shareholder
// row must NOT panic (nil deref in runMatching) and must transfer shares.
//
// Before the fix, FindShareholder returns (nil, nil) for a first-time buyer,
// the buyer branch reads buyerSH.Shares on a nil pointer and panics. The trade
// row + wallet moves were already committed, so money moved but shares did not.
func TestExchange_CrossingTrade_FirstTimeBuyer_NoPanic(t *testing.T) {
	ts := setupTestServer(t)

	sellerID, sellerToken := createInvestor(t, ts, "ex-seller@test.com", "seller", "20240701", 60_000_000)
	buyerID, buyerToken := createInvestor(t, ts, "ex-buyer@test.com", "buyer", "20240702", 60_000_000)

	// Seller creates a listed company (capital 50M -> auto-list, 10000 founding shares).
	r := ts.post("/api/companies", map[string]interface{}{
		"name":            "체결테스트사",
		"description":     "매칭 패닉 회귀",
		"initial_capital": 50_000_000,
		"logo_url":        "",
	}, sellerToken)
	if !r.Success {
		t.Fatalf("create company: %v", r.Error)
	}
	var c struct {
		ID     int  `json:"id"`
		Listed bool `json:"listed"`
	}
	_ = json.Unmarshal(r.Data, &c)
	if !c.Listed {
		t.Fatalf("company not auto-listed at 50M capital: %+v", c)
	}

	// Seller rests a sell order.
	sr := ts.post("/api/exchange/orders", map[string]interface{}{
		"company_id": c.ID, "order_type": "sell", "shares": 5, "price": 5000,
	}, sellerToken)
	if !sr.Success {
		t.Fatalf("seller sell order: %v", sr.Error)
	}

	// Buyer (first-time, no shareholder row) crosses it.
	br := ts.post("/api/exchange/orders", map[string]interface{}{
		"company_id": c.ID, "order_type": "buy", "shares": 5, "price": 5000,
	}, buyerToken)
	if !br.Success {
		t.Fatalf("buyer crossing order failed (panic/500?): %v", br.Error)
	}

	// Integrity: shares actually transferred.
	buyerSH, err := ts.companyRepo.FindShareholder(c.ID, buyerID)
	if err != nil || buyerSH == nil {
		t.Fatalf("buyer shareholder missing after trade: sh=%v err=%v", buyerSH, err)
	}
	if buyerSH.Shares != 5 {
		t.Errorf("buyer shares = %d, want 5", buyerSH.Shares)
	}
	sellerSH, err := ts.companyRepo.FindShareholder(c.ID, sellerID)
	if err != nil || sellerSH == nil {
		t.Fatalf("seller shareholder missing: err=%v", err)
	}
	if sellerSH.Shares != 9995 {
		t.Errorf("seller shares = %d, want 9995 (10000-5)", sellerSH.Shares)
	}

	// Trade tape recorded.
	tr := ts.get("/api/exchange/trades/"+itoaUD(c.ID), buyerToken)
	if !tr.Success {
		t.Fatalf("get trades: %v", tr.Error)
	}
	var trades []struct {
		PricePerShare int `json:"price_per_share"`
		Shares        int `json:"shares"`
	}
	_ = json.Unmarshal(tr.Data, &trades)
	if len(trades) != 1 || trades[0].Shares != 5 || trades[0].PricePerShare != 5000 {
		t.Errorf("trades = %+v, want 1 trade of 5 @ 5000", trades)
	}
}
