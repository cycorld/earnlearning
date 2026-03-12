package user

import "time"

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleStudent Role = "student"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type User struct {
	ID         int       `json:"id"`
	Email      string    `json:"email"`
	Password   string    `json:"-"`
	Name       string    `json:"name"`
	Department string    `json:"department"`
	StudentID  string    `json:"student_id"`
	Role       Role      `json:"role"`
	Status     Status    `json:"status"`
	Bio        string    `json:"bio"`
	AvatarURL  string    `json:"avatar_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// StudentIDDisplay returns the student ID display based on the viewer's role.
// Admin sees the full student ID; students see only the first 2 digits + "학번".
func (u *User) StudentIDDisplay(viewerRole string) string {
	if viewerRole == string(RoleAdmin) {
		return u.StudentID
	}
	if len(u.StudentID) >= 2 {
		return u.StudentID[:2] + "학번"
	}
	return u.StudentID
}
