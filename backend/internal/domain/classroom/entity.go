package classroom

import "time"

type Classroom struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	CreatedBy      int       `json:"created_by"`
	InitialCapital int       `json:"initial_capital"`
	Settings       string    `json:"settings"`
	CreatedAt      time.Time `json:"created_at"`
}

type ClassroomMember struct {
	ID          int       `json:"id"`
	ClassroomID int       `json:"classroom_id"`
	UserID      int       `json:"user_id"`
	JoinedAt    time.Time `json:"joined_at"`
}

// MemberDashboard is an enriched member view for admin dashboard.
type MemberDashboard struct {
	UserID         int       `json:"user_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	StudentID      string    `json:"student_id"`
	Department     string    `json:"department"`
	AvatarURL      string    `json:"avatar_url"`
	Status         string    `json:"status"`
	JoinedAt       time.Time `json:"joined_at"`
	Balance        int       `json:"balance"`
	TotalAsset     int       `json:"total_asset"`
	CompanyCount   int       `json:"company_count"`
	LoanCount      int       `json:"loan_count"`
	TotalDebt      int       `json:"total_debt"`
	PostCount      int       `json:"post_count"`
	CompanyNames   string    `json:"company_names"`
}

type Channel struct {
	ID          int    `json:"id"`
	ClassroomID int    `json:"classroom_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ChannelType string `json:"channel_type"`
	WriteRole   string `json:"write_role"`
	SortOrder   int    `json:"sort_order"`
}
