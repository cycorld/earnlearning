package application

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/freelance"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type FreelanceUseCase struct {
	db         *sql.DB
	repo       freelance.Repository
	walletRepo wallet.Repository
	notifUC    *NotificationUseCase
	autoPoster *AutoPoster
}

func NewFreelanceUseCase(db *sql.DB, repo freelance.Repository, wr wallet.Repository, notifUC *NotificationUseCase) *FreelanceUseCase {
	return &FreelanceUseCase{db: db, repo: repo, walletRepo: wr, notifUC: notifUC, autoPoster: NewAutoPoster(db)}
}

// --- Input types ---

type CreateJobInput struct {
	Title          string               `json:"title"`
	Description    string               `json:"description"`
	Budget         int                  `json:"budget"`
	Deadline       string               `json:"deadline"`
	RequiredSkills freelance.SkillsList `json:"required_skills"`
}

type ApplyJobInput struct {
	Proposal string `json:"proposal"`
	Price    int    `json:"price"`
}

type CompleteWorkInput struct {
	Report string `json:"report"`
	Media  string `json:"media"`
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
		MaxWorkers:     1,
		PriceType:      "negotiable",
	}

	id, err := uc.repo.Create(job)
	if err != nil {
		return nil, err
	}

	// Auto-post to 외주마켓 channel
	skills := ""
	if string(input.RequiredSkills) != "" {
		skills = fmt.Sprintf("**필요 스킬:** %s\n", string(input.RequiredSkills))
	}
	content := fmt.Sprintf("## 💼 외주 의뢰: %s\n\n%s\n\n**예산:** %s\n%s\n👉 [자세히 보기](/freelance/%d)",
		input.Title, input.Description, formatMoney(input.Budget), skills, id)
	uc.autoPoster.PostToChannel("market", clientID, content, []string{"외주의뢰", "구인"})

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

	// Notify client about new application
	uc.notify(job.ClientID, notification.NotifJobApplied,
		"새로운 외주 지원이 접수되었습니다",
		fmt.Sprintf("'%s' 의뢰에 새로운 지원이 접수되었습니다.", job.Title),
		"freelance_job", jobID)

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

	// Set freelancer, change job status, reject others
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

	// Notify freelancer
	uc.notify(app.UserID, notification.NotifJobAccepted,
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

	// Auto-post to 외주마켓 channel
	uc.autoPoster.PostToChannel("market", userID, fmt.Sprintf("## ✅ 외주 완료 보고: %s\n\n%s", job.Title, input.Report), []string{"외주완료", "작업보고"})

	// Notify client
	uc.notify(job.ClientID, notification.NotifJobWorkDone,
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
	uc.notify(*job.FreelancerID, notification.NotifJobCompleted,
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
	if job.Status != freelance.StatusOpen && job.Status != freelance.StatusInProgress {
		return fmt.Errorf("취소할 수 없는 상태입니다")
	}

	// Reject all pending/accepted applications
	apps, err := uc.repo.ListApplicationsByJob(jobID)
	if err != nil {
		return err
	}
	for _, app := range apps {
		if app.Status == freelance.AppPending || app.Status == freelance.AppAccepted {
			_ = uc.repo.UpdateApplicationStatus(app.ID, freelance.AppRejected)
		}
	}

	// Refund escrow
	if job.EscrowAmount > 0 {
		clientWallet, err := uc.walletRepo.FindByUserID(job.ClientID)
		if err == nil && clientWallet != nil {
			_ = uc.walletRepo.Credit(clientWallet.ID, job.EscrowAmount, wallet.TxFreelanceEscrow,
				fmt.Sprintf("에스크로 환불: %s", job.Title), "freelance_job", job.ID)
			_ = uc.repo.SetEscrow(jobID, 0)
		}
	}

	// Notify freelancer about cancellation
	if job.FreelancerID != nil {
		uc.notify(*job.FreelancerID, notification.NotifJobCancelled,
			"외주 의뢰가 취소되었습니다",
			fmt.Sprintf("'%s' 의뢰가 취소되었습니다.", job.Title),
			"freelance_job", jobID)
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
	uc.notify(notifyUserID, notification.NotifJobDisputed,
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

func (uc *FreelanceUseCase) notify(userID int, notifType notification.NotifType, title, body, refType string, refID int) {
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(userID, notifType, title, body, refType, refID)
	}
}
