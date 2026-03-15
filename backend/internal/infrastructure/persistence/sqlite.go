package persistence

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_init.sql
var migrationSQL string

func NewDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db.SetMaxOpenConns(1)

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Incremental migrations (safe to re-run; errors ignored for existing columns)
	alterStatements := []string{
		`ALTER TABLE freelance_jobs ADD COLUMN completion_report TEXT DEFAULT ''`,
		`ALTER TABLE freelance_jobs ADD COLUMN completion_media TEXT DEFAULT '[]'`,
		`ALTER TABLE freelance_jobs ADD COLUMN max_workers INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE freelance_jobs ADD COLUMN auto_approve_application INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE job_applications ADD COLUMN escrow_amount INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE job_applications ADD COLUMN work_completed INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE job_applications ADD COLUMN completion_report TEXT DEFAULT ''`,
		`ALTER TABLE job_applications ADD COLUMN completion_media TEXT DEFAULT '[]'`,
	}
	for _, stmt := range alterStatements {
		db.Exec(stmt) // ignore "duplicate column" errors
	}

	return nil
}
