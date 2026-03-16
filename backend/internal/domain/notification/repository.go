package notification

type Repository interface {
	// Notification operations
	Create(n *Notification) (int, error)
	FindByID(id int) (*Notification, error)
	GetByUserID(userID int, isRead *bool, page, limit int) ([]*Notification, int, error)
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
