package application

import (
	"encoding/json"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type CompanyUsecase struct {
	companyRepo company.CompanyRepository
	userRepo    user.Repository
	walletRepo  wallet.Repository
}

func NewCompanyUsecase(cr company.CompanyRepository, ur user.Repository, wr wallet.Repository) *CompanyUsecase {
	return &CompanyUsecase{
		companyRepo: cr,
		userRepo:    ur,
		walletRepo:  wr,
	}
}

type CreateCompanyInput struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	LogoURL        string `json:"logo_url"`
	InitialCapital int    `json:"initial_capital"`
}

type CompanyOwnerInfo struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	StudentID string `json:"student_id"`
}

type CompanyDetail struct {
	*company.Company
	Owner         *CompanyOwnerInfo    `json:"owner"`
	OwnerName     string               `json:"owner_name"`
	WalletBalance int                  `json:"wallet_balance"`
	Shareholders  []ShareholderDetail  `json:"shareholders"`
}

type ShareholderDetail struct {
	*company.Shareholder
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
}

type MyCompanyItem struct {
	*company.Company
	MyShares      int     `json:"my_shares"`
	MyPercentage  float64 `json:"my_percentage"`
	WalletBalance int     `json:"wallet_balance"`
}

func (uc *CompanyUsecase) CreateCompany(userID int, input CreateCompanyInput) (*company.Company, error) {
	// Validate minimum capital
	if input.InitialCapital < company.MinInitialCapital {
		return nil, company.ErrMinCapital
	}

	// Check personal wallet balance
	w, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("지갑 조회 실패: %w", err)
	}
	if w.Balance < input.InitialCapital {
		return nil, company.ErrInsufficientFunds
	}

	// Debit personal wallet
	err = uc.walletRepo.Debit(w.ID, input.InitialCapital, wallet.TxCompanyFunding,
		fmt.Sprintf("회사 설립: %s", input.Name), "company", 0)
	if err != nil {
		return nil, fmt.Errorf("출금 실패: %w", err)
	}

	// Create company
	c := &company.Company{
		OwnerID:        userID,
		Name:           input.Name,
		Description:    input.Description,
		LogoURL:        input.LogoURL,
		InitialCapital: input.InitialCapital,
		TotalCapital:   input.InitialCapital,
		TotalShares:    company.DefaultTotalShares,
		Valuation:      input.InitialCapital,
		Listed:         false,
		BusinessCard:   "{}",
		Status:         "active",
	}

	companyID, err := uc.companyRepo.Create(c)
	if err != nil {
		// Refund on failure
		_ = uc.walletRepo.Credit(w.ID, input.InitialCapital, wallet.TxCompanyFunding,
			fmt.Sprintf("회사 설립 실패 환불: %s", input.Name), "company", 0)
		return nil, fmt.Errorf("회사 생성 실패: %w", err)
	}
	c.ID = companyID

	// Create company wallet with initial capital
	_, err = uc.companyRepo.CreateCompanyWallet(companyID, input.InitialCapital)
	if err != nil {
		return nil, fmt.Errorf("회사 지갑 생성 실패: %w", err)
	}

	// Create founder shareholder (10000 shares, founding)
	_, err = uc.companyRepo.CreateShareholder(&company.Shareholder{
		CompanyID:       companyID,
		UserID:          userID,
		Shares:          company.DefaultTotalShares,
		AcquisitionType: "founding",
	})
	if err != nil {
		return nil, fmt.Errorf("주주 등록 실패: %w", err)
	}

	// Check listing eligibility
	if c.CheckListing() {
		_ = uc.companyRepo.UpdateListed(companyID, true)
		c.Listed = true
	}

	return c, nil
}

func (uc *CompanyUsecase) GetCompany(companyID int) (*CompanyDetail, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, err
	}

	// Get owner info
	owner, err := uc.userRepo.FindByID(c.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("소유자 조회 실패: %w", err)
	}

	// Get shareholders
	shareholders, err := uc.companyRepo.FindShareholdersByCompanyID(companyID)
	if err != nil {
		return nil, fmt.Errorf("주주 조회 실패: %w", err)
	}

	shDetails := make([]ShareholderDetail, 0, len(shareholders))
	for _, sh := range shareholders {
		u, err := uc.userRepo.FindByID(sh.UserID)
		userName := "알 수 없음"
		if err == nil {
			userName = u.Name
		}
		shDetails = append(shDetails, ShareholderDetail{
			Shareholder: sh,
			Name:        userName,
			Percentage:  sh.Percentage(c.TotalShares),
		})
	}

	// Get company wallet balance
	walletBalance := 0
	cw, err := uc.companyRepo.FindCompanyWallet(companyID)
	if err == nil {
		walletBalance = cw.Balance
	}

	return &CompanyDetail{
		Company: c,
		Owner: &CompanyOwnerInfo{
			ID:        owner.ID,
			Name:      owner.Name,
			StudentID: owner.StudentID,
		},
		OwnerName:     owner.Name,
		WalletBalance: walletBalance,
		Shareholders:  shDetails,
	}, nil
}

