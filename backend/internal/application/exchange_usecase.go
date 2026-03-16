package application

import (
	"fmt"

	"database/sql"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/exchange"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

// ShareholderUpdater provides direct shareholder share updates.
type ShareholderUpdater interface {
	UpdateShareholderShares(companyID, userID, shares int) error
}

type ExchangeUseCase struct {
	db                 *sql.DB
	exchangeRepo       exchange.Repository
	companyRepo        company.CompanyRepository
	walletRepo         wallet.Repository
	shareholderUpdater ShareholderUpdater
	notifUC            *NotificationUseCase
	autoPoster         *AutoPoster
}

func NewExchangeUseCase(er exchange.Repository, cr company.CompanyRepository, wr wallet.Repository) *ExchangeUseCase {
	return &ExchangeUseCase{
		exchangeRepo: er,
		companyRepo:  cr,
		walletRepo:   wr,
	}
}

func (uc *ExchangeUseCase) SetShareholderUpdater(updater ShareholderUpdater) {
	uc.shareholderUpdater = updater
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

func (uc *ExchangeUseCase) runMatching(order *exchange.StockOrder, comp *company.Company) ([]*exchange.StockTrade, error) {
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

	var trades []*exchange.StockTrade

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

		tradeID, err := uc.exchangeRepo.CreateTrade(trade)
		if err != nil {
			return trades, err
		}
		trade.ID = tradeID

		// Update buy order
		match.BuyOrder.RemainingShares -= match.Shares
		if match.BuyOrder.RemainingShares == 0 {
			match.BuyOrder.Status = exchange.OrderStatusFilled
		} else {
			match.BuyOrder.Status = exchange.OrderStatusPartial
		}
		if err := uc.exchangeRepo.UpdateOrder(match.BuyOrder); err != nil {
			return trades, err
		}

		// Update sell order
		match.SellOrder.RemainingShares -= match.Shares
		if match.SellOrder.RemainingShares == 0 {
			match.SellOrder.Status = exchange.OrderStatusFilled
		} else {
			match.SellOrder.Status = exchange.OrderStatusPartial
		}
		if err := uc.exchangeRepo.UpdateOrder(match.SellOrder); err != nil {
			return trades, err
		}

		// Debit buyer wallet
		buyerWallet, err := uc.walletRepo.FindByUserID(match.BuyOrder.UserID)
		if err != nil {
			return trades, err
		}
		if err := uc.walletRepo.Debit(buyerWallet.ID, trade.TotalAmount, wallet.TxStockBuy,
			fmt.Sprintf("%s 주식 %d주 매수", comp.Name, match.Shares), "trade", tradeID); err != nil {
			return trades, err
		}

		// Credit seller wallet
		sellerWallet, err := uc.walletRepo.FindByUserID(match.SellOrder.UserID)
		if err != nil {
			return trades, err
		}
		if err := uc.walletRepo.Credit(sellerWallet.ID, trade.TotalAmount, wallet.TxStockSell,
			fmt.Sprintf("%s 주식 %d주 매도", comp.Name, match.Shares), "trade", tradeID); err != nil {
			return trades, err
		}

		// Update shareholders: buyer +shares
		buyerSH, err := uc.companyRepo.FindShareholder(order.CompanyID, match.BuyOrder.UserID)
		if err != nil {
			// Create new shareholder
			_, err = uc.companyRepo.CreateShareholder(&company.Shareholder{
				CompanyID:       order.CompanyID,
				UserID:          match.BuyOrder.UserID,
				Shares:          match.Shares,
				AcquisitionType: "trade",
			})
			if err != nil {
				return trades, err
			}
		} else {
			newShares := buyerSH.Shares + match.Shares
			if uc.shareholderUpdater != nil {
				if err := uc.shareholderUpdater.UpdateShareholderShares(order.CompanyID, match.BuyOrder.UserID, newShares); err != nil {
					return trades, err
				}
			}
		}

		// Update shareholders: seller -shares (delete if 0)
		sellerSH, err := uc.companyRepo.FindShareholder(order.CompanyID, match.SellOrder.UserID)
		if err == nil && uc.shareholderUpdater != nil {
			newShares := sellerSH.Shares - match.Shares
			if err := uc.shareholderUpdater.UpdateShareholderShares(order.CompanyID, match.SellOrder.UserID, newShares); err != nil {
				return trades, err
			}
		}

		// Update company valuation = trade_price * total_shares
		comp.Valuation = trade.PricePerShare * comp.TotalShares
		if err := uc.companyRepo.Update(comp); err != nil {
			return trades, err
		}

		// Notify buyer and seller
		uc.notify(match.BuyOrder.UserID, notification.NotifStockTrade,
			"주식 매수 체결",
			fmt.Sprintf("%s %d주를 %s에 매수했습니다.", comp.Name, match.Shares, formatMoney(trade.TotalAmount)),
			"trade", tradeID)
		uc.notify(match.SellOrder.UserID, notification.NotifStockTrade,
			"주식 매도 체결",
			fmt.Sprintf("%s %d주를 %s에 매도했습니다.", comp.Name, match.Shares, formatMoney(trade.TotalAmount)),
			"trade", tradeID)

		// Auto-post to 거래소 channel
		if uc.autoPoster != nil {
			content := fmt.Sprintf("## 📊 거래 체결: %s\n\n**%d주** × **%s** = **%s**",
				comp.Name, match.Shares, formatMoney(match.Price), formatMoney(trade.TotalAmount))
			uc.autoPoster.PostToChannelAsAdmin("exchange", content, []string{"거래체결", comp.Name})
		}

		trades = append(trades, trade)
	}

	return trades, nil
}
