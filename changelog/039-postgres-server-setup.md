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

---

## 📚 심화 학습: PgBouncer 의 두 가지 함정 자세히

이 섹션은 **PgBouncer 를 처음 쓰는 사람** 을 위한 해설이에요. 위에서 언급한
"auth_query + auth_dbname" 과 "SCRAM vs cleartext" 이 왜 필요한지,
**왜 기본 설정으로는 안 되는지** 바닥부터 풀어서 설명할게요.

### 포트 5432 vs 6432 — 왜 바운서는 6432?

- **5432**: PostgreSQL의 **공식 디폴트 포트**
- **6432**: **PgBouncer의 관례적 기본 포트** (공식 표준은 아니지만 커뮤니티가 통일해서 씀)

우리 서버:
```
외부 ──TCP 6432──▶ PgBouncer ──localhost:5432──▶ PostgreSQL
                   (0.0.0.0)                     (내부 전용)
```

외부에서는 **6432만** 열려 있고, 5432는 AWS Security Group 에 안 열려 있어서
PG 직접 접속 불가. 학생들은 무조건 PgBouncer 를 거쳐요.
6432 관례를 따른 이유는 그냥 문서와 도구가 모두 그걸 가정하기 때문이에요.

### PgBouncer 가 왜 중간에 끼는가

PostgreSQL의 **커넥션은 비싼 자원**이에요. 커넥션 하나당:
- 서버 프로세스 1개 생성 (PG는 프로세스 기반 아키텍처)
- RAM 5~10MB 고정 할당
- 컨텍스트 스위칭 비용

학생 100명이 각자 웹앱에서 DB에 붙으면 PG가 버벅여요. PgBouncer는 **커넥션 재활용기**:

```
학생1 ──┐
학생2 ──┼──▶ [PgBouncer: client slot 200개] ──▶ [PG: 실제 server conn 10개]
...    │          (싸다, slot 당 ~10KB)           (비싸다, conn 당 ~10MB)
학생100─┘
```

**transaction pooling** 모드: 한 학생의 트랜잭션이 끝나자마자 그 서버 커넥션을
다른 학생에게 넘겨줘요. 100명이 "접속한 느낌"은 주되 실제 PG 서버는 10개만 open.

### 인증 흐름: 로그인이 사실은 두 번 일어난다

학생 `seowon_todoapp` 이 `psql -h db.earnlearning.com -p 6432 ...` 를 실행하면:

```
[1] 학생 ──"나 seowon_todoapp, 비번 xxx"──▶ PgBouncer
[2]                                        PgBouncer: "이 유저 진짜 있어?"
[3]                                        PgBouncer ──▶ PG ("auth_query")
[4]                                        PG: "있음. SCRAM 해시는 이거"
[5]                                        PgBouncer: 해시로 학생 비번 검증
[6] 학생 ◀────────"OK, 인증 통과"────── PgBouncer
[7]                                        PgBouncer ──▶ PG
                                              "seowon_todoapp 으로 세션 오픈"
[8]                                        양쪽 세션 연결 완료
```

**두 번의 "로그인"** 이 일어나요:
- **클라이언트 ↔ PgBouncer** (3~6번): 학생이 PgBouncer에 자기 맞다고 증명
- **PgBouncer ↔ PG** (7번): PgBouncer 가 PG에 "학생 대신" 들어감

두 로그인 모두 PostgreSQL의 **SCRAM-SHA-256** 프로토콜을 써요 (요즘 PG 기본).

### SCRAM 이 뭔데?

**S**alted **C**hallenge **R**esponse **A**uthentication **M**echanism. 핵심:
비밀번호를 **네트워크에 절대 평문으로 안 보내는** 인증 프로토콜.

단순화한 흐름:
```
서버: "랜덤 challenge 줄게: ABC123"
클라이언트: 비밀번호를 해시 + challenge 섞어서 proof 계산
클라이언트: "proof: XYZ789"
서버: 저장된 비번 해시로 같은 계산 → proof 일치하면 OK
```

PG의 `pg_shadow` 에는 비밀번호가 이런 형태로 저장돼요:
```
SCRAM-SHA-256$4096:<salt>$<StoredKey>:<ServerKey>
```

