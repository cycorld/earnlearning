package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/wallet"
)

type WalletRepo struct {
	db *sql.DB
}

func NewWalletRepo(db *sql.DB) *WalletRepo {
	return &WalletRepo{db: db}
}

func (r *WalletRepo) FindByUserID(userID int) (*wallet.Wallet, error) {
	w := &wallet.Wallet{}
	err := r.db.QueryRow(
		"SELECT id, user_id, balance FROM wallets WHERE user_id = ?", userID,
	).Scan(&w.ID, &w.UserID, &w.Balance)
	if err == sql.ErrNoRows {
		return nil, wallet.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *WalletRepo) CreateWallet(userID int) (int, error) {
	result, err := r.db.Exec(
		"INSERT INTO wallets (user_id, balance) VALUES (?, 0)", userID,
	)
	if err != nil {
		// If wallet already exists, return the existing one
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			w, findErr := r.FindByUserID(userID)
			if findErr != nil {
				return 0, findErr
			}
			return w.ID, nil
		}
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *WalletRepo) Credit(walletID int, amount int, txType wallet.TxType, description, refType string, refID int) error {
	if amount <= 0 {
		return wallet.ErrInvalidAmount
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE wallets SET balance = balance + ? WHERE id = ?", amount, walletID)
	if err != nil {
		return err
	}

	var newBalance int
	err = tx.QueryRow("SELECT balance FROM wallets WHERE id = ?", walletID).Scan(&newBalance)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (wallet_id, amount, balance_after, tx_type, description, reference_type, reference_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		walletID, amount, newBalance, string(txType), description, refType, refID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *WalletRepo) Debit(walletID int, amount int, txType wallet.TxType, description, refType string, refID int) error {
	if amount <= 0 {
		return wallet.ErrInvalidAmount
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check balance (admin transfers bypass this via use case layer)
	var currentBalance int
	err = tx.QueryRow("SELECT balance FROM wallets WHERE id = ?", walletID).Scan(&currentBalance)
	if err != nil {
		return err
	}

	// For admin_transfer, allow negative balance
	if txType != wallet.TxAdminTransfer && currentBalance < amount {
		return wallet.ErrInsufficientFunds
	}

	_, err = tx.Exec("UPDATE wallets SET balance = balance - ? WHERE id = ?", amount, walletID)
	if err != nil {
		return err
	}

	var newBalance int
	err = tx.QueryRow("SELECT balance FROM wallets WHERE id = ?", walletID).Scan(&newBalance)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (wallet_id, amount, balance_after, tx_type, description, reference_type, reference_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		walletID, -amount, newBalance, string(txType), description, refType, refID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *WalletRepo) GetTransactions(walletID int, filter wallet.TransactionFilter, page, limit int) ([]*wallet.Transaction, int, error) {
	where := []string{"wallet_id = ?"}
	args := []interface{}{walletID}

	if filter.TxType != "" {
		where = append(where, "tx_type = ?")
		args = append(args, filter.TxType)
	}
	if filter.StartDate != nil {
		where = append(where, "created_at >= ?")
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != nil {
		where = append(where, "created_at <= ?")
		args = append(args, filter.EndDate)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	queryArgs := append(args, limit, offset)
	rows, err := r.db.Query(
		fmt.Sprintf(
			`SELECT id, wallet_id, amount, balance_after, tx_type, description, reference_type, reference_id, created_at
			 FROM transactions WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, whereClause,
		),
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txs []*wallet.Transaction
	for rows.Next() {
		t := &wallet.Transaction{}
		if err := rows.Scan(&t.ID, &t.WalletID, &t.Amount, &t.BalanceAfter, &t.TxType,
			&t.Description, &t.ReferenceType, &t.ReferenceID, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		txs = append(txs, t)
	}
	return txs, total, rows.Err()
}

func (r *WalletRepo) GetRanking(limit int) ([]*wallet.RankEntry, error) {
	rows, err := r.db.Query(
		`SELECT u.id, u.name, w.balance
		 FROM wallets w
		 INNER JOIN users u ON u.id = w.user_id
		 WHERE u.role = 'student' AND u.status = 'approved'
		 ORDER BY w.balance DESC
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*wallet.RankEntry
	rank := 0
	for rows.Next() {
		rank++
		e := &wallet.RankEntry{Rank: rank}
		if err := rows.Scan(&e.UserID, &e.UserName, &e.Cash); err != nil {
			return nil, err
		}
		e.TotalAsset = e.Cash // Simplified; full asset calc is in GetAssetBreakdown
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *WalletRepo) GetAssetBreakdown(userID int) (*wallet.AssetBreakdown, error) {
	ab := &wallet.AssetBreakdown{}

	// Cash: wallet balance
	err := r.db.QueryRow("SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = ?", userID).Scan(&ab.Cash)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// StockValue: sum of (shares * company.valuation / company.total_shares) for each shareholding
	err = r.db.QueryRow(
		`SELECT COALESCE(SUM(s.shares * c.valuation / c.total_shares), 0)
		 FROM shareholders s
		 INNER JOIN companies c ON c.id = s.company_id
		 WHERE s.user_id = ? AND c.status = 'active'`, userID,
	).Scan(&ab.StockValue)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// CompanyEquity: sum of (company_wallet.balance * shares / company.total_shares)
	err = r.db.QueryRow(
		`SELECT COALESCE(SUM(cw.balance * s.shares / c.total_shares), 0)
		 FROM shareholders s
		 INNER JOIN companies c ON c.id = s.company_id
		 INNER JOIN company_wallets cw ON cw.company_id = c.id
		 WHERE s.user_id = ? AND c.status = 'active'`, userID,
	).Scan(&ab.CompanyEquity)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// TotalDebt: sum of remaining for active/overdue loans
	err = r.db.QueryRow(
		`SELECT COALESCE(SUM(remaining), 0)
		 FROM loans
		 WHERE borrower_id = ? AND status IN ('active', 'overdue')`, userID,
	).Scan(&ab.TotalDebt)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	ab.Total = ab.Cash + ab.StockValue + ab.CompanyEquity - ab.TotalDebt
	return ab, nil
}
