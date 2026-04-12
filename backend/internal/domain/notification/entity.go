package notification

import "time"

type NotifType string

const (
	NotifInvestmentFunded  NotifType = "investment_funded"
	NotifInvestmentReceived NotifType = "investment_received"
	NotifDividendReceived  NotifType = "dividend_received"
	NotifJobAccepted       NotifType = "job_accepted"
	NotifJobWorkDone       NotifType = "job_work_done"
	NotifJobCompleted      NotifType = "job_completed"
	NotifLoanApproved      NotifType = "loan_approved"
	NotifLoanOverdue       NotifType = "loan_overdue"
	NotifStockTrade        NotifType = "stock_trade"
	NotifAdminTransfer     NotifType = "admin_transfer"
	NotifAssignmentGraded  NotifType = "assignment_graded"
	NotifUserApproved      NotifType = "user_approved"
	NotifNewAssignment     NotifType = "new_assignment"
	NotifKpiRevenue        NotifType = "kpi_revenue"

	// Comment
	NotifNewComment        NotifType = "new_comment"

	// Freelance
	NotifJobApplied        NotifType = "job_applied"
	NotifJobDisputed       NotifType = "job_disputed"
	NotifJobCancelled      NotifType = "job_cancelled"

	// Grant
	NotifGrantApproved     NotifType = "grant_approved"
	NotifGrantApplied      NotifType = "grant_applied"
	NotifGrantClosed       NotifType = "grant_closed"

	// Disclosure
	NotifDisclosureApproved NotifType = "disclosure_approved"
	NotifDisclosureRejected NotifType = "disclosure_rejected"

	// Shareholder proposal (주주총회)
	NotifProposalStarted  NotifType = "proposal_started"
	NotifProposalClosed   NotifType = "proposal_closed"
	NotifLiquidationPayout NotifType = "liquidation_payout"

	// DM
	NotifNewDM             NotifType = "new_dm"
)

// PushEligibleTypes are notification types that should trigger push notifications.
var PushEligibleTypes = map[NotifType]bool{
	NotifInvestmentFunded:  true,
	NotifInvestmentReceived: true,
	NotifDividendReceived:  true,
	NotifJobAccepted:       true,
	NotifJobWorkDone:       true,
	NotifJobCompleted:      true,
	NotifLoanApproved:      true,
	NotifLoanOverdue:       true,
	NotifStockTrade:        true,
	NotifAdminTransfer:     true,
	NotifAssignmentGraded:  true,
	NotifUserApproved:      true,
	NotifNewAssignment:     true,
	NotifKpiRevenue:        true,
	NotifNewComment:        true,
	NotifJobApplied:        true,
	NotifJobDisputed:       true,
	NotifJobCancelled:      true,
	NotifGrantApproved:     true,
	NotifGrantApplied:      true,
	NotifGrantClosed:       true,
	NotifNewDM:             true,
	NotifDisclosureApproved: true,
	NotifDisclosureRejected: true,
	NotifProposalStarted:    true,
	NotifProposalClosed:     true,
	NotifLiquidationPayout:  true,
}

type Notification struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	NotifType     NotifType `json:"notif_type"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   int       `json:"reference_id"`
	IsRead        bool      `json:"is_read"`
	CreatedAt     time.Time `json:"created_at"`
}

type PushSubscription struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// EmailPreference stores per-user email notification settings.
// All fields default to true (email notifications enabled by default).
type EmailPreference struct {
	UserID       int  `json:"user_id"`
	EmailEnabled bool `json:"email_enabled"` // master switch
}
