package application

import (
	"fmt"
	"strings"

	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/domain/proposal"
	"github.com/earnlearning/backend/internal/domain/user"
)

// ChatProposalUseCase — #106. 학생이 챗봇으로 정리해 보내는 교수님께의 제안/버그.
// 명명: 기존 `ProposalUseCase` (shareholder 주주총회) 와 구분.
type ChatProposalUseCase struct {
	repo     proposal.Repository
	userRepo user.Repository
	notifUC  *NotificationUseCase
	adminID  int
}

func NewChatProposalUseCase(repo proposal.Repository, userRepo user.Repository, notifUC *NotificationUseCase, adminID int) *ChatProposalUseCase {
	if adminID == 0 {
		adminID = 1
	}
	return &ChatProposalUseCase{repo: repo, userRepo: userRepo, notifUC: notifUC, adminID: adminID}
}

type CreateChatProposalInput struct {
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Attachments []string `json:"attachments"`
}

func (uc *ChatProposalUseCase) Create(userID int, in CreateChatProposalInput) (*proposal.Proposal, error) {
	if !proposal.IsValidCategory(in.Category) {
		return nil, fmt.Errorf("category 는 feature/bug/general 중 하나여야 합니다")
	}
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("제목이 필요합니다")
	}
	if strings.TrimSpace(in.Body) == "" {
		return nil, fmt.Errorf("내용이 필요합니다")
	}
	p := &proposal.Proposal{
		UserID:      userID,
		Category:    proposal.Category(in.Category),
		Title:       in.Title,
		Body:        in.Body,
		Attachments: in.Attachments,
		Status:      proposal.StatusOpen,
	}
	id, err := uc.repo.Create(p)
	if err != nil {
		return nil, err
	}
	p.ID = id

	if uc.notifUC != nil && uc.adminID > 0 && uc.adminID != userID {
		title := "새 제안 도착"
		body := fmt.Sprintf("[%s] %s", in.Category, in.Title)
		if u, _ := uc.userRepo.FindByID(userID); u != nil {
			body = fmt.Sprintf("%s · %s", u.Name, body)
		}
		_ = uc.notifUC.CreateNotification(uc.adminID, notification.NotifProposalSubmitted, title, body, "proposal", id)
	}
	return uc.repo.FindByID(id)
}

func (uc *ChatProposalUseCase) ListMine(userID int, limit int) ([]*proposal.Proposal, error) {
	if limit <= 0 {
		limit = 20
	}
	return uc.repo.List(proposal.Filter{UserID: userID, Limit: limit})
}

func (uc *ChatProposalUseCase) AdminList(filter proposal.Filter) ([]*proposal.Proposal, error) {
	return uc.repo.List(filter)
}

func (uc *ChatProposalUseCase) AdminCount(filter proposal.Filter) (int, error) {
	return uc.repo.Count(filter)
}

func (uc *ChatProposalUseCase) Get(userID int, id int, isAdmin bool) (*proposal.Proposal, error) {
	p, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if !isAdmin && p.UserID != userID {
		return nil, proposal.ErrNotFound
	}
	return p, nil
}

type UpdateChatProposalInput struct {
	Status     string `json:"status"`
	AdminNote  string `json:"admin_note"`
	TicketLink string `json:"ticket_link"`
}

func (uc *ChatProposalUseCase) AdminUpdate(id int, in UpdateChatProposalInput) (*proposal.Proposal, error) {
	if !proposal.IsValidStatus(in.Status) {
		return nil, fmt.Errorf("status 는 open/reviewing/resolved/wontfix 중 하나여야 합니다")
	}
	if err := uc.repo.UpdateStatus(id, proposal.Status(in.Status), in.AdminNote, in.TicketLink); err != nil {
		return nil, err
	}
	return uc.repo.FindByID(id)
}
