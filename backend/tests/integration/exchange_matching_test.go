package integration

import (
	"encoding/json"
	"testing"
)

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
