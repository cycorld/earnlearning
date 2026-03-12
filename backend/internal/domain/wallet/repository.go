package wallet

import "time"

type TransactionFilter struct {
	TxType    string
	StartDate *time.Time
	EndDate   *time.Time
}

type Repository interface {
	FindByUserID(userID int) (*Wallet, error)
	CreateWallet(userID int) (int, error)
	Credit(walletID int, amount int, txType TxType, description, refType string, refID int) error
	Debit(walletID int, amount int, txType TxType, description, refType string, refID int) error
	GetTransactions(walletID int, filter TransactionFilter, page, limit int) ([]*Transaction, int, error)
	GetRanking(limit int) ([]*RankEntry, error)
	GetAssetBreakdown(userID int) (*AssetBreakdown, error)
}
