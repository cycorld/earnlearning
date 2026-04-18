package application

import (
	"fmt"
	"time"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

// LiquidationTaxPercent is the portion of company assets consumed as tax during
// liquidation (#023). The remaining 80% is distributed to shareholders.
const LiquidationTaxPercent = 20

// liquidationTaxNotice is prepended to every liquidation proposal's description
// so voters always see the tax + distribution policy before casting a vote (#033).
const liquidationTaxNotice = "⚠️ **청산 안내**: 가결 시 회사 자산의 **20%는 세금**으로 납부되고, 나머지 **80%가 주주별 지분율에 따라 자동 분배**됩니다. 가결 즉시 집행되며 회사는 영구 정지됩니다.\n\n---\n\n"

// =============================================================================
// Shareholder proposal (주주총회 안건) & voting usecase
// These methods live on CompanyUsecase to reuse its wiring (repos + notifUC).
// =============================================================================

type CreateProposalInput struct {
	ProposalType  string `json:"proposal_type"`  // 'general' | 'liquidation' (default 'general')
	Title         string `json:"title"`
	Description   string `json:"description"`
	PassThreshold int    `json:"pass_threshold"` // 1-100; default depends on type
	// DurationDays is how many days the voting stays open. Default 7.
	DurationDays int `json:"duration_days"`
}

// ProposalDetail bundles a proposal with tally + voter info for the frontend.
type ProposalDetail struct {
	*company.Proposal
	CompanyName  string                `json:"company_name"`
	ProposerName string                `json:"proposer_name"`
	Tally        company.ProposalTally `json:"tally"`
	MyVote       *company.Vote         `json:"my_vote,omitempty"`
	Votes        []*VoteDetail         `json:"votes,omitempty"`
}

type VoteDetail struct {
	*company.Vote
	UserName string `json:"user_name"`
}

// CreateProposal creates a new shareholder proposal. Only shareholders of the
// company can create proposals. Typically the owner, but any shareholder is
// allowed so the owner cannot block e.g. liquidation votes.
func (uc *CompanyUsecase) CreateProposal(companyID, userID int, input CreateProposalInput) (*company.Proposal, error) {
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, err
	}
	if c.Status != "active" {
		return nil, fmt.Errorf("활성 상태가 아닌 회사에는 안건을 상정할 수 없습니다")
	}

	// Proposer must be a shareholder
	sh, err := uc.companyRepo.FindShareholder(companyID, userID)
	if err != nil || sh == nil || sh.Shares <= 0 {
		return nil, company.ErrNotShareholder
	}

	if input.Title == "" {
		return nil, fmt.Errorf("안건 제목을 입력해주세요")
	}

	proposalType := input.ProposalType
	if proposalType == "" {
		proposalType = company.ProposalTypeGeneral
	}
	if proposalType != company.ProposalTypeGeneral && proposalType != company.ProposalTypeLiquidation {
		return nil, fmt.Errorf("지원하지 않는 안건 종류입니다: %s", proposalType)
	}

	threshold := input.PassThreshold
	if threshold <= 0 {
		// defaults per type
		if proposalType == company.ProposalTypeLiquidation {
			threshold = 70
		} else {
			threshold = 50
		}
	}
	if threshold < 1 || threshold > 100 {
		return nil, fmt.Errorf("가결 기준(%%)은 1~100 사이여야 합니다")
	}

	// Only one active proposal of the same type at a time
	if existing, err := uc.companyRepo.FindActiveProposalByCompanyAndType(companyID, proposalType); err == nil && existing != nil {
		return nil, fmt.Errorf("이미 진행 중인 %s 안건이 있습니다", proposalType)
	}

	duration := input.DurationDays
	if duration <= 0 {
		duration = 7
	}
	now := time.Now()
	end := now.AddDate(0, 0, duration)

	description := input.Description
	if proposalType == company.ProposalTypeLiquidation {
		description = liquidationTaxNotice + description
	}

	p := &company.Proposal{
		CompanyID:     companyID,
		ProposerID:    userID,
		ProposalType:  proposalType,
		Title:         input.Title,
		Description:   description,
		PassThreshold: threshold,
		Status:        company.ProposalStatusActive,
		StartDate:     now,
		EndDate:       end,
	}
	id, err := uc.companyRepo.CreateProposal(p)
	if err != nil {
		return nil, fmt.Errorf("안건 생성 실패: %w", err)
	}
	p.ID = id

	// Notify all shareholders that a new proposal has been opened
	if uc.notifUC != nil {
		shareholders, _ := uc.companyRepo.FindShareholdersByCompanyID(companyID)
		title := fmt.Sprintf("[%s] 새 주주총회 안건", c.Name)
		body := fmt.Sprintf("%s (가결 기준 %d%%)", p.Title, p.PassThreshold)
		for _, s := range shareholders {
			if s.Shares <= 0 {
				continue
			}
			_ = uc.notifUC.CreateNotification(
				s.UserID,
				notification.NotifProposalStarted,
				title, body, "proposal", p.ID,
			)
		}
	}

	return p, nil
}

