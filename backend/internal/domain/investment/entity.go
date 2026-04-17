package investment

import "time"

type RoundStatus string

const (
	RoundOpen      RoundStatus = "open"
	RoundFunded    RoundStatus = "funded"
	RoundFailed    RoundStatus = "failed"
	RoundCancelled RoundStatus = "cancelled"
)

type InvestmentRound struct {
	ID              int         `json:"id"`
	CompanyID       int         `json:"company_id"`
	PostID          *int        `json:"post_id"`
	TargetAmount    int         `json:"target_amount"`
	OfferedPercent  float64     `json:"offered_percent"`
	CurrentAmount   int         `json:"current_amount"`
	PricePerShare   float64     `json:"price_per_share"`
	NewShares       int         `json:"new_shares"`
	Status          RoundStatus `json:"status"`
	ExpiresAt       *time.Time  `json:"expires_at"`
	CreatedAt       time.Time   `json:"created_at"`
	FundedAt        *time.Time  `json:"funded_at"`

	// Joined fields (flat — kept for backward compat with OAuth clients).
	CompanyName string `json:"company_name,omitempty"`
	OwnerName   string `json:"owner_name,omitempty"`

	// Nested shape consumed by the InvestPage / InvestDetailPage UI.
	Company *RoundCompany `json:"company,omitempty"`
	Owner   *RoundOwner   `json:"owner,omitempty"`

	// Derived: shares remaining = new_shares - sum(investments.shares).
	// Only populated by GetRound / single-fetch endpoints.
	SoldShares      int `json:"sold_shares"`
	RemainingShares int `json:"remaining_shares"`
}

// RoundCompany is the company slice attached to an InvestmentRound response.
type RoundCompany struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Valuation int    `json:"valuation"`
	LogoURL   string `json:"logo_url"`
}

// RoundOwner is the company owner slice attached to an InvestmentRound response.
type RoundOwner struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Investment struct {
	ID        int       `json:"id"`
	RoundID   int       `json:"round_id"`
	UserID    int       `json:"user_id"`
	Amount    int       `json:"amount"`
	Shares    int       `json:"shares"`
	CreatedAt time.Time `json:"created_at"`

	// Joined fields
	UserName    string `json:"user_name,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	CompanyID   int    `json:"company_id,omitempty"`
}

type Dividend struct {
	ID          int       `json:"id"`
	CompanyID   int       `json:"company_id"`
	TotalAmount int       `json:"total_amount"`
	ExecutedBy  int       `json:"executed_by"`
	CreatedAt   time.Time `json:"created_at"`

	// Joined fields
	CompanyName string             `json:"company_name,omitempty"`
	Payments    []*DividendPayment `json:"payments,omitempty"`
}

type DividendPayment struct {
	ID         int       `json:"id"`
	DividendID int       `json:"dividend_id"`
	UserID     int       `json:"user_id"`
	Shares     int       `json:"shares"`
	Amount     int       `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`

	// Joined fields
	UserName    string `json:"user_name,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
}

type KpiRule struct {
	ID              int       `json:"id"`
	CompanyID       int       `json:"company_id"`
	RuleDescription string    `json:"rule_description"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
}

type KpiRevenue struct {
	ID        int       `json:"id"`
	CompanyID int       `json:"company_id"`
	KpiRuleID *int      `json:"kpi_rule_id"`
	Amount    int       `json:"amount"`
	Memo      string    `json:"memo"`
	CreatedBy int       `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// PortfolioItem represents a user's investment position in a company.
// Shape is tailored to what InvestPage.tsx expects: nested company object and
// derived fields like profit / dividends_received so the UI can render without
// further client-side math.
type PortfolioItem struct {
	Company           PortfolioCompany `json:"company"`
	TotalShares       int              `json:"total_shares"`
	Shares            int              `json:"shares"`           // user's shares (was user_shares)
	InvestedAmount    int              `json:"invested_amount"`  // capital put in
	CurrentValue      int              `json:"current_value"`    // mark-to-market
	Profit            int              `json:"profit"`           // current_value - invested_amount
	DividendsReceived int              `json:"dividends_received"` // total dividends received from this company
	Percentage        float64          `json:"percentage"`
}

// PortfolioCompany is the embedded company reference inside a PortfolioItem.
type PortfolioCompany struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Valuation int    `json:"valuation"`
	LogoURL   string `json:"logo_url"`
}
