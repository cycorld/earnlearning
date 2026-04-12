package application

import (
	"encoding/json"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/user"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type CompanyUsecase struct {
	companyRepo company.CompanyRepository
	userRepo    user.Repository
	walletRepo  wallet.Repository
	notifUC     *NotificationUseCase
}

func (uc *CompanyUsecase) SetNotificationUseCase(n *NotificationUseCase) {
	uc.notifUC = n
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

// UpdateCompanyInput 는 기업 정보 부분 업데이트용. 빈 문자열은 "변경 안 함" 의미가
// 아니라 명시적 값으로 처리한다 (description / logo_url 은 빈 값이 정상). name 만
// 빈 문자열이면 변경 안 함으로 본다.
type UpdateCompanyInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	ServiceURL  string `json:"service_url"`
}

func (uc *CompanyUsecase) UpdateCompany(companyID, userID int, input UpdateCompanyInput) error {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return err
	}
	if c.OwnerID != userID {
		return company.ErrNotOwner
	}

	// name 은 빈 문자열이면 기존 값 유지 (호환성 — 기존 클라이언트가 안 보낼 수 있음)
	if input.Name != "" {
		c.Name = input.Name
	}
	c.Description = input.Description
	c.LogoURL = input.LogoURL
	c.ServiceURL = input.ServiceURL
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

// PublicCompanyItem 은 학생용 전체 기업 목록의 한 항목.
// owner 정보를 함께 노출 (이름/학번) 해서 카드 UI 에서 바로 표시 가능.
type PublicCompanyItem struct {
	*company.Company
	OwnerName     string `json:"owner_name"`
	OwnerStudent  string `json:"owner_student_id"`
	WalletBalance int    `json:"wallet_balance"`
}

// GetAllCompaniesWithOwners 는 모든 회사 + 소유자 정보를 반환한다.
// 학생/관리자 누구나 호출 가능.
func (uc *CompanyUsecase) GetAllCompaniesWithOwners() ([]*PublicCompanyItem, error) {
	companies, err := uc.companyRepo.FindAll()
	if err != nil {
		return nil, err
	}

	items := make([]*PublicCompanyItem, 0, len(companies))
	for _, c := range companies {
		item := &PublicCompanyItem{Company: c}

		// 소유자 정보
		owner, err := uc.userRepo.FindByID(c.OwnerID)
		if err == nil && owner != nil {
			item.OwnerName = owner.Name
			item.OwnerStudent = owner.StudentID
		}

		// 회사 지갑 잔액
		cw, err := uc.companyRepo.FindCompanyWallet(c.ID)
		if err == nil && cw != nil {
			item.WalletBalance = cw.Balance
		}

		items = append(items, item)
	}
	return items, nil
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
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
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
			ID:    owner.ID,
			Name:  owner.Name,
			Email: owner.Email,
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

// Disclosure (공시) operations

type CreateDisclosureInput struct {
	Content    string `json:"content"`
	PeriodFrom string `json:"period_from"`
	PeriodTo   string `json:"period_to"`
}

type DisclosureDetail struct {
	*company.Disclosure
	CompanyName string `json:"company_name"`
	AuthorName  string `json:"author_name"`
}

func (uc *CompanyUsecase) CreateDisclosure(companyID, userID int, input CreateDisclosureInput) (*company.Disclosure, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, err
	}
	if c.OwnerID != userID {
		return nil, company.ErrNotOwner
	}
	if input.Content == "" || input.PeriodFrom == "" || input.PeriodTo == "" {
		return nil, fmt.Errorf("공시 내용, 기간 시작/종료일을 모두 입력해주세요")
	}

	d := &company.Disclosure{
		CompanyID:  companyID,
		AuthorID:   userID,
		Content:    input.Content,
		PeriodFrom: input.PeriodFrom,
		PeriodTo:   input.PeriodTo,
		Status:     "pending",
	}
	id, err := uc.companyRepo.CreateDisclosure(d)
	if err != nil {
		return nil, fmt.Errorf("공시 생성 실패: %w", err)
	}
	d.ID = id
	return d, nil
}

func (uc *CompanyUsecase) GetDisclosuresByCompanyID(companyID int) ([]*DisclosureDetail, error) {
	disclosures, err := uc.companyRepo.FindDisclosuresByCompanyID(companyID)
	if err != nil {
		return nil, err
	}
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, err
	}

	details := make([]*DisclosureDetail, 0, len(disclosures))
	for _, d := range disclosures {
		authorName := "알 수 없음"
		if u, err := uc.userRepo.FindByID(d.AuthorID); err == nil {
			authorName = u.Name
		}
		details = append(details, &DisclosureDetail{
			Disclosure:  d,
			CompanyName: c.Name,
			AuthorName:  authorName,
		})
	}
	return details, nil
}

