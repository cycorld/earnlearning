package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/wallet"
)

type WalletRepo struct {
	db DBTX
}

func NewWalletRepo(db *sql.DB) *WalletRepo {
	return &WalletRepo{db: db}
}

// WithTx returns a repo bound to tx so its writes join the caller's transaction (#142).
func (r *WalletRepo) WithTx(tx *sql.Tx) wallet.Repository {
	return &WalletRepo{db: tx}
}

// FindByUserID resolves the user's wallet for their active classroom (#159).
// Priority: active classroom wallet → unassigned(classroom_id=0) → lowest classroom_id.
func (r *WalletRepo) FindByUserID(userID int) (*wallet.Wallet, error) {
	w := &wallet.Wallet{}
	err := r.db.QueryRow(
		`SELECT w.id, w.user_id, w.classroom_id, w.balance
		 FROM wallets w
		 JOIN users u ON u.id = w.user_id
		 WHERE w.user_id = ?
		 ORDER BY (w.classroom_id = u.active_classroom_id) DESC, w.classroom_id ASC
		 LIMIT 1`, userID,
	).Scan(&w.ID, &w.UserID, &w.ClassroomID, &w.Balance)
	if err == sql.ErrNoRows {
		return nil, wallet.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// CreateWallet creates a wallet scoped to the user's active classroom (0 if none).
func (r *WalletRepo) CreateWallet(userID int) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO wallets (user_id, classroom_id, balance)
		 VALUES (?, (SELECT COALESCE(active_classroom_id, 0) FROM users WHERE id = ?), 0)`,
		userID, userID,
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

// FindByUserAndClassroom — (user, classroom) 지갑 조회 (#159).
func (r *WalletRepo) FindByUserAndClassroom(userID, classroomID int) (*wallet.Wallet, error) {
	w := &wallet.Wallet{}
	err := r.db.QueryRow(
		"SELECT id, user_id, classroom_id, balance FROM wallets WHERE user_id = ? AND classroom_id = ?",
		userID, classroomID,
	).Scan(&w.ID, &w.UserID, &w.ClassroomID, &w.Balance)
	if err == sql.ErrNoRows {
		return nil, wallet.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

// EnsureClassroomWallet returns the wallet for (user, classroom).
// 없으면 미배정(classroom_id=0) 지갑을 해당 강의실로 귀속시키거나 새로 만든다.
// isNew=true 면 이 강의실에 처음 묶인 지갑 → 호출부에서 초기자본을 지급해야 한다.
func (r *WalletRepo) EnsureClassroomWallet(userID, classroomID int) (int, bool, error) {
	var id int
	err := r.db.QueryRow(
		"SELECT id FROM wallets WHERE user_id = ? AND classroom_id = ?", userID, classroomID,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if err != sql.ErrNoRows {
		return 0, false, err
	}

	// 미배정 지갑 귀속 (승인 시 만들어진 지갑 재사용)
	res, err := r.db.Exec(
		"UPDATE wallets SET classroom_id = ? WHERE user_id = ? AND classroom_id = 0", classroomID, userID,
	)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			if err := r.db.QueryRow(
				"SELECT id FROM wallets WHERE user_id = ? AND classroom_id = ?", userID, classroomID,
			).Scan(&id); err != nil {
				return 0, false, err
			}
			return id, true, nil
		}
	} else if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return 0, false, err
	}

	res, err = r.db.Exec(
		"INSERT INTO wallets (user_id, classroom_id, balance) VALUES (?, ?, 0)", userID, classroomID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			err = r.db.QueryRow(
				"SELECT id FROM wallets WHERE user_id = ? AND classroom_id = ?", userID, classroomID,
			).Scan(&id)
			return id, false, err
		}
		return 0, false, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, false, err
	}
	return int(newID), true, nil
}

func (r *WalletRepo) Credit(walletID int, amount int, txType wallet.TxType, description, refType string, refID int) error {
	if amount <= 0 {
		return wallet.ErrInvalidAmount
	}

	return withDBTx(r.db, func(tx DBTX) error {
		_, err := tx.Exec("UPDATE wallets SET balance = balance + ? WHERE id = ?", amount, walletID)
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
		return err
	})
}

func (r *WalletRepo) Debit(walletID int, amount int, txType wallet.TxType, description, refType string, refID int) error {
	if amount <= 0 {
		return wallet.ErrInvalidAmount
	}

	return withDBTx(r.db, func(tx DBTX) error {
		// Check balance (admin transfers bypass this via use case layer)
		var currentBalance int
		err := tx.QueryRow("SELECT balance FROM wallets WHERE id = ?", walletID).Scan(&currentBalance)
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
		return err
	})
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

// GetActiveClassroomID — 유저의 활성 강의실 (0 = 미설정) (#159).
func (r *WalletRepo) GetActiveClassroomID(userID int) (int, error) {
	var id int
	err := r.db.QueryRow(
		"SELECT COALESCE(active_classroom_id, 0) FROM users WHERE id = ?", userID,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// GetRankingForUser ranks student wallets in the requester's active classroom (#159).
func (r *WalletRepo) GetRankingForUser(requesterID, limit int) ([]*wallet.RankEntry, error) {
	rows, err := r.db.Query(
		`SELECT u.id, u.name, w.balance
		 FROM wallets w
		 INNER JOIN users u ON u.id = w.user_id
		 WHERE u.role = 'student' AND u.status = 'approved'
		   AND w.classroom_id = (SELECT COALESCE(active_classroom_id, 0) FROM users WHERE id = ?)
		 ORDER BY w.balance DESC
		 LIMIT ?`, requesterID, limit,
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

	// Cash: 활성 강의실 지갑 잔액 (#159 — FindByUserID 와 동일한 우선순위:
	// 활성 강의실 일치 지갑 우선, 없으면 최소 classroom_id 지갑).
	// 이 지갑의 classroom_id 를 "유효 강의실"로 잡아 이하 주식·지분·부채 집계도
	// 같은 강의실로 스코프한다 (누수 방지 #159).
	var effClassroom int
	err := r.db.QueryRow(
		`SELECT COALESCE(w.balance, 0), w.classroom_id
		 FROM wallets w
		 JOIN users u ON u.id = w.user_id
		 WHERE w.user_id = ?
		 ORDER BY (w.classroom_id = u.active_classroom_id) DESC, w.classroom_id ASC
		 LIMIT 1`, userID).Scan(&ab.Cash, &effClassroom)
	if err == sql.ErrNoRows {
		// 지갑이 없으면(강의실 미소속) 모든 자산 0.
		return ab, nil
	}
	if err != nil {
		return nil, err
	}

	// StockValue: 유효 강의실 회사 지분 평가액 (shares * valuation / total_shares)
	err = r.db.QueryRow(
		`SELECT COALESCE(SUM(s.shares * c.valuation / c.total_shares), 0)
		 FROM shareholders s
		 INNER JOIN companies c ON c.id = s.company_id
		 WHERE s.user_id = ? AND c.status = 'active' AND c.classroom_id = ?`, userID, effClassroom,
	).Scan(&ab.StockValue)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// CompanyEquity: 유효 강의실 회사 지갑 지분 (company_wallet.balance * shares / total_shares)
	err = r.db.QueryRow(
		`SELECT COALESCE(SUM(cw.balance * s.shares / c.total_shares), 0)
		 FROM shareholders s
		 INNER JOIN companies c ON c.id = s.company_id
		 INNER JOIN company_wallets cw ON cw.company_id = c.id
		 WHERE s.user_id = ? AND c.status = 'active' AND c.classroom_id = ?`, userID, effClassroom,
	).Scan(&ab.CompanyEquity)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// TotalDebt: 유효 강의실 대출 잔액 합 (active/overdue)
	err = r.db.QueryRow(
		`SELECT COALESCE(SUM(remaining), 0)
		 FROM loans
		 WHERE borrower_id = ? AND status IN ('active', 'overdue') AND classroom_id = ?`, userID, effClassroom,
	).Scan(&ab.TotalDebt)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	ab.Total = ab.Cash + ab.StockValue + ab.CompanyEquity - ab.TotalDebt
	return ab, nil
}
