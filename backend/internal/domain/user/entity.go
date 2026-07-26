package user

import (
	"strconv"
	"time"
)

// OAuthSubject 는 OAuth userinfo 의 `sub` 클레임 문자열이다 (#181).
// Rybbit SSO 는 (providerId="earnlearning", accountId=sub) 로 계정을 영구 연결하고,
// 프로비저닝 API 의 user.id 도 반드시 같은 값이어야 한다. 형식을 바꾸면 기존
// Rybbit 계정 연결이 전부 끊어지므로 절대 변경하지 않는다.
func OAuthSubject(id int) string { return strconv.Itoa(id) }

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
	ID         int    `json:"id"`
	Email      string `json:"email"`
	Password   string `json:"-"`
	Name       string `json:"name"`
	Department string `json:"department"`
	StudentID  string `json:"student_id"`
	Role       Role   `json:"role"`
	Status     Status `json:"status"`
	Bio        string `json:"bio"`
	AvatarURL  string `json:"avatar_url"`
	// #159 활성(현재) 강의실. 0 = 미설정
	ActiveClassroomID int       `json:"active_classroom_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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