// CastVote records a shareholder vote. Shares-at-vote are snapshotted from
// current shareholder record.
func (uc *CompanyUsecase) CastVote(proposalID, userID int, choice string) (*company.Vote, error) {
	if choice != company.VoteChoiceYes && choice != company.VoteChoiceNo {
		return nil, fmt.Errorf("choice는 'yes' 또는 'no' 이어야 합니다")
	}

	p, err := uc.companyRepo.FindProposalByID(proposalID)
	if err != nil {
		return nil, err
	}
	if p.Status != company.ProposalStatusActive {
		return nil, company.ErrProposalClosed
	}
	if time.Now().After(p.EndDate) {
		// Auto-close expired proposal on write access
		_, _ = uc.closeProposal(p)
		return nil, company.ErrProposalClosed
	}

	sh, err := uc.companyRepo.FindShareholder(p.CompanyID, userID)
	if err != nil || sh == nil || sh.Shares <= 0 {
		return nil, company.ErrNotShareholder
	}

	v := &company.Vote{
		ProposalID:   proposalID,
		UserID:       userID,
		Choice:       choice,
		SharesAtVote: sh.Shares,
	}
	id, err := uc.companyRepo.CreateVote(v)
	if err != nil {
		return nil, err
	}
	v.ID = id

	// After a vote, re-tally. If threshold is already reached (or impossible),
	// auto-close so the result is final.
	_, _ = uc.maybeAutoClose(p)

	return v, nil
}

// GetProposalsByCompanyID returns all proposals for a company, enriched with
// tally + proposer name. If viewerUserID > 0, also attaches MyVote.
func (uc *CompanyUsecase) GetProposalsByCompanyID(companyID, viewerUserID int) ([]*ProposalDetail, error) {
	proposals, err := uc.companyRepo.FindProposalsByCompanyID(companyID)
	if err != nil {
		return nil, err
	}
	c, err := uc.companyRepo.FindByID(companyID)
	if err != nil {
		return nil, err
	}

	details := make([]*ProposalDetail, 0, len(proposals))
	for _, p := range proposals {
		// Auto-close expired proposals on read too
		if p.Status == company.ProposalStatusActive && time.Now().After(p.EndDate) {
			if closed, err := uc.closeProposal(p); err == nil {
				p = closed
			}
		}
		tally, _ := uc.tallyProposal(p, c)

		proposerName := "알 수 없음"
		if u, err := uc.userRepo.FindByID(p.ProposerID); err == nil {
			proposerName = u.Name
		}

		var myVote *company.Vote
		if viewerUserID > 0 {
			v, _ := uc.companyRepo.FindVote(p.ID, viewerUserID)
			myVote = v
		}

		details = append(details, &ProposalDetail{
			Proposal:     p,
			CompanyName:  c.Name,
			ProposerName: proposerName,
			Tally:        tally,
			MyVote:       myVote,
		})
	}
	return details, nil
}

