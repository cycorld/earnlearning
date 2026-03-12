package persistence

import (
	"database/sql"
	"fmt"

	"github.com/earnlearning/backend/internal/domain/notification"
)

type NotificationRepo struct {
	db *sql.DB
}

func NewNotificationRepo(db *sql.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(n *notification.Notification) (int, error) {
	result, err := r.db.Exec(`
		INSERT INTO notifications (user_id, notif_type, title, body, reference_type, reference_id, is_read)
		VALUES (?, ?, ?, ?, ?, ?, 0)`,
		n.UserID, n.NotifType, n.Title, n.Body, n.ReferenceType, n.ReferenceID,
	)
	if err != nil {
		return 0, fmt.Errorf("create notification: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *NotificationRepo) FindByID(id int) (*notification.Notification, error) {
	n := &notification.Notification{}
	err := r.db.QueryRow(`
		SELECT id, user_id, notif_type, title, body, reference_type, reference_id, is_read, created_at
		FROM notifications WHERE id = ?`, id,
	).Scan(&n.ID, &n.UserID, &n.NotifType, &n.Title, &n.Body, &n.ReferenceType, &n.ReferenceID, &n.IsRead, &n.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("알림을 찾을 수 없습니다")
		}
		return nil, fmt.Errorf("find notification: %w", err)
	}
	return n, nil
}

func (r *NotificationRepo) GetByUserID(userID int, isRead *bool, page, limit int) ([]*notification.Notification, int, error) {
	baseQuery := "FROM notifications WHERE user_id = ?"
	args := []interface{}{userID}

	if isRead != nil {
		baseQuery += " AND is_read = ?"
		if *isRead {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	offset := (page - 1) * limit
	queryArgs := append(args, limit, offset)
	rows, err := r.db.Query(`
		SELECT id, user_id, notif_type, title, body, reference_type, reference_id, is_read, created_at
		`+baseQuery+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, queryArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*notification.Notification
	for rows.Next() {
		n := &notification.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.NotifType, &n.Title, &n.Body, &n.ReferenceType, &n.ReferenceID, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, n)
	}

	if notifications == nil {
		notifications = []*notification.Notification{}
	}

	return notifications, total, nil
}

func (r *NotificationRepo) MarkRead(id int) error {
	_, err := r.db.Exec("UPDATE notifications SET is_read = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(userID int) error {
	_, err := r.db.Exec("UPDATE notifications SET is_read = 1 WHERE user_id = ? AND is_read = 0", userID)
	if err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	return nil
}

func (r *NotificationRepo) GetUnreadCount(userID int) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = 0", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) SaveSubscription(sub *notification.PushSubscription) error {
	_, err := r.db.Exec(`
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, user_agent)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET user_id = ?, p256dh = ?, auth = ?, user_agent = ?`,
		sub.UserID, sub.Endpoint, sub.P256dh, sub.Auth, sub.UserAgent,
		sub.UserID, sub.P256dh, sub.Auth, sub.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("save subscription: %w", err)
	}
	return nil
}

func (r *NotificationRepo) DeleteSubscription(userID int, endpoint string) error {
	_, err := r.db.Exec("DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?", userID, endpoint)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}

func (r *NotificationRepo) GetSubscriptionsByUserID(userID int) ([]*notification.PushSubscription, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, endpoint, p256dh, auth, user_agent, created_at
		FROM push_subscriptions WHERE user_id = ?`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*notification.PushSubscription
	for rows.Next() {
		s := &notification.PushSubscription{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.UserAgent, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *NotificationRepo) DeleteSubscriptionByEndpoint(endpoint string) error {
	_, err := r.db.Exec("DELETE FROM push_subscriptions WHERE endpoint = ?", endpoint)
	if err != nil {
		return fmt.Errorf("delete subscription by endpoint: %w", err)
	}
	return nil
}
