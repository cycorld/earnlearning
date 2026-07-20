package wallet

import (
	"database/sql"
	"time"
)

type TransactionFilter struct {
	TxType    string
	StartDate *time.Time
	EndDate   *time.Time
}

type Repository interface {
	// WithTx returns a Repository whose writes run inside tx (#142).
	WithTx(tx *sql.Tx) Repository

	// FindByUserID returns the wallet for the user's active classroom (#159).
	// Fallback order: active classroom wallet → unassigned(0) wallet → lowest classroom_id.
	FindByUserID(userID int) (*Wallet, error)
	// CreateWallet creates a wallet scoped to the user's current active classroom (0 if none).
	CreateWallet(userID int) (int, error)
	// EnsureClassroomWallet returns the wallet for (user, classroom), adopting the
	// user's unassigned wallet or creating a new one. isNew=true means the wallet
	// was newly bound to this classroom (initial capital not yet granted).
	EnsureClassroomWallet(userID, classroomID int) (walletID int, isNew bool, err error)
	Credit(walletID int, amount int, txType TxType, description, refType string, refID int) error
	Debit(walletID int, amount int, txType TxType, description, refType string, refID int) error
	GetTransactions(walletID int, filter TransactionFilter, page, limit int) ([]*Transaction, int, error)
	// GetRankingForUser ranks wallets in the requester's active classroom (#159).
	GetRankingForUser(requesterID, limit int) ([]*RankEntry, error)
	GetAssetBreakdown(userID int) (*AssetBreakdown, error)
}