이건 **검증용 해시**여서, 서버는 이걸로 **검증** 만 할 수 있고,
새 proof 를 **생성** 하려면 원본 평문 비번이 필요해요. 기억해두세요 — 이게 함정 1의 핵심.

### 함정 1: pgbouncer_auth 는 왜 cleartext 로 저장해야 하나

위 인증 흐름의 **7번 단계** 에 주목:

```
PgBouncer ──"나는 pgbouncer_auth 야, 비번 증명할게"──▶ PG 로그인
```

PgBouncer 가 PG 에 **클라이언트 역할** 로 로그인해야 해요 (auth_query 실행을 위해).
그러면 PgBouncer 는 `pgbouncer_auth` 의 **평문 비번** 을 알고 있어야 SCRAM proof 를
계산할 수 있어요.

내가 처음 install.sh 에 짠 (틀린) 코드:
```bash
# 잘못된 접근
scram_hash=$(psql -c "SELECT passwd FROM pg_shadow WHERE usename='pgbouncer_auth'")
cat > userlist.txt <<EOF
"pgbouncer_auth" "${scram_hash}"   # ← SCRAM 해시를 userlist 에 넣음
EOF
```

실행하면 PgBouncer 가 PG 에 붙을 때 이런 에러:
```
ERROR: cannot do SCRAM authentication: password is SCRAM secret
       but client authentication did not provide SCRAM keys
```

번역: "나한테 저장된 건 **검증용** 해시뿐이야. 이걸로 **새 proof** 를 못 만들어서
PG 에 로그인 못 해."

**수정**:
```bash
cat > userlist.txt <<EOF
"pgbouncer_auth" "K0c8N6B3vUN..."   # ← 평문 비번
EOF
chmod 600 userlist.txt              # ← postgres OS 유저만 읽기
```

PgBouncer 가 평문으로 SCRAM proof 를 계산해서 PG 에 무사히 로그인.

> 💡 **학생 비번** 은 userlist 에 **안 들어가요**. 학생 비번은 PG 의 `pg_shadow` 에
> SCRAM 해시로만 저장돼요. PgBouncer 는 auth_query 로 해시만 가져와서 학생의 SCRAM
> **검증** 만 하면 되니까 해시로 충분해요. 검증은 해시로 가능, 생성은 평문 필요 —
> 이 비대칭성이 SCRAM 의 핵심이에요.

**요약 표**:
| 누구 | 어디에 저장 | 형태 | 이유 |
|------|-----|-----|-----|
| 학생들 (50명+) | PG `pg_shadow` | SCRAM 해시 | 검증만 하면 됨 |
| `pgbouncer_auth` (1명) | `userlist.txt` | **평문** | PgBouncer→PG 로그인 시 새 proof 생성 필요 |

그래서 **cleartext 가 필요한 딱 1개 계정** 만 userlist 에 넣어요. 이 계정은 최소
권한 (pg_shadow 조회 함수 1개 실행만 가능) 이라 새어도 피해가 제한적이에요.

### 함정 2: auth_dbname 안 넣으면 격리가 왜 깨지나

이건 PgBouncer 디폴트 동작이 **직관과 달라서** 발생하는 문제.

**auth_query 란?** PgBouncer 가 "이 학생 비번 해시 알려줘" 할 때 쓰는 SQL:
```sql
SELECT usename, passwd FROM public.pgbouncer_get_auth($1)
```

이 쿼리가 **어느 DB 에서** 실행될까요?

**디폴트 동작**: "학생이 접속하려던 DB 에서" 실행.

학생이 `seowon_todoapp` DB 에 가려고 하면, PgBouncer 는:
1. "학생이 `seowon_todoapp` 가고 싶구나"
2. `pgbouncer_auth` 로 **`seowon_todoapp` DB** 에 접속
3. 거기서 `SELECT ... FROM public.pgbouncer_get_auth('seowon_todoapp')` 실행

**문제**: 3번 을 하려면 `pgbouncer_auth` 가 `seowon_todoapp` DB 에
**CONNECT 권한** 이 있어야 하고, 거기에 `public.pgbouncer_get_auth` 함수가
**설치** 되어 있어야 해요.

