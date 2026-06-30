package persistence

import "database/sql"

// DBTX is the read/write surface common to *sql.DB and *sql.Tx. A repo holding a
// DBTX runs the same queries whether it owns the connection (*sql.DB) or borrows a
// caller's transaction (*sql.Tx) — which lets the exchange matching engine wrap an
// entire multi-step settlement in one transaction (#142).
type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// withDBTx runs fn inside a transaction. If x is already a *sql.Tx it reuses it —
// the caller's outer transaction supplies atomicity, so it neither begins nor
// commits. If x is a *sql.DB it opens its own transaction and commits on success or
// rolls back on error. This lets a repo method (e.g. Debit/Credit) stay atomic when
// called standalone yet compose into a larger transaction when called via WithTx.
func withDBTx(x DBTX, fn func(DBTX) error) error {
	if tx, ok := x.(*sql.Tx); ok {
		return fn(tx)
	}
	db := x.(*sql.DB)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
