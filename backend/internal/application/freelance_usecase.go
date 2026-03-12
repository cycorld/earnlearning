package application

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/freelance"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type FreelanceUseCase struct {
	db         *sql.DB
	repo       freelance.Repository
	walletRepo wallet.Repository
}

func NewFreelanceUseCase(db *sql.DB, repo freelance.Repository, wr wallet.Repository) *FreelanceUseCase {
	return &FreelanceUseCase{db: db, repo: repo, walletRepo: wr}
}

// --- Input types ---

type CreateJobInput struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Budget         int    `json:"budget"`
	Deadline       string `json:"deadline"`
	RequiredSkills string `json:"required_skills"`
}

type ApplyJobInput struct {
	Proposal string `json:"proposal"`
	Price    int    `json:"price"`
}

type ReviewJobInput struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// --- Use case methods ---

func (uc *FreelanceUseCase) ListJobs(status, skills string, minBudget, page, limit int) ([]*freelance.FreelanceJob, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	filter := freelance.JobFilter{
		Status:    status,
		Skills:    skills,
		MinBudget: minBudget,
	}
	return uc.repo.List(filter, page, limit)
}

func (uc *FreelanceUseCase) CreateJob(input CreateJobInput, clientID int) (*freelance.FreelanceJob, error) {
	if input.Budget <= 0 {
		return nil, fmt.Errorf("예산은 0보다 커야 합니다")
	}
	job := &freelance.FreelanceJob{
		ClientID:       clientID,
		Title:          input.Title,
		Description:    input.Description,
		Budget:         input.Budget,
		RequiredSkills: input.RequiredSkills,
		Status:         freelance.StatusOpen,
	}

	id, err := uc.repo.Create(job)
	if err != nil {
		return nil, err
	}
	return uc.repo.FindByID(id)
}

func (uc *FreelanceUseCase) GetJob(jobID, userID int) (*freelance.FreelanceJob, error) {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return nil, err
	}

	// Only the client can see applications
	if job.ClientID == userID {
		apps, err := uc.repo.ListApplicationsByJob(jobID)
		if err != nil {
			return nil, err
		}
		job.Applications = apps
	}
	return job, nil
}

func (uc *FreelanceUseCase) ApplyToJob(jobID int, input ApplyJobInput, userID int) (*freelance.JobApplication, error) {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != freelance.StatusOpen {
		return nil, freelance.ErrJobNotOpen
	}
	if job.ClientID == userID {
		return nil, freelance.ErrCannotApplyOwnJob
	}

	existing, err := uc.repo.FindApplicationByJobAndUser(jobID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, freelance.ErrAlreadyApplied
	}

	app := &freelance.JobApplication{
		JobID:    jobID,
		UserID:   userID,
		Proposal: input.Proposal,
		Price:    input.Price,
		Status:   freelance.AppPending,
	}
	id, err := uc.repo.CreateApplication(app)
	if err != nil {
		return nil, err
	}
	app.ID = id
	return app, nil
}

func (uc *FreelanceUseCase) AcceptApplication(jobID, applicationID, userID int) error {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return err
	}
	if job.ClientID != userID {
		return freelance.ErrNotClient
	}
	if job.Status != freelance.StatusOpen {
		return freelance.ErrJobNotOpen
	}

	app, err := uc.repo.FindApplicationByID(applicationID)
	if err != nil {
		return err
	}
	if app.JobID != jobID {
		return freelance.ErrApplicationNotFound
	}

	// Check client balance >= agreed price
	clientWallet, err := uc.walletRepo.FindByUserID(userID)
	if err != nil {
		return freelance.ErrInsufficientFunds
	}
	if clientWallet.Balance < app.Price {
		return freelance.ErrInsufficientFunds
	}

	// Debit escrow from client wallet
	err = uc.walletRepo.Debit(clientWallet.ID, app.Price, wallet.TxFreelanceEscrow,
		fmt.Sprintf("외주 에스크로: %s", job.Title), "freelance_job", job.ID)
	if err != nil {
		return err
	}

	// Update job: set freelancer, agreed_price, escrow, status=in_progress
	if err := uc.repo.SetFreelancer(jobID, app.UserID, app.Price); err != nil {
		return err
	}
	if err := uc.repo.SetEscrow(jobID, app.Price); err != nil {
		return err
	}

	// Accept this application, reject others
	if err := uc.repo.UpdateApplicationStatus(applicationID, freelance.AppAccepted); err != nil {
		return err
	}

	// Reject other pending applications
	apps, err := uc.repo.ListApplicationsByJob(jobID)
	if err != nil {
		return err
	}
	for _, a := range apps {
		if a.ID != applicationID && a.Status == freelance.AppPending {
			_ = uc.repo.UpdateApplicationStatus(a.ID, freelance.AppRejected)
		}
	}

	// Create notification for freelancer
	uc.createNotification(app.UserID, "job_accepted",
		"외주 지원이 수락되었습니다",
		fmt.Sprintf("'%s' 작업에 대한 지원이 수락되었습니다.", job.Title),
		"freelance_job", jobID)

	return nil
}

