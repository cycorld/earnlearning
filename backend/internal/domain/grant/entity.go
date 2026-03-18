package grant

import "time"

type GrantStatus string

const (
	StatusOpen   GrantStatus = "open"
	StatusClosed GrantStatus = "closed"
)

type ApplicationStatus string

const (
	AppPending  ApplicationStatus = "pending"
	AppApproved ApplicationStatus = "approved"
	AppRejected ApplicationStatus = "rejected"
)

// UserRef is a nested user reference for JSON output.
type UserRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Grant struct {
	ID            int         `json:"id"`
	AdminID       int         `json:"-"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Reward        int         `json:"reward"`
	MaxApplicants int         `json:"max_applicants"` // 0 = unlimited
	Status        GrantStatus `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`

	// Nested references for JSON output
	Admin            *UserRef             `json:"admin,omitempty"`
	Applications     []*GrantApplication  `json:"applications,omitempty"`
	ApplicationCount *int                 `json:"application_count,omitempty"`
	ApprovedCount    *int                 `json:"approved_count,omitempty"`

	// Internal
	AdminName string `json:"-"`
}

type GrantApplication struct {
	ID        int               `json:"id"`
	GrantID   int               `json:"grant_id"`
	UserID    int               `json:"-"`
	Proposal  string            `json:"proposal"`
	Status    ApplicationStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`

	// Nested reference
	User     *UserRef `json:"user,omitempty"`
	UserName string   `json:"-"`
}
