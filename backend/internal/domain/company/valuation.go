package company

// CalculateValuation returns the company valuation.
// Initial valuation equals initial_capital.
// After investment rounds, valuation = total_capital / offered_percent.
// This is a simplified version for the MVP.
func CalculateValuation(totalCapital int) int {
	return totalCapital
}

// PricePerShare returns the price per share based on current valuation and total shares.
func PricePerShare(valuation, totalShares int) float64 {
	if totalShares == 0 {
		return 0
	}
	return float64(valuation) / float64(totalShares)
}

// ListingThreshold is the minimum total_capital required for listing.
const ListingThreshold = 50000000

// MinInitialCapital is the minimum initial capital to create a company.
const MinInitialCapital = 1000000

// DefaultTotalShares is the default number of shares for a new company.
const DefaultTotalShares = 10000
