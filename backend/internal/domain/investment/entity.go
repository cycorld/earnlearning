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

	// Joined fields
	CompanyName string `json:"company_name,omitempty"`
	OwnerName   string `json:"owner_name,omitempty"`
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

// PortfolioItem represents a user's investment in a specific company.
type PortfolioItem struct {
	CompanyID    int     `json:"company_id"`
	CompanyName  string  `json:"company_name"`
	TotalShares  int     `json:"total_shares"`
	UserShares   int     `json:"user_shares"`
	Invested     int     `json:"invested"`
	CurrentValue int     `json:"current_value"`
	Percentage   float64 `json:"percentage"`
}
