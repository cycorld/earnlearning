package company

import (
	"database/sql"
	"time"
)

// CompanyRepository defines the persistence interface for the company domain.
type CompanyRepository interface {
	// WithTx returns a CompanyRepository whose writes run inside tx (#142).
	WithTx(tx *sql.Tx) CompanyRepository

	Create(c *Company) (int, error)
	FindByID(id int) (*Company, error)
	FindByOwnerID(ownerID int) ([]*Company, error)
	FindAll(classroomID int) ([]*Company, error)
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
	GetCompanyTransactions(walletID int, page, limit int) ([]*CompanyTransaction, int, error)

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

// CompanyServiceRepository — #181 회사 등록 서비스 저장소.
// 중복 URL 은 (company_id, normalized_url) UNIQUE 인덱스가 최종 방어선이며,
// 위반 시 ErrDuplicateServiceURL 을 반환한다.
type CompanyServiceRepository interface {
	Create(s *CompanyService) (int, error)
	// FindByID 는 없으면 ErrServiceNotFound.
	FindByID(id int) (*CompanyService, error)
	// FindByCompanyID 는 id 오름차순 정렬.
	FindByCompanyID(companyID int) ([]*CompanyService, error)
	// FindByCompanyAndNormalizedURL 은 중복 사전 검사용 (없으면 ErrServiceNotFound).
	FindByCompanyAndNormalizedURL(companyID int, normalizedURL string) (*CompanyService, error)
	// Update 는 이름/URL/검증 상태/연동 상태를 한 번에 덮어쓴다 (updated_at 자동 갱신).
	Update(s *CompanyService) error
	// UpdateValidation 은 검증 결과만 기록한다 (서버 시각 기준).
	UpdateValidation(id int, status string, checkedAt *time.Time, detail string) error
	// UpdateRybbit 은 연동 성공 후에만 호출되는 단일 UPDATE (fail-closed).
	UpdateRybbit(id int, status, siteID string, connectedAt *time.Time) error
	// Delete 는 대상이 없으면 ErrServiceNotFound.
	Delete(id int) error
}
