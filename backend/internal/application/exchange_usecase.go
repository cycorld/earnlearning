package application

import (
	"fmt"

	"database/sql"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/exchange"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type ExchangeUseCase struct {
	db           *sql.DB
	exchangeRepo exchange.Repository
	companyRepo  company.CompanyRepository
	walletRepo   wallet.Repository
	notifUC      *NotificationUseCase
	autoPoster   *AutoPoster
}

func NewExchangeUseCase(er exchange.Repository, cr company.CompanyRepository, wr wallet.Repository) *ExchangeUseCase {
	return &ExchangeUseCase{
		exchangeRepo: er,
		companyRepo:  cr,
		walletRepo:   wr,
	}
}

func (uc *ExchangeUseCase) SetDB(db *sql.DB) {
	uc.db = db
	uc.autoPoster = NewAutoPoster(db)
}

func (uc *ExchangeUseCase) SetNotificationUseCase(notifUC *NotificationUseCase) {
	uc.notifUC = notifUC
}

func (uc *ExchangeUseCase) notify(userID int, notifType notification.NotifType, title, body, refType string, refID int) {
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(userID, notifType, title, body, refType, refID)
	}
}

func (uc *ExchangeUseCase) ListCompanies() ([]*exchange.ListedCompany, error) {
	return uc.exchangeRepo.GetListedCompanies()
}

func (uc *ExchangeUseCase) GetOrderbook(companyID int) (*exchange.Orderbook, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, exchange.ErrCompanyNotFound
	}
	if !c.Listed {
		return nil, exchange.ErrCompanyNotListed
	}
	return uc.exchangeRepo.GetOrderbook(companyID)
}

// Position is the user's tradeable position in one company — mirrors the limits
// PlaceOrder validates against (available_cash for buys, available_shares for sells).
type Position struct {
	Shares          int `json:"shares"`           // 보유 주식
	AvailableShares int `json:"available_shares"` // 보유 − 미체결 매도
	Balance         int `json:"balance"`          // 지갑 잔액
	AvailableCash   int `json:"available_cash"`   // 잔액 − 미체결 매수
}

func (uc *ExchangeUseCase) GetPosition(companyID, userID int) (*Position, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, exchange.ErrCompanyNotFound
	}
	if !c.Listed {
		return nil, exchange.ErrCompanyNotListed
	}

	pos := &Position{}
	if w, err := uc.walletRepo.FindByUserID(userID); err == nil && w != nil {
		pos.Balance = w.Balance
	}
	pendingBuy, err := uc.exchangeRepo.GetPendingBuyTotal(userID)
	if err != nil {
		return nil, err
	}
	pos.AvailableCash = max(0, pos.Balance-pendingBuy)

	if sh, err := uc.companyRepo.FindShareholder(companyID, userID); err == nil && sh != nil {
		pos.Shares = sh.Shares
	}
	pendingSell, err := uc.exchangeRepo.GetPendingSellShares(userID, companyID)
	if err != nil {
		return nil, err
	}
	pos.AvailableShares = max(0, pos.Shares-pendingSell)

	return pos, nil
}

func (uc *ExchangeUseCase) GetCompanyTrades(companyID, limit int) ([]*exchange.StockTrade, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, exchange.ErrCompanyNotFound
	}
	if !c.Listed {
		return nil, exchange.ErrCompanyNotListed
	}
	return uc.exchangeRepo.GetCompanyTrades(companyID, limit)
}

type PlaceOrderInput struct {
	CompanyID int    `json:"company_id"`
	OrderType string `json:"order_type"`
	Shares    int    `json:"shares"`
	Price     int    `json:"price"`
}

type PlaceOrderResult struct {
	Order  *exchange.StockOrder   `json:"order"`
	Trades []*exchange.StockTrade `json:"trades"`
}

