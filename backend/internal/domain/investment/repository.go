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
	HasOpenRound(companyID int) (bool, error)

	// Investments
	CreateInvestment(inv *Investment) (int, error)
	ListByUser(userID int) ([]*Investment, error)

	// Dividends
	CreateDividend(d *Dividend) (int, error)
	CreateDividendPayment(p *DividendPayment) (int, error)
	ListDividendsByUser(userID int) ([]*DividendPayment, error)

	// KPI
	CreateKpiRule(rule *KpiRule) (int, error)
	ListKpiRules(companyID int) ([]*KpiRule, error)
	CreateKpiRevenue(rev *KpiRevenue) (int, error)
}