func (uc *CompanyUsecase) GetAllDisclosures() ([]*DisclosureDetail, error) {
	disclosures, err := uc.companyRepo.FindAllDisclosures()
	if err != nil {
		return nil, err
	}

	details := make([]*DisclosureDetail, 0, len(disclosures))
	for _, d := range disclosures {
		companyName := "알 수 없음"
		if c, err := uc.companyRepo.FindByID(d.CompanyID); err == nil {
			companyName = c.Name
		}
		authorName := "알 수 없음"
		if u, err := uc.userRepo.FindByID(d.AuthorID); err == nil {
			authorName = u.Name
		}
		details = append(details, &DisclosureDetail{
			Disclosure:  d,
			CompanyName: companyName,
			AuthorName:  authorName,
		})
	}
	return details, nil
}

type ApproveDisclosureInput struct {
	Reward    int    `json:"reward"`
	AdminNote string `json:"admin_note"`
}

func (uc *CompanyUsecase) ApproveDisclosure(disclosureID int, input ApproveDisclosureInput) error {
	d, err := uc.companyRepo.FindDisclosureByID(disclosureID)
	if err != nil {
		return err
	}
	if d.Status != "pending" {
		return fmt.Errorf("이미 처리된 공시입니다 (상태: %s)", d.Status)
	}

	// Update status
	if err := uc.companyRepo.UpdateDisclosureStatus(disclosureID, "approved", input.Reward, input.AdminNote); err != nil {
		return fmt.Errorf("공시 상태 업데이트 실패: %w", err)
	}

	// Credit company wallet if reward > 0
	if input.Reward > 0 {
		cw, err := uc.companyRepo.FindCompanyWallet(d.CompanyID)
		if err != nil {
			return fmt.Errorf("회사 지갑 조회 실패: %w", err)
		}
		err = uc.companyRepo.CreditCompanyWallet(cw.ID, input.Reward,
			"disclosure_reward",
			fmt.Sprintf("공시 승인 수익금 (%s ~ %s)", d.PeriodFrom, d.PeriodTo),
			"disclosure", disclosureID,
		)
		if err != nil {
			return fmt.Errorf("수익금 입금 실패: %w", err)
		}
	}

	// Send notification to company owner
	if uc.notifUC != nil {
		c, _ := uc.companyRepo.FindByID(d.CompanyID)
		if c != nil {
			title := fmt.Sprintf("[%s] 공시가 승인되었습니다", c.Name)
			body := fmt.Sprintf("공시(%s ~ %s)가 승인되었습니다.", d.PeriodFrom, d.PeriodTo)
			if input.Reward > 0 {
				body += fmt.Sprintf(" 수익금 %d원이 법인 계좌에 입금되었습니다.", input.Reward)
			}
			if input.AdminNote != "" {
				body += " 코멘트: " + input.AdminNote
			}
			_ = uc.notifUC.CreateNotification(c.OwnerID, notification.NotifDisclosureApproved, title, body, "company", c.ID)
		}
	}

	return nil
}

func (uc *CompanyUsecase) RejectDisclosure(disclosureID int, adminNote string) error {
	d, err := uc.companyRepo.FindDisclosureByID(disclosureID)
	if err != nil {
		return err
	}
	if d.Status != "pending" {
		return fmt.Errorf("이미 처리된 공시입니다 (상태: %s)", d.Status)
	}

	if err := uc.companyRepo.UpdateDisclosureStatus(disclosureID, "rejected", 0, adminNote); err != nil {
		return err
	}

	// Send notification to company owner
	if uc.notifUC != nil {
		c, _ := uc.companyRepo.FindByID(d.CompanyID)
		if c != nil {
			title := fmt.Sprintf("[%s] 공시가 거절되었습니다", c.Name)
			body := fmt.Sprintf("공시(%s ~ %s)가 거절되었습니다.", d.PeriodFrom, d.PeriodTo)
			if adminNote != "" {
				body += " 사유: " + adminNote
			}
			_ = uc.notifUC.CreateNotification(c.OwnerID, notification.NotifDisclosureRejected, title, body, "company", c.ID)
		}
	}

	return nil
}