func (uc *ExchangeUseCase) PlaceOrder(input PlaceOrderInput, userID int) (*PlaceOrderResult, error) {
	if input.Shares <= 0 {
		return nil, exchange.ErrInvalidShares
	}
	if input.Price <= 0 {
		return nil, exchange.ErrInvalidPrice
	}

	// Check company is listed
	comp, err := uc.companyRepo.FindByID(input.CompanyID)
	if err != nil {
		return nil, exchange.ErrCompanyNotFound
	}
	if !comp.Listed {
		return nil, exchange.ErrCompanyNotListed
	}

	orderType := exchange.OrderType(input.OrderType)

	// Validate balance/shares
	if orderType == exchange.OrderTypeBuy {
		w, err := uc.walletRepo.FindByUserID(userID)
		if err != nil {
			return nil, fmt.Errorf("지갑을 찾을 수 없습니다")
		}
		pendingBuy, err := uc.exchangeRepo.GetPendingBuyTotal(userID)
		if err != nil {
			return nil, err
		}
		availableBalance := w.Balance - pendingBuy
		requiredAmount := input.Shares * input.Price
		if availableBalance < requiredAmount {
			return nil, exchange.ErrInsufficientBalance
		}
	} else if orderType == exchange.OrderTypeSell {
		sh, err := uc.companyRepo.FindShareholder(input.CompanyID, userID)
		if err != nil || sh == nil {
			return nil, exchange.ErrInsufficientShares
		}
		pendingSell, err := uc.exchangeRepo.GetPendingSellShares(userID, input.CompanyID)
		if err != nil {
			return nil, err
		}
		availableShares := sh.Shares - pendingSell
		if availableShares < input.Shares {
			return nil, exchange.ErrInsufficientShares
		}
	}

	// Create the order
	order := &exchange.StockOrder{
		CompanyID:       input.CompanyID,
		UserID:          userID,
		OrderType:       orderType,
		Shares:          input.Shares,
		RemainingShares: input.Shares,
		PricePerShare:   input.Price,
		Status:          exchange.OrderStatusOpen,
	}

	orderID, err := uc.exchangeRepo.CreateOrder(order)
	if err != nil {
		return nil, err
	}
	order.ID = orderID

	// Run matching engine
	trades, err := uc.runMatching(order, comp)
	if err != nil {
		return nil, err
	}

	// Re-read order to get updated state
	order, err = uc.exchangeRepo.FindOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	return &PlaceOrderResult{Order: order, Trades: trades}, nil
}

func (uc *ExchangeUseCase) CancelOrder(orderID, userID int) error {
	order, err := uc.exchangeRepo.FindOrderByID(orderID)
	if err != nil {
		return exchange.ErrOrderNotFound
	}
	if order.UserID != userID {
		return exchange.ErrNotOrderOwner
	}
	if order.Status != exchange.OrderStatusOpen && order.Status != exchange.OrderStatusPartial {
		return exchange.ErrOrderNotCancellable
	}
	return uc.exchangeRepo.CancelOrder(orderID)
}

type MyOrdersResult struct {
	Orders     []*exchange.StockOrder `json:"orders"`
	Total      int                    `json:"total"`
	TotalPages int                    `json:"total_pages"`
}

