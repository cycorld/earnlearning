package user

import "time"

type Repository interface {
	Create(u *User) (int, error)
	FindByID(id int) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByStatus(status Status) ([]*User, error)
	ListAll(page, limit int) ([]*User, int, error)
	UpdateStatus(id int, status Status) error
	UpdateAvatarURL(id int, avatarURL string) error
	GetUserActivity(userID int) (*UserActivity, error)

	// #128 비밀번호 재설정
	UpdatePassword(id int, passwordHash string) error
	// SaveResetToken은 해당 사용자의 기존 토큰을 모두 무효화하고 새 토큰 해시를 저장한다.
	SaveResetToken(userID int, tokenHash string, expiresAt time.Time) error
	// ConsumeResetToken은 유효(미사용·미만료)한 토큰을 사용 처리하고 user_id를 반환한다.
	// 유효하지 않으면 ErrInvalidResetToken.
	ConsumeResetToken(tokenHash string) (int, error)
}

type ActivityPost struct {
	ID        int    `json:"id"`
	Content   string `json:"content"`
	PostType  string `json:"post_type"`
	Channel   string `json:"channel"`
	LikeCount int    `json:"like_count"`
	CreatedAt string `json:"created_at"`
}

type ActivityFreelanceJob struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Budget      int    `json:"budget"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type ActivityGrantApp struct {
	ID        int    `json:"id"`
	GrantID   int    `json:"grant_id"`
	GrantTitle string `json:"grant_title"`
	Status    string `json:"status"`
	Proposal  string `json:"proposal"`
	CreatedAt string `json:"created_at"`
}

type UserActivity struct {
	Posts         []ActivityPost         `json:"posts"`
	FreelanceJobs []ActivityFreelanceJob `json:"freelance_jobs"`
	GrantApps     []ActivityGrantApp     `json:"grant_apps"`
}
