package exchange

type Repository interface {
	// Order operations
	CreateOrder(order *StockOrder) (int, error)
	FindOrderByID(id int) (*StockOrder, error)
	UpdateOrder(order *StockOrder) error
	CancelOrder(id int) error

	// Matching queries
	FindMatchingBuyOrders(companyID int, maxPrice int, excludeUserID int) ([]*StockOrder, error)
	FindMatchingSellOrders(companyID int, minPrice int, excludeUserID int) ([]*StockOrder, error)

	// Trade operations
	CreateTrade(trade *StockTrade) (int, error)

	// Query operations
	GetOrderbook(companyID int) (*Orderbook, error)
	GetUserOrders(userID int, status string, companyID int, page, limit int) ([]*StockOrder, int, error)
	GetListedCompanies() ([]*ListedCompany, error)

	// Pending order calculations
	GetPendingBuyTotal(userID int) (int, error)
	GetPendingSellShares(userID int, companyID int) (int, error)
}
