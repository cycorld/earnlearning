package application

import (
	"errors"
	"time"

	"github.com/earnlearning/backend/internal/domain/dm"
	"github.com/earnlearning/backend/internal/domain/user"
)

type DMUseCase struct {
	repo     dm.Repository
	userRepo user.Repository
	hub      WSBroadcaster
}

func NewDMUseCase(repo dm.Repository, userRepo user.Repository, hub WSBroadcaster) *DMUseCase {
	return &DMUseCase{repo: repo, userRepo: userRepo, hub: hub}
}

type SendDMInput struct {
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
}

var (
	ErrCannotDMSelf  = errors.New("자기 자신에게 메시지를 보낼 수 없습니다")
	ErrEmptyMessage  = errors.New("메시지 내용을 입력하세요")
	ErrUserNotFound  = errors.New("사용자를 찾을 수 없습니다")
)

func (uc *DMUseCase) SendMessage(senderID int, input SendDMInput) (*dm.Message, error) {
	if senderID == input.ReceiverID {
		return nil, ErrCannotDMSelf
	}
	if input.Content == "" {
		return nil, ErrEmptyMessage
	}

	// Verify receiver exists
	if _, err := uc.userRepo.FindByID(input.ReceiverID); err != nil {
		return nil, ErrUserNotFound
	}

	msg := &dm.Message{
		SenderID:   senderID,
		ReceiverID: input.ReceiverID,
		Content:    input.Content,
	}
	id, err := uc.repo.SendMessage(msg)
	if err != nil {
		return nil, err
	}
	msg.ID = id
	msg.CreatedAt = time.Now()

	// Send real-time notification via WebSocket
	if uc.hub != nil {
		uc.hub.SendToUser(input.ReceiverID, map[string]interface{}{
			"event": "dm",
			"data":  msg,
		})
	}

	return msg, nil
}

func (uc *DMUseCase) GetMessages(userID, peerID, limit, beforeID int) ([]*dm.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return uc.repo.GetMessages(userID, peerID, limit, beforeID)
}

func (uc *DMUseCase) GetConversations(userID int) ([]*dm.Conversation, error) {
	return uc.repo.GetConversations(userID)
}

func (uc *DMUseCase) MarkAsRead(userID, peerID int) error {
	return uc.repo.MarkAsRead(userID, peerID)
}

func (uc *DMUseCase) GetUnreadCount(userID int) (int, error) {
	return uc.repo.GetUnreadCount(userID)
}
