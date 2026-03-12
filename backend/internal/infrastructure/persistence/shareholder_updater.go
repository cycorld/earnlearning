package persistence

import (
	"database/sql"
	"fmt"
)

// ShareholderUpdaterImpl implements application.ShareholderUpdater
// for direct shareholder share updates needed by the exchange matching engine.
type ShareholderUpdaterImpl struct {
	db *sql.DB
}

func NewShareholderUpdater(db *sql.DB) *ShareholderUpdaterImpl {
	return &ShareholderUpdaterImpl{db: db}
}

func (u *ShareholderUpdaterImpl) UpdateShareholderShares(companyID, userID, shares int) error {
	if shares <= 0 {
		// Delete shareholder record if shares drop to 0 or below
		_, err := u.db.Exec("DELETE FROM shareholders WHERE company_id = ? AND user_id = ?", companyID, userID)
		if err != nil {
			return fmt.Errorf("delete shareholder: %w", err)
		}
		return nil
	}

	_, err := u.db.Exec("UPDATE shareholders SET shares = ? WHERE company_id = ? AND user_id = ?", shares, companyID, userID)
	if err != nil {
		return fmt.Errorf("update shareholder shares: %w", err)
	}
	return nil
}
