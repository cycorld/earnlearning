package persistence

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/post"
)

type PostRepo struct {
	db *sql.DB
}

func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

// Channel operations

func (r *PostRepo) GetChannels(classroomID int) ([]*post.Channel, error) {
	rows, err := r.db.Query(`
		SELECT id, classroom_id, name, slug, channel_type, write_role, sort_order
		FROM channels WHERE classroom_id = ? ORDER BY sort_order`, classroomID)
	if err != nil {
		return nil, fmt.Errorf("query channels: %w", err)
	}
	defer rows.Close()

	var channels []*post.Channel
	for rows.Next() {
		ch := &post.Channel{}
		if err := rows.Scan(&ch.ID, &ch.ClassroomID, &ch.Name, &ch.Slug, &ch.ChannelType, &ch.WriteRole, &ch.SortOrder); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

func (r *PostRepo) FindChannelByID(channelID int) (*post.Channel, error) {
	ch := &post.Channel{}
	err := r.db.QueryRow(`
		SELECT id, classroom_id, name, slug, channel_type, write_role, sort_order
		FROM channels WHERE id = ?`, channelID).Scan(
		&ch.ID, &ch.ClassroomID, &ch.Name, &ch.Slug, &ch.ChannelType, &ch.WriteRole, &ch.SortOrder,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("채널을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, fmt.Errorf("query channel: %w", err)
	}
	return ch, nil
}

// Post operations

func (r *PostRepo) CreatePost(p *post.Post) (int, error) {
	pinned := 0
	if p.Pinned {
		pinned = 1
	}
	res, err := r.db.Exec(`
		INSERT INTO posts (channel_id, author_id, content, post_type, media, tags, like_count, comment_count, pinned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ChannelID, p.AuthorID, p.Content, p.PostType, p.Media, p.Tags,
		p.LikeCount, p.CommentCount, pinned,
	)
	if err != nil {
		return 0, fmt.Errorf("insert post: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *PostRepo) FindPostByID(postID int) (*post.Post, error) {
	p := &post.Post{}
	var pinned int
	err := r.db.QueryRow(`
		SELECT p.id, p.channel_id, p.author_id, p.content, p.post_type, p.media, p.tags,
		       p.like_count, p.comment_count, p.pinned, p.created_at, p.updated_at,
		       u.name, u.avatar_url, u.student_id, u.department
		FROM posts p
		JOIN users u ON u.id = p.author_id
		WHERE p.id = ?`, postID).Scan(
		&p.ID, &p.ChannelID, &p.AuthorID, &p.Content, &p.PostType, &p.Media, &p.Tags,
		&p.LikeCount, &p.CommentCount, &pinned, &p.CreatedAt, &p.UpdatedAt,
		&p.AuthorName, &p.AuthorAvatar, &p.AuthorStudentID, &p.AuthorDepartment,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("게시글을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, fmt.Errorf("query post: %w", err)
	}
	p.Pinned = pinned == 1
	return p, nil
}

// FindPostByIDWithViewer returns a single post enriched with author, channel
// name, and the viewer's is_liked flag — same shape as GetPosts rows, so the
// frontend can render a deep-linked detail page identically to a feed card.
func (r *PostRepo) FindPostByIDWithViewer(postID, viewerUserID int) (*post.Post, error) {
	p := &post.Post{}
	var pinned, isLiked int
	err := r.db.QueryRow(`
		SELECT p.id, p.channel_id, p.author_id, p.content, p.post_type, p.media, p.tags,
		       p.like_count, p.comment_count, p.pinned, p.created_at, p.updated_at,
		       u.name, u.avatar_url, u.student_id, u.department,
		       CASE WHEN pl.id IS NOT NULL THEN 1 ELSE 0 END as is_liked,
		       COALESCE(ch.name, '') as channel_name
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN channels ch ON ch.id = p.channel_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = ?
		WHERE p.id = ?`, viewerUserID, postID).Scan(
		&p.ID, &p.ChannelID, &p.AuthorID, &p.Content, &p.PostType, &p.Media, &p.Tags,
		&p.LikeCount, &p.CommentCount, &pinned, &p.CreatedAt, &p.UpdatedAt,
		&p.AuthorName, &p.AuthorAvatar, &p.AuthorStudentID, &p.AuthorDepartment, &isLiked,
		&p.ChannelName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query post by id: %w", err)
	}
	p.Pinned = pinned == 1
	p.IsLiked = isLiked == 1
	return p, nil
}

func (r *PostRepo) UpdatePost(postID int, content string, tags string) error {
	_, err := r.db.Exec("UPDATE posts SET content = ?, tags = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", content, tags, postID)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	return nil
}

func (r *PostRepo) DeletePost(postID int) error {
	_, err := r.db.Exec("DELETE FROM comments WHERE post_id = ?", postID)
	if err != nil {
		return fmt.Errorf("delete post comments: %w", err)
	}
	_, err = r.db.Exec("DELETE FROM post_likes WHERE post_id = ?", postID)
	if err != nil {
		return fmt.Errorf("delete post likes: %w", err)
	}
	_, err = r.db.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	return nil
}

func (r *PostRepo) GetPosts(classroomID, channelID int, page, limit int, tag string, currentUserID int) ([]*post.Post, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause based on classroomID or channelID
	var countWhere string
	var countArgs []interface{}
	if channelID > 0 {
		countWhere = "WHERE p.channel_id = ?"
		countArgs = []interface{}{channelID}
	} else if classroomID > 0 {
		countWhere = "WHERE p.channel_id IN (SELECT id FROM channels WHERE classroom_id = ?)"
		countArgs = []interface{}{classroomID}
	} else {
		countWhere = "WHERE 1=0"
	}

	if tag != "" {
		countWhere += " AND p.tags LIKE ?"
		countArgs = append(countArgs, "%\""+tag+"\"%")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM posts p " + countWhere

	var total int
	if err := r.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	// Build main query WHERE clause
	var queryWhere string
	var filterArgs []interface{}
	if channelID > 0 {
		queryWhere = "WHERE p.channel_id = ?"
		filterArgs = []interface{}{channelID}
	} else if classroomID > 0 {
		queryWhere = "WHERE p.channel_id IN (SELECT id FROM channels WHERE classroom_id = ?)"
		filterArgs = []interface{}{classroomID}
	} else {
		queryWhere = "WHERE 1=0"
	}

	// Query posts
	query := `
		SELECT p.id, p.channel_id, p.author_id, p.content, p.post_type, p.media, p.tags,
		       p.like_count, p.comment_count, p.pinned, p.created_at, p.updated_at,
		       u.name, u.avatar_url, u.student_id, u.department,
		       CASE WHEN pl.id IS NOT NULL THEN 1 ELSE 0 END as is_liked,
		       COALESCE(ch.name, '') as channel_name
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN channels ch ON ch.id = p.channel_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = ?
		` + queryWhere
	args := append([]interface{}{currentUserID}, filterArgs...)

	if tag != "" {
		query += " AND p.tags LIKE ?"
		args = append(args, "%\""+tag+"\"%")
	}

	query += " ORDER BY p.pinned DESC, p.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []*post.Post
	for rows.Next() {
		p := &post.Post{}
		var pinned, isLiked int
		if err := rows.Scan(
			&p.ID, &p.ChannelID, &p.AuthorID, &p.Content, &p.PostType, &p.Media, &p.Tags,
			&p.LikeCount, &p.CommentCount, &pinned, &p.CreatedAt, &p.UpdatedAt,
			&p.AuthorName, &p.AuthorAvatar, &p.AuthorStudentID, &p.AuthorDepartment, &isLiked,
			&p.ChannelName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan post: %w", err)
		}
		p.Pinned = pinned == 1
		p.IsLiked = isLiked == 1
		posts = append(posts, p)
	}
	return posts, total, nil
}

// Like operations

func (r *PostRepo) LikePost(postID, userID int) error {
	_, err := r.db.Exec("INSERT INTO post_likes (post_id, user_id) VALUES (?, ?)", postID, userID)
	if err != nil {
		return fmt.Errorf("insert like: %w", err)
	}
	return nil
}

func (r *PostRepo) UnlikePost(postID, userID int) error {
	_, err := r.db.Exec("DELETE FROM post_likes WHERE post_id = ? AND user_id = ?", postID, userID)
	if err != nil {
		return fmt.Errorf("delete like: %w", err)
	}
	return nil
}

func (r *PostRepo) IsLiked(postID, userID int) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?", postID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check like: %w", err)
	}
	return count > 0, nil
}

func (r *PostRepo) IncrementLikeCount(postID int) error {
	_, err := r.db.Exec("UPDATE posts SET like_count = like_count + 1 WHERE id = ?", postID)
	return err
}

func (r *PostRepo) DecrementLikeCount(postID int) error {
	_, err := r.db.Exec("UPDATE posts SET like_count = CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END WHERE id = ?", postID)
	return err
}

// Comment operations

func (r *PostRepo) CreateComment(c *post.Comment) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO comments (post_id, author_id, content, media)
		VALUES (?, ?, ?, ?)`,
		c.PostID, c.AuthorID, c.Content, c.Media,
	)
	if err != nil {
		return 0, fmt.Errorf("insert comment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *PostRepo) GetComments(postID int) ([]*post.Comment, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.post_id, c.author_id, c.content, c.media, c.created_at,
		       u.name, u.avatar_url, u.student_id, u.department
		FROM comments c
		JOIN users u ON u.id = c.author_id
		WHERE c.post_id = ? ORDER BY c.created_at`, postID)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()

	var comments []*post.Comment
	for rows.Next() {
		c := &post.Comment{}
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorID, &c.Content, &c.Media, &c.CreatedAt, &c.AuthorName, &c.AuthorAvatar, &c.AuthorStudentID, &c.AuthorDepartment); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *PostRepo) FindCommentByID(commentID int) (*post.Comment, error) {
	c := &post.Comment{}
	err := r.db.QueryRow(`
		SELECT c.id, c.post_id, c.author_id, c.content, c.media, c.created_at,
		       u.name, u.avatar_url, u.student_id, u.department
		FROM comments c
		JOIN users u ON u.id = c.author_id
		WHERE c.id = ?`, commentID).Scan(
		&c.ID, &c.PostID, &c.AuthorID, &c.Content, &c.Media, &c.CreatedAt,
		&c.AuthorName, &c.AuthorAvatar, &c.AuthorStudentID, &c.AuthorDepartment,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("댓글을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, fmt.Errorf("query comment: %w", err)
	}
	return c, nil
}

func (r *PostRepo) DeleteComment(commentID int) error {
	_, err := r.db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}

func (r *PostRepo) IncrementCommentCount(postID int) error {
	_, err := r.db.Exec("UPDATE posts SET comment_count = comment_count + 1 WHERE id = ?", postID)
	return err
}

func (r *PostRepo) DecrementCommentCount(postID int) error {
	_, err := r.db.Exec("UPDATE posts SET comment_count = CASE WHEN comment_count > 0 THEN comment_count - 1 ELSE 0 END WHERE id = ?", postID)
	return err
}

// Assignment operations

func (r *PostRepo) CreateAssignment(a *post.Assignment) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO assignments (post_id, deadline, reward_amount, max_score)
		VALUES (?, ?, ?, ?)`,
		a.PostID, a.Deadline, a.RewardAmount, a.MaxScore,
	)
	if err != nil {
		return 0, fmt.Errorf("insert assignment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *PostRepo) FindAssignmentByID(assignmentID int) (*post.Assignment, error) {
	a := &post.Assignment{}
	err := r.db.QueryRow(`
		SELECT id, post_id, deadline, reward_amount, max_score
		FROM assignments WHERE id = ?`, assignmentID).Scan(
		&a.ID, &a.PostID, &a.Deadline, &a.RewardAmount, &a.MaxScore,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("과제를 찾을 수 없습니다")
	}
	if err != nil {
		return nil, fmt.Errorf("query assignment: %w", err)
	}
	return a, nil
}

func (r *PostRepo) FindAssignmentByPostID(postID int) (*post.Assignment, error) {
	a := &post.Assignment{}
	err := r.db.QueryRow(`
		SELECT id, post_id, deadline, reward_amount, max_score
		FROM assignments WHERE post_id = ?`, postID).Scan(
		&a.ID, &a.PostID, &a.Deadline, &a.RewardAmount, &a.MaxScore,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query assignment by post: %w", err)
	}
	return a, nil
}

// Submission operations

func (r *PostRepo) CreateSubmission(s *post.Submission) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO submissions (assignment_id, student_id, comment_id, content, files)
		VALUES (?, ?, ?, ?, ?)`,
		s.AssignmentID, s.StudentID, s.CommentID, s.Content, s.Files,
	)
	if err != nil {
		return 0, fmt.Errorf("insert submission: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *PostRepo) FindSubmission(assignmentID, studentID int) (*post.Submission, error) {
	s := &post.Submission{}
	err := r.db.QueryRow(`
		SELECT id, assignment_id, student_id, comment_id, content, files, grade, rewarded, submitted_at
		FROM submissions WHERE assignment_id = ? AND student_id = ?`, assignmentID, studentID).Scan(
		&s.ID, &s.AssignmentID, &s.StudentID, &s.CommentID, &s.Content, &s.Files,
		&s.Grade, &s.Rewarded, &s.SubmittedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query submission: %w", err)
	}
	return s, nil
}

func (r *PostRepo) FindSubmissionByID(submissionID int) (*post.Submission, error) {
	s := &post.Submission{}
	err := r.db.QueryRow(`
		SELECT s.id, s.assignment_id, s.student_id, s.comment_id, s.content, s.files, s.grade, s.rewarded, s.submitted_at,
		       u.name
		FROM submissions s
		JOIN users u ON u.id = s.student_id
		WHERE s.id = ?`, submissionID).Scan(
		&s.ID, &s.AssignmentID, &s.StudentID, &s.CommentID, &s.Content, &s.Files,
		&s.Grade, &s.Rewarded, &s.SubmittedAt, &s.StudentName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("제출물을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, fmt.Errorf("query submission: %w", err)
	}
	return s, nil
}

func (r *PostRepo) UpdateSubmissionGrade(submissionID int, grade int, rewarded bool) error {
	rewardedInt := 0
	if rewarded {
		rewardedInt = 1
	}
	_, err := r.db.Exec("UPDATE submissions SET grade = ?, rewarded = ? WHERE id = ?", grade, rewardedInt, submissionID)
	if err != nil {
		return fmt.Errorf("update grade: %w", err)
	}
	return nil
}

func (r *PostRepo) GetSubmissions(assignmentID int) ([]*post.Submission, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.assignment_id, s.student_id, s.comment_id, s.content, s.files, s.grade, s.rewarded, s.submitted_at,
		       u.name
		FROM submissions s
		JOIN users u ON u.id = s.student_id
		WHERE s.assignment_id = ? ORDER BY s.submitted_at`, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("query submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*post.Submission
	for rows.Next() {
		s := &post.Submission{}
		if err := rows.Scan(
			&s.ID, &s.AssignmentID, &s.StudentID, &s.CommentID, &s.Content, &s.Files,
			&s.Grade, &s.Rewarded, &s.SubmittedAt, &s.StudentName,
		); err != nil {
			return nil, fmt.Errorf("scan submission: %w", err)
		}
		submissions = append(submissions, s)
	}
	return submissions, nil
}

// Upload operations

func (r *PostRepo) CreateUpload(u *post.Upload) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO uploads (user_id, filename, stored_name, mime_type, size, path)
		VALUES (?, ?, ?, ?, ?, ?)`,
		u.UserID, u.Filename, u.StoredName, u.MimeType, u.Size, u.Path,
	)
	if err != nil {
		return 0, fmt.Errorf("insert upload: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return int(id), nil
}

func (r *PostRepo) FindUploadByID(id int) (*post.Upload, error) {
	u := &post.Upload{}
	err := r.db.QueryRow(`
		SELECT id, user_id, filename, stored_name, mime_type, size, path, created_at
		FROM uploads WHERE id = ?`, id).Scan(
		&u.ID, &u.UserID, &u.Filename, &u.StoredName, &u.MimeType, &u.Size, &u.Path, &u.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("파일을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, fmt.Errorf("query upload: %w", err)
	}
	return u, nil
}
