package persistence

import (
	"database/sql"
	"fmt"

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
		       total_capital, total_shares, valuation, listed, business_card, status, created_at
		FROM companies WHERE id = ?`, id).Scan(
		&c.ID, &c.OwnerID, &c.Name, &c.Description, &c.LogoURL,
		&c.InitialCapital, &c.TotalCapital, &c.TotalShares,
		&c.Valuation, &listed, &c.BusinessCard, &c.Status, &c.CreatedAt,
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
		       total_capital, total_shares, valuation, listed, business_card, status, created_at
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
			&c.Valuation, &listed, &c.BusinessCard, &c.Status, &c.CreatedAt,
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
		       total_capital, total_shares, valuation, listed, business_card, status, created_at
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
			&c.Valuation, &listed, &c.BusinessCard, &c.Status, &c.CreatedAt,
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
		UPDATE companies SET description = ?, logo_url = ?, business_card = ?
		WHERE id = ?`,
		c.Description, c.LogoURL, c.BusinessCard, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update company: %w", err)
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
