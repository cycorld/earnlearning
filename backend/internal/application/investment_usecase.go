package application

import (
	"database/sql"
	"fmt"
	"math"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/investment"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type InvestmentUseCase struct {
	db          *sql.DB
	repo        investment.Repository
	companyRepo company.CompanyRepository
	walletRepo  wallet.Repository
	notifUC     *NotificationUseCase
	autoPoster  *AutoPoster
}

func NewInvestmentUseCase(
	db *sql.DB,
	repo investment.Repository,
	cr company.CompanyRepository,
	wr wallet.Repository,
) *InvestmentUseCase {
	return &InvestmentUseCase{db: db, repo: repo, companyRepo: cr, walletRepo: wr, autoPoster: NewAutoPoster(db)}
}

func (uc *InvestmentUseCase) SetNotificationUseCase(notifUC *NotificationUseCase) {
	uc.notifUC = notifUC
}

// --- Input types ---

type CreateRoundInput struct {
	CompanyID      int     `json:"company_id"`
	TargetAmount   int     `json:"target_amount"`
	OfferedPercent float64 `json:"offered_percent"`
}

type InvestInput struct {
	RoundID int `json:"round_id"`
}

type ExecuteDividendInput struct {
	CompanyID   int `json:"company_id"`
	TotalAmount int `json:"total_amount"`
}

type CreateKpiRuleInput struct {
	CompanyID       int    `json:"company_id"`
	RuleDescription string `json:"rule_description"`
}

type AddKpiRevenueInput struct {
	CompanyID int    `json:"company_id"`
	KpiRuleID *int   `json:"kpi_rule_id"`
	Amount    int    `json:"amount"`
	Memo      string `json:"memo"`
}

// --- Use case methods ---

func (uc *InvestmentUseCase) CreateRound(input CreateRoundInput, userID int) (*investment.InvestmentRound, error) {
	// Validate
	c, err := uc.companyRepo.FindByID(input.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyNotFound
	}
	if c.OwnerID != userID {
		return nil, investment.ErrNotOwner
	}
	if input.OfferedPercent <= 0 || input.OfferedPercent >= 1 {
		return nil, investment.ErrInvalidPercent
	}
	if input.TargetAmount <= 0 {
		return nil, investment.ErrInvalidAmount
	}

	// Check no open round exists
	hasOpen, err := uc.repo.HasOpenRound(input.CompanyID)
	if err != nil {
		return nil, err
	}
	if hasOpen {
		return nil, investment.ErrOpenRoundExists
	}

	// Calculate new_shares and price_per_share
	// new_shares = total_shares * offered_percent / (1 - offered_percent)
	newShares := int(math.Round(float64(c.TotalShares) * input.OfferedPercent / (1 - input.OfferedPercent)))
	if newShares <= 0 {
		newShares = 1
	}
	// price_per_share = target_amount / new_shares
	pricePerShare := float64(input.TargetAmount) / float64(newShares)

	round := &investment.InvestmentRound{
		CompanyID:      input.CompanyID,
		TargetAmount:   input.TargetAmount,
		OfferedPercent: input.OfferedPercent,
		CurrentAmount:  0,
		PricePerShare:  pricePerShare,
		NewShares:      newShares,
		Status:         investment.RoundOpen,
	}

	id, err := uc.repo.CreateRound(round)
	if err != nil {
		return nil, err
	}

	// Auto-post to 투자라운지 channel
	content := fmt.Sprintf("## 📈 투자 라운드 오픈: %s\n\n**목표 금액:** %s\n**지분 제공:** %.1f%%\n**주당 가격:** %s\n\n👉 [투자하러 가기](/investment)",
		c.Name, formatMoney(input.TargetAmount), input.OfferedPercent*100, formatMoney(int(pricePerShare)))
	uc.autoPoster.PostToChannel("invest", userID, content, []string{"투자라운드", c.Name})

	return uc.repo.FindRoundByID(id)
}

