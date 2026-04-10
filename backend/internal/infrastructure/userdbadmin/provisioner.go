// Package userdbadmin 은 학생 개인 PG 데이터베이스를 프로비저닝 한다.
// PG admin 계정 (postgres 슈퍼유저) 으로 직접 연결해서 CREATE ROLE / CREATE DATABASE /
// REVOKE / GRANT 등의 DDL 을 실행한다.
//
// DDL 은 PgBouncer transaction pooling 모드에서 동작하지 않으므로 반드시 PG 원본
// 포트(5432) 에 직접 연결해야 한다. PgBouncer 가 아닌 admin 연결을 써야 한다는 뜻.
package userdbadmin

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/lib/pq"

	"github.com/earnlearning/backend/internal/domain/userdb"
)

// Provisioner 는 PG 서버에 학생 DB 를 생성/삭제/재발급 하는 인터페이스.
type Provisioner interface {
	Create(username, projectName string) (*CreatedDB, error)
	Delete(dbName, pgUsername string) error
	Rotate(pgUsername string) (newPassword string, err error)
}

// CreatedDB 는 Create 결과. Host/Port 는 학생이 접속할 공개 주소.
type CreatedDB struct {
	DBName     string
	PGUsername string
	Password   string
	Host       string
	Port       int
}

// Config 로 PGProvisioner 를 만든다.
type Config struct {
	// AdminDSN 예: postgres://postgres:pw@host.docker.internal:5432/postgres?sslmode=disable
	// 빈 값이면 NoopProvisioner 가 반환된다.
	AdminDSN string
	// PublicHost 는 학생에게 안내할 접속 주소 (기본: db.earnlearning.com)
	PublicHost string
	// PublicPort 는 학생에게 안내할 포트 (기본: 6432 = PgBouncer)
	PublicPort int
}

// New 는 설정에 따라 실제 PG 프로비저너 또는 NoopProvisioner 를 반환한다.
// DSN 이 비어 있으면 (로컬 개발 / 테스트) Noop 을 반환하고 에러는 없다.
func New(cfg Config) (Provisioner, error) {
	if cfg.AdminDSN == "" {
		return NewNoop(), nil
	}
	if cfg.PublicHost == "" {
		cfg.PublicHost = "db.earnlearning.com"
	}
	if cfg.PublicPort == 0 {
		cfg.PublicPort = 6432
	}

	// URL 형태 DSN 만 지원 (postgres://user:pw@host:port/db?params)
	if _, err := url.Parse(cfg.AdminDSN); err != nil {
		return nil, fmt.Errorf("admin DSN: %w", err)
	}

	db, err := sql.Open("postgres", cfg.AdminDSN)
	if err != nil {
		return nil, fmt.Errorf("open pg admin: %w", err)
	}
	// 풀은 작게 — admin 연결은 DDL 용이라 동시성 낮음
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping pg admin: %w", err)
	}

	return &PGProvisioner{
		db:         db,
		adminDSN:   cfg.AdminDSN,
		publicHost: cfg.PublicHost,
		publicPort: cfg.PublicPort,
	}, nil
}

// nameRE 는 username/projectName 검증용 정규식.
// PG identifier 제한 + 보수적 sanitization.
var nameRE = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

// ValidateName 은 username/projectName 이 안전한지 확인한다.
func ValidateName(s string) error {
	if !nameRE.MatchString(s) {
		return fmt.Errorf("이름 형식이 올바르지 않습니다: 소문자 시작, 소문자/숫자/밑줄, 3~32자")
	}
	return nil
}

// BuildDBName 은 username, projectName 을 합쳐 실제 PG DB 이름을 만든다.
func BuildDBName(username, projectName string) (string, error) {
	if err := ValidateName(username); err != nil {
		return "", fmt.Errorf("username: %w", err)
	}
	if err := ValidateName(projectName); err != nil {
		return "", fmt.Errorf("projectName: %w", err)
	}
	name := username + "_" + projectName
	if len(name) > 63 {
		return "", fmt.Errorf("DB 이름이 63자를 초과합니다")
	}
	return name, nil
}

