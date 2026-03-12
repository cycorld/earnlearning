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

type Channel struct {
	ID          int    `json:"id"`
	ClassroomID int    `json:"classroom_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ChannelType string `json:"channel_type"`
	WriteRole   string `json:"write_role"`
	SortOrder   int    `json:"sort_order"`
}