func (uc *InvestmentUseCase) Invest(roundID, userID int) (*investment.Investment, error) {
	round, err := uc.repo.FindRoundByID(roundID)
	if err != nil {
		return nil, err
	}
	if round.Status != investment.RoundOpen {
		return nil, investment.ErrRoundNotOpen
	}

	// Get company to check ownership
	c, err := uc.companyRepo.FindByID(round.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyNotFound
	}
	if c.OwnerID == userID {
		return nil, investment.ErrCannotInvestOwnCompany
	}

	// 1 round = 1 investor, check balance >= target_amount
	investAmount := round.TargetAmount
	w, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, investment.ErrInsufficientFunds
	}
	if w.Balance < investAmount {
		return nil, investment.ErrInsufficientFunds
	}

	// Debit investor wallet
	err = uc.walletRepo.Debit(w.ID, investAmount, wallet.TxInvestment,
		fmt.Sprintf("%s 투자", c.Name), "investment_round", round.ID)
	if err != nil {
		return nil, err
	}

	// Fund round immediately
	if err := uc.repo.UpdateRoundFunded(round.ID, investAmount); err != nil {
		return nil, err
	}

	// Update company: total_shares, total_capital, valuation
	newTotalShares := c.TotalShares + round.NewShares
	newTotalCapital := c.TotalCapital + investAmount
	// Post-money valuation = target_amount / offered_percent
	newValuation := int(math.Round(float64(investAmount) / round.OfferedPercent))

	if err := uc.companyRepo.UpdateTotalShares(round.CompanyID, newTotalShares); err != nil {
		return nil, err
	}
	if err := uc.companyRepo.UpdateCapitalAndValuation(round.CompanyID, newTotalCapital, newValuation); err != nil {
		return nil, err
	}

	// Upsert shareholder
	if err := uc.companyRepo.UpsertShareholder(round.CompanyID, userID, round.NewShares, "investment"); err != nil {
		return nil, err
	}

	// Credit company wallet
	companyWallet, err := uc.companyRepo.FindCompanyWallet(round.CompanyID)
	if err != nil {
		return nil, err
	}
	err = uc.companyRepo.CreditCompanyWallet(companyWallet.ID, investAmount, "investment",
		fmt.Sprintf("투자 유치: %d원", investAmount), "investment_round", round.ID)
	if err != nil {
		return nil, err
	}

	// Check listing
	updatedCompany, err := uc.companyRepo.FindByID(round.CompanyID)
	if err == nil && updatedCompany.CheckListing() && !updatedCompany.Listed {
		_ = uc.companyRepo.UpdateListed(round.CompanyID, true)
	}

	// Create investment record
	inv := &investment.Investment{
		RoundID: round.ID,
		UserID:  userID,
		Amount:  investAmount,
		Shares:  round.NewShares,
	}
	invID, err := uc.repo.CreateInvestment(inv)
	if err != nil {
		return nil, err
	}
	inv.ID = invID

	// Notify company owner
	uc.notify(c.OwnerID, "investment_funded",
		"투자가 완료되었습니다",
		fmt.Sprintf("%s에 %d원 투자가 완료되었습니다.", c.Name, investAmount),
		"investment_round", round.ID)

	// Notify investor
	uc.notify(userID, "investment_received",
		"투자가 완료되었습니다",
		fmt.Sprintf("%s에 %s을 투자하여 %d주를 취득했습니다.", c.Name, formatMoney(investAmount), round.NewShares),
		"investment_round", round.ID)

	return inv, nil
}

func (uc *InvestmentUseCase) ListRounds(companyID int, status string, page, limit int) ([]*investment.InvestmentRound, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	filter := investment.RoundFilter{
		CompanyID: companyID,
		Status:    status,
	}
	return uc.repo.ListRounds(filter, page, limit)
}

func (uc *InvestmentUseCase) GetPortfolio(userID int) ([]*investment.PortfolioItem, error) {
	invs, err := uc.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	// Aggregate by company
	companyMap := make(map[int]*investment.PortfolioItem)
	for _, inv := range invs {
		item, exists := companyMap[inv.CompanyID]
		if !exists {
			item = &investment.PortfolioItem{
				CompanyID:   inv.CompanyID,
				CompanyName: inv.CompanyName,
			}
			companyMap[inv.CompanyID] = item
		}
		item.UserShares += inv.Shares
		item.Invested += inv.Amount
	}

	var portfolio []*investment.PortfolioItem
	for _, item := range companyMap {
		// Get current company data for valuation
		c, err := uc.companyRepo.FindByID(item.CompanyID)
		if err == nil {
			item.TotalShares = c.TotalShares
			if c.TotalShares > 0 {
				item.Percentage = float64(item.UserShares) / float64(c.TotalShares) * 100
				item.CurrentValue = int(float64(c.Valuation) * float64(item.UserShares) / float64(c.TotalShares))
			}
		}
		portfolio = append(portfolio, item)
	}
	return portfolio, nil
}

