package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/investment"
)

type InvestmentRepo struct {
	db *sql.DB
}

func NewInvestmentRepo(db *sql.DB) *InvestmentRepo {
	return &InvestmentRepo{db: db}
}

// --- Rounds ---

func (r *InvestmentRepo) CreateRound(round *investment.InvestmentRound) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO investment_rounds (company_id, post_id, target_amount, offered_percent,
			current_amount, price_per_share, new_shares, status, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		round.CompanyID, round.PostID, round.TargetAmount, round.OfferedPercent,
		round.CurrentAmount, round.PricePerShare, round.NewShares, round.Status, round.ExpiresAt,
	)
	if err != nil {
		return 0, fmt.Errorf("create round: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *InvestmentRepo) FindRoundByID(id int) (*investment.InvestmentRound, error) {
	round := &investment.InvestmentRound{}
	var postID sql.NullInt64
	var expiresAt sql.NullTime
	var fundedAt sql.NullTime
	var companyValuation int
	var companyLogoURL string
	var ownerID int

	err := r.db.QueryRow(`
		SELECT ir.id, ir.company_id, ir.post_id, ir.target_amount, ir.offered_percent,
			   ir.current_amount, ir.price_per_share, ir.new_shares, ir.status,
			   ir.expires_at, ir.created_at, ir.funded_at,
			   c.name, c.valuation, COALESCE(c.logo_url, ''), u.id, u.name
		FROM investment_rounds ir
		JOIN companies c ON c.id = ir.company_id
		JOIN users u ON u.id = c.owner_id
		WHERE ir.id = ?`, id).Scan(
		&round.ID, &round.CompanyID, &postID, &round.TargetAmount, &round.OfferedPercent,
		&round.CurrentAmount, &round.PricePerShare, &round.NewShares, &round.Status,
		&expiresAt, &round.CreatedAt, &fundedAt,
		&round.CompanyName, &companyValuation, &companyLogoURL, &ownerID, &round.OwnerName,
	)
	if err == sql.ErrNoRows {
		return nil, investment.ErrRoundNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find round: %w", err)
	}
	if postID.Valid {
		pid := int(postID.Int64)
		round.PostID = &pid
	}
	if expiresAt.Valid {
		round.ExpiresAt = &expiresAt.Time
	}
	if fundedAt.Valid {
		round.FundedAt = &fundedAt.Time
	}
	round.Company = &investment.RoundCompany{
		ID: round.CompanyID, Name: round.CompanyName,
		Valuation: companyValuation, LogoURL: companyLogoURL,
	}
	round.Owner = &investment.RoundOwner{ID: ownerID, Name: round.OwnerName}

	// Sold/remaining shares (derived). FindRoundByID is the hot path for the
	// detail page which absolutely needs this.
	sold, _ := r.SumSharesByRound(round.ID)
	round.SoldShares = sold
	round.RemainingShares = round.NewShares - sold
	if round.RemainingShares < 0 {
		round.RemainingShares = 0
	}
	return round, nil
}

func (r *InvestmentRepo) ListRounds(filter investment.RoundFilter, page, limit int) ([]*investment.InvestmentRound, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if filter.CompanyID > 0 {
		where = append(where, "ir.company_id = ?")
		args = append(args, filter.CompanyID)
	}
	if filter.Status != "" {
		where = append(where, "ir.status = ?")
		args = append(args, filter.Status)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM investment_rounds ir WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count rounds: %w", err)
	}

	offset := (page - 1) * limit
	queryArgs := append(args, limit, offset)

	rows, err := r.db.Query(`
		SELECT ir.id, ir.company_id, ir.post_id, ir.target_amount, ir.offered_percent,
			   ir.current_amount, ir.price_per_share, ir.new_shares, ir.status,
			   ir.expires_at, ir.created_at, ir.funded_at,
			   c.name, c.valuation, COALESCE(c.logo_url, ''), u.id, u.name
		FROM investment_rounds ir
		JOIN companies c ON c.id = ir.company_id
		JOIN users u ON u.id = c.owner_id
		WHERE `+whereClause+`
		ORDER BY ir.created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rounds: %w", err)
	}
	defer rows.Close()

	var rounds []*investment.InvestmentRound
	for rows.Next() {
		round := &investment.InvestmentRound{}
		var postID sql.NullInt64
		var expiresAt sql.NullTime
		var fundedAt sql.NullTime
		var companyValuation int
		var companyLogoURL string
		var ownerID int

		if err := rows.Scan(
			&round.ID, &round.CompanyID, &postID, &round.TargetAmount, &round.OfferedPercent,
			&round.CurrentAmount, &round.PricePerShare, &round.NewShares, &round.Status,
			&expiresAt, &round.CreatedAt, &fundedAt,
			&round.CompanyName, &companyValuation, &companyLogoURL, &ownerID, &round.OwnerName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan round: %w", err)
		}
		if postID.Valid {
			pid := int(postID.Int64)
			round.PostID = &pid
		}
		if expiresAt.Valid {
			round.ExpiresAt = &expiresAt.Time
		}
		if fundedAt.Valid {
			round.FundedAt = &fundedAt.Time
		}
		round.Company = &investment.RoundCompany{
			ID: round.CompanyID, Name: round.CompanyName,
			Valuation: companyValuation, LogoURL: companyLogoURL,
		}
		round.Owner = &investment.RoundOwner{ID: ownerID, Name: round.OwnerName}
		rounds = append(rounds, round)
	}
	return rounds, total, nil
}

func (r *InvestmentRepo) UpdateRoundStatus(id int, status investment.RoundStatus) error {
	_, err := r.db.Exec("UPDATE investment_rounds SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *InvestmentRepo) UpdateRoundFunded(id int, amount int) error {
	_, err := r.db.Exec(
		"UPDATE investment_rounds SET current_amount = ?, status = 'funded', funded_at = CURRENT_TIMESTAMP WHERE id = ?",
		amount, id,
	)
	return err
}

func (r *InvestmentRepo) UpdateRoundCurrentAmount(id int, currentAmount int) error {
	_, err := r.db.Exec(
		"UPDATE investment_rounds SET current_amount = ? WHERE id = ?",
		currentAmount, id,
	)
	return err
}

func (r *InvestmentRepo) SumSharesByRound(roundID int) (int, error) {
	var total sql.NullInt64
	err := r.db.QueryRow(
		"SELECT COALESCE(SUM(shares), 0) FROM investments WHERE round_id = ?",
		roundID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return int(total.Int64), nil
}

// SumDividendsByUserAndCompany joins dividend_payments → dividends on
// dividend_id to scope by company_id.
func (r *InvestmentRepo) SumDividendsByUserAndCompany(userID, companyID int) (int, error) {
	var total sql.NullInt64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(dp.amount), 0)
		FROM dividend_payments dp
		JOIN dividends d ON d.id = dp.dividend_id
		WHERE dp.user_id = ? AND d.company_id = ?`,
		userID, companyID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return int(total.Int64), nil
}

func (r *InvestmentRepo) HasOpenRound(companyID int) (bool, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM investment_rounds WHERE company_id = ? AND status = 'open'",
		companyID,
	).Scan(&count)
	return count > 0, err
}

// --- Investments ---

func (r *InvestmentRepo) CreateInvestment(inv *investment.Investment) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO investments (round_id, user_id, amount, shares)
		VALUES (?, ?, ?, ?)`,
		inv.RoundID, inv.UserID, inv.Amount, inv.Shares,
	)
	if err != nil {
		return 0, fmt.Errorf("create investment: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *InvestmentRepo) ListByUser(userID int) ([]*investment.Investment, error) {
	rows, err := r.db.Query(`
		SELECT i.id, i.round_id, i.user_id, i.amount, i.shares, i.created_at,
			   u.name AS user_name, c.name AS company_name, c.id AS company_id
		FROM investments i
		JOIN users u ON u.id = i.user_id
		JOIN investment_rounds ir ON ir.id = i.round_id
		JOIN companies c ON c.id = ir.company_id
		WHERE i.user_id = ?
		ORDER BY i.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list investments: %w", err)
	}
	defer rows.Close()

	var invs []*investment.Investment
	for rows.Next() {
		inv := &investment.Investment{}
		if err := rows.Scan(
			&inv.ID, &inv.RoundID, &inv.UserID, &inv.Amount, &inv.Shares, &inv.CreatedAt,
			&inv.UserName, &inv.CompanyName, &inv.CompanyID,
		); err != nil {
			return nil, fmt.Errorf("scan investment: %w", err)
		}
		invs = append(invs, inv)
	}
	return invs, nil
}

