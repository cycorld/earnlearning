package application

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/grant"
	"github.com/earnlearning/backend/internal/domain/wallet"
)

type GrantUseCase struct {
	db         *sql.DB
	repo       grant.Repository
	walletRepo wallet.Repository
}

func NewGrantUseCase(db *sql.DB, repo grant.Repository, wr wallet.Repository) *GrantUseCase {
	return &GrantUseCase{db: db, repo: repo, walletRepo: wr}
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
	uc.createNotification(app.UserID, "grant_approved",
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

	return uc.repo.UpdateStatus(grantID, grant.StatusClosed)
}

func (uc *GrantUseCase) createNotification(userID int, notifType, title, body, refType string, refID int) {
	_, _ = uc.db.Exec(`
		INSERT INTO notifications (user_id, notif_type, title, body, reference_type, reference_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, notifType, title, body, refType, refID)
}
