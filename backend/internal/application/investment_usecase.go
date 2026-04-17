package application

import (
	"database/sql"
	"fmt"
	"math"
	"time"

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
	Shares  int `json:"shares"` // number of shares to purchase (partial invest)
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
	content := fmt.Sprintf("## 📈 투자 라운드 오픈: %s\n\n**목표 금액:** %s\n**지분 제공:** %.1f%%\n**주당 가격:** %s\n\n👉 [투자하러 가기](/invest/%d)",
		c.Name, formatMoney(input.TargetAmount), input.OfferedPercent*100, formatMoney(int(pricePerShare)), id)
	uc.autoPoster.PostToChannel("invest", userID, content, []string{"투자라운드", c.Name})

	return uc.repo.FindRoundByID(id)
}

// Invest supports partial investment: one round can be filled by multiple
// investors, each buying a subset of the round's new_shares. When the final
// share sells, the round is marked as `funded` and the company's valuation
// jumps to the post-money value.
func (uc *InvestmentUseCase) Invest(roundID, userID, shares int) (*investment.Investment, error) {
	round, err := uc.repo.FindRoundByID(roundID)
	if err != nil {
		return nil, err
	}
	// Auto-expire if past expiry before any write.
	round, err = uc.maybeAutoExpire(round)
	if err != nil {
		return nil, err
	}
	if round.Status != investment.RoundOpen {
		return nil, investment.ErrRoundNotOpen
	}
	if shares <= 0 {
		return nil, investment.ErrInvalidShares
	}

	// Company ownership guard
	c, err := uc.companyRepo.FindByID(round.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyNotFound
	}
	if c.OwnerID == userID {
		return nil, investment.ErrCannotInvestOwnCompany
	}

	// Compute remaining shares = new_shares - sum(investments.shares)
	sold, err := uc.repo.SumSharesByRound(round.ID)
	if err != nil {
		return nil, err
	}
	remaining := round.NewShares - sold
	if remaining <= 0 {
		return nil, investment.ErrRoundNotOpen
	}
	if shares > remaining {
		return nil, investment.ErrOverSubscribed
	}

	// Compute this investor's payment.
	// - Last investor pays `target - current_amount` exactly (eats rounding).
	// - Others pay round(shares * price_per_share).
	var investAmount int
	if shares == remaining {
		investAmount = round.TargetAmount - round.CurrentAmount
		if investAmount < 0 {
			investAmount = 0
		}
	} else {
		investAmount = int(math.Round(float64(shares) * round.PricePerShare))
	}

	// Wallet balance check + debit
	w, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, investment.ErrInsufficientFunds
	}
	if w.Balance < investAmount {
		return nil, investment.ErrInsufficientFunds
	}
	if err := uc.walletRepo.Debit(w.ID, investAmount, wallet.TxInvestment,
		fmt.Sprintf("%s 투자", c.Name), "investment_round", round.ID); err != nil {
		return nil, err
	}

	// Update round.current_amount. Close the round only if it's now fully sold.
	newCurrentAmount := round.CurrentAmount + investAmount
	isFinalSale := shares == remaining
	if isFinalSale {
		if err := uc.repo.UpdateRoundFunded(round.ID, newCurrentAmount); err != nil {
			return nil, err
		}
	} else {
		if err := uc.repo.UpdateRoundCurrentAmount(round.ID, newCurrentAmount); err != nil {
			return nil, err
		}
	}

	// Update company aggregates.
	newTotalShares := c.TotalShares + shares
	newTotalCapital := c.TotalCapital + investAmount
	if err := uc.companyRepo.UpdateTotalShares(round.CompanyID, newTotalShares); err != nil {
		return nil, err
	}
	// Valuation: only finalize on full close using post-money formula.
	// For partial fills we leave the current valuation untouched (UpdateCapital-
	// AndValuation takes both, so pass existing valuation through).
	newValuation := c.Valuation
	if isFinalSale {
		newValuation = int(math.Round(float64(round.TargetAmount) / round.OfferedPercent))
	}
	if err := uc.companyRepo.UpdateCapitalAndValuation(round.CompanyID, newTotalCapital, newValuation); err != nil {
		return nil, err
	}

	// Upsert shareholder (additive upsert adds to any existing position).
	if err := uc.companyRepo.UpsertShareholder(round.CompanyID, userID, shares, "investment"); err != nil {
		return nil, err
	}

	// Credit company wallet
	companyWallet, err := uc.companyRepo.FindCompanyWallet(round.CompanyID)
	if err != nil {
		return nil, err
	}
	if err := uc.companyRepo.CreditCompanyWallet(companyWallet.ID, investAmount, "investment",
		fmt.Sprintf("투자 유치: %d원", investAmount), "investment_round", round.ID); err != nil {
		return nil, err
	}

	// Auto-list check
	if updated, err := uc.companyRepo.FindByID(round.CompanyID); err == nil && updated.CheckListing() && !updated.Listed {
		_ = uc.companyRepo.UpdateListed(round.CompanyID, true)
	}

	// Record this individual investment
	inv := &investment.Investment{
		RoundID: round.ID,
		UserID:  userID,
		Amount:  investAmount,
		Shares:  shares,
	}
	invID, err := uc.repo.CreateInvestment(inv)
	if err != nil {
		return nil, err
	}
	inv.ID = invID

	// Notifications
	if isFinalSale {
		uc.notify(c.OwnerID, "investment_funded",
			"투자 라운드 마감",
			fmt.Sprintf("%s 라운드가 목표 금액 %s을 달성하여 마감되었습니다.", c.Name, formatMoney(round.TargetAmount)),
			"investment_round", round.ID)
	} else {
		uc.notify(c.OwnerID, "investment_funded",
			"새 투자가 들어왔습니다",
			fmt.Sprintf("%s에 %s이 투자되었습니다. (누적 %s / 목표 %s)",
				c.Name, formatMoney(investAmount), formatMoney(newCurrentAmount), formatMoney(round.TargetAmount)),
			"investment_round", round.ID)
	}
	uc.notify(userID, "investment_received",
		"투자가 완료되었습니다",
		fmt.Sprintf("%s에 %s을 투자하여 %d주를 취득했습니다.", c.Name, formatMoney(investAmount), shares),
		"investment_round", round.ID)

	return inv, nil
}