// --- Dividends ---

func (r *InvestmentRepo) CreateDividend(d *investment.Dividend) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO dividends (company_id, total_amount, executed_by)
		VALUES (?, ?, ?)`,
		d.CompanyID, d.TotalAmount, d.ExecutedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("create dividend: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *InvestmentRepo) CreateDividendPayment(p *investment.DividendPayment) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO dividend_payments (dividend_id, user_id, shares, amount)
		VALUES (?, ?, ?, ?)`,
		p.DividendID, p.UserID, p.Shares, p.Amount,
	)
	if err != nil {
		return 0, fmt.Errorf("create dividend payment: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *InvestmentRepo) ListDividendsByUser(userID int) ([]*investment.DividendPayment, error) {
	rows, err := r.db.Query(`
		SELECT dp.id, dp.dividend_id, dp.user_id, dp.shares, dp.amount, dp.created_at,
			   u.name AS user_name, c.name AS company_name
		FROM dividend_payments dp
		JOIN dividends d ON d.id = dp.dividend_id
		JOIN companies c ON c.id = d.company_id
		JOIN users u ON u.id = dp.user_id
		WHERE dp.user_id = ?
		ORDER BY dp.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list dividends: %w", err)
	}
	defer rows.Close()

	var payments []*investment.DividendPayment
	for rows.Next() {
		p := &investment.DividendPayment{}
		if err := rows.Scan(
			&p.ID, &p.DividendID, &p.UserID, &p.Shares, &p.Amount, &p.CreatedAt,
			&p.UserName, &p.CompanyName,
		); err != nil {
			return nil, fmt.Errorf("scan dividend payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}

// --- KPI ---

func (r *InvestmentRepo) CreateKpiRule(rule *investment.KpiRule) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO kpi_rules (company_id, rule_description, active)
		VALUES (?, ?, ?)`,
		rule.CompanyID, rule.RuleDescription, rule.Active,
	)
	if err != nil {
		return 0, fmt.Errorf("create kpi rule: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *InvestmentRepo) ListKpiRules(companyID int) ([]*investment.KpiRule, error) {
	rows, err := r.db.Query(`
		SELECT id, company_id, rule_description, active, created_at
		FROM kpi_rules
		WHERE company_id = ?
		ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list kpi rules: %w", err)
	}
	defer rows.Close()

	var rules []*investment.KpiRule
	for rows.Next() {
		rule := &investment.KpiRule{}
		if err := rows.Scan(
			&rule.ID, &rule.CompanyID, &rule.RuleDescription, &rule.Active, &rule.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan kpi rule: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *InvestmentRepo) CreateKpiRevenue(rev *investment.KpiRevenue) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO kpi_revenues (company_id, kpi_rule_id, amount, memo, created_by)
		VALUES (?, ?, ?, ?, ?)`,
		rev.CompanyID, rev.KpiRuleID, rev.Amount, rev.Memo, rev.CreatedBy,
	)
	if err != nil {
		return 0, fmt.Errorf("create kpi revenue: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}
