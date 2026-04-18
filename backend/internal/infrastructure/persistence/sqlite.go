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

	// Create company_disclosures table (idempotent)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS company_disclosures (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		company_id  INTEGER NOT NULL REFERENCES companies(id),
		author_id   INTEGER NOT NULL REFERENCES users(id),
		content     TEXT NOT NULL,
		period_from DATE NOT NULL,
		period_to   DATE NOT NULL,
		status      TEXT NOT NULL DEFAULT 'pending',
		reward      INTEGER DEFAULT 0,
		admin_note  TEXT DEFAULT '',
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create company_disclosures: %w", err)
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

	// Shareholder proposals & votes (#022 주주총회 투표 시스템)
	proposalTables := []string{
		`CREATE TABLE IF NOT EXISTS shareholder_proposals (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			company_id     INTEGER NOT NULL REFERENCES companies(id),
			proposer_id    INTEGER NOT NULL REFERENCES users(id),
			proposal_type  TEXT NOT NULL DEFAULT 'general'
			               CHECK (proposal_type IN ('general', 'liquidation')),
			title          TEXT NOT NULL,
			description    TEXT NOT NULL DEFAULT '',
			pass_threshold INTEGER NOT NULL DEFAULT 50,
			status         TEXT NOT NULL DEFAULT 'active'
			               CHECK (status IN ('active', 'passed', 'rejected', 'cancelled', 'executed')),
			start_date     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			end_date       DATETIME NOT NULL,
			result_note    TEXT NOT NULL DEFAULT '',
			created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			closed_at      DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shareholder_proposals_company ON shareholder_proposals(company_id, status)`,
		`CREATE TABLE IF NOT EXISTS shareholder_votes (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			proposal_id    INTEGER NOT NULL REFERENCES shareholder_proposals(id),
			user_id        INTEGER NOT NULL REFERENCES users(id),
			choice         TEXT NOT NULL CHECK (choice IN ('yes', 'no')),
			shares_at_vote INTEGER NOT NULL,
			created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(proposal_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shareholder_votes_proposal ON shareholder_votes(proposal_id)`,
	}
	for _, stmt := range proposalTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create proposal tables: %w", err)
		}
	}

	// LLM API keys + daily billing usage (#068 LLM API 키 발급 + 자정 과금)
	llmTables := []string{
		`CREATE TABLE IF NOT EXISTS llm_api_keys (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id           INTEGER NOT NULL REFERENCES users(id),
			proxy_student_id  INTEGER NOT NULL,
			proxy_key_id      INTEGER NOT NULL,
			prefix            TEXT NOT NULL,
			label             TEXT NOT NULL DEFAULT '',
			issued_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			revoked_at        DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_api_keys_user ON llm_api_keys(user_id, revoked_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_api_keys_proxy ON llm_api_keys(proxy_key_id)`,
		`CREATE TABLE IF NOT EXISTS llm_daily_usage (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id           INTEGER NOT NULL REFERENCES users(id),
			usage_date        DATE NOT NULL,
			prompt_tokens     INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			cache_hits        INTEGER NOT NULL DEFAULT 0,
			cache_tokens      INTEGER NOT NULL DEFAULT 0,
			requests          INTEGER NOT NULL DEFAULT 0,
			cost_krw          INTEGER NOT NULL DEFAULT 0,
			debited_krw       INTEGER NOT NULL DEFAULT 0,
			debt_krw          INTEGER NOT NULL DEFAULT 0,
			billed_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, usage_date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_llm_daily_usage_user_date ON llm_daily_usage(user_id, usage_date DESC)`,
	}
	for _, stmt := range llmTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create llm tables: %w", err)
		}
	}
	// cache_tokens 는 #068 중간 업데이트로 추가됨. 기존에 테이블이 이미 만들어진
	// 환경(stage 등)을 위해 ALTER 로 보정한다. 중복 컬럼 에러는 무시.
	db.Exec(`ALTER TABLE llm_daily_usage ADD COLUMN cache_tokens INTEGER NOT NULL DEFAULT 0`)

	// Chatbot TA (#071)
	chatTables := []string{
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id           INTEGER NOT NULL REFERENCES users(id),
			title             TEXT NOT NULL DEFAULT '',
			active_skill_id   INTEGER,
			tokens_used       INTEGER NOT NULL DEFAULT 0,
			created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_message_at   DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_recent ON chat_sessions(user_id, last_message_at DESC)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id         INTEGER NOT NULL REFERENCES chat_sessions(id),
			role               TEXT NOT NULL CHECK (role IN ('system','user','assistant','tool')),
			content            TEXT NOT NULL,
			reasoning_content  TEXT DEFAULT '',
			model              TEXT DEFAULT '',
			prompt_tokens      INTEGER DEFAULT 0,
			completion_tokens  INTEGER DEFAULT 0,
			cache_tokens       INTEGER DEFAULT 0,
			tool_calls         TEXT DEFAULT '[]',
			tool_call_id       TEXT DEFAULT '',
			created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS chat_skills (
			id                        INTEGER PRIMARY KEY AUTOINCREMENT,
			slug                      TEXT UNIQUE NOT NULL,
			name                      TEXT NOT NULL,
			description               TEXT NOT NULL DEFAULT '',
			system_prompt             TEXT NOT NULL,
			default_model             TEXT NOT NULL DEFAULT 'qwen-chat',
			default_reasoning_effort  TEXT DEFAULT '',
			tools_allowed             TEXT NOT NULL DEFAULT '[]',
			wiki_scope                TEXT NOT NULL DEFAULT '[]',
			enabled                   INTEGER NOT NULL DEFAULT 1,
			admin_only                INTEGER NOT NULL DEFAULT 0,
			created_by                INTEGER REFERENCES users(id),
			updated_at                DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// Wiki meta (git md 파일이 source-of-truth, 이건 파일 → FTS5 인덱스 메타)
		`CREATE TABLE IF NOT EXISTS chat_wiki_meta (
			slug            TEXT PRIMARY KEY,
			path            TEXT NOT NULL,
			title           TEXT NOT NULL,
			notion_page_id  TEXT DEFAULT '',
			synced_at       DATETIME,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// FTS5 가상 테이블 (본문 검색)
		`CREATE VIRTUAL TABLE IF NOT EXISTS chat_wiki_docs USING fts5(
			slug UNINDEXED,
			title,
			body,
			tokenize = 'unicode61 remove_diacritics 2'
		)`,
		// 챗봇 사용량 (학교 부담, 관리자 모니터링용)
		`CREATE TABLE IF NOT EXISTS chat_usage (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id           INTEGER NOT NULL REFERENCES users(id),
			usage_date        DATE NOT NULL,
			requests          INTEGER NOT NULL DEFAULT 0,
			prompt_tokens     INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			cache_tokens      INTEGER NOT NULL DEFAULT 0,
			cost_krw          INTEGER NOT NULL DEFAULT 0,
			UNIQUE(user_id, usage_date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_usage_date ON chat_usage(usage_date DESC)`,
		// Service-level config (#076) — 챗봇이 llm-proxy 호출용 서비스 키 저장
		`CREATE TABLE IF NOT EXISTS chat_service_config (
			key        TEXT PRIMARY KEY,
			value      TEXT NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range chatTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create chat tables: %w", err)
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