즉 학생이 DB 만들 때마다:
```sql
GRANT CONNECT ON DATABASE seowon_todoapp TO pgbouncer_auth;
CREATE FUNCTION seowon_todoapp.public.pgbouncer_get_auth(...) ...;
```

이렇게 해야 하는데, 이건 격리를 근본적으로 깨요:
- `pgbouncer_auth` 가 **모든 학생 DB** 에 접근 가능 (탈취 시 파급 큼)
- 학생이 실수로 `pgbouncer_auth` 에 과한 권한 부여 가능
- DB 생성 스크립트 복잡해짐

실제 우리가 본 에러:
```
testuser_myproj/pgbouncer_auth@127.0.0.1:5432 →
  FATAL: permission denied for database "testuser_myproj"
```
PgBouncer 가 학생 DB에 auth_query 때문에 들어가려 했는데 권한 없어서 실패.

**해결**: PgBouncer `[databases]` 에 한 줄 추가.
```ini
[databases]
* = host=127.0.0.1 port=5432 auth_dbname=postgres
```

이러면 PgBouncer 는:
1. 학생이 `seowon_todoapp` 가고 싶어함
2. auth_query 를 **무조건 `postgres` DB 에서** 돌림
3. `postgres.public.pgbouncer_get_auth('seowon_todoapp')` 호출
4. 함수가 `pg_shadow` 전역 카탈로그 조회 → 해시 반환
5. PgBouncer 가 해시로 학생 SCRAM 검증
6. **이때서야** 학생 인증 완료. 그 다음 `seowon_todoapp` DB 로 실제 세션 연결

변화:
| 항목 | auth_dbname 없음 | auth_dbname=postgres |
|------|---|---|
| pgbouncer_auth 필요 권한 | 모든 학생 DB CONNECT | **postgres DB** CONNECT 만 |
| 함수 설치 위치 | 모든 학생 DB | **postgres DB 1곳** |
| 학생 DB 생성 시 GRANT | 필요 | **불필요** |
| 격리 | 깨짐 | **유지** |

> 💡 `pg_shadow` 는 **cluster-wide** (전체 PG 인스턴스) 카탈로그라 어느 DB 에서
> 조회해도 같은 결과가 나와요. 그래서 postgres DB 에서 조회해도 다른 DB 의 유저
> 해시를 얻을 수 있어요. 이게 이 트릭의 핵심.

### 호텔 비유로 한 번 더 정리

**PgBouncer = 호텔 프론트데스크**.
- 학생 = 투숙객 (100명)
- PG 서버 = 객실 (실제 10개)
- PgBouncer = 프론트데스크 직원

투숙객이 올 때마다 방을 새로 짓는 대신, 프론트가 빈 방을 배정하고 퇴실 시 회수
(transaction pooling).

- **함정 1**: 프론트 직원은 **호텔 금고 마스터키(cleartext)** 를 갖고 있어야 직원
  휴게실(PG) 에 들어갈 수 있어요. 방 카드키 지문 사진(SCRAM 해시)만 가지고는 문이
  안 열려요.
- **함정 2**: 투숙객 신원 조회는 **프론트 로비(postgres DB)** 에서 하지, 각 객실
  (학생 DB) 에 찾아가서 하지 않아요. 안 그러면 프론트 직원이 모든 객실 마스터키를
  들고 다녀야 하니까.

---

## 🔒 보안 평가: 학생 관점 공격 표면 분석

설치 직후 `attacker_test` 계정으로 직접 공격을 시도해서 실측했어요. 아래는 그 결과.

### ✅ 막혀 있는 공격 (Safe)

