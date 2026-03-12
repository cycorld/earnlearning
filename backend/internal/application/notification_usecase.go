package application

import (
	"github.com/earnlearning/backend/internal/domain/notification"
	"github.com/earnlearning/backend/internal/infrastructure/push"
)

// WSBroadcaster is an interface for sending WebSocket messages to users.
type WSBroadcaster interface {
	SendToUser(userID int, message interface{})
}

type NotificationUseCase struct {
	notifRepo   notification.Repository
	pushService *push.WebPushService
	wsBroadcast WSBroadcaster
}

func NewNotificationUseCase(repo notification.Repository, pushSvc *push.WebPushService, ws WSBroadcaster) *NotificationUseCase {
	return &NotificationUseCase{
		notifRepo:   repo,
		pushService: pushSvc,
		wsBroadcast: ws,
	}
}

type NotificationListResult struct {
	Data         []*notification.Notification `json:"data"`
	UnreadCount  int                          `json:"unread_count"`
	Pagination   PaginationInfo               `json:"pagination"`
}

type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func (uc *NotificationUseCase) GetNotifications(userID int, isRead *bool, page, limit int) (*NotificationListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, err := uc.notifRepo.GetByUserID(userID, isRead, page, limit)
	if err != nil {
		return nil, err
	}

	unreadCount, err := uc.notifRepo.GetUnreadCount(userID)
	if err != nil {
		return nil, err
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	if notifications == nil {
		notifications = []*notification.Notification{}
	}

	return &NotificationListResult{
		Data:        notifications,
		UnreadCount: unreadCount,
		Pagination: PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *NotificationUseCase) MarkRead(notifID, userID int) error {
	n, err := uc.notifRepo.FindByID(notifID)
	if err != nil {
		return err
	}
	if n.UserID != userID {
		return err
	}
	return uc.notifRepo.MarkRead(notifID)
}

func (uc *NotificationUseCase) MarkAllRead(userID int) error {
	return uc.notifRepo.MarkAllRead(userID)
}

// CreateNotification creates a notification and sends it via WebSocket and Push.
func (uc *NotificationUseCase) CreateNotification(userID int, notifType notification.NotifType, title, body, refType string, refID int) error {
	n := &notification.Notification{
		UserID:        userID,
		NotifType:     notifType,
		Title:         title,
		Body:          body,
		ReferenceType: refType,
		ReferenceID:   refID,
	}

	id, err := uc.notifRepo.Create(n)
	if err != nil {
		return err
	}
	n.ID = id

	// Send via WebSocket
	if uc.wsBroadcast != nil {
		wsMsg := map[string]interface{}{
			"type": "notification",
			"data": n,
		}
		uc.wsBroadcast.SendToUser(userID, wsMsg)
	}

	// Send via Web Push if applicable
	if notification.PushEligibleTypes[notifType] && uc.pushService != nil {
		payload := uc.pushService.FormatPayload(n)
		go uc.pushService.SendToUser(userID, payload)
	}

	return nil
}

type SubscribePushInput struct {
	Endpoint  string `json:"endpoint"`
	P256dh    string `json:"p256dh"`
	Auth      string `json:"auth"`
	UserAgent string `json:"user_agent"`
}

func (uc *NotificationUseCase) SubscribePush(userID int, input SubscribePushInput) error {
	sub := &notification.PushSubscription{
		UserID:    userID,
		Endpoint:  input.Endpoint,
		P256dh:    input.P256dh,
		Auth:      input.Auth,
		UserAgent: input.UserAgent,
	}
	return uc.notifRepo.SaveSubscription(sub)
}

type UnsubscribePushInput struct {
	Endpoint string `json:"endpoint"`
}

func (uc *NotificationUseCase) UnsubscribePush(userID int, input UnsubscribePushInput) error {
	return uc.notifRepo.DeleteSubscription(userID, input.Endpoint)
}

func (uc *NotificationUseCase) GetVAPIDPublicKey() string {
	if uc.pushService != nil {
		return uc.pushService.GetVAPIDPublicKey()
	}
	return ""
}