func (uc *ExchangeUseCase) GetMyOrders(userID int, status string, companyID, page, limit int) (*MyOrdersResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	orders, total, err := uc.exchangeRepo.GetUserOrders(userID, status, companyID, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	if orders == nil {
		orders = []*exchange.StockOrder{}
	}

	return &MyOrdersResult{
		Orders:     orders,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// tradeNotice captures the post-commit side effects (notification + auto-post) for
// one executed trade, so they fire only after the matching transaction commits.
type tradeNotice struct {
	buyerID, sellerID int
	shares            int
	price             int
	totalAmount       int
	tradeID           int
}

func (uc *ExchangeUseCase) runMatching(order *exchange.StockOrder, comp *company.Company) ([]*exchange.StockTrade, error) {
	// Compute matches first (pure, no writes) so an order that crosses nothing
	// skips the transaction entirely.
	var matches []*exchange.MatchResult
	if order.OrderType == exchange.OrderTypeBuy {
		sellOrders, err := uc.exchangeRepo.FindMatchingSellOrders(order.CompanyID, order.PricePerShare, order.UserID)
		if err != nil {
			return nil, err
		}
		matches = exchange.MatchBuyOrder(order, sellOrders)
	} else {
		buyOrders, err := uc.exchangeRepo.FindMatchingBuyOrders(order.CompanyID, order.PricePerShare, order.UserID)
		if err != nil {
			return nil, err
		}
		matches = exchange.MatchSellOrder(order, buyOrders)
	}
	if len(matches) == 0 {
		return nil, nil
	}

	// #142: settle every match inside ONE transaction — trade rows, order updates,
	// wallet debit/credit, shareholder transfer and valuation all commit together or
	// roll back together, so a failure mid-loop can never leave "money moved but
	// shares didn't" half-state. uc.db has SetMaxOpenConns(1), so any DB access via a
	// non-tx repo while this tx is open would deadlock — hence the matching repos are
	// rebound to tx and the external side effects (notify/auto-post, which use other
	// connections) are deferred until after commit.
	tx, err := uc.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	exchangeRepo := uc.exchangeRepo.WithTx(tx)
	walletRepo := uc.walletRepo.WithTx(tx)
	companyRepo := uc.companyRepo.WithTx(tx)

	var trades []*exchange.StockTrade
	var notices []tradeNotice

	for _, match := range matches {
		trade := &exchange.StockTrade{
			CompanyID:     order.CompanyID,
			BuyOrderID:    match.BuyOrder.ID,
			SellOrderID:   match.SellOrder.ID,
			BuyerID:       match.BuyOrder.UserID,
			SellerID:      match.SellOrder.UserID,
			Shares:        match.Shares,
			PricePerShare: match.Price,
			TotalAmount:   match.Shares * match.Price,
		}

		tradeID, err := exchangeRepo.CreateTrade(trade)
		if err != nil {
			return nil, err
		}
		trade.ID = tradeID

		// Update buy order
		match.BuyOrder.RemainingShares -= match.Shares
		if match.BuyOrder.RemainingShares == 0 {
			match.BuyOrder.Status = exchange.OrderStatusFilled
		} else {
			match.BuyOrder.Status = exchange.OrderStatusPartial
		}
		if err := exchangeRepo.UpdateOrder(match.BuyOrder); err != nil {
			return nil, err
		}

		// Update sell order
		match.SellOrder.RemainingShares -= match.Shares
		if match.SellOrder.RemainingShares == 0 {
			match.SellOrder.Status = exchange.OrderStatusFilled
		} else {
			match.SellOrder.Status = exchange.OrderStatusPartial
		}
		if err := exchangeRepo.UpdateOrder(match.SellOrder); err != nil {
			return nil, err
		}

		// Debit buyer wallet
		buyerWallet, err := walletRepo.FindByUserID(match.BuyOrder.UserID)
		if err != nil {
			return nil, err
		}
		if err := walletRepo.Debit(buyerWallet.ID, trade.TotalAmount, wallet.TxStockBuy,
			fmt.Sprintf("%s 주식 %d주 매수", comp.Name, match.Shares), "trade", tradeID); err != nil {
			return nil, err
		}

		// Credit seller wallet
		sellerWallet, err := walletRepo.FindByUserID(match.SellOrder.UserID)
		if err != nil {
			return nil, err
		}
		if err := walletRepo.Credit(sellerWallet.ID, trade.TotalAmount, wallet.TxStockSell,
			fmt.Sprintf("%s 주식 %d주 매도", comp.Name, match.Shares), "trade", tradeID); err != nil {
			return nil, err
		}

		// Transfer shares: buyer +shares (Upsert inserts a first-time buyer or adds to
		// an existing position), seller -shares (Subtract deletes the row at zero).
		if err := companyRepo.UpsertShareholder(order.CompanyID, match.BuyOrder.UserID, match.Shares, "trade"); err != nil {
			return nil, err
		}
		if err := companyRepo.SubtractShareholderShares(order.CompanyID, match.SellOrder.UserID, match.Shares); err != nil {
			return nil, err
		}

		// Update company valuation = trade_price * total_shares
		comp.Valuation = trade.PricePerShare * comp.TotalShares
		if err := companyRepo.Update(comp); err != nil {
			return nil, err
		}

		notices = append(notices, tradeNotice{
			buyerID:     match.BuyOrder.UserID,
			sellerID:    match.SellOrder.UserID,
			shares:      match.Shares,
			price:       match.Price,
			totalAmount: trade.TotalAmount,
			tradeID:     tradeID,
		})
		trades = append(trades, trade)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Post-commit side effects: never run inside the tx (separate DB connections
	// would deadlock under SetMaxOpenConns(1)), and never fire for a trade that
	// rolled back. Best-effort — a failed notification must not undo a settled trade.
	for _, n := range notices {
		uc.notify(n.buyerID, notification.NotifStockTrade,
			"주식 매수 체결",
			fmt.Sprintf("%s %d주를 %s에 매수했습니다.", comp.Name, n.shares, formatMoney(n.totalAmount)),
			"trade", n.tradeID)
		uc.notify(n.sellerID, notification.NotifStockTrade,
			"주식 매도 체결",
			fmt.Sprintf("%s %d주를 %s에 매도했습니다.", comp.Name, n.shares, formatMoney(n.totalAmount)),
			"trade", n.tradeID)

		if uc.autoPoster != nil {
			content := fmt.Sprintf("## 📊 거래 체결: %s\n\n**%d주** × **%s** = **%s**",
				comp.Name, n.shares, formatMoney(n.price), formatMoney(n.totalAmount))
			uc.autoPoster.PostToChannelAsAdmin("exchange", content, []string{"거래체결", comp.Name})
		}
	}

	return trades, nil
}
