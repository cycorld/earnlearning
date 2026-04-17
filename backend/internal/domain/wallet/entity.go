package wallet

import "time"

type TxType string

const (
	TxInitialCapital  TxType = "initial_capital"
	TxAdminTransfer   TxType = "admin_transfer"
	TxFreelanceEscrow TxType = "freelance_escrow"
	TxFreelancePay    TxType = "freelance_pay"
	TxFreelanceRefund TxType = "freelance_refund"
	TxInvestment      TxType = "investment"
	TxDividend        TxType = "dividend"
	TxStockBuy        TxType = "stock_buy"
	TxStockSell       TxType = "stock_sell"
	TxLoanDisburse    TxType = "loan_disburse"
	TxLoanRepay       TxType = "loan_repay"
	TxLoanInterest    TxType = "loan_interest"
	TxLoanPenalty     TxType = "loan_penalty"
	TxAssignReward    TxType = "assign_reward"
	TxKpiRevenue      TxType = "kpi_revenue"
	TxCompanyFunding  TxType = "company_funding"
	TxTransfer        TxType = "transfer"
	TxCompanyTransfer TxType = "company_transfer"
	TxLikeReward      TxType = "like_reward"
	TxCommentReward   TxType = "comment_reward"
	// Company liquidation (#023)
	TxLiquidationPayout TxType = "liquidation_payout"
	TxLiquidationTax    TxType = "liquidation_tax"
)

type Wallet struct {
	ID      int `json:"id"`
	UserID  int `json:"user_id"`
	Balance int `json:"balance"`
}

type Transaction struct {
	ID            int       `json:"id"`
	WalletID      int       `json:"wallet_id"`
	Amount        int       `json:"amount"`
	BalanceAfter  int       `json:"balance_after"`
	TxType        TxType    `json:"tx_type"`
	Description   string    `json:"description"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   int       `json:"reference_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type AssetBreakdown struct {
	Cash          int `json:"cash"`
	StockValue    int `json:"stock_value"`
	CompanyEquity int `json:"company_equity"`
	TotalDebt     int `json:"total_debt"`
	Total         int `json:"total"`
}

type RankEntry struct {
	Rank       int    `json:"rank"`
	UserID     int    `json:"user_id"`
	UserName   string `json:"user_name"`
	TotalAsset int    `json:"total_asset"`
	Cash       int    `json:"cash"`
}