// GetProposal returns a single proposal with tally + full vote list.
func (uc *CompanyUsecase) GetProposal(proposalID, viewerUserID int) (*ProposalDetail, error) {
	p, err := uc.companyRepo.FindProposalByID(proposalID)
	if err != nil {
		return nil, err
	}
	if p.Status == company.ProposalStatusActive && time.Now().After(p.EndDate) {
		if closed, err := uc.closeProposal(p); err == nil {
			p = closed
		}
	}

	c, err := uc.companyRepo.FindByID(p.CompanyID)
	if err != nil {
		return nil, err
	}
	tally, _ := uc.tallyProposal(p, c)

	votes, _ := uc.companyRepo.FindVotesByProposalID(p.ID)
	voteDetails := make([]*VoteDetail, 0, len(votes))
	for _, v := range votes {
		name := "알 수 없음"
		if u, err := uc.userRepo.FindByID(v.UserID); err == nil {
			name = u.Name
		}
		voteDetails = append(voteDetails, &VoteDetail{Vote: v, UserName: name})
	}

	proposerName := "알 수 없음"
	if u, err := uc.userRepo.FindByID(p.ProposerID); err == nil {
		proposerName = u.Name
	}

	var myVote *company.Vote
	if viewerUserID > 0 {
		myVote, _ = uc.companyRepo.FindVote(p.ID, viewerUserID)
	}

	return &ProposalDetail{
		Proposal:     p,
		CompanyName:  c.Name,
		ProposerName: proposerName,
		Tally:        tally,
		MyVote:       myVote,
		Votes:        voteDetails,
	}, nil
}

// CancelProposal lets the proposer cancel their own active proposal.
func (uc *CompanyUsecase) CancelProposal(proposalID, userID int) error {
	p, err := uc.companyRepo.FindProposalByID(proposalID)
	if err != nil {
		return err
	}
	if p.ProposerID != userID {
		return fmt.Errorf("안건 상정자만 취소할 수 있습니다")
	}
	if p.Status != company.ProposalStatusActive {
		return company.ErrProposalClosed
	}
	now := time.Now()
	return uc.companyRepo.UpdateProposalStatus(p.ID, company.ProposalStatusCancelled, "상정자가 취소함", &now)
}

// =============================================================================
// Internal helpers
// =============================================================================

// tallyProposal computes vote tally and projected outcome.
func (uc *CompanyUsecase) tallyProposal(p *company.Proposal, c *company.Company) (company.ProposalTally, error) {
	votes, err := uc.companyRepo.FindVotesByProposalID(p.ID)
	if err != nil {
		return company.ProposalTally{}, err
	}

	yes := 0
	no := 0
	for _, v := range votes {
		switch v.Choice {
		case company.VoteChoiceYes:
			yes += v.SharesAtVote
		case company.VoteChoiceNo:
			no += v.SharesAtVote
		}
	}

	total := c.TotalShares
	if total <= 0 {
		total = 1
	}
	yesPct := float64(yes) / float64(total) * 100
	noPct := float64(no) / float64(total) * 100

	projected := "pending"
	if yesPct >= float64(p.PassThreshold) {
		projected = company.ProposalStatusPassed
	} else if noPct > float64(100-p.PassThreshold) {
		// Even if all remaining shares voted yes, threshold can't be reached
		projected = company.ProposalStatusRejected
	}

	return company.ProposalTally{
		YesShares:       yes,
		NoShares:        no,
		TotalShares:     total,
		YesPercent:      yesPct,
		NoPercent:       noPct,
		ProjectedStatus: projected,
	}, nil
}

