package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/exchange"
)

type ExchangeRepo struct {
	db *sql.DB
}

func NewExchangeRepo(db *sql.DB) *ExchangeRepo {
	return &ExchangeRepo{db: db}
}

func (r *ExchangeRepo) CreateOrder(order *exchange.StockOrder) (int, error) {
	result, err := r.db.Exec(`
		INSERT INTO stock_orders (company_id, user_id, order_type, shares, remaining_shares, price_per_share, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		order.CompanyID, order.UserID, order.OrderType, order.Shares, order.RemainingShares, order.PricePerShare, order.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("create order: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *ExchangeRepo) FindOrderByID(id int) (*exchange.StockOrder, error) {
	o := &exchange.StockOrder{}
	err := r.db.QueryRow(`
		SELECT id, company_id, user_id, order_type, shares, remaining_shares, price_per_share, status, created_at, updated_at
		FROM stock_orders WHERE id = ?`, id,
	).Scan(&o.ID, &o.CompanyID, &o.UserID, &o.OrderType, &o.Shares, &o.RemainingShares, &o.PricePerShare, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, exchange.ErrOrderNotFound
		}
		return nil, fmt.Errorf("find order: %w", err)
	}
	return o, nil
}

func (r *ExchangeRepo) UpdateOrder(order *exchange.StockOrder) error {
	_, err := r.db.Exec(`
		UPDATE stock_orders SET remaining_shares = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		order.RemainingShares, order.Status, order.ID,
	)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	return nil
}

