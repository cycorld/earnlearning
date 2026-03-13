package company

// CompanyRepository defines the persistence interface for the company domain.
type CompanyRepository interface {
	Create(c *Company) (int, error)
	FindByID(id int) (*Company, error)
	FindByOwnerID(ownerID int) ([]*Company, error)
	FindAll() ([]*Company, error)
	Update(c *Company) error
	UpdateListed(companyID int, listed bool) error

	// Shareholder operations
	CreateShareholder(s *Shareholder) (int, error)
	FindShareholdersByCompanyID(companyID int) ([]*Shareholder, error)
	FindShareholder(companyID, userID int) (*Shareholder, error)
	UpsertShareholder(companyID, userID, shares int, acquisitionType string) error
	UpdateTotalShares(companyID, totalShares int) error
	UpdateCapitalAndValuation(companyID, totalCapital, valuation int) error

	// Company wallet operations
	CreateCompanyWallet(companyID int, initialBalance int) (int, error)
	FindCompanyWallet(companyID int) (*CompanyWallet, error)
	CreditCompanyWallet(walletID int, amount int, txType string, desc string, refType string, refID int) error
	DebitCompanyWallet(walletID int, amount int, txType string, desc string, refType string, refID int) error
}
