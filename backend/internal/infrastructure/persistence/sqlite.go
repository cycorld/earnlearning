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
		`ALTER TABLE freelance_jobs ADD COLUMN price_type TEXT NOT NULL DEFAULT 'negotiable'`,
		`ALTER TABLE companies ADD COLUMN service_url TEXT DEFAULT ''`,
	}
	for _, stmt := range alterStatements {
		db.Exec(stmt) // ignore "duplicate column" errors
	}

	// Create email_preferences table (idempotent)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS email_preferences (
		user_id INTEGER PRIMARY KEY REFERENCES users(id),
		email_enabled INTEGER NOT NULL DEFAULT 1
	)`)
	if err != nil {
		return fmt.Errorf("create email_preferences table: %w", err)
	}

	// Create grants tables (idempotent)
	grantTables := []string{
		`CREATE TABLE IF NOT EXISTS grants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			admin_id INTEGER NOT NULL REFERENCES users(id),
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			reward INTEGER NOT NULL DEFAULT 0,
			max_applicants INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'open',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS grant_applications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			grant_id INTEGER NOT NULL REFERENCES grants(id),
			user_id INTEGER NOT NULL REFERENCES users(id),
			proposal TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(grant_id, user_id)
		)`,
	}
	for _, stmt := range grantTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create grant tables: %w", err)
		}
	}

	// Create OAuth tables (idempotent)
	oauthTables := []string{
		`CREATE TABLE IF NOT EXISTS oauth_clients (
			id TEXT PRIMARY KEY,
			secret_hash TEXT NOT NULL,
			user_id INTEGER REFERENCES users(id),
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			redirect_uris TEXT DEFAULT '[]',
			scopes TEXT DEFAULT '[]',
			status TEXT DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
			code TEXT PRIMARY KEY,
			client_id TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			redirect_uri TEXT NOT NULL,
			scopes TEXT DEFAULT '[]',
			code_challenge TEXT DEFAULT '',
			code_challenge_method TEXT DEFAULT '',
			expires_at DATETIME NOT NULL,
			used INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			access_token TEXT UNIQUE,
			refresh_token TEXT UNIQUE,
			scopes TEXT DEFAULT '[]',
			expires_at DATETIME NOT NULL,
			revoked INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range oauthTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create oauth tables: %w", err)
		}
	}

	// User databases (학생 개인 PG DB 프로비저닝 참조)
	userDBTables := []string{
		`CREATE TABLE IF NOT EXISTS user_databases (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id      INTEGER NOT NULL REFERENCES users(id),
			project_name TEXT NOT NULL,
			db_name      TEXT NOT NULL UNIQUE,
			pg_username  TEXT NOT NULL UNIQUE,
			host         TEXT NOT NULL,
			port         INTEGER NOT NULL DEFAULT 6432,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_rotated DATETIME,
			UNIQUE(user_id, project_name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_databases_user ON user_databases(user_id)`,
	}
	for _, stmt := range userDBTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create user_databases tables: %w", err)
		}
	}

	// DM tables
	dmTables := []string{
		`CREATE TABLE IF NOT EXISTS dm_messages (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_id   INTEGER NOT NULL REFERENCES users(id),
			receiver_id INTEGER NOT NULL REFERENCES users(id),
			content     TEXT NOT NULL,
			is_read     INTEGER NOT NULL DEFAULT 0,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dm_sender_receiver ON dm_messages(sender_id, receiver_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_dm_receiver_unread ON dm_messages(receiver_id, is_read)`,
	}
	for _, stmt := range dmTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create dm tables: %w", err)
		}
	}

	return nil
}
