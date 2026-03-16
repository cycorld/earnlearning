package application

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/grant"
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type GrantUseCase struct {
	db         *sql.DB
	repo       grant.Repository
	walletRepo wallet.Repository
	notifUC    *NotificationUseCase
	autoPoster *AutoPoster
}

func NewGrantUseCase(db *sql.DB, repo grant.Repository, wr wallet.Repository, notifUC *NotificationUseCase) *GrantUseCase {
	return &GrantUseCase{db: db, repo: repo, walletRepo: wr, notifUC: notifUC, autoPoster: NewAutoPoster(db)}
}

// --- Input types ---

type CreateGrantInput struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Reward        int    `json:"reward"`
	MaxApplicants int    `json:"max_applicants"`
}

type ApplyGrantInput struct {
	Proposal string `json:"proposal"`
}

// --- Use case methods ---

func (uc *GrantUseCase) CreateGrant(input CreateGrantInput, adminID int) (*grant.Grant, error) {
	if input.Reward <= 0 {
		return nil, fmt.Errorf("보상은 0보다 커야 합니다")
	}

	g := &grant.Grant{
		AdminID:       adminID,
		Title:         input.Title,
		Description:   input.Description,
		Reward:        input.Reward,
		MaxApplicants: input.MaxApplicants,
		Status:        grant.StatusOpen,
	}

	id, err := uc.repo.Create(g)
	if err != nil {
		return nil, err
	}

	// Auto-post to 과제 channel
	content := fmt.Sprintf("## 📋 새 정부과제 공고: %s\n\n%s\n\n**보상:** %s | **모집 인원:** %d명\n\n👉 [지원하러 가기](/grants/%d)",
		input.Title, input.Description, formatMoney(input.Reward), input.MaxApplicants, id)
	uc.autoPoster.PostToChannelAsAdmin("assignment", content, []string{"정부과제", "공고"})

	return uc.repo.FindByID(id)
}

func (uc *GrantUseCase) GetGrant(grantID int) (*grant.Grant, error) {
	g, err := uc.repo.FindByID(grantID)
	if err != nil {
		return nil, err
	}
	apps, err := uc.repo.ListApplicationsByGrant(grantID)
	if err != nil {
		return nil, err
	}
	g.Applications = apps
	return g, nil
}

func (uc *GrantUseCase) ListGrants(status string, page, limit int) ([]*grant.Grant, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	filter := grant.GrantFilter{Status: status}
	return uc.repo.List(filter, page, limit)
}

func (uc *GrantUseCase) ApplyToGrant(grantID int, input ApplyGrantInput, userID int) (*grant.GrantApplication, error) {
	g, err := uc.repo.FindByID(grantID)
	if err != nil {
		return nil, err
	}
	if g.Status != grant.StatusOpen {
		return nil, grant.ErrGrantNotOpen
	}
	if g.AdminID == userID {
		return nil, grant.ErrCannotApplyOwnGrant
	}

	existing, err := uc.repo.FindApplicationByGrantAndUser(grantID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, grant.ErrAlreadyApplied
	}

	app := &grant.GrantApplication{
		GrantID:  grantID,
		UserID:   userID,
		Proposal: input.Proposal,
		Status:   grant.AppPending,
	}
	id, err := uc.repo.CreateApplication(app)
	if err != nil {
		return nil, err
	}
	app.ID = id

	// Notify admin about new application
	uc.notify(g.AdminID, notification.NotifGrantApplied,
		"새로운 과제 지원이 접수되었습니다",
		fmt.Sprintf("'%s' 과제에 새로운 지원이 접수되었습니다.", g.Title),
		"grant", grantID)

	return app, nil
}

func (uc *GrantUseCase) ApproveApplication(grantID, applicationID, adminID int) error {
	g, err := uc.repo.FindByID(grantID)
	if err != nil {
		return err
	}
	if g.AdminID != adminID {
		return grant.ErrNotAdmin
	}

	app, err := uc.repo.FindApplicationByID(applicationID)
	if err != nil {
		return err
	}
	if app.GrantID != grantID {
		return grant.ErrApplicationNotFound
	}
	if app.Status != grant.AppPending {
		return grant.ErrNotApproved
	}

	// Update application status
	if err := uc.repo.UpdateApplicationStatus(applicationID, grant.AppApproved); err != nil {
		return err
	}

	// Credit reward to applicant's wallet from admin's wallet
	adminWallet, err := uc.walletRepo.FindByUserID(adminID)
	if err != nil {
		return err
	}

	// Debit from admin
	err = uc.walletRepo.Debit(adminWallet.ID, g.Reward, wallet.TxFreelanceEscrow,
		fmt.Sprintf("정부과제 보상: %s", g.Title), "grant", g.ID)
	if err != nil {
		return err
	}

	// Credit to applicant
	applicantWallet, err := uc.walletRepo.FindByUserID(app.UserID)
	if err != nil {
		return err
	}
	err = uc.walletRepo.Credit(applicantWallet.ID, g.Reward, wallet.TxFreelancePay,
		fmt.Sprintf("정부과제 보상: %s", g.Title), "grant", g.ID)
	if err != nil {
		return err
	}

	// Notify applicant
	uc.notify(app.UserID, notification.NotifGrantApproved,
		"정부과제가 승인되었습니다",
		fmt.Sprintf("'%s' 과제가 승인되어 %d원이 지급되었습니다.", g.Title, g.Reward),
		"grant", grantID)

	return nil
}

func (uc *GrantUseCase) CloseGrant(grantID, adminID int) error {
	g, err := uc.repo.FindByID(grantID)
	if err != nil {
		return err
	}
	if g.AdminID != adminID {
		return grant.ErrNotAdmin
	}
	if g.Status != grant.StatusOpen {
		return grant.ErrGrantNotOpen
	}

	if err := uc.repo.UpdateStatus(grantID, grant.StatusClosed); err != nil {
		return err
	}

	// Notify pending applicants about closure
	apps, err := uc.repo.ListApplicationsByGrant(grantID)
	if err == nil {
		for _, app := range apps {
			if app.Status == grant.AppPending {
				uc.notify(app.UserID, notification.NotifGrantClosed,
					"정부과제 모집이 종료되었습니다",
					fmt.Sprintf("'%s' 과제 모집이 종료되었습니다.", g.Title),
					"grant", grantID)
			}
		}
	}

	return nil
}

func (uc *GrantUseCase) notify(userID int, notifType notification.NotifType, title, body, refType string, refID int) {
	if uc.notifUC != nil {
		_ = uc.notifUC.CreateNotification(userID, notifType, title, body, refType, refID)
	}
}
