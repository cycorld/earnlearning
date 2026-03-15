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
	Title                  string               `json:"title"`
	Description            string               `json:"description"`
	Budget                 int                  `json:"budget"`
	Deadline               string               `json:"deadline"`
	RequiredSkills         freelance.SkillsList `json:"required_skills"`
	MaxWorkers             *int                 `json:"max_workers"`
	AutoApproveApplication bool                 `json:"auto_approve_application"`
	PriceType              string               `json:"price_type"`
}

type ApplyJobInput struct {
	Proposal string `json:"proposal"`
	Price    int    `json:"price"`
}

type CompleteWorkInput struct {
	Report        string `json:"report"`
	Media         string `json:"media"`
	ApplicationID int    `json:"application_id"`
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

	maxWorkers := 1 // default: traditional single-worker mode
	if input.MaxWorkers != nil {
		maxWorkers = *input.MaxWorkers
	}

	priceType := "negotiable" // default
	if input.PriceType != "" {
		if input.PriceType != "fixed" && input.PriceType != "negotiable" {
			return nil, freelance.ErrInvalidPriceType
		}
		priceType = input.PriceType
	}

	job := &freelance.FreelanceJob{
		ClientID:               clientID,
		Title:                  input.Title,
		Description:            input.Description,
		Budget:                 input.Budget,
		RequiredSkills:         input.RequiredSkills,
		Status:                 freelance.StatusOpen,
		MaxWorkers:             maxWorkers,
		AutoApproveApplication: input.AutoApproveApplication,
		PriceType:              priceType,
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

func (uc *FreelanceUseCase) ListApplications(jobID int) ([]*freelance.JobApplication, error) {
	_, err := uc.repo.FindByID(jobID)
	if err != nil {
		return nil, err
	}
	return uc.repo.ListApplicationsByJob(jobID)
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

	// Fixed price: worker must apply with exact budget price
	if job.PriceType == "fixed" && input.Price != job.Budget {
		return nil, freelance.ErrFixedPriceMismatch
	}

	// Check max_workers limit (0 = unlimited)
	if job.MaxWorkers > 0 {
		acceptedCount, err := uc.repo.CountAcceptedApplications(jobID)
		if err != nil {
			return nil, err
		}
		if acceptedCount >= job.MaxWorkers {
			return nil, freelance.ErrMaxWorkersReached
		}
	}

	initialStatus := freelance.AppPending
	if job.AutoApproveApplication {
		initialStatus = freelance.AppAccepted
	}

	app := &freelance.JobApplication{
		JobID:    jobID,
		UserID:   userID,
		Proposal: input.Proposal,
		Price:    input.Price,
		Status:   initialStatus,
	}
	id, err := uc.repo.CreateApplication(app)
	if err != nil {
		return nil, err
	}
	app.ID = id

	// If auto-approved in assignment mode, debit escrow per application
	if job.AutoApproveApplication {
		clientWallet, err := uc.walletRepo.FindByUserID(job.ClientID)
		if err != nil {
			return nil, err
		}
		if clientWallet.Balance >= app.Price {
			_ = uc.walletRepo.Debit(clientWallet.ID, app.Price, wallet.TxFreelanceEscrow,
				fmt.Sprintf("과제 에스크로: %s", job.Title), "freelance_job", job.ID)
			_ = uc.repo.SetApplicationEscrow(app.ID, app.Price)
		}
	}

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

	// Accept the application
	if err := uc.repo.UpdateApplicationStatus(applicationID, freelance.AppAccepted); err != nil {
		return err
	}

	isAssignmentMode := job.MaxWorkers != 1

	if isAssignmentMode {
		// Assignment mode: per-application escrow, job stays open
		_ = uc.repo.SetApplicationEscrow(app.ID, app.Price)
	} else {
		// Traditional mode: set freelancer, change job status, reject others
		if err := uc.repo.SetFreelancer(jobID, app.UserID, app.Price); err != nil {
			return err
		}
		if err := uc.repo.SetEscrow(jobID, app.Price); err != nil {
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
	}

	// Create notification for freelancer
	uc.createNotification(app.UserID, "job_accepted",
		"외주 지원이 수락되었습니다",
		fmt.Sprintf("'%s' 작업에 대한 지원이 수락되었습니다.", job.Title),
		"freelance_job", jobID)

	return nil
}

func (uc *FreelanceUseCase) CompleteWork(jobID, userID int, input CompleteWorkInput) error {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return err
	}

	isAssignmentMode := job.MaxWorkers != 1

	if isAssignmentMode {
		// Assignment mode: per-application completion
		if job.Status != freelance.StatusOpen {
			return freelance.ErrJobNotOpen
		}

		// Find the worker's application
		var app *freelance.JobApplication
		if input.ApplicationID > 0 {
			app, err = uc.repo.FindApplicationByID(input.ApplicationID)
			if err != nil {
				return err
			}
		} else {
			app, err = uc.repo.FindApplicationByJobAndUser(jobID, userID)
			if err != nil {
				return err
			}
		}
		if app == nil || app.JobID != jobID {
			return freelance.ErrApplicationNotFound
		}
		if app.UserID != userID {
			return freelance.ErrNotFreelancer
		}
		if app.Status != freelance.AppAccepted {
			return freelance.ErrNotFreelancer
		}

		media := input.Media
		if media == "" {
			media = "[]"
		}
		if err := uc.repo.SetApplicationWorkCompleted(app.ID, input.Report, media); err != nil {
			return err
		}
	} else {
		// Traditional mode
		if job.Status != freelance.StatusInProgress {
			return freelance.ErrJobNotInProgress
		}
		if job.FreelancerID == nil || *job.FreelancerID != userID {
			return freelance.ErrNotFreelancer
		}

		if err := uc.repo.SetWorkCompleted(jobID); err != nil {
			return err
		}

		// Save completion report
		if input.Report != "" {
			media := input.Media
			if media == "" {
				media = "[]"
			}
			uc.repo.SaveCompletionReport(jobID, input.Report, media)
		}
	}

	// Auto-post to 외주마켓 channel
	uc.postToMarketChannel(job, userID, input)

	// Notify client
	uc.createNotification(job.ClientID, "work_completed",
		"외주 작업이 완료되었습니다",
		fmt.Sprintf("'%s' 작업이 완료되었습니다. 검수 후 승인해주세요.", job.Title),
		"freelance_job", jobID)

	return nil
}

func (uc *FreelanceUseCase) postToMarketChannel(job *freelance.FreelanceJob, userID int, input CompleteWorkInput) {
	// Find market channel across all classrooms the user belongs to
	var channelID int
	err := uc.db.QueryRow(`
		SELECT c.id FROM channels c
		JOIN classroom_members cm ON cm.classroom_id = c.classroom_id
		WHERE c.slug = 'market' AND cm.user_id = ?
		LIMIT 1`, userID).Scan(&channelID)
	if err != nil || channelID == 0 {
		return
	}

	report := input.Report
	if report == "" {
		report = "(보고서 없음)"
	}

	content := fmt.Sprintf("## 📋 외주 완료 보고: %s\n\n**의뢰자:** 사용자 #%d\n**합의 금액:** %d원\n\n---\n\n%s",
		job.Title, job.ClientID, job.AgreedPrice, report)

	media := input.Media
	if media == "" {
		media = "[]"
	}

	tags := `["외주완료","작업보고"]`

	_, _ = uc.db.Exec(`
		INSERT INTO posts (channel_id, author_id, content, post_type, media, tags)
		VALUES (?, ?, ?, 'normal', ?, ?)`,
		channelID, userID, content, media, tags)
}

type ApproveJobInput struct {
	ApplicationID int `json:"application_id"`
}

func (uc *FreelanceUseCase) ApproveJob(jobID, userID int, input *ApproveJobInput) error {
	job, err := uc.repo.FindByID(jobID)
	if err != nil {
		return err
	}
	if job.ClientID != userID {
		return freelance.ErrNotClient
	}

	isAssignmentMode := job.MaxWorkers != 1

	if isAssignmentMode {
		// Assignment mode: per-application approval
		if job.Status != freelance.StatusOpen {
			return freelance.ErrJobNotOpen
		}
		if input == nil || input.ApplicationID == 0 {
			return freelance.ErrApplicationNotFound
		}

		app, err := uc.repo.FindApplicationByID(input.ApplicationID)
		if err != nil {
			return err
		}
		if app.JobID != jobID {
			return freelance.ErrApplicationNotFound
		}
		if !app.WorkCompleted {
			return freelance.ErrWorkNotCompleted
		}

		// Transfer per-application escrow to freelancer
		freelancerWallet, err := uc.walletRepo.FindByUserID(app.UserID)
		if err != nil {
			return err
		}
		escrow := app.EscrowAmount
		if escrow == 0 {
			escrow = app.Price
		}
		err = uc.walletRepo.Credit(freelancerWallet.ID, escrow, wallet.TxFreelancePay,
			fmt.Sprintf("과제 대금: %s", job.Title), "freelance_job", job.ID)
		if err != nil {
			return err
		}

		// Clear application escrow
		_ = uc.repo.SetApplicationEscrow(app.ID, 0)

		// Notify freelancer
		uc.createNotification(app.UserID, "job_approved",
			"과제 대금이 지급되었습니다",
			fmt.Sprintf("'%s' 과제가 승인되어 %d원이 지급되었습니다.", job.Title, escrow),
			"freelance_job", jobID)
	} else {
		// Traditional mode
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
	}

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
