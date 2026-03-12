package persistence

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/freelance"
)

type FreelanceRepo struct {
	db *sql.DB
}

func NewFreelanceRepo(db *sql.DB) *FreelanceRepo {
	return &FreelanceRepo{db: db}
}

func (r *FreelanceRepo) Create(job *freelance.FreelanceJob) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO freelance_jobs (client_id, title, description, budget, deadline, required_skills, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		job.ClientID, job.Title, job.Description, job.Budget, job.Deadline, job.RequiredSkills, job.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("create freelance job: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *FreelanceRepo) FindByID(id int) (*freelance.FreelanceJob, error) {
	job := &freelance.FreelanceJob{}
	var freelancerID sql.NullInt64
	var deadline sql.NullTime
	var completedAt sql.NullTime
	var freelancerName sql.NullString
	var clientName string

	err := r.db.QueryRow(`
		SELECT j.id, j.client_id, j.title, j.description, j.budget, j.deadline,
			   j.required_skills, j.status, j.freelancer_id, j.escrow_amount,
			   j.agreed_price, j.work_completed, j.created_at, j.completed_at,
			   u1.name AS client_name,
			   u2.name AS freelancer_name
		FROM freelance_jobs j
		JOIN users u1 ON u1.id = j.client_id
		LEFT JOIN users u2 ON u2.id = j.freelancer_id
		WHERE j.id = ?`, id).Scan(
		&job.ID, &job.ClientID, &job.Title, &job.Description, &job.Budget, &deadline,
		&job.RequiredSkills, &job.Status, &freelancerID, &job.EscrowAmount,
		&job.AgreedPrice, &job.WorkCompleted, &job.CreatedAt, &completedAt,
		&clientName, &freelancerName,
	)
	if err == sql.ErrNoRows {
		return nil, freelance.ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find freelance job: %w", err)
	}
	if freelancerID.Valid {
		fid := int(freelancerID.Int64)
		job.FreelancerID = &fid
	}
	if deadline.Valid {
		job.Deadline = &deadline.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if freelancerName.Valid {
		job.FreelancerName = freelancerName.String
	}
	// Populate nested client reference
	job.Client = &freelance.UserRef{
		ID:   job.ClientID,
		Name: clientName,
	}
	job.ClientName = clientName
	return job, nil
}

func (r *FreelanceRepo) List(filter freelance.JobFilter, page, limit int) ([]*freelance.FreelanceJob, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if filter.Status != "" {
		where = append(where, "j.status = ?")
		args = append(args, filter.Status)
	}
	if filter.Skills != "" {
		where = append(where, "j.required_skills LIKE ?")
		args = append(args, "%"+filter.Skills+"%")
	}
	if filter.MinBudget > 0 {
		where = append(where, "j.budget >= ?")
		args = append(args, filter.MinBudget)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM freelance_jobs j WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count freelance jobs: %w", err)
	}

	offset := (page - 1) * limit
	queryArgs := append(args, limit, offset)

	rows, err := r.db.Query(`
		SELECT j.id, j.client_id, j.title, j.description, j.budget, j.deadline,
			   j.required_skills, j.status, j.freelancer_id, j.escrow_amount,
			   j.agreed_price, j.work_completed, j.created_at, j.completed_at,
			   u.name AS client_name,
			   (SELECT COUNT(*) FROM job_applications WHERE job_id = j.id) AS application_count
		FROM freelance_jobs j
		JOIN users u ON u.id = j.client_id
		WHERE `+whereClause+`
		ORDER BY j.created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list freelance jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*freelance.FreelanceJob
	for rows.Next() {
		job := &freelance.FreelanceJob{}
		var freelancerID sql.NullInt64
		var deadline sql.NullTime
		var completedAt sql.NullTime
		var clientName string
		var appCount int

		if err := rows.Scan(
			&job.ID, &job.ClientID, &job.Title, &job.Description, &job.Budget, &deadline,
			&job.RequiredSkills, &job.Status, &freelancerID, &job.EscrowAmount,
			&job.AgreedPrice, &job.WorkCompleted, &job.CreatedAt, &completedAt,
			&clientName, &appCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan freelance job: %w", err)
		}
		if freelancerID.Valid {
			fid := int(freelancerID.Int64)
			job.FreelancerID = &fid
		}
		if deadline.Valid {
			job.Deadline = &deadline.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		// Populate nested client reference and application count
		job.Client = &freelance.UserRef{
			ID:   job.ClientID,
			Name: clientName,
		}
		job.ClientName = clientName
		job.ApplicationCount = &appCount
		jobs = append(jobs, job)
	}
	return jobs, total, nil
}

func (r *FreelanceRepo) UpdateStatus(id int, status freelance.JobStatus) error {
	_, err := r.db.Exec("UPDATE freelance_jobs SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *FreelanceRepo) SetFreelancer(jobID, freelancerID, agreedPrice int) error {
	_, err := r.db.Exec(
		"UPDATE freelance_jobs SET freelancer_id = ?, agreed_price = ?, status = 'in_progress' WHERE id = ?",
		freelancerID, agreedPrice, jobID,
	)
	return err
}

func (r *FreelanceRepo) SetEscrow(jobID, amount int) error {
	_, err := r.db.Exec("UPDATE freelance_jobs SET escrow_amount = ? WHERE id = ?", amount, jobID)
	return err
}

func (r *FreelanceRepo) SetWorkCompleted(jobID int) error {
	_, err := r.db.Exec("UPDATE freelance_jobs SET work_completed = 1 WHERE id = ?", jobID)
	return err
}

func (r *FreelanceRepo) SetCompleted(jobID int) error {
	_, err := r.db.Exec(
		"UPDATE freelance_jobs SET status = 'completed', completed_at = CURRENT_TIMESTAMP WHERE id = ?",
		jobID,
	)
	return err
}

// --- Applications ---

func (r *FreelanceRepo) CreateApplication(app *freelance.JobApplication) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO job_applications (job_id, user_id, proposal, price, status)
		VALUES (?, ?, ?, ?, ?)`,
		app.JobID, app.UserID, app.Proposal, app.Price, app.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("create application: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *FreelanceRepo) FindApplicationByID(id int) (*freelance.JobApplication, error) {
	app := &freelance.JobApplication{}
	var userName string
	err := r.db.QueryRow(`
		SELECT a.id, a.job_id, a.user_id, a.proposal, a.price, a.status, a.created_at,
			   u.name AS user_name
		FROM job_applications a
		JOIN users u ON u.id = a.user_id
		WHERE a.id = ?`, id).Scan(
		&app.ID, &app.JobID, &app.UserID, &app.Proposal, &app.Price, &app.Status, &app.CreatedAt,
		&userName,
	)
	if err == sql.ErrNoRows {
		return nil, freelance.ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find application: %w", err)
	}
	app.UserName = userName
	app.User = &freelance.UserRef{
		ID:   app.UserID,
		Name: userName,
	}
	return app, nil
}

func (r *FreelanceRepo) FindApplicationByJobAndUser(jobID, userID int) (*freelance.JobApplication, error) {
	app := &freelance.JobApplication{}
	var userName string
	err := r.db.QueryRow(`
		SELECT a.id, a.job_id, a.user_id, a.proposal, a.price, a.status, a.created_at,
			   u.name AS user_name
		FROM job_applications a
		JOIN users u ON u.id = a.user_id
		WHERE a.job_id = ? AND a.user_id = ?`, jobID, userID).Scan(
		&app.ID, &app.JobID, &app.UserID, &app.Proposal, &app.Price, &app.Status, &app.CreatedAt,
		&userName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find application by job and user: %w", err)
	}
	app.UserName = userName
	app.User = &freelance.UserRef{
		ID:   app.UserID,
		Name: userName,
	}
	return app, nil
}

func (r *FreelanceRepo) ListApplicationsByJob(jobID int) ([]*freelance.JobApplication, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.job_id, a.user_id, a.proposal, a.price, a.status, a.created_at,
			   u.name AS user_name
		FROM job_applications a
		JOIN users u ON u.id = a.user_id
		WHERE a.job_id = ?
		ORDER BY a.created_at ASC`, jobID)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []*freelance.JobApplication
	for rows.Next() {
		app := &freelance.JobApplication{}
		var userName string
		if err := rows.Scan(
			&app.ID, &app.JobID, &app.UserID, &app.Proposal, &app.Price, &app.Status, &app.CreatedAt,
			&userName,
		); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		app.UserName = userName
		app.User = &freelance.UserRef{
			ID:   app.UserID,
			Name: userName,
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (r *FreelanceRepo) UpdateApplicationStatus(id int, status freelance.ApplicationStatus) error {
	_, err := r.db.Exec("UPDATE job_applications SET status = ? WHERE id = ?", status, id)
	return err
}

// --- Reviews ---

func (r *FreelanceRepo) CreateReview(review *freelance.FreelanceReview) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO freelance_reviews (job_id, reviewer_id, reviewee_id, rating, comment)
		VALUES (?, ?, ?, ?, ?)`,
		review.JobID, review.ReviewerID, review.RevieweeID, review.Rating, review.Comment,
	)
	if err != nil {
		return 0, fmt.Errorf("create review: %w", err)
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (r *FreelanceRepo) FindReviewByJobAndReviewer(jobID, reviewerID int) (*freelance.FreelanceReview, error) {
	review := &freelance.FreelanceReview{}
	err := r.db.QueryRow(`
		SELECT id, job_id, reviewer_id, reviewee_id, rating, comment, created_at
		FROM freelance_reviews
		WHERE job_id = ? AND reviewer_id = ?`, jobID, reviewerID).Scan(
		&review.ID, &review.JobID, &review.ReviewerID, &review.RevieweeID,
		&review.Rating, &review.Comment, &review.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find review: %w", err)
	}
	return review, nil
}

func (r *FreelanceRepo) ListReviewsByUser(userID int) ([]*freelance.FreelanceReview, error) {
	rows, err := r.db.Query(`
		SELECT r.id, r.job_id, r.reviewer_id, r.reviewee_id, r.rating, r.comment, r.created_at,
			   u.name AS reviewer_name
		FROM freelance_reviews r
		JOIN users u ON u.id = r.reviewer_id
		WHERE r.reviewee_id = ?
		ORDER BY r.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*freelance.FreelanceReview
	for rows.Next() {
		review := &freelance.FreelanceReview{}
		if err := rows.Scan(
			&review.ID, &review.JobID, &review.ReviewerID, &review.RevieweeID,
			&review.Rating, &review.Comment, &review.CreatedAt,
			&review.ReviewerName,
		); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, review)
	}
	return reviews, nil
}
