package application

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/post"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

var tagRegex = regexp.MustCompile(`#([^\s#]+)`)

type PostUsecase struct {
	postRepo   post.PostRepository
	walletRepo wallet.Repository
	notifUC    *NotificationUseCase
}

func NewPostUsecase(pr post.PostRepository, wr wallet.Repository) *PostUsecase {
	return &PostUsecase{
		postRepo:   pr,
		walletRepo: wr,
	}
}

func (uc *PostUsecase) SetNotificationUseCase(notifUC *NotificationUseCase) {
	uc.notifUC = notifUC
}

func (uc *PostUsecase) GetChannels(classroomID int) ([]*post.Channel, error) {
	return uc.postRepo.GetChannels(classroomID)
}

type GetPostsInput struct {
	ClassroomID   int    `json:"classroom_id"`
	ChannelID     int    `json:"channel_id"`
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Tag           string `json:"tag"`
	CurrentUserID int    `json:"-"`
}

type PostsResult struct {
	Posts []*post.Post `json:"posts"`
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

func (uc *PostUsecase) GetPosts(input GetPostsInput) (*PostsResult, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.Limit < 1 || input.Limit > 50 {
		input.Limit = 20
	}

	posts, total, err := uc.postRepo.GetPosts(input.ClassroomID, input.ChannelID, input.Page, input.Limit, input.Tag, input.CurrentUserID)
	if err != nil {
		return nil, err
	}

	if posts == nil {
		posts = []*post.Post{}
	}

	return &PostsResult{
		Posts: posts,
		Total: total,
		Page:  input.Page,
		Limit: input.Limit,
	}, nil
}

type CreatePostInput struct {
	ChannelID int    `json:"channel_id"`
	Content   string `json:"content"`
	PostType  string `json:"post_type"`
	Media     string `json:"media"`
	Tags      string `json:"tags"`
}

func (uc *PostUsecase) CreatePost(userID int, role string, input CreatePostInput) (*post.Post, error) {
	// Get channel to check write_role
	ch, err := uc.postRepo.FindChannelByID(input.ChannelID)
	if err != nil {
		return nil, err
	}

	// Check write permission
	if ch.WriteRole == "admin" && role != "admin" {
		return nil, fmt.Errorf("이 채널에는 관리자만 글을 작성할 수 있습니다")
	}

	// Merge user-provided tags with auto-extracted tags from content
	autoTags := extractTags(input.Content)
	seen := make(map[string]bool)
	var tags []string
	// User-provided tags (from JSON array string)
	if input.Tags != "" {
		var userTags []string
		if json.Unmarshal([]byte(input.Tags), &userTags) == nil {
			for _, t := range userTags {
				t = strings.TrimSpace(t)
				if t != "" && !seen[t] {
					seen[t] = true
					tags = append(tags, t)
				}
			}
		}
	}
	// Auto-extracted #tags from content
	for _, t := range autoTags {
		if !seen[t] {
			seen[t] = true
			tags = append(tags, t)
		}
	}
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, _ := json.Marshal(tags)

	if input.PostType == "" {
		input.PostType = "normal"
	}
	if input.Media == "" {
		input.Media = "[]"
	}

	p := &post.Post{
		ChannelID: input.ChannelID,
		AuthorID:  userID,
		Content:   input.Content,
		PostType:  input.PostType,
		Media:     input.Media,
		Tags:      string(tagsJSON),
	}

	postID, err := uc.postRepo.CreatePost(p)
	if err != nil {
		return nil, fmt.Errorf("게시글 작성 실패: %w", err)
	}
	p.ID = postID

	return p, nil
}

func (uc *PostUsecase) LikePost(postID, userID int) (bool, error) {
	// Check if post exists
	_, err := uc.postRepo.FindPostByID(postID)
	if err != nil {
		return false, err
	}

	// Toggle like
	liked, err := uc.postRepo.IsLiked(postID, userID)
	if err != nil {
		return false, err
	}

	if liked {
		if err := uc.postRepo.UnlikePost(postID, userID); err != nil {
			return false, err
		}
		if err := uc.postRepo.DecrementLikeCount(postID); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := uc.postRepo.LikePost(postID, userID); err != nil {
		return false, err
	}
	if err := uc.postRepo.IncrementLikeCount(postID); err != nil {
		return false, err
	}
	return true, nil
}

type CreateCommentInput struct {
	PostID  int    `json:"post_id"`
	Content string `json:"content"`
	Media   string `json:"media"`
}

func (uc *PostUsecase) CreateComment(userID int, input CreateCommentInput) (*post.Comment, error) {
	// Check if post exists
	p, err := uc.postRepo.FindPostByID(input.PostID)
	if err != nil {
		return nil, err
	}

	if input.Media == "" {
		input.Media = "[]"
	}

	c := &post.Comment{
		PostID:   input.PostID,
		AuthorID: userID,
		Content:  input.Content,
		Media:    input.Media,
	}

	commentID, err := uc.postRepo.CreateComment(c)
	if err != nil {
		return nil, fmt.Errorf("댓글 작성 실패: %w", err)
	}
	c.ID = commentID
	c.CreatedAt = time.Now()

	_ = uc.postRepo.IncrementCommentCount(input.PostID)

	// If this is an assignment post, auto-create submission
	if p.PostType == "assignment" {
		assignment, err := uc.postRepo.FindAssignmentByPostID(input.PostID)
		if err == nil && assignment != nil {
			existing, _ := uc.postRepo.FindSubmission(assignment.ID, userID)
			if existing == nil {
				_, _ = uc.postRepo.CreateSubmission(&post.Submission{
					AssignmentID: assignment.ID,
					StudentID:    userID,
					CommentID:    commentID,
					Content:      input.Content,
					Files:        input.Media,
				})
			}
		}
	}

	return c, nil
}

type CreateAssignmentInput struct {
	ChannelID    int       `json:"channel_id"`
	Content      string    `json:"content"`
	Media        string    `json:"media"`
	Deadline     time.Time `json:"deadline"`
	RewardAmount int       `json:"reward_amount"`
	MaxScore     int       `json:"max_score"`
}

func (uc *PostUsecase) CreateAssignment(userID int, input CreateAssignmentInput) (*post.Post, *post.Assignment, error) {
	// Extract tags
	tags := extractTags(input.Content)
	tagsJSON, _ := json.Marshal(tags)

	if input.Media == "" {
		input.Media = "[]"
	}
	if input.MaxScore == 0 {
		input.MaxScore = 100
	}

	// Create post first
	p := &post.Post{
		ChannelID: input.ChannelID,
		AuthorID:  userID,
		Content:   input.Content,
		PostType:  "assignment",
		Media:     input.Media,
		Tags:      string(tagsJSON),
	}

	postID, err := uc.postRepo.CreatePost(p)
	if err != nil {
		return nil, nil, fmt.Errorf("과제 게시글 작성 실패: %w", err)
	}
	p.ID = postID

	// Create assignment record
	a := &post.Assignment{
		PostID:       postID,
		Deadline:     input.Deadline,
		RewardAmount: input.RewardAmount,
		MaxScore:     input.MaxScore,
	}

	assignmentID, err := uc.postRepo.CreateAssignment(a)
	if err != nil {
		return nil, nil, fmt.Errorf("과제 생성 실패: %w", err)
	}
	a.ID = assignmentID

	return p, a, nil
}

type SubmitAssignmentInput struct {
	AssignmentID int    `json:"assignment_id"`
	Content      string `json:"content"`
	Files        string `json:"files"`
}

func (uc *PostUsecase) SubmitAssignment(userID int, input SubmitAssignmentInput) (*post.Submission, error) {
	// Check assignment exists
	assignment, err := uc.postRepo.FindAssignmentByID(input.AssignmentID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate submission
	existing, _ := uc.postRepo.FindSubmission(input.AssignmentID, userID)
	if existing != nil {
		return nil, fmt.Errorf("이미 제출한 과제입니다")
	}

	if input.Files == "" {
		input.Files = "[]"
	}

	// Create comment on the assignment post
	comment := &post.Comment{
		PostID:   assignment.PostID,
		AuthorID: userID,
		Content:  input.Content,
		Media:    input.Files,
	}

	commentID, err := uc.postRepo.CreateComment(comment)
	if err != nil {
		return nil, fmt.Errorf("댓글 작성 실패: %w", err)
	}
	_ = uc.postRepo.IncrementCommentCount(assignment.PostID)

	// Create submission
	s := &post.Submission{
		AssignmentID: input.AssignmentID,
		StudentID:    userID,
		CommentID:    commentID,
		Content:      input.Content,
		Files:        input.Files,
	}

	submissionID, err := uc.postRepo.CreateSubmission(s)
	if err != nil {
		return nil, fmt.Errorf("과제 제출 실패: %w", err)
	}
	s.ID = submissionID

	return s, nil
}

type GradeAssignmentInput struct {
	SubmissionID int `json:"submission_id"`
	Grade        int `json:"grade"`
}

func (uc *PostUsecase) GradeAssignment(input GradeAssignmentInput) error {
	// Get submission
	submission, err := uc.postRepo.FindSubmissionByID(input.SubmissionID)
	if err != nil {
		return err
	}

	// Get assignment for reward calculation
	assignment, err := uc.postRepo.FindAssignmentByID(submission.AssignmentID)
	if err != nil {
		return err
	}

	// Validate grade
	if input.Grade < 0 || input.Grade > assignment.MaxScore {
		return fmt.Errorf("점수는 0~%d 사이여야 합니다", assignment.MaxScore)
	}

	// Calculate reward: reward_amount * grade / max_score
	reward := 0
	if assignment.RewardAmount > 0 && assignment.MaxScore > 0 {
		reward = assignment.RewardAmount * input.Grade / assignment.MaxScore
	}

	// Credit student's personal wallet if reward > 0
	rewarded := false
	if reward > 0 {
		// Ensure wallet exists, create if not
		w, err := uc.walletRepo.FindByUserID(submission.StudentID)
		if err != nil {
			// Try to create wallet
			walletID, createErr := uc.walletRepo.CreateWallet(submission.StudentID)
			if createErr != nil {
				return uc.postRepo.UpdateSubmissionGrade(input.SubmissionID, input.Grade, false)
			}
			err = uc.walletRepo.Credit(walletID, reward, wallet.TxAssignReward,
				fmt.Sprintf("과제 보상 (점수: %d/%d)", input.Grade, assignment.MaxScore),
				"submission", submission.ID)
			if err == nil {
				rewarded = true
			}
		} else {
			err = uc.walletRepo.Credit(w.ID, reward, wallet.TxAssignReward,
				fmt.Sprintf("과제 보상 (점수: %d/%d)", input.Grade, assignment.MaxScore),
				"submission", submission.ID)
			if err == nil {
				rewarded = true
			}
		}
	}

	// Notify student about grade
	if uc.notifUC != nil {
		msg := fmt.Sprintf("과제 점수: %d/%d점", input.Grade, assignment.MaxScore)
		if reward > 0 {
			msg += fmt.Sprintf(" (보상: %d원)", reward)
		}
		_ = uc.notifUC.CreateNotification(submission.StudentID, notification.NotifAssignmentGraded,
			"과제가 채점되었습니다", msg, "assignment", assignment.ID)
	}

	// Update submission grade
	return uc.postRepo.UpdateSubmissionGrade(input.SubmissionID, input.Grade, rewarded)
}

func (uc *PostUsecase) GetComments(postID int) ([]*post.Comment, error) {
	return uc.postRepo.GetComments(postID)
}

func (uc *PostUsecase) GetSubmissions(assignmentID int) ([]*post.Submission, error) {
	return uc.postRepo.GetSubmissions(assignmentID)
}

// extractTags parses #태그 patterns from content.
func extractTags(content string) []string {
	matches := tagRegex.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := m[1]
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	if tags == nil {
		tags = []string{}
	}
	return tags
}
