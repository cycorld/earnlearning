package llm

import "time"

type Repository interface {
	// Keys
	UpsertKey(k *UserKey) (int, error)
	FindActiveKeyByUserID(userID int) (*UserKey, error)
	FindKeyByProxyKeyID(proxyKeyID int) (*UserKey, error)
	MarkKeyRevoked(id int, revokedAt time.Time) error
	ListAllActiveKeys() ([]*UserKey, error)

	// Daily usage / billing
	UpsertDailyUsage(u *DailyUsage) error
	FindDailyUsage(userID int, usageDate time.Time) (*DailyUsage, error)
	ListDailyUsage(userID int, days int) ([]*DailyUsage, error)
	SumUsageSince(userID int, since time.Time) (cost, debt int, err error)
	SumUsageAllTime(userID int) (cost, debt int, err error)
}
