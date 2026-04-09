---
id: 013
title: LMS 개인 DB 프로비저닝 API + 프로필 UI
priority: high
type: feat
branch: feat/student-db-provisioning
created: 2026-04-10
---

## 개요

LMS 프로필 페이지에서 학생이 본인 전용 PostgreSQL DB를
원클릭으로 생성/관리할 수 있는 기능. 티켓 #012 (서버 설치)에 의존.

## 선행 조건

- #012 완료: 서버에 PG + PgBouncer 구동 중, `earnlearning-db` 스크립트 동작
- 백엔드에서 PG admin 연결 가능 (`DATABASE_ADMIN_URL` 환경변수)

## 사용자 플로우

1. 학생이 `/profile` 진입 → "내 데이터베이스" 섹션 확인
2. [+ 새 DB 만들기] → 프로젝트명 입력 (`todo_app`, `portfolio` 등)
3. 서버가 PG 유저/DB 생성 → 접속정보 카드 표시 (1회만 비밀번호 전체 노출)
4. 이후에는 host/port/dbname/user는 상시 조회, password는 재발급 버튼으로만
5. 각 DB 카드에서: psql/Node.js/Python 코드 스니펫 복사, 삭제

## API 설계

모두 JWT 인증 (본인 DB만 조작 가능).

| Method | Path | 설명 |
|--------|------|------|
| `GET` | `/api/users/me/databases` | 내 DB 목록 |
| `POST` | `/api/users/me/databases` | 신규 DB 생성 `{project_name}` |
| `DELETE` | `/api/users/me/databases/:id` | DB 삭제 |
| `POST` | `/api/users/me/databases/:id/rotate` | 비밀번호 재발급 |

### 응답 예시
```json
{
  "id": 1,
  "project_name": "todo_app",
  "db_name": "seowon_todoapp",
  "username": "seowon_todoapp",
  "host": "db.earnlearning.com",
  "port": 6432,
  "password": "xXx...",  // 생성/rotate 시에만 반환
  "created_at": "2026-04-10T12:00:00Z"
}
```

## DB 스키마 (SQLite — LMS 본 DB)

```sql
-- sqlite.go alterStatements 에 추가
CREATE TABLE IF NOT EXISTS user_databases (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id       INTEGER NOT NULL,
  project_name  TEXT NOT NULL,          -- "todo_app"
  db_name       TEXT UNIQUE NOT NULL,   -- "seowon_todoapp"
  pg_username   TEXT UNIQUE NOT NULL,   -- "seowon_todoapp"
  created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
  last_rotated  DATETIME
);
CREATE INDEX idx_user_databases_user ON user_databases(user_id);
```

> 비밀번호는 **저장하지 않음** — 생성/rotate 응답 시 1회 노출. 잊어버리면 rotate.

## 백엔드 구조 (Go / Echo — 기존 아키텍처 준수)

```
backend/internal/
├── domain/
│   └── userdb.go                   # UserDatabase entity
├── application/
│   └── userdb_usecase.go           # Create/Delete/Rotate/List
├── infrastructure/
│   ├── pgadmin/
│   │   └── provisioner.go          # PG admin conn wrapper
│   └── repository/
│       └── userdb_sqlite.go
└── interfaces/http/handler/
    └── userdb_handler.go
```

### provisioner.go 핵심
```go
type Provisioner struct {
    adminDB *sql.DB  // admin conn to PG
}

func (p *Provisioner) CreateDB(user, proj string) (password string, err error)
func (p *Provisioner) DropDB(user, proj string) error
func (p *Provisioner) RotatePassword(user, proj string) (string, error)
```

- PG admin은 별도 conn pool (max 2), idle timeout 짧게
- DDL은 트랜잭션 불가 → 실패 시 수동 cleanup
- `CREATE USER` → `CREATE DATABASE` → `REVOKE/GRANT` 순
- 이름 검증: `^[a-z][a-z0-9_]{2,31}$`
- 사용자당 최대 3 DB 제한