func (r *ExchangeRepo) CancelOrder(id int) error {
	_, err := r.db.Exec(`
		UPDATE stock_orders SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	return nil
}

// FindMatchingSellOrders returns open/partial sell orders for a company
// with price <= maxPrice, sorted by price ASC (cheapest first).
func (r *ExchangeRepo) FindMatchingSellOrders(companyID int, maxPrice int, excludeUserID int) ([]*exchange.StockOrder, error) {
	rows, err := r.db.Query(`
		SELECT id, company_id, user_id, order_type, shares, remaining_shares, price_per_share, status, created_at, updated_at
		FROM stock_orders
		WHERE company_id = ? AND order_type = 'sell' AND status IN ('open', 'partial')
		  AND price_per_share <= ? AND user_id != ?
		ORDER BY price_per_share ASC, created_at ASC`,
		companyID, maxPrice, excludeUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("find matching sell orders: %w", err)
	}
	defer rows.Close()

	return scanOrders(rows)
}

// FindMatchingBuyOrders returns open/partial buy orders for a company
// with price >= minPrice, sorted by price DESC (highest first).
func (r *ExchangeRepo) FindMatchingBuyOrders(companyID int, minPrice int, excludeUserID int) ([]*exchange.StockOrder, error) {
	rows, err := r.db.Query(`
		SELECT id, company_id, user_id, order_type, shares, remaining_shares, price_per_share, status, created_at, updated_at
		FROM stock_orders
		WHERE company_id = ? AND order_type = 'buy' AND status IN ('open', 'partial')
		  AND price_per_share >= ? AND user_id != ?
		ORDER BY price_per_share DESC, created_at ASC`,
		companyID, minPrice, excludeUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("find matching buy orders: %w", err)
	}
	defer rows.Close()

	return scanOrders(rows)
}

func (r *ExchangeRepo) CreateTrade(trade *exchange.StockTrade) (int, error) {
	result, err := r.db.Exec(`
		INSERT INTO stock_trades (company_id, buy_order_id, sell_order_id, buyer_id, seller_id, shares, price_per_share, total_amount)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		trade.CompanyID, trade.BuyOrderID, trade.SellOrderID, trade.BuyerID, trade.SellerID,
		trade.Shares, trade.PricePerShare, trade.TotalAmount,
	)
	if err != nil {
		return 0, fmt.Errorf("create trade: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *ExchangeRepo) GetOrderbook(companyID int) (*exchange.Orderbook, error) {
	ob := &exchange.Orderbook{
		Asks: []*exchange.OrderbookEntry{},
		Bids: []*exchange.OrderbookEntry{},
	}

	// Asks (sell orders, price ASC)
	askRows, err := r.db.Query(`
		SELECT price_per_share, SUM(remaining_shares), COUNT(*)
		FROM stock_orders
		WHERE company_id = ? AND order_type = 'sell' AND status IN ('open', 'partial')
		GROUP BY price_per_share
		ORDER BY price_per_share ASC
		LIMIT 20`, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("get asks: %w", err)
	}
	defer askRows.Close()

	for askRows.Next() {
		e := &exchange.OrderbookEntry{}
		if err := askRows.Scan(&e.Price, &e.Shares, &e.Count); err != nil {
			return nil, err
		}
		ob.Asks = append(ob.Asks, e)
	}

	// Bids (buy orders, price DESC)
	bidRows, err := r.db.Query(`
		SELECT price_per_share, SUM(remaining_shares), COUNT(*)
		FROM stock_orders
		WHERE company_id = ? AND order_type = 'buy' AND status IN ('open', 'partial')
		GROUP BY price_per_share
		ORDER BY price_per_share DESC
		LIMIT 20`, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("get bids: %w", err)
	}
	defer bidRows.Close()

	for bidRows.Next() {
		e := &exchange.OrderbookEntry{}
		if err := bidRows.Scan(&e.Price, &e.Shares, &e.Count); err != nil {
			return nil, err
		}
		ob.Bids = append(ob.Bids, e)
	}

	return ob, nil
}

func (r *ExchangeRepo) GetUserOrders(userID int, status string, companyID int, page, limit int) ([]*exchange.StockOrder, int, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "user_id = ?")
	args = append(args, userID)

	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if companyID > 0 {
		conditions = append(conditions, "company_id = ?")
		args = append(args, companyID)
	}

	where := strings.Join(conditions, " AND ")

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM stock_orders WHERE "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	offset := (page - 1) * limit
	queryArgs := append(args, limit, offset)
	rows, err := r.db.Query(`
		SELECT id, company_id, user_id, order_type, shares, remaining_shares, price_per_share, status, created_at, updated_at
		FROM stock_orders WHERE `+where+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get user orders: %w", err)
	}
	defer rows.Close()

	orders, err := scanOrders(rows)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *ExchangeRepo) GetListedCompanies() ([]*exchange.ListedCompany, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.name, c.logo_url, c.total_shares,
			COALESCE((SELECT price_per_share FROM stock_trades WHERE company_id = c.id ORDER BY created_at DESC LIMIT 1), 0) as last_price,
			COALESCE(
				CASE
					WHEN (SELECT price_per_share FROM stock_trades WHERE company_id = c.id ORDER BY created_at DESC LIMIT 1 OFFSET 1) > 0
					THEN ((SELECT price_per_share FROM stock_trades WHERE company_id = c.id ORDER BY created_at DESC LIMIT 1) * 100.0 /
						  (SELECT price_per_share FROM stock_trades WHERE company_id = c.id ORDER BY created_at DESC LIMIT 1 OFFSET 1)) - 100
					ELSE 0
				END, 0) as change_percent,
			COALESCE((SELECT SUM(shares) FROM stock_trades WHERE company_id = c.id AND created_at >= datetime('now', '-24 hours')), 0) as volume_24h
		FROM companies c
		WHERE c.listed = 1 AND c.status = 'active'
		ORDER BY c.name`)
	if err != nil {
		return nil, fmt.Errorf("get listed companies: %w", err)
	}
	defer rows.Close()

	var companies []*exchange.ListedCompany
	for rows.Next() {
		lc := &exchange.ListedCompany{}
		if err := rows.Scan(&lc.ID, &lc.Name, &lc.LogoURL, &lc.TotalShares, &lc.LastPrice, &lc.ChangePercent, &lc.Volume24h); err != nil {
			return nil, err
		}
		lc.MarketCap = lc.LastPrice * lc.TotalShares
		companies = append(companies, lc)
	}

	if companies == nil {
		companies = []*exchange.ListedCompany{}
	}

	return companies, nil
}

func (r *ExchangeRepo) GetPendingBuyTotal(userID int) (int, error) {
	var total sql.NullInt64
	err := r.db.QueryRow(`
		SELECT SUM(remaining_shares * price_per_share)
		FROM stock_orders
		WHERE user_id = ? AND order_type = 'buy' AND status IN ('open', 'partial')`, userID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get pending buy total: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}

func (r *ExchangeRepo) GetPendingSellShares(userID int, companyID int) (int, error) {
	var total sql.NullInt64
	err := r.db.QueryRow(`
		SELECT SUM(remaining_shares)
		FROM stock_orders
		WHERE user_id = ? AND company_id = ? AND order_type = 'sell' AND status IN ('open', 'partial')`, userID, companyID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get pending sell shares: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}

func scanOrders(rows *sql.Rows) ([]*exchange.StockOrder, error) {
	var orders []*exchange.StockOrder
	for rows.Next() {
		o := &exchange.StockOrder{}
		if err := rows.Scan(&o.ID, &o.CompanyID, &o.UserID, &o.OrderType, &o.Shares, &o.RemainingShares,
			&o.PricePerShare, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}
