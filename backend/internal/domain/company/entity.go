package company

import "time"

type Company struct {
	ID             int       `json:"id"`
	OwnerID        int       `json:"owner_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	LogoURL        string    `json:"logo_url"`
	InitialCapital int       `json:"initial_capital"`
	TotalCapital   int       `json:"total_capital"`
	TotalShares    int       `json:"total_shares"`
	Valuation      int       `json:"valuation"`
	Listed         bool      `json:"listed"`
	BusinessCard   string    `json:"business_card"`
	ServiceURL     string    `json:"service_url"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// CheckListing checks if company meets listing requirements.
// A company is listed when total_capital >= 50,000,000.
func (c *Company) CheckListing() bool {
	return c.TotalCapital >= 50000000
}

type Shareholder struct {
	ID              int       `json:"id"`
	CompanyID       int       `json:"company_id"`
	UserID          int       `json:"user_id"`
	Shares          int       `json:"shares"`
	AcquisitionType string    `json:"acquisition_type"`
	AcquiredAt      time.Time `json:"acquired_at"`
}

// Percentage returns the shareholder's ownership percentage.
func (s *Shareholder) Percentage(totalShares int) float64 {
	if totalShares == 0 {
		return 0
	}
	return float64(s.Shares) / float64(totalShares) * 100
}

// BusinessCard is a value object for company business cards.
type BusinessCard struct {
	CompanyName string `json:"company_name"`
	OwnerName   string `json:"owner_name"`
	Title       string `json:"title"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	Website     string `json:"website"`
	LogoURL     string `json:"logo_url"`
}

type Disclosure struct {
	ID         int       `json:"id"`
	CompanyID  int       `json:"company_id"`
	AuthorID   int       `json:"author_id"`
	Content    string    `json:"content"`
	PeriodFrom string    `json:"period_from"`
	PeriodTo   string    `json:"period_to"`
	Status     string    `json:"status"`
	Reward     int       `json:"reward"`
	AdminNote  string    `json:"admin_note"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CompanyWallet struct {
	ID        int `json:"id"`
	CompanyID int `json:"company_id"`
	Balance   int `json:"balance"`
}

type CompanyTransaction struct {
	ID              int       `json:"id"`
	CompanyWalletID int       `json:"company_wallet_id"`
	Amount          int       `json:"amount"`
	BalanceAfter    int       `json:"balance_after"`
	TxType          string    `json:"tx_type"`
	Description     string    `json:"description"`
	ReferenceType   string    `json:"reference_type"`
	ReferenceID     int       `json:"reference_id"`
	CreatedAt       time.Time `json:"created_at"`
}
