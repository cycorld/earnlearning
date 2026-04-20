package proposal

import "time"

type Category string

const (
	CategoryFeature Category = "feature"
	CategoryBug     Category = "bug"
	CategoryGeneral Category = "general"
)

func IsValidCategory(c string) bool {
	switch Category(c) {
	case CategoryFeature, CategoryBug, CategoryGeneral:
		return true
	}
	return false
}

type Status string

const (
	StatusOpen      Status = "open"
	StatusReviewing Status = "reviewing"
	StatusResolved  Status = "resolved"
	StatusWontfix   Status = "wontfix"
)

func IsValidStatus(s string) bool {
	switch Status(s) {
	case StatusOpen, StatusReviewing, StatusResolved, StatusWontfix:
		return true
	}
	return false
}

// UserRef — admin 페이지에서 누가 제출했는지 노출.
type UserRef struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	StudentID  string `json:"student_id"`
	Department string `json:"department"`
}

type Proposal struct {
	ID          int       `json:"id"`
	UserID      int       `json:"-"`
	Category    Category  `json:"category"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Attachments []string  `json:"attachments"` // /uploads/xxx.png URLs
	Status      Status    `json:"status"`
	AdminNote   string    `json:"admin_note"`
	TicketLink  string    `json:"ticket_link"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	User *UserRef `json:"user,omitempty"`
}

type Filter struct {
	Status   string
	Category string
	UserID   int // 0 = all
	Limit    int
	Offset   int
}