// maybeAutoClose closes the proposal if the current tally is final
// (passed or impossible-to-pass) OR the end date has passed.
func (uc *CompanyUsecase) maybeAutoClose(p *company.Proposal) (*company.Proposal, error) {
	c, err := uc.companyRepo.FindByID(p.CompanyID)
	if err != nil {
		return p, err
	}
	tally, err := uc.tallyProposal(p, c)
	if err != nil {
		return p, err
	}
	if tally.ProjectedStatus == "pending" && !time.Now().After(p.EndDate) {
		return p, nil
	}
	return uc.closeProposal(p)
}

// closeProposal finalizes a proposal and sends notifications to shareholders.
// Idempotent — if already closed, returns as-is.
func (uc *CompanyUsecase) closeProposal(p *company.Proposal) (*company.Proposal, error) {
	if p.Status != company.ProposalStatusActive {
		return p, nil
	}
	c, err := uc.companyRepo.FindByID(p.CompanyID)
	if err != nil {
		return p, err
	}
	tally, err := uc.tallyProposal(p, c)
	if err != nil {
		return p, err
	}

	finalStatus := company.ProposalStatusRejected
	resultNote := fmt.Sprintf("기간 만료 (찬성 %.1f%% / 기준 %d%%)", tally.YesPercent, p.PassThreshold)
	if tally.ProjectedStatus == company.ProposalStatusPassed || tally.YesPercent >= float64(p.PassThreshold) {
		finalStatus = company.ProposalStatusPassed
		resultNote = fmt.Sprintf("가결 (찬성 %.1f%% / 기준 %d%%)", tally.YesPercent, p.PassThreshold)
	} else if tally.ProjectedStatus == company.ProposalStatusRejected {
		resultNote = fmt.Sprintf("부결 확정 (반대 %.1f%%)", tally.NoPercent)
	}

	now := time.Now()
	if err := uc.companyRepo.UpdateProposalStatus(p.ID, finalStatus, resultNote, &now); err != nil {
		return p, err
	}
	p.Status = finalStatus
	p.ResultNote = resultNote
	p.ClosedAt = &now

	// #033: auto-execute passed liquidation proposals so shareholders don't
	// have to wait for the owner to manually trigger distribution.
	if finalStatus == company.ProposalStatusPassed &&
		p.ProposalType == company.ProposalTypeLiquidation &&
		c.Status == "active" {
		if _, err := uc.executeLiquidationCore(p, c, p.ProposerID); err != nil {
			// Do not roll back the pass status — log and let someone retry via
			// the manual /execute endpoint.
			fmt.Printf("[liquidation] auto-execute failed for proposal %d: %v\n", p.ID, err)
		} else {
			// Reload to reflect executed status for the caller.
			if refreshed, err := uc.companyRepo.FindProposalByID(p.ID); err == nil {
				p = refreshed
			}
			// Close-outcome notification is subsumed by per-shareholder
			// payout notifications emitted inside executeLiquidationCore.
			return p, nil
		}
	}

	// Notify shareholders of the outcome
	if uc.notifUC != nil {
		shareholders, _ := uc.companyRepo.FindShareholdersByCompanyID(p.CompanyID)
		title := fmt.Sprintf("[%s] 주주총회 결과", c.Name)
		outcome := "가결"
		if finalStatus == company.ProposalStatusRejected {
			outcome = "부결"
		}
		body := fmt.Sprintf("%s — %s (%s)", p.Title, outcome, resultNote)
		for _, s := range shareholders {
			if s.Shares <= 0 {
				continue
			}
			_ = uc.notifUC.CreateNotification(
				s.UserID,
				notification.NotifProposalClosed,
				title, body, "proposal", p.ID,
			)
		}
	}

	return p, nil
}

// =============================================================================
// Company liquidation execution (#023)
// =============================================================================

// LiquidationPayout describes what each shareholder received during liquidation.
type LiquidationPayout struct {
	UserID   int    `json:"user_id"`
	UserName string `json:"user_name"`
	Shares   int    `json:"shares"`
	Amount   int    `json:"amount"`
}