| # | 공격 | 결과 | 방어 원리 |
|---|------|------|----------|
| 1 | `SELECT * FROM pg_shadow` (타인 비번 해시) | ERROR: permission denied | PG 기본: pg_shadow 는 superuser only |
| 2 | `SELECT * FROM pg_authid` (비번 원본) | ERROR: permission denied | 동일 |
| 3 | `pg_read_file('/etc/passwd')` | ERROR: permission denied | `pg_read_server_files` 롤 필요 |
| 4 | `COPY t FROM PROGRAM 'cat /etc/passwd'` | ERROR: permission denied | `pg_execute_server_program` 롤 필요 |
| 5 | `CREATE EXTENSION plpython3u` | ERROR: not available | postgresql-plpython3 미설치 |
| 6 | `SELECT * FROM postgres.public.xxx()` (cross-DB) | ERROR: cross-database references are not implemented | PostgreSQL 자체가 cross-DB 쿼리 미지원 |
| 7 | 다른 학생 DB 접속 (올바른 유저/DB 조합이지만 비번 모름) | FATAL: SASL authentication failed | SCRAM 검증 실패 |
| 8 | 본인 유저로 남의 DB 접속 (비번 맞음, DB 다름) | FATAL: permission denied for database "victim_app" | `REVOKE CONNECT ... FROM PUBLIC` |
| 9 | `pgbouncer_auth` 로 외부 로그인 (비번 모름) | SASL failed | pgbouncer_auth 는 PG 서버 직접 로그인 가능하지만 비번 알아야 함 |
| 10 | 슈퍼유저 승격 (`ALTER ROLE ... SUPERUSER`) | ERROR: must be superuser | PG 기본 보호 |

### ⚠️ 정보 노출 (Info Disclosure, 차단 불가)

| # | 노출 항목 | 심각도 | 영향 |
|---|-----------|--------|------|
| 1 | `SELECT datname FROM pg_database` | **낮음** | 다른 학생 DB 이름 열람 가능 (예: `seowon_todoapp`, `minji_portfolio`) |
| 2 | `SELECT rolname FROM pg_roles` | **낮음** | 다른 학생 유저명 열람 가능 |
| 3 | `has_database_privilege(user, db, 'CONNECT')` | **낮음** | ACL 매트릭스 확인 가능 |

**왜 못 막나**: `pg_database`, `pg_roles` 는 PostgreSQL 의 **시스템 카탈로그**로
기본적으로 모든 로그인 유저에게 SELECT 권한이 있어요. PostgreSQL 의 많은 클라이언트
툴(psql, DBeaver, Prisma 등)이 이 카탈로그를 읽어서 기능을 제공하기 때문에, REVOKE
하면 툴들이 깨져요.

**영향 평가**: 이름만 노출되고 데이터/비번은 안 나와요. 보안 사고라기보다는
**프라이버시 레벨의 노출**. 실무 DB 서비스(Supabase, Neon 등) 도 같은 수준.

### ⚠️ 리소스 고갈 공격 (Resource Exhaustion)

| # | 공격 | 방어 상태 | 부연 설명 |
|---|------|----------|----------|
| 1 | **디스크 채우기** (`COPY` 수 GB 삽입) | ⚠️ **부분 방어** | per-DB 쿼터 없음. 한 학생이 19GB 채우면 전체 다운. **TODO: tablespace quota 또는 파일시스템 quota 도입 검토** |
| 2 | 커넥션 독점 | ✅ 방어됨 | `CONNECTION LIMIT 10` per 유저 |
| 3 | 무한루프 쿼리 (CPU) | ✅ 방어됨 | `statement_timeout = 30s` 자동 킬 |
| 4 | 유휴 트랜잭션 (락 홀드) | ✅ 방어됨 | `idle_in_transaction_session_timeout = 60s` |
| 5 | 메모리 폭식 (큰 sort/hash) | ✅ 방어됨 | `work_mem = 2MB` 초과분은 임시 파일로 (느려지지만 OOM 방지) |
| 6 | 동시 커넥션 폭주 (max_connections 고갈) | ✅ 방어됨 | PgBouncer가 200 client → 10 server 로 버퍼 |
| 7 | 로그 플러딩 (`RAISE NOTICE` 반복) | ⚠️ **부분 방어** | PG 로그는 파일 로테이트되지만 디스크 IO 부담 가능 |

### ⚠️ 크리덴셜 관련 리스크

