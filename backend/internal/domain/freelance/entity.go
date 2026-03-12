package freelance

import (
	"encoding/json"
	"strings"
	"time"
)

// SkillsList is a comma-separated string that serializes as a JSON array.
type SkillsList string

func (s SkillsList) MarshalJSON() ([]byte, error) {
	str := string(s)
	if str == "" {
		return json.Marshal([]string{})
	}
	parts := strings.Split(str, ",")
	trimmed := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			trimmed = append(trimmed, p)
		}
	}
	return json.Marshal(trimmed)
}

func (s *SkillsList) UnmarshalJSON(data []byte) error {
	// Try array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = SkillsList(strings.Join(arr, ","))
		return nil
	}
	// Fallback to string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = SkillsList(str)
	return nil
}

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

// UserRef is a nested user reference for JSON output.
type UserRef struct {
	ID   int     `json:"id"`
	Name string  `json:"name"`
	Rating *float64 `json:"rating,omitempty"`
}

type FreelanceJob struct {
	ID             int        `json:"id"`
	ClientID       int        `json:"-"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Budget         int        `json:"budget"`
	Deadline       *time.Time `json:"deadline"`
	RequiredSkills SkillsList `json:"required_skills"`
	Status         JobStatus  `json:"status"`
	FreelancerID   *int       `json:"freelancer_id"`
	EscrowAmount   int        `json:"escrow_amount"`
	AgreedPrice    int        `json:"agreed_price"`
	WorkCompleted  bool       `json:"work_completed"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`

	// Nested references for JSON output
	Client           *UserRef          `json:"client,omitempty"`
	Applications     []*JobApplication `json:"applications,omitempty"`
	ApplicationCount *int              `json:"application_count,omitempty"`

	// Internal joined fields (not serialized directly)
	ClientName     string `json:"-"`
	FreelancerName string `json:"freelancer_name,omitempty"`
}

type JobApplication struct {
	ID        int               `json:"id"`
	JobID     int               `json:"job_id"`
	UserID    int               `json:"-"`
	Proposal  string            `json:"proposal"`
	Price     int               `json:"price"`
	Status    ApplicationStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`

	// Nested reference for JSON output
	User *UserRef `json:"user,omitempty"`

	// Internal joined field (not serialized directly)
	UserName string `json:"-"`
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