// LiquidationResult summarizes the distribution of a dissolved company's assets.
type LiquidationResult struct {
	CompanyID      int                 `json:"company_id"`
	CompanyName    string              `json:"company_name"`
	TotalBalance   int                 `json:"total_balance"`
	Tax            int                 `json:"tax"`
	Distributable  int                 `json:"distributable"`
	Payouts        []LiquidationPayout `json:"payouts"`
	ResidualTax    int                 `json:"residual_tax"` // extra tax from rounding
	ExecutedAt     time.Time           `json:"executed_at"`
}

// ExecuteLiquidation consumes a passed liquidation proposal and distributes
// the company's assets. Any user can trigger execution once a proposal passes,
// but typically the company owner is the one to call it.
//
// Steps:
//  1. Validate proposal: type=liquidation, status=passed
//  2. Compute tax (20% of wallet balance) and per-shareholder payouts
//  3. Debit full balance from company wallet
//  4. Credit each shareholder's personal wallet with their share
//  5. Mark company as 'dissolved'
//  6. Mark proposal as 'executed'
//  7. Create disclosure summarizing the liquidation
//  8. Notify all shareholders with their individual payout amounts
func (uc *CompanyUsecase) ExecuteLiquidation(proposalID, userID int) (*LiquidationResult, error) {
	p, err := uc.companyRepo.FindProposalByID(proposalID)
	if err != nil {
		return nil, err
	}
	if p.ProposalType != company.ProposalTypeLiquidation {
		return nil, fmt.Errorf("청산 안건이 아닙니다")
	}

	c, err := uc.companyRepo.FindByID(p.CompanyID)
	if err != nil {
		return nil, err
	}

	// Permission check runs before status checks so non-shareholders get a
	// clear 'not shareholder' error regardless of proposal state.
	trigger, err := uc.companyRepo.FindShareholder(c.ID, userID)
	if err != nil || trigger == nil || trigger.Shares <= 0 {
		return nil, company.ErrNotShareholder
	}

	if p.Status != company.ProposalStatusPassed {
		return nil, fmt.Errorf("가결된 청산 안건만 집행할 수 있습니다 (현재 상태: %s)", p.Status)
	}
	if c.Status == "dissolved" {
		return nil, fmt.Errorf("이미 청산된 회사입니다")
	}

	return uc.executeLiquidationCore(p, c, userID)
}