| # | 리스크 | 방어 상태 | 완화 |
|---|--------|----------|------|
| 1 | 학생이 본인 비번을 GitHub 에 commit | ⚠️ **교육 이슈** | 가이드에 `.env` 사용 강조. 모니터링 불가 |
| 2 | 학생 비번 약함 | ✅ 방어됨 | 서버가 랜덤 24자로 생성, 학생 임의 변경 불가 |
| 3 | `pgbouncer_auth` cleartext 노출 | ✅ 방어됨 | `/etc/pgbouncer/userlist.txt` chmod 600, postgres OS 유저 소유 |
| 4 | EC2 root 침탈 | ❌ **전면 노출** | root = 모든 DB 접근. 일반적인 신뢰 경계 |
| 5 | 재사용 공격 (유출된 비번) | ⚠️ **수동 대응** | `earnlearning-db rotate` 로 로테이션 가능. 주기적 강제 로테이션은 없음 |
| 6 | pgbouncer_auth 가 탈취되면? | ⚠️ **제한적 피해** | 전체 학생의 SCRAM 해시 열람 가능. 해시는 SCRAM 이라 브루트포스 필요 — 즉각적 계정 탈취는 아님 |

### ⚠️ 네트워크 공격

| # | 공격 | 방어 상태 |
|---|------|----------|
| 1 | 포트 스캔 / DDoS | ⚠️ AWS Shield Standard 만 있음. SG 로 6432 만 개방 |
| 2 | 평문 스니핑 | ⚠️ **부분 방어** | PG 는 SSL 지원하지만 self-signed cert. 학생 클라이언트가 `sslmode=require` 안 주면 평문 |
| 3 | 무차별 대입 (brute force) | ❌ **미방어** | fail2ban 미설치. **TODO** |
| 4 | SQL injection (학생 앱) | ❌ 학생 책임 | 학생 앱이 취약하면 자기 DB 가 털림 (타인 DB 는 무관) |

### 📊 종합 리스크 매트릭스

| 영역 | 리스크 | 등급 |
|------|--------|------|
| 타인 DB 데이터 탈취 | 거의 불가 (비번 모르고 PG 격리 잘 됨) | 🟢 **LOW** |
| 서버 장악 (RCE) | 없음 (superuser 불가, 파일 접근 불가) | 🟢 **LOW** |
| 정보 노출 (DB/유저 이름) | 이름만 노출, 데이터 안전 | 🟡 **MEDIUM-LOW** |
| 디스크 고갈로 서버 다운 | 가능 (쿼터 없음) | 🟡 **MEDIUM** |
| 무차별 대입 | 가능 (fail2ban 없음) | 🟡 **MEDIUM** |
| 학생 본인 실수 (비번 유출) | 교육/가이드로만 커버 | 🟡 **MEDIUM** |
| 전체 평가 (교육 환경 기준) | 허용 가능 | 🟢 **ACCEPTABLE** |

### 🔧 개선 TODO (우선순위 순)

1. **[high] fail2ban 설치**: `/var/log/postgresql/postgresql-16-main.log` 모니터링
   → 10회 이상 인증 실패 IP 차단
2. **[high] 디스크 쿼터**: 학생 DB 당 100MB 제한. PG 는 native quota 가 없으니
   주기적 `pg_database_size()` 체크 후 경고/제한
3. **[med] SSL cert 정식화**: Let's Encrypt 로 `db.earnlearning.com` 인증서 발급
   → `sslmode=verify-full` 가이드
4. **[med] pgbouncer_auth 주기 로테이션**: 월 1회 비번 교체 (cron)
5. **[med] 학생 비번 주기 로테이션 강제**: 학기 중 1회 필수 rotate
6. **[low] 시스템 카탈로그 노출 완화**: 불가능하지만 `pg_stat_activity` 에 RLS 비슷한 필터 적용 가능 (검토)
7. **[low] 로그 수집 중앙화**: Cloudwatch 나 Loki 로 접속 로그 분석

### 결론

**교육 환경 기준에서 acceptable**. 실무 수준의 보안이 필요한 프로덕션 데이터를
저장하면 안 돼요 (학생용이라는 전제). 학생들에게는 가이드에서 다음을 명시:
- 개인정보, 결제정보 등 민감 데이터 저장 금지
- 이 DB는 학습/포트폴리오용
- 비밀번호를 코드/GitHub에 포함 금지
- 문제 발생 시 즉시 rotate

---

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