func (uc *CompanyUsecase) UpdateCompany(companyID, userID int, description, logoURL string) error {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return err
	}
	if c.OwnerID != userID {
		return company.ErrNotOwner
	}

	c.Description = description
	c.LogoURL = logoURL
	return uc.companyRepo.Update(c)
}

func (uc *CompanyUsecase) GetMyCompanies(userID int) ([]*MyCompanyItem, error) {
	companies, err := uc.companyRepo.FindByOwnerID(userID)
	if err != nil {
		return nil, err
	}

	items := make([]*MyCompanyItem, 0, len(companies))
	for _, c := range companies {
		sh, err := uc.companyRepo.FindShareholder(c.ID, userID)
		myShares := 0
		myPct := 0.0
		if err == nil && sh != nil {
			myShares = sh.Shares
			myPct = sh.Percentage(c.TotalShares)
		}

		walletBalance := 0
		cw, err := uc.companyRepo.FindCompanyWallet(c.ID)
		if err == nil {
			walletBalance = cw.Balance
		}

		items = append(items, &MyCompanyItem{
			Company:       c,
			MyShares:      myShares,
			MyPercentage:  myPct,
			WalletBalance: walletBalance,
		})
	}

	return items, nil
}

func (uc *CompanyUsecase) GetAllCompanies() ([]*company.Company, error) {
	return uc.companyRepo.FindAll()
}

func (uc *CompanyUsecase) CreateBusinessCard(companyID, userID int, card company.BusinessCard) error {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return err
	}
	if c.OwnerID != userID {
		return company.ErrNotOwner
	}

	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("명함 데이터 직렬화 실패: %w", err)
	}

	c.BusinessCard = string(cardJSON)
	return uc.companyRepo.Update(c)
}

type BusinessCardOwner struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type BusinessCardCompany struct {
	ID           int                `json:"id"`
	OwnerID      int                `json:"owner_id"`
	Owner        *BusinessCardOwner `json:"owner"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	LogoURL      string             `json:"logo_url"`
	TotalCapital int                `json:"total_capital"`
	TotalShares  int                `json:"total_shares"`
	Valuation    int                `json:"valuation"`
	Listed       bool               `json:"listed"`
	Status       string             `json:"status"`
}

type BusinessCardResponse struct {
	Company BusinessCardCompany `json:"company"`
}

func (uc *CompanyUsecase) GetBusinessCard(companyID int) (*BusinessCardResponse, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, err
	}

	bc := BusinessCardCompany{
		ID:           c.ID,
		OwnerID:      c.OwnerID,
		Name:         c.Name,
		Description:  c.Description,
		LogoURL:      c.LogoURL,
		TotalCapital: c.TotalCapital,
		TotalShares:  c.TotalShares,
		Valuation:    c.Valuation,
		Listed:       c.Listed,
		Status:       c.Status,
	}

	// Get owner info
	owner, err := uc.userRepo.FindByID(c.OwnerID)
	if err == nil {
		bc.Owner = &BusinessCardOwner{
			ID:   owner.ID,
			Name: owner.Name,
		}
	}

	return &BusinessCardResponse{
		Company: bc,
	}, nil
}

func (uc *CompanyUsecase) DownloadBusinessCard(companyID int) ([]byte, string, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, "", err
	}

	var card company.BusinessCard
	if err := json.Unmarshal([]byte(c.BusinessCard), &card); err != nil {
		card = company.BusinessCard{CompanyName: c.Name}
	}

	// Simple text-based placeholder for business card PDF
	content := fmt.Sprintf(`
=====================================
        %s
=====================================
  %s - %s
  Email: %s
  Phone: %s
  Address: %s
  Website: %s
=====================================
`, card.CompanyName, card.OwnerName, card.Title,
		card.Email, card.Phone, card.Address, card.Website)

	return []byte(content), c.Name + "_명함.txt", nil
}