// executeLiquidationCore performs the actual distribution + state changes.
// Assumes caller has already validated proposal type/status, company status,
// and any required permission checks. Used by both ExecuteLiquidation (manual
// API trigger) and closeProposal (automatic on vote pass, #033).
func (uc *CompanyUsecase) executeLiquidationCore(p *company.Proposal, c *company.Company, actorID int) (*LiquidationResult, error) {
	// Wallet balance
	cw, err := uc.companyRepo.FindCompanyWallet(c.ID)
	if err != nil {
		return nil, fmt.Errorf("회사 지갑 조회 실패: %w", err)
	}
	balance := cw.Balance

	// Tax (integer division rounds down, remainder is burned as residual tax)
	tax := balance * LiquidationTaxPercent / 100
	distributable := balance - tax

	// Compute per-shareholder payouts
	shareholders, err := uc.companyRepo.FindShareholdersByCompanyID(c.ID)
	if err != nil {
		return nil, fmt.Errorf("주주 조회 실패: %w", err)
	}

	totalShares := c.TotalShares
	if totalShares <= 0 {
		// Fallback: sum of shares
		for _, s := range shareholders {
			totalShares += s.Shares
		}
	}

	payouts := make([]LiquidationPayout, 0, len(shareholders))
	totalPaid := 0
	for _, s := range shareholders {
		if s.Shares <= 0 {
			continue
		}
		amount := distributable * s.Shares / totalShares
		if amount <= 0 {
			continue
		}
		name := "알 수 없음"
		if u, err := uc.userRepo.FindByID(s.UserID); err == nil {
			name = u.Name
		}
		payouts = append(payouts, LiquidationPayout{
			UserID:   s.UserID,
			UserName: name,
			Shares:   s.Shares,
			Amount:   amount,
		})
		totalPaid += amount
	}

	// Rounding residual goes to tax
	residualTax := distributable - totalPaid
	if residualTax < 0 {
		residualTax = 0
	}
	tax += residualTax

	// Debit full balance from company wallet (this will also log a tx)
	if balance > 0 {
		if err := uc.companyRepo.DebitCompanyWallet(
			cw.ID, balance,
			string(wallet.TxLiquidationTax),
			fmt.Sprintf("청산 집행 (세금 %d원 + 주주 분배 %d원)", tax, totalPaid),
			"proposal", p.ID,
		); err != nil {
			return nil, fmt.Errorf("회사 잔액 차감 실패: %w", err)
		}
	}

	// Credit each shareholder's personal wallet
	for _, payout := range payouts {
		w, err := uc.walletRepo.FindByUserID(payout.UserID)
		if err != nil {
			// This shouldn't happen for approved users but we log + continue
			continue
		}
		desc := fmt.Sprintf("[%s] 청산 분배금 (지분 %d주)", c.Name, payout.Shares)
		if err := uc.walletRepo.Credit(w.ID, payout.Amount, wallet.TxLiquidationPayout, desc, "proposal", p.ID); err != nil {
			return nil, fmt.Errorf("주주 %d 분배금 지급 실패: %w", payout.UserID, err)
		}
	}

	// Mark company as dissolved
	if err := uc.companyRepo.UpdateStatus(c.ID, "dissolved"); err != nil {
		return nil, fmt.Errorf("회사 상태 업데이트 실패: %w", err)
	}

	// Mark proposal as executed
	now := time.Now()
	if err := uc.companyRepo.UpdateProposalStatus(p.ID, company.ProposalStatusExecuted,
		fmt.Sprintf("청산 집행 완료 (세금 %d원, 분배 %d원)", tax, totalPaid),
		&now,
	); err != nil {
		return nil, fmt.Errorf("안건 상태 업데이트 실패: %w", err)
	}

	// Create a disclosure for the public record
	disclosureContent := fmt.Sprintf(
		"## 회사 청산 공시\n\n"+
			"**회사명**: %s\n"+
			"**총 자산**: %d원\n"+
			"**세금 (20%%)**: %d원\n"+
			"**주주 분배 총액**: %d원\n\n"+
			"### 주주별 분배 내역\n\n",
		c.Name, balance, tax, totalPaid,
	)
	for _, payout := range payouts {
		disclosureContent += fmt.Sprintf("- %s: %d주 → %d원\n", payout.UserName, payout.Shares, payout.Amount)
	}
	_, _ = uc.companyRepo.CreateDisclosure(&company.Disclosure{
		CompanyID:  c.ID,
		AuthorID:   actorID,
		Content:    disclosureContent,
		PeriodFrom: now.Format("2006-01-02"),
		PeriodTo:   now.Format("2006-01-02"),
		Status:     "approved", // auto-approved, system-generated
	})

	// Notify each shareholder with their payout
	if uc.notifUC != nil {
		for _, payout := range payouts {
			title := fmt.Sprintf("[%s] 청산 분배금 입금", c.Name)
			body := fmt.Sprintf("청산 분배금 %d원이 지갑에 입금되었습니다. (지분 %d주)", payout.Amount, payout.Shares)
			_ = uc.notifUC.CreateNotification(
				payout.UserID,
				notification.NotifLiquidationPayout,
				title, body, "wallet", 0,
			)
		}
	}

	return &LiquidationResult{
		CompanyID:     c.ID,
		CompanyName:   c.Name,
		TotalBalance:  balance,
		Tax:           tax,
		Distributable: distributable,
		Payouts:       payouts,
		ResidualTax:   residualTax,
		ExecutedAt:    now,
	}, nil
}
