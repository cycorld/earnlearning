package post

import (
	"encoding/json"
	"time"
)

type Channel struct {
	ID          int    `json:"id"`
	ClassroomID int    `json:"classroom_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	ChannelType string `json:"channel_type"`
	WriteRole   string `json:"write_role"`
	SortOrder   int    `json:"sort_order"`
}

type Post struct {
	ID           int       `json:"id"`
	ChannelID    int       `json:"channel_id"`
	AuthorID     int       `json:"-"`
	Content      string    `json:"content"`
	PostType     string    `json:"post_type"`
	Media        string    `json:"media"`
	Tags         string    `json:"tags"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	Pinned       bool      `json:"pinned"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Joined fields (not stored in posts table, not serialized directly)
	AuthorName      string `json:"-"`
	AuthorAvatar    string `json:"-"`
	AuthorStudentID string `json:"-"`
	IsLiked         bool   `json:"is_liked"`
}

type PostAuthor struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	StudentID string `json:"student_id"`
}

func (p Post) MarshalJSON() ([]byte, error) {
	type Alias Post
	return json.Marshal(&struct {
		Alias
		Author PostAuthor `json:"author"`
	}{
		Alias: (Alias)(p),
		Author: PostAuthor{
			ID:        p.AuthorID,
			Name:      p.AuthorName,
			AvatarURL: p.AuthorAvatar,
			StudentID: p.AuthorStudentID,
		},
	})
}

type Comment struct {
	ID        int       `json:"id"`
	PostID    int       `json:"post_id"`
	AuthorID  int       `json:"-"`
	Content   string    `json:"content"`
	Media     string    `json:"media"`
	CreatedAt time.Time `json:"created_at"`

	// Joined fields (not serialized directly)
	AuthorName   string `json:"-"`
	AuthorAvatar string `json:"-"`
}

type CommentAuthor struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

func (c Comment) MarshalJSON() ([]byte, error) {
	type Alias Comment
	return json.Marshal(&struct {
		Alias
		Author CommentAuthor `json:"author"`
	}{
		Alias: (Alias)(c),
		Author: CommentAuthor{
			ID:        c.AuthorID,
			Name:      c.AuthorName,
			AvatarURL: c.AuthorAvatar,
		},
	})
}

type Assignment struct {
	ID           int       `json:"id"`
	PostID       int       `json:"post_id"`
	Deadline     time.Time `json:"deadline"`
	RewardAmount int       `json:"reward_amount"`
	MaxScore     int       `json:"max_score"`
}

type Submission struct {
	ID           int       `json:"id"`
	AssignmentID int       `json:"assignment_id"`
	StudentID    int       `json:"student_id"`
	CommentID    int       `json:"comment_id"`
	Content      string    `json:"content"`
	Files        string    `json:"files"`
	Grade        *int      `json:"grade"`
	Rewarded     bool      `json:"rewarded"`
	SubmittedAt  time.Time `json:"submitted_at"`

	// Joined fields
	StudentName string `json:"student_name,omitempty"`
}

type Upload struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Filename   string    `json:"filename"`
	StoredName string    `json:"stored_name"`
	MimeType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	Path       string    `json:"path"`
	CreatedAt  time.Time `json:"created_at"`
}
