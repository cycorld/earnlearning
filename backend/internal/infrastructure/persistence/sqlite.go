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
		// #120 — 회고 에세이 AI 평가. student_milestones 가 fresh CREATE 된 DB 는 이미 컬럼 포함;
		// #119 로 이미 만들어진 DB 만 이 ALTER 가 효과.
		// 테이블이 아직 없는 fresh DB 에서는 에러 무시 → 이후 CREATE 단계에서 컬럼 포함되어 생성.
		`ALTER TABLE student_milestones ADD COLUMN ai_score INTEGER`,
		`ALTER TABLE student_milestones ADD COLUMN ai_reasoning TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE student_milestones ADD COLUMN ai_signals TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE student_milestones ADD COLUMN ai_evaluated_at DATETIME`,
		// #132 멘션 알림 — 클릭 이동 시 페이지 내 anchor (예: comment-12)
		`ALTER TABLE notifications ADD COLUMN anchor TEXT NOT NULL DEFAULT ''`,
		// #159 멀티 강의실 — 유저의 활성(현재) 강의실. 0 = 미설정
		`ALTER TABLE users ADD COLUMN active_classroom_id INTEGER NOT NULL DEFAULT 0`,
		// #159 Phase 2 — 금융 도메인 루트 엔티티 강의실 스코핑 (0 = 미배정, 백필로 채움)
		`ALTER TABLE companies ADD COLUMN classroom_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE freelance_jobs ADD COLUMN classroom_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE loans ADD COLUMN classroom_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE grants ADD COLUMN classroom_id INTEGER NOT NULL DEFAULT 0`,
		// #166 메일 승인 플로우 + 멀티 메일함(user/company/shared). 신규 CREATE 는 컬럼 포함;
		// 이 ALTER 는 이전 CREATE 로 만들어진 로컬 개발 DB 보호용 (프로덕션 미배포). 에러 무시.
		`ALTER TABLE mail_addresses ADD COLUMN status TEXT NOT NULL DEFAULT 'pending'`,
		`ALTER TABLE mail_addresses ADD COLUMN owner_type TEXT NOT NULL DEFAULT 'user'`,
		`ALTER TABLE mail_addresses ADD COLUMN owner_id INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE mail_addresses ADD COLUMN display_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE emails ADD COLUMN address_id INTEGER NOT NULL DEFAULT 0`,
		// #171 표시용 헤더 From (봉투 from_addr 과 별도 저장)
		`ALTER TABLE emails ADD COLUMN header_from TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE emails ADD COLUMN header_from_name TEXT NOT NULL DEFAULT ''`,
		// owner_id 백필: 개인 주소는 owner_id = 기존 user_id (idempotent).
		`UPDATE mail_addresses SET owner_id = user_id WHERE owner_type = 'user' AND owner_id = 0`,
	}
	for _, stmt := range alterStatements {
		db.Exec(stmt) // ignore "duplicate column" errors
	}

	// #159 강의실별 지갑: wallets UNIQUE(user_id) → UNIQUE(user_id, classroom_id) 리빌드
	if err := migrateWalletsPerClassroom(db); err != nil {
		return fmt.Errorf("migrate wallets per classroom: %w", err)
	}

	// #159 활성 강의실 백필: 미설정(0) 유저를 첫 멤버십 강의실로
	_, err = db.Exec(`UPDATE users SET active_classroom_id =
		COALESCE((SELECT MIN(cm.classroom_id) FROM classroom_members cm WHERE cm.user_id = users.id), 0)
		WHERE active_classroom_id = 0`)
	if err != nil {
		return fmt.Errorf("backfill active_classroom_id: %w", err)
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
			classroom_id INTEGER NOT NULL DEFAULT 0,
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

	// chat_messages.attachments (#106) — 학생이 챗봇에 첨부한 이미지 URL JSON 배열.
	// save_proposal 시 최근 학생 메시지의 attachments 를 모아 proposal 에 첨부.
	db.Exec(`ALTER TABLE chat_messages ADD COLUMN attachments TEXT DEFAULT '[]'`)

	// Proposals (#106) — 학생이 챗봇으로 정리한 교수님께의 제안/버그 리포트
	feedbackTables := []string{
		`CREATE TABLE IF NOT EXISTS proposals (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id     INTEGER NOT NULL REFERENCES users(id),
			category    TEXT NOT NULL CHECK (category IN ('feature','bug','general')),
			title       TEXT NOT NULL,
			body        TEXT NOT NULL,
			attachments TEXT NOT NULL DEFAULT '[]',
			status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','reviewing','resolved','wontfix')),
			admin_note  TEXT NOT NULL DEFAULT '',
			ticket_link TEXT NOT NULL DEFAULT '',
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_proposals_user ON proposals(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_proposals_status ON proposals(status, created_at DESC)`,
	}
	for _, stmt := range feedbackTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create proposal tables: %w", err)
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

	// #119 학생 4대 평가지표 (1차 MVP / 2차 MVP / 사업계획서 / 회고 발표)
	// #120 ai_* 컬럼 — 회고 에세이 AI 작성 확률 평가
	milestoneTables := []string{
		`CREATE TABLE IF NOT EXISTS student_milestones (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			student_id      INTEGER NOT NULL REFERENCES users(id),
			milestone_type  TEXT NOT NULL CHECK (milestone_type IN ('mvp1', 'mvp2', 'business_plan', 'retrospective')),
			source_type     TEXT NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual', 'company', 'grant')),
			source_ref_id   INTEGER,
			url             TEXT NOT NULL DEFAULT '',
			content         TEXT NOT NULL DEFAULT '',
			status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
			admin_note      TEXT NOT NULL DEFAULT '',
			approved_by     INTEGER REFERENCES users(id),
			approved_at     DATETIME,
			ai_score        INTEGER,
			ai_reasoning    TEXT NOT NULL DEFAULT '',
			ai_signals      TEXT NOT NULL DEFAULT '',
			ai_evaluated_at DATETIME,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(student_id, milestone_type)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_student_milestones_student ON student_milestones(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_student_milestones_status ON student_milestones(status)`,
	}
	for _, stmt := range milestoneTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create student_milestones tables: %w", err)
		}
	}

	// #125 business_plan 비공개 첨부 파일. data/private_uploads/ 에 저장 (static 서빙 X).
	// 접근은 업로더 본인 + 관리자만 (인증 다운로드 엔드포인트 경유).
	milestoneFileTables := []string{
		`CREATE TABLE IF NOT EXISTS milestone_files (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			student_id     INTEGER NOT NULL REFERENCES users(id),
			milestone_type TEXT NOT NULL DEFAULT 'business_plan',
			filename       TEXT NOT NULL,
			stored_name    TEXT NOT NULL,
			mime_type      TEXT NOT NULL DEFAULT '',
			size           INTEGER NOT NULL DEFAULT 0,
			path           TEXT NOT NULL,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_milestone_files_student ON milestone_files(student_id, milestone_type)`,
	}
	for _, stmt := range milestoneFileTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create milestone_files tables: %w", err)
		}
	}

	// #128 비밀번호 재설정 토큰. 원본 토큰은 이메일로만 전달, DB에는 SHA-256 해시만 저장.
	passwordResetTables := []string{
		`CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used       INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_password_reset_user ON password_reset_tokens(user_id)`,
	}
	for _, stmt := range passwordResetTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create password_reset_tokens tables: %w", err)
		}
	}

	// #166 멀티 메일함 — 주소(user/company/shared) / 메일 / 첨부 / 공용 권한 (idempotent).
	// 프로덕션 미배포이므로 CREATE 를 멀티 메일함 구조로 재정의한다. 로컬 개발 DB 는 위 ALTER 로 보강됨.
	mailTables := []string{
		`CREATE TABLE IF NOT EXISTS mail_addresses (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_type   TEXT NOT NULL DEFAULT 'user',   -- 'user' | 'company' | 'shared'
			owner_id     INTEGER NOT NULL DEFAULT 0,      -- user id | company id | 생성 관리자 id
			user_id      INTEGER NOT NULL DEFAULT 0,      -- 알림 수신 책임 유저 (하위호환 유지)
			local_part   TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL DEFAULT '',        -- shared 표시명
			status       TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'approved' | 'rejected'
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// 소유 주체별 1주소 (shared 는 관리자당 여러 개 허용 → 부분 인덱스로 제외).
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_mail_addresses_owner
			ON mail_addresses(owner_type, owner_id) WHERE owner_type != 'shared'`,
		`CREATE TABLE IF NOT EXISTS emails (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			address_id    INTEGER NOT NULL DEFAULT 0,
			header_from   TEXT NOT NULL DEFAULT '',
			header_from_name TEXT NOT NULL DEFAULT '',
			owner_user_id INTEGER NOT NULL DEFAULT 0,
			direction     TEXT NOT NULL CHECK(direction IN ('in','out')),
			from_addr     TEXT NOT NULL,
			to_addr       TEXT NOT NULL,
			subject       TEXT NOT NULL DEFAULT '',
			body_text     TEXT NOT NULL DEFAULT '',
			body_html     TEXT NOT NULL DEFAULT '',
			message_id    TEXT NOT NULL DEFAULT '',
			in_reply_to   TEXT NOT NULL DEFAULT '',
			refs          TEXT NOT NULL DEFAULT '',
			read          INTEGER NOT NULL DEFAULT 0,
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_owner ON emails(owner_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_address ON emails(address_id, direction, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS mail_attachments (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			email_id    INTEGER NOT NULL REFERENCES emails(id),
			filename    TEXT NOT NULL,
			mime        TEXT NOT NULL DEFAULT '',
			size        INTEGER NOT NULL DEFAULT 0,
			stored_path TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mail_attachments_email ON mail_attachments(email_id)`,
		// 공용(shared) 메일함 접근 권한. 회수는 revoked=1 (삭제 금지).
		`CREATE TABLE IF NOT EXISTS mail_address_grants (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			address_id INTEGER NOT NULL,
			user_id    INTEGER NOT NULL,
			revoked    INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(address_id, user_id)
		)`,
	}
	for _, stmt := range mailTables {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create mail tables: %w", err)
		}
	}

	// #159 Phase 2 백필: 미배정(0) 도메인 엔티티를 소유자의 첫 멤버십 강의실로.
	// grants 테이블은 위에서 생성되므로 반드시 마지막에 실행. 멱등 (0인 행만 갱신).
	classroomBackfills := []string{
		`UPDATE companies SET classroom_id =
			COALESCE((SELECT MIN(cm.classroom_id) FROM classroom_members cm WHERE cm.user_id = companies.owner_id), 0)
			WHERE classroom_id = 0`,
		`UPDATE freelance_jobs SET classroom_id =
			COALESCE((SELECT MIN(cm.classroom_id) FROM classroom_members cm WHERE cm.user_id = freelance_jobs.client_id), 0)
			WHERE classroom_id = 0`,
		`UPDATE loans SET classroom_id =
			COALESCE((SELECT MIN(cm.classroom_id) FROM classroom_members cm WHERE cm.user_id = loans.borrower_id), 0)
			WHERE classroom_id = 0`,
		`UPDATE grants SET classroom_id =
			COALESCE((SELECT MIN(id) FROM classrooms), 0)
			WHERE classroom_id = 0`,
	}
	for _, stmt := range classroomBackfills {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("classroom backfill: %w", err)
		}
	}

	return nil
}

