package persistence

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/company"
)

type CompanyRepo struct {
	db *sql.DB
}

func NewCompanyRepo(db *sql.DB) *CompanyRepo {
	return &CompanyRepo{db: db}
}

func (r *CompanyRepo) Create(c *company.Company) (int, error) {
	listed := 0
	if c.Listed {
		listed = 1
	}
	res, err := r.db.Exec(`
		INSERT INTO companies (owner_id, name, description, logo_url, initial_capital, total_capital, total_shares, valuation, listed, business_card, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.OwnerID, c.Name, c.Description, c.LogoURL,
		c.InitialCapital, c.TotalCapital, c.TotalShares,
		c.Valuation, listed, c.BusinessCard, c.Status,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: companies.name") {
			return 0, company.ErrDuplicateName
		}
		return 0, fmt.Errorf("insert company: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *CompanyRepo) FindByID(id int) (*company.Company, error) {
	c := &company.Company{}
	var listed int
	err := r.db.QueryRow(`
		SELECT id, owner_id, name, description, logo_url, initial_capital,
		       total_capital, total_shares, valuation, listed, business_card, service_url, status, created_at
		FROM companies WHERE id = ?`, id).Scan(
		&c.ID, &c.OwnerID, &c.Name, &c.Description, &c.LogoURL,
		&c.InitialCapital, &c.TotalCapital, &c.TotalShares,
		&c.Valuation, &listed, &c.BusinessCard, &c.ServiceURL, &c.Status, &c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, company.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query company: %w", err)
	}
	c.Listed = listed == 1
	return c, nil
}

func (r *CompanyRepo) FindByOwnerID(ownerID int) ([]*company.Company, error) {
	rows, err := r.db.Query(`
		SELECT id, owner_id, name, description, logo_url, initial_capital,
		       total_capital, total_shares, valuation, listed, business_card, service_url, status, created_at
		FROM companies WHERE owner_id = ? ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query companies: %w", err)
	}
	defer rows.Close()

	var companies []*company.Company
	for rows.Next() {
		c := &company.Company{}
		var listed int
		if err := rows.Scan(
			&c.ID, &c.OwnerID, &c.Name, &c.Description, &c.LogoURL,
			&c.InitialCapital, &c.TotalCapital, &c.TotalShares,
			&c.Valuation, &listed, &c.BusinessCard, &c.ServiceURL, &c.Status, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		c.Listed = listed == 1
		companies = append(companies, c)
	}
	return companies, nil
}

func (r *CompanyRepo) FindAll() ([]*company.Company, error) {
	rows, err := r.db.Query(`
		SELECT id, owner_id, name, description, logo_url, initial_capital,
		       total_capital, total_shares, valuation, listed, business_card, service_url, status, created_at
		FROM companies ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query all companies: %w", err)
	}
	defer rows.Close()

	var companies []*company.Company
	for rows.Next() {
		c := &company.Company{}
		var listed int
		if err := rows.Scan(
			&c.ID, &c.OwnerID, &c.Name, &c.Description, &c.LogoURL,
			&c.InitialCapital, &c.TotalCapital, &c.TotalShares,
			&c.Valuation, &listed, &c.BusinessCard, &c.ServiceURL, &c.Status, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		c.Listed = listed == 1
		companies = append(companies, c)
	}
	return companies, nil
}

func (r *CompanyRepo) Update(c *company.Company) error {
	_, err := r.db.Exec(`
		UPDATE companies SET name = ?, description = ?, logo_url = ?, business_card = ?, service_url = ?
		WHERE id = ?`,
		c.Name, c.Description, c.LogoURL, c.BusinessCard, c.ServiceURL, c.ID,
	)
	if err != nil {
		// SQLite UNIQUE constraint 위반 → ErrDuplicateName 으로 매핑
		// (companies.name 이 UNIQUE)
		if strings.Contains(err.Error(), "UNIQUE constraint failed: companies.name") {
			return company.ErrDuplicateName
		}
		return fmt.Errorf("update company: %w", err)
	}
	return nil
}

func (r *CompanyRepo) UpdateStatus(companyID int, status string) error {
	_, err := r.db.Exec("UPDATE companies SET status = ? WHERE id = ?", status, companyID)
	if err != nil {
		return fmt.Errorf("update company status: %w", err)
	}
	return nil
}

func (r *CompanyRepo) UpdateListed(companyID int, listed bool) error {
	listedInt := 0
	if listed {
		listedInt = 1
	}
	_, err := r.db.Exec("UPDATE companies SET listed = ? WHERE id = ?", listedInt, companyID)
	if err != nil {
		return fmt.Errorf("update listed: %w", err)
	}
	return nil
}

// Shareholder operations

func (r *CompanyRepo) CreateShareholder(s *company.Shareholder) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO shareholders (company_id, user_id, shares, acquisition_type)
		VALUES (?, ?, ?, ?)`,
		s.CompanyID, s.UserID, s.Shares, s.AcquisitionType,
	)
	if err != nil {
		return 0, fmt.Errorf("insert shareholder: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *CompanyRepo) FindShareholdersByCompanyID(companyID int) ([]*company.Shareholder, error) {
	rows, err := r.db.Query(`
		SELECT id, company_id, user_id, shares, acquisition_type, acquired_at
		FROM shareholders WHERE company_id = ? ORDER BY shares DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("query shareholders: %w", err)
	}
	defer rows.Close()

	var shareholders []*company.Shareholder
	for rows.Next() {
		s := &company.Shareholder{}
		if err := rows.Scan(&s.ID, &s.CompanyID, &s.UserID, &s.Shares, &s.AcquisitionType, &s.AcquiredAt); err != nil {
			return nil, fmt.Errorf("scan shareholder: %w", err)
		}
		shareholders = append(shareholders, s)
	}
	return shareholders, nil
}

func (r *CompanyRepo) FindShareholder(companyID, userID int) (*company.Shareholder, error) {
	s := &company.Shareholder{}
	err := r.db.QueryRow(`
		SELECT id, company_id, user_id, shares, acquisition_type, acquired_at
		FROM shareholders WHERE company_id = ? AND user_id = ?`, companyID, userID).Scan(
		&s.ID, &s.CompanyID, &s.UserID, &s.Shares, &s.AcquisitionType, &s.AcquiredAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query shareholder: %w", err)
	}
	return s, nil
}

// SubtractShareholderShares decrements a shareholder's shares by `shares`.
// When the resulting balance is <= 0, the row is deleted entirely so stale
// "0 share" ghost rows don't pollute the shareholder list.
func (r *CompanyRepo) SubtractShareholderShares(companyID, userID, shares int) error {
	var current int
	err := r.db.QueryRow(
		"SELECT shares FROM shareholders WHERE company_id = ? AND user_id = ?",
		companyID, userID,
	).Scan(&current)
	if err == sql.ErrNoRows {
		return nil // nothing to do
	}
	if err != nil {
		return fmt.Errorf("find shareholder for decrement: %w", err)
	}
	if current-shares <= 0 {
		_, err = r.db.Exec(
			"DELETE FROM shareholders WHERE company_id = ? AND user_id = ?",
			companyID, userID,
		)
	} else {
		_, err = r.db.Exec(
			"UPDATE shareholders SET shares = shares - ? WHERE company_id = ? AND user_id = ?",
			shares, companyID, userID,
		)
	}
	if err != nil {
		return fmt.Errorf("decrement shareholder: %w", err)
	}
	return nil
}

func (r *CompanyRepo) UpsertShareholder(companyID, userID, shares int, acquisitionType string) error {
	_, err := r.db.Exec(`
		INSERT INTO shareholders (company_id, user_id, shares, acquisition_type)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(company_id, user_id) DO UPDATE SET shares = shares + ?`,
		companyID, userID, shares, acquisitionType, shares,
	)
	if err != nil {
		return fmt.Errorf("upsert shareholder: %w", err)
	}
	return nil
}

func (r *CompanyRepo) UpdateTotalShares(companyID, totalShares int) error {
	_, err := r.db.Exec("UPDATE companies SET total_shares = ? WHERE id = ?", totalShares, companyID)
	if err != nil {
		return fmt.Errorf("update total shares: %w", err)
	}
	return nil
}

func (r *CompanyRepo) UpdateCapitalAndValuation(companyID, totalCapital, valuation int) error {
	_, err := r.db.Exec("UPDATE companies SET total_capital = ?, valuation = ? WHERE id = ?", totalCapital, valuation, companyID)
	if err != nil {
		return fmt.Errorf("update capital and valuation: %w", err)
	}
	return nil
}

// Company wallet operations

func (r *CompanyRepo) CreateCompanyWallet(companyID int, initialBalance int) (int, error) {
	res, err := r.db.Exec("INSERT INTO company_wallets (company_id, balance) VALUES (?, ?)", companyID, initialBalance)
	if err != nil {
		return 0, fmt.Errorf("insert company wallet: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *CompanyRepo) FindCompanyWallet(companyID int) (*company.CompanyWallet, error) {
	w := &company.CompanyWallet{}
	err := r.db.QueryRow("SELECT id, company_id, balance FROM company_wallets WHERE company_id = ?", companyID).Scan(
		&w.ID, &w.CompanyID, &w.Balance,
	)
	if err == sql.ErrNoRows {
		return nil, company.ErrWalletNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query company wallet: %w", err)
	}
	return w, nil
}

func (r *CompanyRepo) CreditCompanyWallet(walletID int, amount int, txType string, desc string, refType string, refID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int
	err = tx.QueryRow("SELECT balance FROM company_wallets WHERE id = ?", walletID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	newBalance := balance + amount
	_, err = tx.Exec("UPDATE company_wallets SET balance = ? WHERE id = ?", newBalance, walletID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO company_transactions (company_wallet_id, amount, balance_after, tx_type, description, reference_type, reference_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		walletID, amount, newBalance, txType, desc, refType, refID,
	)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	return tx.Commit()
}

// Disclosure operations

func (r *CompanyRepo) CreateDisclosure(d *company.Disclosure) (int, error) {
	status := d.Status
	if status == "" {
		status = "pending"
	}
	res, err := r.db.Exec(`
		INSERT INTO company_disclosures (company_id, author_id, content, period_from, period_to, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		d.CompanyID, d.AuthorID, d.Content, d.PeriodFrom, d.PeriodTo, status,
	)
	if err != nil {
		return 0, fmt.Errorf("insert disclosure: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *CompanyRepo) FindDisclosureByID(id int) (*company.Disclosure, error) {
	d := &company.Disclosure{}
	err := r.db.QueryRow(`
		SELECT id, company_id, author_id, content, period_from, period_to,
		       status, reward, admin_note, created_at, updated_at
		FROM company_disclosures WHERE id = ?`, id).Scan(
		&d.ID, &d.CompanyID, &d.AuthorID, &d.Content, &d.PeriodFrom, &d.PeriodTo,
		&d.Status, &d.Reward, &d.AdminNote, &d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, company.ErrDisclosureNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query disclosure: %w", err)
	}
	return d, nil
}

func (r *CompanyRepo) FindDisclosuresByCompanyID(companyID int) ([]*company.Disclosure, error) {
	rows, err := r.db.Query(`
		SELECT id, company_id, author_id, content, period_from, period_to,
		       status, reward, admin_note, created_at, updated_at
		FROM company_disclosures WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("query disclosures: %w", err)
	}
	defer rows.Close()

	var disclosures []*company.Disclosure
	for rows.Next() {
		d := &company.Disclosure{}
		if err := rows.Scan(
			&d.ID, &d.CompanyID, &d.AuthorID, &d.Content, &d.PeriodFrom, &d.PeriodTo,
			&d.Status, &d.Reward, &d.AdminNote, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan disclosure: %w", err)
		}
		disclosures = append(disclosures, d)
	}
	return disclosures, nil
}

func (r *CompanyRepo) FindAllDisclosures() ([]*company.Disclosure, error) {
	rows, err := r.db.Query(`
		SELECT id, company_id, author_id, content, period_from, period_to,
		       status, reward, admin_note, created_at, updated_at
		FROM company_disclosures ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query all disclosures: %w", err)
	}
	defer rows.Close()

	var disclosures []*company.Disclosure
	for rows.Next() {
		d := &company.Disclosure{}
		if err := rows.Scan(
			&d.ID, &d.CompanyID, &d.AuthorID, &d.Content, &d.PeriodFrom, &d.PeriodTo,
			&d.Status, &d.Reward, &d.AdminNote, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan disclosure: %w", err)
		}
		disclosures = append(disclosures, d)
	}
	return disclosures, nil
}

func (r *CompanyRepo) UpdateDisclosureStatus(id int, status string, reward int, adminNote string) error {
	_, err := r.db.Exec(`
		UPDATE company_disclosures SET status = ?, reward = ?, admin_note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		status, reward, adminNote, id,
	)
	if err != nil {
		return fmt.Errorf("update disclosure status: %w", err)
	}
	return nil
}

// =============================================================================
// Proposal (주주총회 안건) operations
// =============================================================================

func (r *CompanyRepo) CreateProposal(p *company.Proposal) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO shareholder_proposals
			(company_id, proposer_id, proposal_type, title, description,
			 pass_threshold, status, start_date, end_date, result_note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.CompanyID, p.ProposerID, p.ProposalType, p.Title, p.Description,
		p.PassThreshold, p.Status, p.StartDate, p.EndDate, p.ResultNote,
	)
	if err != nil {
		return 0, fmt.Errorf("insert proposal: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *CompanyRepo) FindProposalByID(id int) (*company.Proposal, error) {
	p := &company.Proposal{}
	var closedAt sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, company_id, proposer_id, proposal_type, title, description,
		       pass_threshold, status, start_date, end_date, result_note, created_at, closed_at
		FROM shareholder_proposals WHERE id = ?`, id).Scan(
		&p.ID, &p.CompanyID, &p.ProposerID, &p.ProposalType, &p.Title, &p.Description,
		&p.PassThreshold, &p.Status, &p.StartDate, &p.EndDate, &p.ResultNote, &p.CreatedAt, &closedAt,
	)
	if err == sql.ErrNoRows {
		return nil, company.ErrProposalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query proposal: %w", err)
	}
	if closedAt.Valid {
		t := closedAt.Time
		p.ClosedAt = &t
	}
	return p, nil
}

func (r *CompanyRepo) FindProposalsByCompanyID(companyID int) ([]*company.Proposal, error) {
	rows, err := r.db.Query(`
		SELECT id, company_id, proposer_id, proposal_type, title, description,
		       pass_threshold, status, start_date, end_date, result_note, created_at, closed_at
		FROM shareholder_proposals WHERE company_id = ? ORDER BY created_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("query proposals: %w", err)
	}
	defer rows.Close()

	var proposals []*company.Proposal
	for rows.Next() {
		p := &company.Proposal{}
		var closedAt sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.ProposerID, &p.ProposalType, &p.Title, &p.Description,
			&p.PassThreshold, &p.Status, &p.StartDate, &p.EndDate, &p.ResultNote, &p.CreatedAt, &closedAt,
		); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		if closedAt.Valid {
			t := closedAt.Time
			p.ClosedAt = &t
		}
		proposals = append(proposals, p)
	}
	return proposals, nil
}

func (r *CompanyRepo) FindActiveProposalByCompanyAndType(companyID int, proposalType string) (*company.Proposal, error) {
	p := &company.Proposal{}
	var closedAt sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, company_id, proposer_id, proposal_type, title, description,
		       pass_threshold, status, start_date, end_date, result_note, created_at, closed_at
		FROM shareholder_proposals
		WHERE company_id = ? AND proposal_type = ? AND status = 'active'
		ORDER BY created_at DESC LIMIT 1`, companyID, proposalType).Scan(
		&p.ID, &p.CompanyID, &p.ProposerID, &p.ProposalType, &p.Title, &p.Description,
		&p.PassThreshold, &p.Status, &p.StartDate, &p.EndDate, &p.ResultNote, &p.CreatedAt, &closedAt,
	)
	if err == sql.ErrNoRows {
		return nil, company.ErrProposalNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query active proposal: %w", err)
	}
	if closedAt.Valid {
		t := closedAt.Time
		p.ClosedAt = &t
	}
	return p, nil
}

func (r *CompanyRepo) UpdateProposalStatus(id int, status string, resultNote string, closedAt *time.Time) error {
	var ca interface{}
	if closedAt != nil {
		ca = *closedAt
	}
	_, err := r.db.Exec(`
		UPDATE shareholder_proposals
		SET status = ?, result_note = ?, closed_at = ?
		WHERE id = ?`, status, resultNote, ca, id)
	if err != nil {
		return fmt.Errorf("update proposal status: %w", err)
	}
	return nil
}

// =============================================================================
// Vote operations
// =============================================================================

func (r *CompanyRepo) CreateVote(v *company.Vote) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO shareholder_votes (proposal_id, user_id, choice, shares_at_vote)
		VALUES (?, ?, ?, ?)`,
		v.ProposalID, v.UserID, v.Choice, v.SharesAtVote,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return 0, company.ErrAlreadyVoted
		}
		return 0, fmt.Errorf("insert vote: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *CompanyRepo) FindVote(proposalID, userID int) (*company.Vote, error) {
	v := &company.Vote{}
	err := r.db.QueryRow(`
		SELECT id, proposal_id, user_id, choice, shares_at_vote, created_at
		FROM shareholder_votes WHERE proposal_id = ? AND user_id = ?`, proposalID, userID).Scan(
		&v.ID, &v.ProposalID, &v.UserID, &v.Choice, &v.SharesAtVote, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query vote: %w", err)
	}
	return v, nil
}

func (r *CompanyRepo) FindVotesByProposalID(proposalID int) ([]*company.Vote, error) {
	rows, err := r.db.Query(`
		SELECT id, proposal_id, user_id, choice, shares_at_vote, created_at
		FROM shareholder_votes WHERE proposal_id = ? ORDER BY created_at ASC`, proposalID)
	if err != nil {
		return nil, fmt.Errorf("query votes: %w", err)
	}
	defer rows.Close()

	var votes []*company.Vote
	for rows.Next() {
		v := &company.Vote{}
		if err := rows.Scan(&v.ID, &v.ProposalID, &v.UserID, &v.Choice, &v.SharesAtVote, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan vote: %w", err)
		}
		votes = append(votes, v)
	}
	return votes, nil
}

func (r *CompanyRepo) GetCompanyTransactions(walletID int, page, limit int) ([]*company.CompanyTransaction, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	var total int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM company_transactions WHERE company_wallet_id = ?", walletID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count company txs: %w", err)
	}
	offset := (page - 1) * limit
	rows, err := r.db.Query(`
		SELECT id, company_wallet_id, amount, balance_after, tx_type, description, reference_type, reference_id, created_at
		FROM company_transactions
		WHERE company_wallet_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, walletID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query company txs: %w", err)
	}
	defer rows.Close()

	var txs []*company.CompanyTransaction
	for rows.Next() {
		t := &company.CompanyTransaction{}
		if err := rows.Scan(&t.ID, &t.CompanyWalletID, &t.Amount, &t.BalanceAfter, &t.TxType,
			&t.Description, &t.ReferenceType, &t.ReferenceID, &t.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan company tx: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, total, rows.Err()
}

func (r *CompanyRepo) DebitCompanyWallet(walletID int, amount int, txType string, desc string, refType string, refID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var balance int
	err = tx.QueryRow("SELECT balance FROM company_wallets WHERE id = ?", walletID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	if balance < amount {
		return company.ErrInsufficientFunds
	}

	newBalance := balance - amount
	_, err = tx.Exec("UPDATE company_wallets SET balance = ? WHERE id = ?", newBalance, walletID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO company_transactions (company_wallet_id, amount, balance_after, tx_type, description, reference_type, reference_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		walletID, -amount, newBalance, txType, desc, refType, refID,
	)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	return tx.Commit()
}
