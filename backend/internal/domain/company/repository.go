package company

import "time"

// CompanyRepository defines the persistence interface for the company domain.
type CompanyRepository interface {
	Create(c *Company) (int, error)
	FindByID(id int) (*Company, error)
	FindByOwnerID(ownerID int) ([]*Company, error)
	FindAll() ([]*Company, error)
	Update(c *Company) error
	UpdateListed(companyID int, listed bool) error
	UpdateStatus(companyID int, status string) error

	// Shareholder operations
	CreateShareholder(s *Shareholder) (int, error)
	FindShareholdersByCompanyID(companyID int) ([]*Shareholder, error)
	FindShareholder(companyID, userID int) (*Shareholder, error)
	UpsertShareholder(companyID, userID, shares int, acquisitionType string) error
	// SubtractShareholderShares decrements a shareholder's position. If the
	// result is <= 0, the row is deleted. Used for refund/cancel flows.
	SubtractShareholderShares(companyID, userID, shares int) error
	UpdateTotalShares(companyID, totalShares int) error
	UpdateCapitalAndValuation(companyID, totalCapital, valuation int) error

	// Company wallet operations
	CreateCompanyWallet(companyID int, initialBalance int) (int, error)
	FindCompanyWallet(companyID int) (*CompanyWallet, error)
	CreditCompanyWallet(walletID int, amount int, txType string, desc string, refType string, refID int) error
	DebitCompanyWallet(walletID int, amount int, txType string, desc string, refType string, refID int) error

	// Disclosure operations
	CreateDisclosure(d *Disclosure) (int, error)
	FindDisclosureByID(id int) (*Disclosure, error)
	FindDisclosuresByCompanyID(companyID int) ([]*Disclosure, error)
	FindAllDisclosures() ([]*Disclosure, error)
	UpdateDisclosureStatus(id int, status string, reward int, adminNote string) error

	// Proposal (주주총회) operations
	CreateProposal(p *Proposal) (int, error)
	FindProposalByID(id int) (*Proposal, error)
	FindProposalsByCompanyID(companyID int) ([]*Proposal, error)
	FindActiveProposalByCompanyAndType(companyID int, proposalType string) (*Proposal, error)
	UpdateProposalStatus(id int, status string, resultNote string, closedAt *time.Time) error

	// Vote operations
	CreateVote(v *Vote) (int, error)
	FindVote(proposalID, userID int) (*Vote, error)
	FindVotesByProposalID(proposalID int) ([]*Vote, error)
}