// --- PGProvisioner (실제) ---

type PGProvisioner struct {
	db         *sql.DB
	adminDSN   string
	publicHost string
	publicPort int
}

// quoteIdent 는 PG identifier 를 안전하게 인용한다.
// database/sql 은 identifier 를 바인딩 파라미터로 전달할 수 없으므로 수동.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// generatePassword 는 24자 랜덤 비밀번호를 만든다.
func generatePassword() (string, error) {
	b := make([]byte, 18) // 18 bytes → 24 base64 chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	// `_` 와 `-` 만 남도록 (DSN 에서 이스케이프 불필요)
	return s, nil
}

func (p *PGProvisioner) Create(username, projectName string) (*CreatedDB, error) {
	dbName, err := BuildDBName(username, projectName)
	if err != nil {
		return nil, err
	}

	password, err := generatePassword()
	if err != nil {
		return nil, fmt.Errorf("generate password: %w", err)
	}

	qDB := quoteIdent(dbName)
	qUser := quoteIdent(dbName) // username == dbName

	// 1. CREATE ROLE
	_, err = p.db.Exec(fmt.Sprintf(`CREATE ROLE %s WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION INHERIT CONNECTION LIMIT 10 PASSWORD '%s'`,
		qUser, escapeLiteral(password)))
	if err != nil {
		if isDuplicateObjectError(err) {
			// 다른 사용자가 같은 slug 로 이미 이 DB 를 만들었음
			return nil, userdb.ErrSlugConflict
		}
		return nil, fmt.Errorf("create role: %w", err)
	}

	// 2. CREATE DATABASE  (rollback on failure: DROP ROLE)
	_, err = p.db.Exec(fmt.Sprintf(`CREATE DATABASE %s OWNER %s ENCODING 'UTF8' LC_COLLATE 'C.UTF-8' LC_CTYPE 'C.UTF-8' TEMPLATE template0`,
		qDB, qUser))
	if err != nil {
		_, _ = p.db.Exec(fmt.Sprintf(`DROP ROLE IF EXISTS %s`, qUser))
		if isDuplicateDatabaseError(err) {
			return nil, userdb.ErrSlugConflict
		}
		return nil, fmt.Errorf("create database: %w", err)
	}

	// 3. DB-level grants: REVOKE PUBLIC, GRANT to owner
	_, err = p.db.Exec(fmt.Sprintf(`REVOKE CONNECT ON DATABASE %s FROM PUBLIC`, qDB))
	if err != nil {
		_ = p.dropAll(dbName, dbName)
		return nil, fmt.Errorf("revoke public: %w", err)
	}
	_, err = p.db.Exec(fmt.Sprintf(`GRANT CONNECT ON DATABASE %s TO %s`, qDB, qUser))
	if err != nil {
		_ = p.dropAll(dbName, dbName)
		return nil, fmt.Errorf("grant connect: %w", err)
	}
	_, err = p.db.Exec(fmt.Sprintf(`GRANT ALL PRIVILEGES ON DATABASE %s TO %s`, qDB, qUser))
	if err != nil {
		_ = p.dropAll(dbName, dbName)
		return nil, fmt.Errorf("grant all: %w", err)
	}

	// 4. Schema-level isolation (connect to target DB as postgres)
	// 별도 연결로 target DB 에 붙어서 public schema 를 학생 소유로 변경
	if err := p.isolatePublicSchema(dbName); err != nil {
		_ = p.dropAll(dbName, dbName)
		return nil, fmt.Errorf("isolate schema: %w", err)
	}

	return &CreatedDB{
		DBName:     dbName,
		PGUsername: dbName,
		Password:   password,
		Host:       p.publicHost,
		Port:       p.publicPort,
	}, nil
}