func (uc *InvestmentUseCase) ExecuteDividend(input ExecuteDividendInput, userID int) (*investment.Dividend, error) {
	// Verify ownership
	c, err := uc.companyRepo.FindByID(input.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyNotFound
	}
	if c.OwnerID != userID {
		return nil, investment.ErrNotOwner
	}
	if input.TotalAmount <= 0 {
		return nil, investment.ErrInvalidAmount
	}

	// Check company wallet balance
	companyWallet, err := uc.companyRepo.FindCompanyWallet(input.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyWalletNotFound
	}
	if companyWallet.Balance < input.TotalAmount {
		return nil, investment.ErrInsufficientFunds
	}

	// Debit company wallet
	err = uc.companyRepo.DebitCompanyWallet(companyWallet.ID, input.TotalAmount, "dividend",
		fmt.Sprintf("배당금 지급: %d원", input.TotalAmount), "dividend", 0)
	if err != nil {
		return nil, err
	}

	// Create dividend record
	dividend := &investment.Dividend{
		CompanyID:   input.CompanyID,
		TotalAmount: input.TotalAmount,
		ExecutedBy:  userID,
	}
	dividendID, err := uc.repo.CreateDividend(dividend)
	if err != nil {
		return nil, err
	}
	dividend.ID = dividendID

	// Get all shareholders
	shareholders, err := uc.companyRepo.FindShareholdersByCompanyID(input.CompanyID)
	if err != nil {
		return nil, err
	}

	// Distribute dividends proportionally using floor division
	var payments []*investment.DividendPayment
	for _, sh := range shareholders {
		// amount = floor(totalAmount * shares / totalShares)
		payAmount := (input.TotalAmount * sh.Shares) / c.TotalShares
		if payAmount <= 0 {
			continue
		}

		// Credit shareholder's wallet
		shWallet, err := uc.walletRepo.FindByUserID(sh.UserID)
		if err != nil {
			continue // skip if no wallet
		}
		err = uc.walletRepo.Credit(shWallet.ID, payAmount, wallet.TxDividend,
			fmt.Sprintf("%s 배당금", c.Name), "dividend", dividendID)
		if err != nil {
			continue
		}

		payment := &investment.DividendPayment{
			DividendID: dividendID,
			UserID:     sh.UserID,
			Shares:     sh.Shares,
			Amount:     payAmount,
		}
		paymentID, err := uc.repo.CreateDividendPayment(payment)
		if err != nil {
			continue
		}
		payment.ID = paymentID
		payments = append(payments, payment)

		// Notify each shareholder
		uc.notify(sh.UserID, "dividend_received",
			"배당금을 받았습니다",
			fmt.Sprintf("%s에서 %s의 배당금을 받았습니다.", c.Name, formatMoney(payAmount)),
			"dividend", dividendID)
	}
	dividend.Payments = payments

	return dividend, nil
}

func (uc *InvestmentUseCase) GetMyDividends(userID int) ([]*investment.DividendPayment, error) {
	return uc.repo.ListDividendsByUser(userID)
}

func (uc *InvestmentUseCase) CreateKpiRule(input CreateKpiRuleInput) (*investment.KpiRule, error) {
	rule := &investment.KpiRule{
		CompanyID:       input.CompanyID,
		RuleDescription: input.RuleDescription,
		Active:          true,
	}
	id, err := uc.repo.CreateKpiRule(rule)
	if err != nil {
		return nil, err
	}
	rule.ID = id
	return rule, nil
}

func (uc *InvestmentUseCase) AddKpiRevenue(input AddKpiRevenueInput, adminUserID int) (*investment.KpiRevenue, error) {
	if input.Amount <= 0 {
		return nil, investment.ErrInvalidAmount
	}

	// Credit company wallet
	companyWallet, err := uc.companyRepo.FindCompanyWallet(input.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyWalletNotFound
	}
	err = uc.companyRepo.CreditCompanyWallet(companyWallet.ID, input.Amount, "kpi_revenue",
		fmt.Sprintf("KPI 매출: %s", input.Memo), "kpi_revenue", 0)
	if err != nil {
		return nil, err
	}

	rev := &investment.KpiRevenue{
		CompanyID: input.CompanyID,
		KpiRuleID: input.KpiRuleID,
		Amount:    input.Amount,
		Memo:      input.Memo,
		CreatedBy: adminUserID,
	}
	id, err := uc.repo.CreateKpiRevenue(rev)
	if err != nil {
		return nil, err
	}
	rev.ID = id
	return rev, nil
}

func (uc *InvestmentUseCase) notify(userID int, notifType, title, body, refType string, refID int) {
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(userID, notification.NotifType(notifType), title, body, refType, refID)
	}
}
