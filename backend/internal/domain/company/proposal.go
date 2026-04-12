package company

import "time"

// Proposal represents a shareholder meeting proposal (주주총회 안건).
type Proposal struct {
	ID             int       `json:"id"`
	CompanyID      int       `json:"company_id"`
	ProposerID     int       `json:"proposer_id"`
	ProposalType   string    `json:"proposal_type"` // 'general', 'liquidation'
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	PassThreshold  int       `json:"pass_threshold"` // percentage (1-100). e.g., 70 for liquidation.
	Status         string    `json:"status"`         // 'active', 'passed', 'rejected', 'cancelled', 'executed'
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	ResultNote     string    `json:"result_note"`
	CreatedAt      time.Time `json:"created_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
}

// Vote represents a single shareholder's vote on a proposal.
type Vote struct {
	ID            int       `json:"id"`
	ProposalID    int       `json:"proposal_id"`
	UserID        int       `json:"user_id"`
	Choice        string    `json:"choice"`         // 'yes' or 'no'
	SharesAtVote  int       `json:"shares_at_vote"` // snapshot of user's shares when voting
	CreatedAt     time.Time `json:"created_at"`
}

// Valid proposal types.
const (
	ProposalTypeGeneral     = "general"
	ProposalTypeLiquidation = "liquidation"
)

// Valid proposal statuses.
const (
	ProposalStatusActive    = "active"
	ProposalStatusPassed    = "passed"
	ProposalStatusRejected  = "rejected"
	ProposalStatusCancelled = "cancelled"
	ProposalStatusExecuted  = "executed"
)

// Valid vote choices.
const (
	VoteChoiceYes = "yes"
	VoteChoiceNo  = "no"
)

// ProposalTally holds vote aggregation for a proposal.
type ProposalTally struct {
	YesShares   int     `json:"yes_shares"`
	NoShares    int     `json:"no_shares"`
	TotalShares int     `json:"total_shares"`
	YesPercent  float64 `json:"yes_percent"`
	NoPercent   float64 `json:"no_percent"`
	// Projected outcome based on current tally and threshold.
	// 'passed' if yes_percent >= threshold
	// 'rejected' if no_percent > (100 - threshold) -- impossible to pass
	// 'pending'  otherwise
	ProjectedStatus string `json:"projected_status"`
}
