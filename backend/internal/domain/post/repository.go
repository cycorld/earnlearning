package post

// PostRepository defines the persistence interface for the feed domain.
type PostRepository interface {
	// Channel operations
	GetChannels(classroomID int) ([]*Channel, error)
	FindChannelByID(channelID int) (*Channel, error)

	// Post operations
	CreatePost(p *Post) (int, error)
	FindPostByID(postID int) (*Post, error)
	UpdatePost(postID int, content string, tags string) error
	DeletePost(postID int) error
	GetPosts(classroomID, channelID int, page, limit int, tag string, currentUserID int) ([]*Post, int, error)

	// Like operations
	LikePost(postID, userID int) error
	UnlikePost(postID, userID int) error
	IsLiked(postID, userID int) (bool, error)
	IncrementLikeCount(postID int) error
	DecrementLikeCount(postID int) error

	// Comment operations
	CreateComment(c *Comment) (int, error)
	GetComments(postID int) ([]*Comment, error)
	FindCommentByID(commentID int) (*Comment, error)
	DeleteComment(commentID int) error
	IncrementCommentCount(postID int) error
	DecrementCommentCount(postID int) error

	// Assignment operations
	CreateAssignment(a *Assignment) (int, error)
	FindAssignmentByID(assignmentID int) (*Assignment, error)
	FindAssignmentByPostID(postID int) (*Assignment, error)

	// Submission operations
	CreateSubmission(s *Submission) (int, error)
	FindSubmission(assignmentID, studentID int) (*Submission, error)
	FindSubmissionByID(submissionID int) (*Submission, error)
	UpdateSubmissionGrade(submissionID int, grade int, rewarded bool) error
	GetSubmissions(assignmentID int) ([]*Submission, error)

	// Upload operations
	CreateUpload(u *Upload) (int, error)
	FindUploadByID(id int) (*Upload, error)
}
