package user

type Repository interface {
	Create(u *User) (int, error)
	FindByID(id int) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByStatus(status Status) ([]*User, error)
	ListAll(page, limit int) ([]*User, int, error)
	UpdateStatus(id int, status Status) error
	GetUserActivity(userID int) (*UserActivity, error)
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
