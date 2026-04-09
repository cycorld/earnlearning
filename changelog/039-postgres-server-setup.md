# 039. 학생용 PostgreSQL 서버 설치 (PgBouncer 포함)

> **날짜**: 2026-04-10
> **태그**: `PostgreSQL`, `PgBouncer`, `인프라`, `DB`, `학생DB`

## 무엇을 했나요?

EarnLearning 서버에 **PostgreSQL 16**과 **PgBouncer**를 설치해서,
학생들이 각자 자기만의 DB(`{username}_{projname}`)를 사용할 수 있는 기반을 만들었어요.
앞으로는 바이브코딩 프로젝트에서 SQLite 파일이 아니라 "진짜 서버 DB"에 연결할 수 있습니다!

> 💡 이 작업은 **인프라 준비** 단계예요. 프로필 페이지에서 클릭 한 번으로 DB를 만드는 UI는
> 다음 티켓(#013)에서 구현됩니다. 지금은 서버에서 CLI로 관리합니다.

## 왜 필요했나요?

### SQLite 의 한계

지금까지 학생들이 바이브코딩한 앱은 대부분 SQLite(파일 하나)를 썼어요.
쉽고 간편하지만 한계가 있어요:

- **동시 접속이 어려움**: 여러 사람이 쓰는 웹앱엔 부족
- **파일 전송 필요**: 배포할 때마다 `.sqlite` 파일을 서버에 올려야 함
- **실무와 괴리**: 실제 회사 서비스는 99% PostgreSQL/MySQL 같은 서버 DB 사용

### "진짜 DB 경험"을 주기 위해

학생들이 `DATABASE_URL=postgresql://...` 연결 문자열을 `.env`에 넣고,
Node.js/Python에서 원격 DB에 연결하는 **실무 워크플로우**를 체험할 수 있게 하고 싶었어요.

## 어떻게 만들었나요?

### 1. 서버 성능 검토

우선 EarnLearning 서버가 PostgreSQL을 돌릴 수 있는지 따져봤어요.

- **서버**: AWS t3.small (vCPU 2개, **RAM 2GB**)
- **이미 돌고 있는 것**: blue(프로덕션) + stage 백엔드 스택, ~730MB 사용
- **남은 메모리**: 약 1.2GB

50명 학생이 동시에 쓰면 커넥션이 60~90개 생길 수 있는데,
PostgreSQL은 커넥션 1개당 5~10MB를 써요. 그대로 두면 RAM이 모자랄 수 있어요.

**해결책 = PgBouncer (커넥션 풀러)**

```
[학생 100명] → [PgBouncer] → [실제 PG 커넥션 10~20개]
```

PgBouncer가 가운데서 커넥션을 "재활용"해주기 때문에,
클라이언트 100명이 들어와도 실제 DB 서버 커넥션은 20개로 압축돼요.

### 2. 설치 자동화 스크립트

`deploy/postgres/install.sh` 에 **멱등한(여러 번 실행해도 안전한)** 설치 스크립트를 작성했어요.

핵심 단계:

1. **swap 2GB 추가**: 피크 타임에 OOM(메모리 부족 사망)을 막기 위해
2. **PostgreSQL 16 설치**: Ubuntu 공식 레포 대신 `apt.postgresql.org` (최신 버전)
3. **postgresql.conf 튜닝**: 작은 서버에 맞춰 값 조정
   ```
   shared_buffers = 128MB        # 기본 25%는 너무 큼
   max_connections = 60
   statement_timeout = 30s       # 무한루프 쿼리 자동 차단
   ```
4. **pg_hba.conf**: 외부 접속 허용 (SCRAM 암호)
5. **PgBouncer 설치 + 설정**: transaction pooling 모드
6. **관리 스크립트 설치**: `/usr/local/bin/earnlearning-db`

### 3. 인증 구조: auth_query 방식

PgBouncer가 학생마다 비밀번호를 관리하면 학생이 새로 추가될 때마다
PgBouncer를 재시작해야 해요. 그래서 **auth_query** 방식을 썼어요:

```
1. 학생이 PgBouncer에 접속 (유저명 + 비번 전달)
2. PgBouncer가 PostgreSQL에 물어봄: "이 유저의 비번 해시가 뭐야?"
3. PostgreSQL이 저장된 SCRAM 해시 반환
4. PgBouncer가 SCRAM 프로토콜로 비번 검증
```

**핵심 설정**: `auth_dbname = postgres`

이 옵션이 없으면 PgBouncer는 **학생 본인의 DB**에 접속해서 auth_query를 실행해요.
그러면 pgbouncer_auth 계정이 모든 학생 DB에 접근 권한을 가져야 하고, 격리가 깨져요.
`auth_dbname=postgres`를 설정하면 항상 postgres 관리 DB에서 auth_query가 실행돼요.

### 4. 권한 격리: 학생끼리 서로 못 보게

각 학생 DB는 다음 SQL로 격리돼요:

```sql
-- 생성 시
CREATE ROLE seowon_todoapp WITH
  LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  CONNECTION LIMIT 10
  PASSWORD '...';

CREATE DATABASE seowon_todoapp OWNER seowon_todoapp;

-- 본인 외에는 DB 접속 불가
REVOKE CONNECT ON DATABASE seowon_todoapp FROM PUBLIC;
GRANT CONNECT ON DATABASE seowon_todoapp TO seowon_todoapp;

-- public schema 소유권도 학생에게
REVOKE ALL ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO seowon_todoapp;

-- postgres 관리 DB 접근 차단 (매우 중요!)
REVOKE CONNECT ON DATABASE postgres FROM PUBLIC;
```

결과:
- ✅ 본인 DB에서는 테이블 생성/삭제/조회 자유롭게
- ❌ 친구 DB에 접속 시도 → `permission denied`
- ❌ postgres 관리 DB 접속 시도 → `permission denied`

### 5. 관리 CLI 스크립트

`/usr/local/bin/earnlearning-db` 를 만들어서 서버에서 CLI로 CRUD가 가능해요:

```bash
sudo earnlearning-db create seowon todoapp    # 생성 (JSON 출력)
sudo earnlearning-db rotate seowon todoapp    # 비밀번호 재발급
sudo earnlearning-db delete seowon todoapp    # 삭제
sudo earnlearning-db list seowon              # 특정 유저 DB 목록
sudo earnlearning-db list-all                  # 전체
```

이름 규칙: `^[a-z][a-z0-9_]{2,31}$` (소문자 + 숫자 + 밑줄, 3~32자).
SQL 인젝션 방지를 위해 스크립트 시작 부분에서 정규식으로 검증해요.

### 6. 네트워크 구성

- **AWS Security Group**: TCP 6432 (PgBouncer 포트) 외부 개방, 5432(PG 원본)는 **닫음**
- **Cloudflare DNS**: `db.earnlearning.com` → EC2 public IP, **Proxy OFF**
  - Cloudflare의 주황색 구름(프록시)은 HTTP/HTTPS만 지원해요. PostgreSQL은 TCP라 Proxy OFF 필수!

### 7. 문서화

- **`docs/POSTGRES_SETUP.md`**: 운영자(저)용 설치/운영 매뉴얼
- **`docs/STUDENT_DB_GUIDE.md`**: 학생용 접속 가이드 (psql/DBeaver/Node/Python 예제)

## 실제로 테스트해봤어요

설치 직후 더미 계정으로 전체 플로우를 검증했어요:

```bash
# 1. 생성
sudo earnlearning-db create testuser myproj
# → {"db_name":"testuser_myproj","password":"K0c8N...","url":"postgresql://..."}

# 2. 로컬(맥북)에서 원격 접속
PGPASSWORD='K0c8N...' psql -h db.earnlearning.com -p 6432 \
  -U testuser_myproj testuser_myproj
# psql (16.13) Type "help" for help.
# testuser_myproj=> SELECT current_user;
#    current_user
# ------------------
#  testuser_myproj

# 3. CRUD 확인
# CREATE TABLE todos (...)
# INSERT INTO todos (...)
# SELECT * FROM todos ✅

# 4. 격리 확인: postgres 관리 DB 접근 시도
PGPASSWORD='K0c8N...' psql -h db.earnlearning.com -p 6432 \
  -U testuser_myproj postgres
# FATAL: permission denied for database "postgres" ✅

# 5. 비밀번호 재발급
sudo earnlearning-db rotate testuser myproj
# → 새 비밀번호 발급 (기존 비번 무효화)

# 6. 삭제
sudo earnlearning-db delete testuser myproj
# → DROP DATABASE + DROP ROLE 완료
```

## 리소스 사용량 (실측)

| 시점 | 사용 메모리 | swap |
|------|------------|------|
| 설치 전 | 726 MB | 0 MB |
| 설치 후(idle) | 752 MB | 6 MB |

PG + PgBouncer가 idle 상태에서 **+26MB** 만 사용. 학생이 실제로 붙으면
커넥션마다 5~10MB가 추가되니 피크 +400MB 수준으로 예상돼요.
2GB swap이 완충 역할을 하고, 기존 blue/stage 서비스엔 영향 없음을 확인했어요.

## 배운 점

### 1. PgBouncer 의 auth_query 함정
**auth_dbname을 지정 안 하면** PgBouncer가 학생 본인 DB에 접속해서 auth_query를 돌리려고 해요.
`pgbouncer_auth` 계정이 모든 DB에 들어갈 수 있어야 하는 셈인데, 이건 격리를 깨요.
`auth_dbname=postgres` 한 줄로 해결돼요.

### 2. SCRAM 해시 vs 평문
PgBouncer가 PG 서버에 로그인할 때는 **평문 비밀번호**가 필요해요.
SCRAM 해시로는 서버 로그인이 안 돼요 (에러: "cannot do SCRAM authentication").
그래서 `pgbouncer_auth`의 비번만 `userlist.txt`에 평문으로 저장하고 (파일 권한 600),
학생 비번은 PG에 SCRAM 해시로 저장해요.

### 3. `postgres` DB 는 기본적으로 PUBLIC 접속 가능
PostgreSQL은 기본 설정에서 `postgres` 관리 DB에 모든 로그인 유저가 접속 가능해요.
명시적으로 `REVOKE CONNECT ON DATABASE postgres FROM PUBLIC`을 해줘야 격리돼요.

### 4. 작은 서버에서 PG 돌리는 법
- `shared_buffers`는 기본값(25%)이 너무 커요. 128MB로 고정.
- `statement_timeout=30s` 로 무한루프 쿼리 자동 차단.
- swap은 꼭 있어야 함 (OOM 예방).

### 5. Cloudflare 는 TCP 프록시 안 함
DB 서버 주소는 Proxy OFF 로 직접 IP를 노출해야 해요.
HTTP/HTTPS 만 주황색 구름(프록시) 가능.

## 사용한 AI 프롬프트

```
ssh earnlearning 에 postgresql 을 설치해서 50명을 위한 계정과
그 계정만이 각각 쓸 수 있는 {username}_{projname} 형태의 데이터베이스를
사용할 수 있게 해주고 싶어. 그리고 해당 계정과 새로운 디비 등록을
우리 lms 의 개인 유저 프로필에서 할 수 있으면 좋을거 같은데,
일단 우리 서버 성능에서 해당 디비 제공이 가능한지
(사용량은 거의 없음-대신 커넥션은 프로젝트 수만큼 나오겠지)
그리고 프로필 구현이 가능한지 알려줘.
```

## 다음 단계

- **#013**: LMS 프로필 페이지에 "내 데이터베이스" 섹션 추가
  - 백엔드 API: `POST/GET/DELETE/ROTATE /api/users/me/databases`
  - 프론트엔드: `ProfilePage`에 카드형 UI + 접속 코드 스니펫 복사 버튼
  - 교육용 가이드 링크
