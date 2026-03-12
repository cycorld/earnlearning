package freelance

import "time"

type JobStatus string

const (
	StatusOpen       JobStatus = "open"
	StatusInProgress JobStatus = "in_progress"
	StatusCompleted  JobStatus = "completed"
	StatusDisputed   JobStatus = "disputed"
	StatusCancelled  JobStatus = "cancelled"
)

type ApplicationStatus string

const (
	AppPending  ApplicationStatus = "pending"
	AppAccepted ApplicationStatus = "accepted"
	AppRejected ApplicationStatus = "rejected"
)

type FreelanceJob struct {
	ID             int        `json:"id"`
	ClientID       int        `json:"client_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Budget         int        `json:"budget"`
	Deadline       *time.Time `json:"deadline"`
	RequiredSkills string     `json:"required_skills"`
	Status         JobStatus  `json:"status"`
	FreelancerID   *int       `json:"freelancer_id"`
	EscrowAmount   int        `json:"escrow_amount"`
	AgreedPrice    int        `json:"agreed_price"`
	WorkCompleted  bool       `json:"work_completed"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`

	// Joined fields
	ClientName     string            `json:"client_name,omitempty"`
	FreelancerName string            `json:"freelancer_name,omitempty"`
	Applications   []*JobApplication `json:"applications,omitempty"`
}

type JobApplication struct {
	ID        int               `json:"id"`
	JobID     int               `json:"job_id"`
	UserID    int               `json:"user_id"`
	Proposal  string            `json:"proposal"`
	Price     int               `json:"price"`
	Status    ApplicationStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`

	// Joined fields
	UserName string `json:"user_name,omitempty"`
}

type FreelanceReview struct {
	ID         int       `json:"id"`
	JobID      int       `json:"job_id"`
	ReviewerID int       `json:"reviewer_id"`
	RevieweeID int       `json:"reviewee_id"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`

	// Joined fields
	ReviewerName string `json:"reviewer_name,omitempty"`
}