// maybeAutoExpire flips an open round to `failed` if expires_at has passed.
// Returns the (possibly updated) round. No-op for non-open rounds.
func (uc *InvestmentUseCase) maybeAutoExpire(round *investment.InvestmentRound) (*investment.InvestmentRound, error) {
	if round.Status != investment.RoundOpen {
		return round, nil
	}
	if round.ExpiresAt == nil || !time.Now().After(*round.ExpiresAt) {
		return round, nil
	}
	if err := uc.repo.UpdateRoundStatus(round.ID, investment.RoundFailed); err != nil {
		return round, err
	}
	round.Status = investment.RoundFailed
	return round, nil
}

// GetRound fetches a single round by ID, auto-expiring if needed.
func (uc *InvestmentUseCase) GetRound(id int) (*investment.InvestmentRound, error) {
	round, err := uc.repo.FindRoundByID(id)
	if err != nil {
		return nil, err
	}
	return uc.maybeAutoExpire(round)
}

// =============================================================================
// Early close — owner closes a partially-funded round and accepts the actual
// capital raised. The company keeps what was paid, investors keep their shares,
// and the company's valuation is re-set using the agreed per-share price.
// =============================================================================

// CloseRoundEarly turns an open round with at least one investor into a
// `funded` state. Remaining un-sold shares are NOT minted. The post-close
// valuation is computed as `price_per_share × total_shares_after` which
// corresponds to the per-share price that investors actually paid.
func (uc *InvestmentUseCase) CloseRoundEarly(roundID, userID int) (*investment.InvestmentRound, error) {
	round, err := uc.repo.FindRoundByID(roundID)
	if err != nil {
		return nil, err
	}
	if round.Status != investment.RoundOpen {
		return nil, investment.ErrRoundNotOpen
	}
	c, err := uc.companyRepo.FindByID(round.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyNotFound
	}
	if c.OwnerID != userID {
		return nil, investment.ErrNotOwner
	}
	if round.SoldShares <= 0 {
		return nil, investment.ErrNoInvestors
	}

	// Mark as funded with the actual raised amount.
	if err := uc.repo.UpdateRoundFunded(round.ID, round.CurrentAmount); err != nil {
		return nil, err
	}

	// Re-price the company. Using price_per_share keeps every investor's
	// implicit valuation consistent with what they agreed to when buying.
	// (Avoids computing target/offered_percent which would overstate it —
	// only a fraction of the offered equity was actually taken.)
	newValuation := int(math.Round(round.PricePerShare * float64(c.TotalShares)))
	if err := uc.companyRepo.UpdateCapitalAndValuation(round.CompanyID, c.TotalCapital, newValuation); err != nil {
		return nil, err
	}
	if updated, err := uc.companyRepo.FindByID(round.CompanyID); err == nil && updated.CheckListing() && !updated.Listed {
		_ = uc.companyRepo.UpdateListed(round.CompanyID, true)
	}

	// Notify all shareholders and the owner.
	title := fmt.Sprintf("[%s] 투자 라운드 조기 마감", c.Name)
	body := fmt.Sprintf("목표 %s 중 %s 유치 후 라운드가 조기 마감되었습니다. 회사 가치가 %s로 재평가됐어요.",
		formatMoney(round.TargetAmount), formatMoney(round.CurrentAmount), formatMoney(newValuation))
	shareholders, _ := uc.companyRepo.FindShareholdersByCompanyID(round.CompanyID)
	for _, sh := range shareholders {
		if sh.Shares <= 0 {
			continue
		}
		uc.notify(sh.UserID, "investment_funded", title, body, "investment_round", round.ID)
	}

	return uc.repo.FindRoundByID(roundID)
}

