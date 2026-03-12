package exchange

// MatchResult represents a single match between a buy and sell order.
type MatchResult struct {
	BuyOrder  *StockOrder
	SellOrder *StockOrder
	Shares    int
	Price     int // matched at existing order's price
}

// MatchBuyOrder finds sell orders that can fill the given buy order.
// Sell orders are sorted by price ASC (cheapest first).
// Match price = sell order's price (existing order's price).
// No self-trading allowed.
func MatchBuyOrder(buyOrder *StockOrder, sellOrders []*StockOrder) []*MatchResult {
	var results []*MatchResult
	remaining := buyOrder.RemainingShares

	for _, sell := range sellOrders {
		if remaining <= 0 {
			break
		}
		// Skip self-trading
		if sell.UserID == buyOrder.UserID {
			continue
		}
		// Sell price must be <= buy price
		if sell.PricePerShare > buyOrder.PricePerShare {
			continue
		}

		matchShares := min(remaining, sell.RemainingShares)
		if matchShares <= 0 {
			continue
		}

		results = append(results, &MatchResult{
			BuyOrder:  buyOrder,
			SellOrder: sell,
			Shares:    matchShares,
			Price:     sell.PricePerShare, // existing order's price
		})

		remaining -= matchShares
	}

	return results
}

// MatchSellOrder finds buy orders that can fill the given sell order.
// Buy orders are sorted by price DESC (highest first).
// Match price = buy order's price (existing order's price).
// No self-trading allowed.
func MatchSellOrder(sellOrder *StockOrder, buyOrders []*StockOrder) []*MatchResult {
	var results []*MatchResult
	remaining := sellOrder.RemainingShares

	for _, buy := range buyOrders {
		if remaining <= 0 {
			break
		}
		// Skip self-trading
		if buy.UserID == sellOrder.UserID {
			continue
		}
		// Buy price must be >= sell price
		if buy.PricePerShare < sellOrder.PricePerShare {
			continue
		}

		matchShares := min(remaining, buy.RemainingShares)
		if matchShares <= 0 {
			continue
		}

		results = append(results, &MatchResult{
			BuyOrder:  buy,
			SellOrder: sellOrder,
			Shares:    matchShares,
			Price:     buy.PricePerShare, // existing order's price
		})

		remaining -= matchShares
	}

	return results
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
