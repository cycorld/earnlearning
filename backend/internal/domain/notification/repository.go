package notification

type Repository interface {
	// Notification operations
	Create(n *Notification) (int, error)
	FindByID(id int) (*Notification, error)
	// notifType: 빈 문자열이면 전체, 값이 있으면 해당 notif_type만 (#132 멘션 탭)
	GetByUserID(userID int, isRead *bool, notifType string, page, limit int) ([]*Notification, int, error)
	MarkRead(id int) error
	MarkAllRead(userID int) error
	GetUnreadCount(userID int) (int, error)

	// Push subscription operations
	SaveSubscription(sub *PushSubscription) error
	DeleteSubscription(userID int, endpoint string) error
	GetSubscriptionsByUserID(userID int) ([]*PushSubscription, error)
	DeleteSubscriptionByEndpoint(endpoint string) error

	// User query for announcements
	GetApprovedUserIDs() ([]int, error)

	// Email preference operations
	GetEmailPreference(userID int) (*EmailPreference, error)
	SaveEmailPreference(pref *EmailPreference) error

	// User email query (for sending emails)
	GetUserEmail(userID int) (string, error)
}