// =============================================================================
// Cancel round with full refund. Undoes every investment:
//   - debits company wallet → credits each investor
//   - decrements each shareholder's shares (deletes row if zero)
//   - rolls back company.total_shares and total_capital
//   - marks round as 'cancelled'
// Valuation is NOT changed — the original pre-round valuation persists.
// =============================================================================

func (uc *InvestmentUseCase) CancelRoundAndRefund(roundID, userID int) (*investment.InvestmentRound, error) {
	round, err := uc.repo.FindRoundByID(roundID)
	if err != nil {
		return nil, err
	}
	if round.Status != investment.RoundOpen {
		return nil, investment.ErrRoundNotOpen
	}
	c, err := uc.companyRepo.FindByID(round.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyNotFound
	}
	if c.OwnerID != userID {
		return nil, investment.ErrNotOwner
	}

	// Guard: company must still have enough to refund everyone.
	companyWallet, err := uc.companyRepo.FindCompanyWallet(round.CompanyID)
	if err != nil {
		return nil, err
	}
	if companyWallet.Balance < round.CurrentAmount {
		return nil, investment.ErrCompanyFundsInsufficient
	}

	investments, err := uc.repo.ListByRound(round.ID)
	if err != nil {
		return nil, err
	}

	// Debit full refund pool from company wallet in one transaction record.
	if round.CurrentAmount > 0 {
		if err := uc.companyRepo.DebitCompanyWallet(companyWallet.ID, round.CurrentAmount,
			"investment_refund",
			fmt.Sprintf("라운드 #%d 취소 — 투자금 환불", round.ID),
			"investment_round", round.ID,
		); err != nil {
			return nil, err
		}
	}

	// Refund each investor and shrink their shareholdings.
	totalSharesReturned := 0
	for _, inv := range investments {
		w, err := uc.walletRepo.FindByUserID(inv.UserID)
		if err != nil {
			continue
		}
		if err := uc.walletRepo.Credit(w.ID, inv.Amount, wallet.TxInvestment,
			fmt.Sprintf("%s 라운드 취소 — 환불", c.Name),
			"investment_round", round.ID,
		); err != nil {
			continue
		}
		_ = uc.companyRepo.SubtractShareholderShares(round.CompanyID, inv.UserID, inv.Shares)
		totalSharesReturned += inv.Shares

		// Notify investor.
		uc.notify(inv.UserID, "investment_received",
			"투자금이 환불되었습니다",
			fmt.Sprintf("%s 라운드가 대표에 의해 취소되어 %s이 환불되었어요.", c.Name, formatMoney(inv.Amount)),
			"investment_round", round.ID,
		)
	}

	// Roll back company aggregates.
	newTotalShares := c.TotalShares - totalSharesReturned
	newTotalCapital := c.TotalCapital - round.CurrentAmount
	if newTotalShares < 0 {
		newTotalShares = 0
	}
	if newTotalCapital < 0 {
		newTotalCapital = 0
	}
	if err := uc.companyRepo.UpdateTotalShares(round.CompanyID, newTotalShares); err != nil {
		return nil, err
	}
	if err := uc.companyRepo.UpdateCapitalAndValuation(round.CompanyID, newTotalCapital, c.Valuation); err != nil {
		return nil, err
	}

	// Mark round as cancelled.
	if err := uc.repo.UpdateRoundStatus(round.ID, investment.RoundCancelled); err != nil {
		return nil, err
	}

	// Notify owner too for audit trail.
	uc.notify(c.OwnerID, "investment_funded",
		"라운드를 취소했습니다",
		fmt.Sprintf("%s 라운드 #%d — 투자자 %d명에게 총 %s 환불 완료.",
			c.Name, round.ID, len(investments), formatMoney(round.CurrentAmount)),
		"investment_round", round.ID,
	)

	return uc.repo.FindRoundByID(roundID)
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

// GetPortfolio returns the user's holdings aggregated per company, shaped for
// the frontend InvestPage (nested company object + derived profit / dividends).
func (uc *InvestmentUseCase) GetPortfolio(userID int) ([]*investment.PortfolioItem, error) {
	invs, err := uc.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	// Aggregate per company
	type agg struct {
		companyID int
		shares    int
		invested  int
	}
	aggs := make(map[int]*agg)
	for _, inv := range invs {
		a, ok := aggs[inv.CompanyID]
		if !ok {
			a = &agg{companyID: inv.CompanyID}
			aggs[inv.CompanyID] = a
		}
		a.shares += inv.Shares
		a.invested += inv.Amount
	}

	portfolio := make([]*investment.PortfolioItem, 0, len(aggs))
	for _, a := range aggs {
		c, err := uc.companyRepo.FindByID(a.companyID)
		if err != nil {
			continue
		}
		var pct float64
		var currentValue int
		if c.TotalShares > 0 {
			pct = float64(a.shares) / float64(c.TotalShares) * 100
			currentValue = int(float64(c.Valuation) * float64(a.shares) / float64(c.TotalShares))
		}
		dividendsReceived, _ := uc.repo.SumDividendsByUserAndCompany(userID, a.companyID)

		portfolio = append(portfolio, &investment.PortfolioItem{
			Company: investment.PortfolioCompany{
				ID:        c.ID,
				Name:      c.Name,
				Valuation: c.Valuation,
				LogoURL:   c.LogoURL,
			},
			TotalShares:       c.TotalShares,
			Shares:            a.shares,
			InvestedAmount:    a.invested,
			CurrentValue:      currentValue,
			Profit:            currentValue - a.invested,
			DividendsReceived: dividendsReceived,
			Percentage:        pct,
		})
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

// CreateKpiRule now requires the caller to be the company owner.
func (uc *InvestmentUseCase) CreateKpiRule(input CreateKpiRuleInput, userID int) (*investment.KpiRule, error) {
	c, err := uc.companyRepo.FindByID(input.CompanyID)
	if err != nil {
		return nil, investment.ErrCompanyNotFound
	}
	if c.OwnerID != userID {
		return nil, investment.ErrNotOwner
	}
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
