package exchange

import "time"

type OrderType string

const (
	OrderTypeBuy  OrderType = "buy"
	OrderTypeSell OrderType = "sell"
)

type OrderStatus string

const (
	OrderStatusOpen      OrderStatus = "open"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type StockOrder struct {
	ID              int         `json:"id"`
	CompanyID       int         `json:"company_id"`
	UserID          int         `json:"user_id"`
	OrderType       OrderType   `json:"order_type"`
	Shares          int         `json:"shares"`
	RemainingShares int         `json:"remaining_shares"`
	PricePerShare   int         `json:"price_per_share"`
	Status          OrderStatus `json:"status"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type StockTrade struct {
	ID            int       `json:"id"`
	CompanyID     int       `json:"company_id"`
	BuyOrderID    int       `json:"buy_order_id"`
	SellOrderID   int       `json:"sell_order_id"`
	BuyerID       int       `json:"buyer_id"`
	SellerID      int       `json:"seller_id"`
	Shares        int       `json:"shares"`
	PricePerShare int       `json:"price_per_share"`
	TotalAmount   int       `json:"total_amount"`
	CreatedAt     time.Time `json:"created_at"`
}

type OrderbookEntry struct {
	Price  int `json:"price"`
	Shares int `json:"shares"`
	Count  int `json:"count"`
}

type Orderbook struct {
	Asks []*OrderbookEntry `json:"asks"`
	Bids []*OrderbookEntry `json:"bids"`
}

type ListedCompany struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	LogoURL       string  `json:"logo_url"`
	TotalShares   int     `json:"total_shares"`
	LastPrice     int     `json:"last_price"`
	ChangePercent float64 `json:"change_percent"`
	Volume24h     int     `json:"volume_24h"`
	MarketCap     int     `json:"market_cap"`
}