## 프론트엔드 (`frontend/src/routes/profile/`)

### UserDatabasesSection.tsx
- 목록 표시 (카드 grid)
- [+ 새 DB] 버튼 → 모달
- 각 카드:
  - 프로젝트명 / db_name
  - 접속정보 표시 토글
  - [📋 psql 복사] [📋 .env 복사] [🔄 비밀번호 재발급] [🗑 삭제]
  - 삭제는 confirm dialog (db_name 타이핑 확인)

### NewDatabaseModal.tsx
- `project_name` 입력 (sanitize 힌트)
- 생성 후 접속정보 전체 표시 (비밀번호 포함)
- "저장했어요" 체크박스 → 확인 후 모달 닫기
- 닫으면 비밀번호 다시 볼 수 없음 경고

### 코드 스니펫 생성
```typescript
function psqlSnippet(db) {
  return `psql -h ${db.host} -p ${db.port} -U ${db.username} ${db.db_name}`;
}
function envSnippet(db, pw) {
  return `DATABASE_URL=postgresql://${db.username}:${pw}@${db.host}:${db.port}/${db.db_name}`;
}
```

## 보안

- 프로젝트명 sanitization (서버/클라 양쪽)
- 사용자당 DB 개수 제한 (max 3, env로 조정)
- Rate limit: 생성 분당 2회 (abuse 방지)
- Admin PG 연결문자열은 `env_file` 로만 주입, 로그 금지
- 삭제 시 해당 DB의 활성 세션 종료: `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ?`
- 학생 본인만 조작 (`user_id = auth.user_id` 체크)

## 테스트 (TDD)

### 회귀 테스트
- [ ] `TestCreateUserDB_Success` — 정상 생성 + 목록 조회
- [ ] `TestCreateUserDB_InvalidName` — 이름 sanitization
- [ ] `TestCreateUserDB_Limit` — 3개 초과 시 거부
- [ ] `TestCreateUserDB_Duplicate` — 같은 project_name 거부
- [ ] `TestDeleteUserDB_NotOwner` — 타인의 DB 삭제 시 403
- [ ] `TestRotatePassword_Success`
- [ ] `TestSmoke` — 관련 핸들러 추가

> PG provisioner 테스트는 mock 또는 testcontainers-go + postgres
> (CLAUDE.md: "integration tests는 real DB" 원칙 — 로컬 CI에 PG 컨테이너)

## 구현 순서

1. 도메인 entity + SQLite 마이그레이션
2. `pgadmin.Provisioner` + 단위 테스트 (testcontainers-go)
3. Usecase + Repository
4. HTTP handler + 라우터 등록
5. 통합 테스트 + 스모크 테스트 업데이트
6. 프론트엔드 API client 추가 (`lib/api.ts`)
7. `UserDatabasesSection` + `NewDatabaseModal` 컴포넌트
8. `ProfilePage.tsx`에 섹션 삽입
9. 교육용 스니펫 복사 UI
10. changelog 작성

## 환경변수 추가

```
# backend env_file
POSTGRES_ADMIN_URL=postgresql://postgres:xxxxx@127.0.0.1:5432/postgres?sslmode=disable
POSTGRES_PUBLIC_HOST=db.earnlearning.com
POSTGRES_PUBLIC_PORT=6432
USER_DB_MAX_PER_USER=3
```

## 완료 기준

- [ ] 학생 계정으로 프로필에서 DB 생성 → 외부에서 psql 접속 성공
- [ ] 비밀번호 재발급 동작
- [ ] 삭제 시 PG에서도 완전히 제거 (활성 세션 종료 포함)
- [ ] 회귀 테스트 통과
- [ ] 스모크 테스트 통과
- [ ] changelog/039-student-db-provisioning.md 작성
- [ ] `docs/STUDENT_DB_GUIDE.md`와 링크된 "개발자 페이지" 혹은 프로필에서 열람 가능
