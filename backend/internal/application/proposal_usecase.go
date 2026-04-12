package application

import (
	"fmt"
	"time"

	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/domain/notification"
)

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

	p := &company.Proposal{
		CompanyID:     companyID,
		ProposerID:    userID,
		ProposalType:  proposalType,
		Title:         input.Title,
		Description:   input.Description,
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