// migrateWalletsPerClassroom — wallets 를 유저 전역 지갑(UNIQUE user_id)에서
// 강의실별 지갑(UNIQUE(user_id, classroom_id))으로 리빌드한다 (#159).
// SQLite 는 ALTER 로 UNIQUE 제약을 제거할 수 없어 새 테이블 복사 방식 사용.
// - id 를 보존해 transactions.wallet_id 참조가 그대로 유효
// - 기존 지갑의 classroom_id 는 유저의 첫 멤버십 강의실로 백필 (없으면 0 = 미배정;
//   미배정 지갑은 첫 강의실 조인 시 해당 강의실로 귀속됨)
// - 구 테이블은 wallets_legacy_159 로 보존 (DROP 금지 규칙)
func migrateWalletsPerClassroom(db *sql.DB) error {
	var hasCol int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('wallets') WHERE name = 'classroom_id'`,
	).Scan(&hasCol); err != nil {
		return fmt.Errorf("inspect wallets schema: %w", err)
	}
	if hasCol > 0 {
		return nil // 이미 리빌드됨
	}

	// RENAME 시 transactions 의 FK 선언이 wallets_legacy_159 로 따라가지 않도록.
	// foreign_keys=ON 상태에서는 legacy_alter_table 과 무관하게 참조가 재작성되므로
	// 마이그레이션 동안 둘 다 조정한다 (트랜잭션 밖에서만 유효).
	// 새 wallets 가 id 를 보존하므로 기존 참조는 계속 유효하다.
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer db.Exec(`PRAGMA foreign_keys = ON`)
	if _, err := db.Exec(`PRAGMA legacy_alter_table = ON`); err != nil {
		return err
	}
	defer db.Exec(`PRAGMA legacy_alter_table = OFF`)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		`CREATE TABLE wallets_new (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id      INTEGER NOT NULL REFERENCES users(id),
			classroom_id INTEGER NOT NULL DEFAULT 0,
			balance      INTEGER NOT NULL DEFAULT 0,
			UNIQUE(user_id, classroom_id)
		)`,
		`INSERT INTO wallets_new (id, user_id, classroom_id, balance)
		 SELECT w.id, w.user_id,
		        COALESCE((SELECT MIN(cm.classroom_id) FROM classroom_members cm WHERE cm.user_id = w.user_id), 0),
		        w.balance
		 FROM wallets w`,
		`ALTER TABLE wallets RENAME TO wallets_legacy_159`,
		`ALTER TABLE wallets_new RENAME TO wallets`,
		`CREATE INDEX IF NOT EXISTS idx_wallets_user ON wallets(user_id)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("wallets rebuild %q: %w", stmt[:30], err)
		}
	}

	return tx.Commit()
}