// isolatePublicSchema 는 target DB 에 admin 으로 접속해서 public 스키마를 학생에게 넘긴다.
// PG 는 연결마다 DB 가 고정되므로, 원본 admin DSN 의 DB 부분만 바꾼 새 연결을 연다.
func (p *PGProvisioner) isolatePublicSchema(dbName string) error {
	dsn, err := rewriteDSNDatabase(p.adminDSN, dbName)
	if err != nil {
		return fmt.Errorf("rewrite dsn: %w", err)
	}
	target, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer target.Close()
	target.SetMaxOpenConns(1)

	if err := target.Ping(); err != nil {
		return err
	}
	qUser := quoteIdent(dbName)
	stmts := []string{
		`REVOKE ALL ON SCHEMA public FROM PUBLIC`,
		fmt.Sprintf(`GRANT ALL ON SCHEMA public TO %s`, qUser),
		fmt.Sprintf(`ALTER SCHEMA public OWNER TO %s`, qUser),
	}
	for _, s := range stmts {
		if _, err := target.Exec(s); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	return nil
}

// rewriteDSNDatabase 는 URL 형태 DSN 의 path(DB 이름)만 교체한 새 DSN 을 반환한다.
// 예: postgres://postgres:pw@host:5432/postgres?sslmode=disable
//  →  postgres://postgres:pw@host:5432/<newDB>?sslmode=disable
func rewriteDSNDatabase(dsn, newDB string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + newDB
	return u.String(), nil
}

func (p *PGProvisioner) Delete(dbName, pgUsername string) error {
	qDB := quoteIdent(dbName)
	qUser := quoteIdent(pgUsername)

	// 활성 세션 종료
	_, err := p.db.Exec(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName)
	if err != nil {
		return fmt.Errorf("terminate sessions: %w", err)
	}

	_, err = p.db.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, qDB))
	if err != nil {
		return fmt.Errorf("drop database: %w", err)
	}
	_, err = p.db.Exec(fmt.Sprintf(`DROP ROLE IF EXISTS %s`, qUser))
	if err != nil {
		return fmt.Errorf("drop role: %w", err)
	}
	return nil
}

func (p *PGProvisioner) Rotate(pgUsername string) (string, error) {
	password, err := generatePassword()
	if err != nil {
		return "", err
	}
	qUser := quoteIdent(pgUsername)
	_, err = p.db.Exec(fmt.Sprintf(`ALTER ROLE %s WITH PASSWORD '%s'`, qUser, escapeLiteral(password)))
	if err != nil {
		return "", fmt.Errorf("rotate: %w", err)
	}
	return password, nil
}

func (p *PGProvisioner) dropAll(dbName, pgUsername string) error {
	return p.Delete(dbName, pgUsername)
}

// escapeLiteral 은 SQL 문자열 리터럴용 ' 이스케이프.
func escapeLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// isDuplicateObjectError 는 lib/pq 에러가 PG SQLSTATE 42710 (duplicate_object,
// 주로 `CREATE ROLE ... already exists` 인지 확인한다.
func isDuplicateObjectError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "42710"
	}
	return false
}

// isDuplicateDatabaseError 는 SQLSTATE 42P04 (duplicate_database) 검사.
func isDuplicateDatabaseError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "42P04"
	}
	return false
}

// --- NoopProvisioner (로컬 개발 / 테스트 용) ---

// NoopProvisioner 는 실제 PG 에 접속하지 않고 성공/실패를 시뮬레이션한다.
// 단위/통합 테스트와 POSTGRES_ADMIN_URL 이 빈 로컬 환경에서 사용.
type NoopProvisioner struct{}

func NewNoop() *NoopProvisioner { return &NoopProvisioner{} }

func (n *NoopProvisioner) Create(username, projectName string) (*CreatedDB, error) {
	dbName, err := BuildDBName(username, projectName)
	if err != nil {
		return nil, err
	}
	return &CreatedDB{
		DBName:     dbName,
		PGUsername: dbName,
		Password:   "noop-password-" + dbName,
		Host:       "db.earnlearning.com",
		Port:       6432,
	}, nil
}

func (n *NoopProvisioner) Delete(dbName, pgUsername string) error { return nil }

func (n *NoopProvisioner) Rotate(pgUsername string) (string, error) {
	return "noop-rotated-" + pgUsername, nil
}
