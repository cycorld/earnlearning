package investment

type RoundFilter struct {
	CompanyID int
	Status    string
}

type Repository interface {
	// Rounds
	CreateRound(round *InvestmentRound) (int, error)
	FindRoundByID(id int) (*InvestmentRound, error)
	ListRounds(filter RoundFilter, page, limit int) ([]*InvestmentRound, int, error)
	UpdateRoundStatus(id int, status RoundStatus) error
	UpdateRoundFunded(id int, amount int) error
	// UpdateRoundCurrentAmount bumps current_amount without changing status.
	// Used for partial investments that don't fully close the round.
	UpdateRoundCurrentAmount(id int, currentAmount int) error
	HasOpenRound(companyID int) (bool, error)

	// Investments
	CreateInvestment(inv *Investment) (int, error)
	ListByUser(userID int) ([]*Investment, error)
	// SumSharesByRound returns total shares already issued through investments
	// for a given round. Used to compute remaining shares for partial invest.
	SumSharesByRound(roundID int) (int, error)
	// SumDividendsByUserAndCompany returns total dividend amount received by
	// this user from a specific company. Drives the portfolio view.
	SumDividendsByUserAndCompany(userID, companyID int) (int, error)

	// Dividends
	CreateDividend(d *Dividend) (int, error)
	CreateDividendPayment(p *DividendPayment) (int, error)
	ListDividendsByUser(userID int) ([]*DividendPayment, error)

	// KPI
	CreateKpiRule(rule *KpiRule) (int, error)
	ListKpiRules(companyID int) ([]*KpiRule, error)
	CreateKpiRevenue(rev *KpiRevenue) (int, error)
}
