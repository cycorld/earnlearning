# 040. LMS 프로필에서 학생 개인 PostgreSQL 프로비저닝

> **날짜**: 2026-04-10
> **태그**: `Feature`, `PostgreSQL`, `프로필`, `학생DB`, `백엔드`, `프론트엔드`

## 무엇을 했나요?

학생들이 **프로필 페이지에서 버튼 한 번**으로 자기만의 PostgreSQL 데이터베이스를
받을 수 있게 만들었어요. 티켓 #012 에서 만든 PG/PgBouncer 인프라 위에,
**CRUD API + 프로필 UI** 를 얹은 작업이에요.

### 사용자 경험

1. `/profile` 진입 → "내 데이터베이스" 섹션
2. `[+ 새 DB]` 클릭 → 프로젝트명 입력 (예: `todoapp`)
3. 생성 완료 모달: host/port/database/username/**password** + `.env` 스니펫 + `psql` 명령어
4. 저장 체크박스 체크 → 닫으면 비밀번호는 다시 볼 수 없음
5. 카드에서: 접속정보 토글(비번 제외), 비밀번호 재발급, 삭제

## 왜 필요했나요?

#012 에서 서버 인프라는 깔았지만, 학생이 DB 를 받으려면 조교가 SSH 로
`sudo earnlearning-db create <학생> <프로젝트>` 를 수동 실행해야 했어요.
50명 × 여러 프로젝트 × 학기 단위 → 스케일 안 되는 수동 작업.

학생이 직접 UI 에서 만들 수 있어야 진짜 교육 도구가 돼요.

## 어떻게 만들었나요?

### 백엔드 (Go / Echo — Clean Architecture 준수)

기존 코드베이스의 **4계층 구조** 를 따랐어요:

```
domain/userdb/           # 엔티티 + 레포지토리 인터페이스 (순수 Go)
application/             # 유즈케이스 (비즈니스 로직)
infrastructure/
  persistence/userdb_repo.go    # SQLite 레포 구현
  userdbadmin/provisioner.go    # PG admin DDL 실행
interfaces/http/handler/userdb_handler.go   # Echo HTTP 핸들러
```

#### 1. Domain layer (`internal/domain/userdb/`)

```go
type UserDatabase struct {
    ID          int
    UserID      int
    ProjectName string     // "todoapp"
    DBName      string     // "seowon_todoapp"
    PGUsername  string
    Host        string     // "db.earnlearning.com"
    Port        int        // 6432
    CreatedAt   time.Time
    LastRotated *time.Time
}
```

**저장 안 하는 것**: `Password`. 생성/재발급 응답에서 1회만 노출해요.
잊어버리면 재발급 (새 비번 생성 → 기존 비번 즉시 무효).

#### 2. SQLite 마이그레이션

`persistence/sqlite.go` 의 `RunMigrations()` 에 idempotent 하게 추가:
```sql
CREATE TABLE IF NOT EXISTS user_databases (
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
);
```

UNIQUE 제약 덕에 중복은 DB 레벨에서 차단, 레포에서 잡아 `ErrDuplicate`.

#### 3. PG Provisioner (`infrastructure/userdbadmin/`)

핵심 인터페이스:
```go
type Provisioner interface {
    Create(username, projectName string) (*CreatedDB, error)
    Delete(dbName, pgUsername string) error
    Rotate(pgUsername string) (newPassword string, err error)
}
```

**두 구현체**:
- **PGProvisioner**: 실제 `lib/pq` 드라이버로 PG 에 연결해서 `CREATE ROLE`, `CREATE DATABASE`, `REVOKE/GRANT`, `ALTER SCHEMA` DDL 실행
- **NoopProvisioner**: 가짜 성공 응답. `POSTGRES_ADMIN_URL` 이 비어 있는 로컬/테스트 환경에서 사용

프로비저너 선택 로직:
```go
func New(cfg Config) (Provisioner, error) {
    if cfg.AdminDSN == "" { return NewNoop(), nil }
    // ... 실제 PG 연결 ...
}
```

#### 4. DDL 은 PgBouncer 를 못 탄다

**중요한 제약**: `CREATE DATABASE` 같은 DDL 은 트랜잭션 안에서 실행될 수 없어요.
PgBouncer 의 `transaction pooling` 은 모든 쿼리를 트랜잭션으로 감싸기 때문에
6432 포트(PgBouncer)로는 DDL 이 실패해요.

**해결**: 백엔드가 PG 원본 포트 `5432` 에 직접 연결합니다.
- 학생 접속: `db.earnlearning.com:6432` (PgBouncer, 외부 개방)
- admin 접속: `host.docker.internal:5432` (호스트 내부, 외부 닫힘)

#### 5. Docker 컨테이너 → 호스트 PG 라우팅

백엔드는 컨테이너로 동작, PG 는 호스트 프로세스. 컨테이너에서 호스트를 부르려면
`extra_hosts` 로 `host.docker.internal` 을 host-gateway 로 매핑해야 해요:

```yaml
# deploy/docker-compose.{blue,green,stage}.yml
backend:
  extra_hosts:
    - "host.docker.internal:host-gateway"
```

그리고 `.env.prod` / `.env.stage` 에:
```bash
POSTGRES_ADMIN_URL=postgres://postgres:PW@host.docker.internal:5432/postgres?sslmode=disable
POSTGRES_PUBLIC_HOST=db.earnlearning.com
POSTGRES_PUBLIC_PORT=6432
USER_DB_MAX_PER_USER=3
```

#### 6. 함정: public schema 소유권 이전

`CREATE DATABASE` 시 지정한 OWNER 는 DB 자체만 소유해요. 안의 `public` 스키마는
기본적으로 `postgres` 가 소유하고, PUBLIC 에 CREATE 권한을 가져요. 학생끼리 격리를
유지하려면 **target DB 에 직접 연결해서** 스키마 권한을 바꿔야 해요:

```go
func (p *PGProvisioner) isolatePublicSchema(dbName string) error {
    // admin DSN 의 DB 부분만 target 으로 교체한 새 연결
    dsn, _ := rewriteDSNDatabase(p.adminDSN, dbName)
    target, _ := sql.Open("postgres", dsn)
    defer target.Close()

    target.Exec("REVOKE ALL ON SCHEMA public FROM PUBLIC")
    target.Exec(fmt.Sprintf("GRANT ALL ON SCHEMA public TO %q", dbName))
    target.Exec(fmt.Sprintf("ALTER SCHEMA public OWNER TO %q", dbName))
}
```

PG 는 **한 연결 = 한 DB** 고정이라 이게 유일한 방법이에요.

#### 7. 유저명 슬러그 (이메일 → PG username)

학생 이메일 `seowon@example.com` 을 PG 안전한 이름으로 변환:
```go
func SlugFromEmail(email string) string {
    local := strings.Split(email, "@")[0]
    s := lowercase(local)
    s = regex.ReplaceAll(s, "[^a-z0-9_]", "")
    if !startsWithLetter(s) { s = "u_" + s }
    if len(s) < 3  { s = s + "_db" }
    if len(s) > 20 { s = s[:20] }
    return s
}
```

최종 DB 이름은 `{slug}_{projectName}` 이고, PG identifier 제한인 63자를 넘으면 거부.

#### 8. 쿼터 & 유효성

- **사용자당 최대 3개** DB (`USER_DB_MAX_PER_USER` env 조정 가능)
- 프로젝트명 정규식: `^[a-z][a-z0-9_]{2,31}$`
- 본인 DB 만 조작 가능 (`if u.UserID != userID → ErrForbidden`)
- 중복 프로젝트명: `UNIQUE(user_id, project_name)` 제약

### 프론트엔드 (React + shadcn/ui)

`UserDatabasesSection.tsx` 한 파일에 필요한 컴포넌트 다 넣었어요:

| 컴포넌트 | 역할 |
|---|---|
| `UserDatabasesSection` | 프로필 카드 (목록 + "새 DB" 버튼) |
| `DatabaseCard` | 각 DB 카드 (접속정보 토글, rotate, delete) |
| `NewDatabaseDialog` | 프로젝트명 입력 모달 |
| `CredentialsDialog` | 생성/재발급 시 비번+스니펫 표시 (1회성) |
| `KV` / `CopyBlock` | 복사 가능한 key-value 행 |

**포인트**:
- 삭제 확정: DB 이름을 정확히 타이핑해야 활성화 (실수 방지)
- 비밀번호는 **생성/재발급 때만** `CredentialsDialog` 에서 표시 → "저장했어요" 체크 → 닫기
- 이후에는 `DatabaseCard` 에서 host/port/db/user 만 보이고 password 줄은 없음
- `.env` 스니펫 + `psql` 명령어 **복사 버튼** 내장 (교육용)

### API 라우트

모두 `approved` 그룹 (JWT 로그인 + 승인된 학생만):

| Method | Path | 설명 |
|---|---|---|
| GET | `/api/users/me/databases` | 내 DB 목록 |
| POST | `/api/users/me/databases` | 신규 생성 (body: `{project_name}`) |
| POST | `/api/users/me/databases/:id/rotate` | 비밀번호 재발급 |
| DELETE | `/api/users/me/databases/:id` | 삭제 |

**참고**: OAuth scope 는 추가하지 않았어요. 이건 학생 본인 도구지 외부 앱이 대신할
기능이 아니에요 (내 DB 만 조작하므로 OAuth delegation 의미가 약함).

## TDD & 테스트

회귀 테스트 7개 작성 (`backend/tests/integration/userdb_test.go`):

1. `TestUserDB_List_Empty` — 초기 목록 비어 있음
2. `TestUserDB_Create_Success` — 생성 후 필수 필드 확인
3. `TestUserDB_Create_InvalidName` — 7가지 잘못된 이름 거부
4. `TestUserDB_Create_Duplicate` — 동일 project_name 재생성 거부
5. `TestUserDB_QuotaExceeded` — 4번째 생성 거부 (max=3)
6. `TestUserDB_Rotate` — rotate 후 비밀번호가 달라야 함
7. `TestUserDB_Delete` + `Delete_NotOwner` — 본인만 삭제 가능

테스트는 **NoopProvisioner** 를 사용해서 실제 PG 없이 통합 테스트. 스모크 테스트에도
`GET /api/users/me/databases` 를 추가했어요.

### E2E 검증 (실제 PG)

SSH 터널 + `POSTGRES_ADMIN_URL` 환경변수로 실제 EC2 PG 에 대해 provisioner 의
Create/Rotate/Delete 전체 플로우를 실행하고 성공 확인:
```
created: &{DBName:e2etest_autotest Password:Q-Ed3G5tqBxPh_BbSfmXMh5w Host:db.earnlearning.com Port:6432}
PASS
```

## 배운 점

### 1. DDL 은 PgBouncer transaction pooling 과 상극
`CREATE DATABASE` 같은 DDL 은 암시적으로 트랜잭션 밖에서 실행돼야 해요. PgBouncer
`transaction` 모드는 모든 쿼리를 트랜잭션으로 감싸므로 DDL 이 실패. 해결: admin 용
연결은 PG 원본 포트(5432) 로, 학생 용 연결은 PgBouncer(6432) 로 **이원화**.

### 2. PG 한 연결 = 한 DB 고정
JDBC/psql 과 달리 `database/sql` (lib/pq) 에서는 한 연결이 한 DB 에 바인딩돼요.
다른 DB 에 DDL 을 쏘려면 **새 연결** 을 열어야 해요. URL DSN 의 path 만 바꿔서
재연결하는 패턴을 썼어요.

### 3. public schema 소유권
새 DB 는 OWNER 가 있어도 안의 `public` 스키마는 `postgres` 소유 + PUBLIC 에 CREATE
권한이 있어요. 격리하려면 target DB 에 연결해서:
```sql
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO student;
ALTER SCHEMA public OWNER TO student;
```

### 4. Clean Architecture 의 가치
"DB 연결 실패 시 Noop 으로 fallback" 이 한 줄로 끝났어요 (`infrastructure` 만 갈아
끼우면 됨). 테스트도 provisioner 인터페이스를 NoopProvisioner 로 주입해서 실제 PG
없이도 API 전체를 검증 가능.

### 5. 비밀번호는 저장하지 않기
SQLite 에 암호화해서 저장하는 대신, **생성/재발급 시 1회만 반환** 하고 잊어버리면
재발급하도록 했어요. 장점:
- 암호화 키 관리 부담 없음
- DB 덤프 유출 시에도 비번 안전
- UX 는 "저장했어요" 체크박스로 커버

## 사용한 AI 프롬프트

```
응 두가지 티켓 만들고 진행해줘.
```

(앞선 리서치 단계에서 "50명 위한 계정 + {username}_{projname} DB, 프로필에서 관리"
요구를 정리 → #012 서버 인프라 → #013 LMS 통합 으로 쪼갰고, 이 티켓은 후자.)

## 다음 단계

- **주기적 백업 cron**: 각 학생 DB 를 주 1회 `pg_dump` 해서 `/backup/` 로
- **디스크 쿼터 모니터링**: `pg_database_size()` 임계값 넘으면 알림
- **fail2ban**: PG 로그 기반 IP 차단
- **OAuth scope 확장 검토**: 외부 앱에서 "내 DB 목록 조회" 정도는 필요할 수 있음