func (uc *FreelanceUseCase) CompleteWork(jobID, userID int) error {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return err
	}
	if job.Status != freelance.StatusInProgress {
		return freelance.ErrJobNotInProgress
	}
	if job.FreelancerID == nil || *job.FreelancerID != userID {
		return freelance.ErrNotFreelancer
	}

	if err := uc.repo.SetWorkCompleted(jobID); err != nil {
		return err
	}

	// Notify client
	uc.createNotification(job.ClientID, "work_completed",
		"외주 작업이 완료되었습니다",
		fmt.Sprintf("'%s' 작업이 완료되었습니다. 검수 후 승인해주세요.", job.Title),
		"freelance_job", jobID)

	return nil
}

func (uc *FreelanceUseCase) ApproveJob(jobID, userID int) error {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return err
	}
	if job.ClientID != userID {
		return freelance.ErrNotClient
	}
	if job.Status != freelance.StatusInProgress {
		return freelance.ErrJobNotInProgress
	}
	if !job.WorkCompleted {
		return freelance.ErrWorkNotCompleted
	}

	// Transfer escrow to freelancer wallet
	freelancerWallet, err := uc.walletRepo.FindByUserID(*job.FreelancerID)
	if err != nil {
		return err
	}
	err = uc.walletRepo.Credit(freelancerWallet.ID, job.EscrowAmount, wallet.TxFreelancePay,
		fmt.Sprintf("외주 대금: %s", job.Title), "freelance_job", job.ID)
	if err != nil {
		return err
	}

	// Clear escrow and set completed
	if err := uc.repo.SetEscrow(jobID, 0); err != nil {
		return err
	}
	if err := uc.repo.SetCompleted(jobID); err != nil {
		return err
	}

	// Notify freelancer
	uc.createNotification(*job.FreelancerID, "job_approved",
		"외주 대금이 지급되었습니다",
		fmt.Sprintf("'%s' 작업이 승인되어 %d원이 지급되었습니다.", job.Title, job.EscrowAmount),
		"freelance_job", jobID)

	return nil
}

func (uc *FreelanceUseCase) CancelJob(jobID, userID int) error {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return err
	}
	if job.ClientID != userID {
		return freelance.ErrNotClient
	}
	if job.Status != freelance.StatusOpen {
		return freelance.ErrJobNotOpen
	}
	return uc.repo.UpdateStatus(jobID, freelance.StatusCancelled)
}

func (uc *FreelanceUseCase) DisputeJob(jobID, userID int) error {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return err
	}
	if job.Status != freelance.StatusInProgress {
		return freelance.ErrJobNotInProgress
	}

	// Either party can dispute
	isClient := job.ClientID == userID
	isFreelancer := job.FreelancerID != nil && *job.FreelancerID == userID
	if !isClient && !isFreelancer {
		return freelance.ErrNotParticipant
	}

	if err := uc.repo.UpdateStatus(jobID, freelance.StatusDisputed); err != nil {
		return err
	}

	// Notify the other party
	var notifyUserID int
	if isClient {
		notifyUserID = *job.FreelancerID
	} else {
		notifyUserID = job.ClientID
	}
	uc.createNotification(notifyUserID, "job_disputed",
		"외주 작업에 분쟁이 제기되었습니다",
		fmt.Sprintf("'%s' 작업에 분쟁이 제기되었습니다.", job.Title),
		"freelance_job", jobID)

	return nil
}

func (uc *FreelanceUseCase) ReviewJob(jobID int, input ReviewJobInput, userID int) (*freelance.FreelanceReview, error) {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != freelance.StatusCompleted {
		return nil, freelance.ErrJobNotCompleted
	}
	if input.Rating < 1 || input.Rating > 5 {
		return nil, freelance.ErrInvalidRating
	}

	// Determine reviewer and reviewee
	isClient := job.ClientID == userID
	isFreelancer := job.FreelancerID != nil && *job.FreelancerID == userID
	if !isClient && !isFreelancer {
		return nil, freelance.ErrNotParticipant
	}

	// Check if already reviewed
	existing, err := uc.repo.FindReviewByJobAndReviewer(jobID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, freelance.ErrAlreadyReviewed
	}

	var revieweeID int
	if isClient {
		revieweeID = *job.FreelancerID
	} else {
		revieweeID = job.ClientID
	}

	review := &freelance.FreelanceReview{
		JobID:      jobID,
		ReviewerID: userID,
		RevieweeID: revieweeID,
		Rating:     input.Rating,
		Comment:    input.Comment,
	}
	id, err := uc.repo.CreateReview(review)
	if err != nil {
		return nil, err
	}
	review.ID = id
	return review, nil
}

func (uc *FreelanceUseCase) createNotification(userID int, notifType, title, body, refType string, refID int) {
	_, _ = uc.db.Exec(`
		INSERT INTO notifications (user_id, notif_type, title, body, reference_type, reference_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, notifType, title, body, refType, refID)
}
