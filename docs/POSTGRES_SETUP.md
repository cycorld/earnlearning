# PostgreSQL 서버 운영 가이드

earnlearning EC2에 설치된 PostgreSQL 16 + PgBouncer의 설치/운영 매뉴얼.
학생들이 `{username}_{projname}` 형태의 개인 DB를 쓸 수 있도록 구성되어 있다.

## 아키텍처

```
[학생 로컬]
    │  psql / DBeaver / Node / Python
    │  (port 6432, scram 암호)
    ▼
[EC2 :6432  PgBouncer]          ← transaction pooling, max 200 client
    │
    ▼  (max_connections=60)
[EC2 :5432  PostgreSQL 16]      ← 실제 DB, /var/lib/postgresql/16/main
```

- **PgBouncer**: 학생 수백 커넥션 → 실제 PG 커넥션 10~20개로 압축
- **auth_query**: PgBouncer가 PG의 `pg_shadow` 를 on-the-fly 조회 → 유저 추가 시 pgbouncer 재로드 불필요
- **메모리 예산**: PG ~360MB + PgBouncer ~20MB (swap 2GB 있음)

## 설치

최초 설치 또는 재설정(멱등):
```bash
# 로컬에서 스크립트 전송
scp deploy/postgres/install.sh deploy/postgres/earnlearning-db.sh earnlearning:/tmp/

# 원격 실행
ssh earnlearning 'sudo bash /tmp/install.sh'
```

설치 스크립트는 다음 작업을 수행한다:
1. swap 2GB 추가 (OOM 방지)
2. PostgreSQL 공식 apt repo 추가 + PG 16 + PgBouncer 설치
3. `postgresql.conf` 튜닝 (t3.small 전용)
4. `pg_hba.conf` 에 외부 접속 허용 규칙 추가
5. `postgres` superuser 비밀번호 생성 → `/root/earnlearning-secrets/postgres_admin_password`
6. `pgbouncer_auth` 역할 생성 + `public.pgbouncer_get_auth()` 함수 (auth_query)
7. `/usr/local/bin/earnlearning-db` 관리 스크립트 설치

## 방화벽 / DNS

### AWS Security Group
| 포트 | 소스 | 목적 |
|------|------|------|
| 6432 | 0.0.0.0/0 | PgBouncer (학생 접속) |
| 5432 | — | **열지 말 것** (PgBouncer 경유만 허용) |

### Cloudflare DNS
- `db.earnlearning.com` → EC2 public IP (**Proxy: OFF**, Cloudflare는 TCP 프록시를 안 해줌)

## 학생 계정 관리

### 생성
```bash
ssh earnlearning 'sudo earnlearning-db create seowon todoapp'
```
출력 예:
```json
{
  "db_name": "seowon_todoapp",
  "username": "seowon_todoapp",
  "password": "abc123xyz...",
  "host": "db.earnlearning.com",
  "port": 6432,
  "psql": "PGPASSWORD='...' psql -h db.earnlearning.com -p 6432 -U seowon_todoapp seowon_todoapp",
  "url": "postgresql://seowon_todoapp:...@db.earnlearning.com:6432/seowon_todoapp"
}
```

### 삭제 (완전 제거)
```bash
ssh earnlearning 'sudo earnlearning-db delete seowon todoapp'
```
활성 세션 강제 종료 후 `DROP DATABASE` + `DROP ROLE`.

⚠️ **주의 (#016)**: `earnlearning-db delete` 는 PG 만 정리, **LMS SQLite 의
`user_databases` 행은 남음** → 학생 프로필에 좀비 카드. 두 가지 처리법:

**A. 운영자가 즉시 정리 (권장)** — admin 토큰으로 DB명 직접 지정:
```bash
ADMIN_TOKEN='Bearer ...'  # admin 로그인 토큰
DB_NAME='seowon_todoapp'
curl -X DELETE -H "Authorization: $ADMIN_TOKEN" \
  "https://earnlearning.com/api/admin/user-databases/by-dbname/$DB_NAME"
```

**B. 일괄 정합성 검사** — 모든 SQLite 행을 PG 와 대조해 고아 자동 제거:
```bash
curl -X POST -H "Authorization: $ADMIN_TOKEN" \
  "https://earnlearning.com/api/admin/user-databases/reconcile"
# → { "checked": N, "removed": M, "errors": K, "orphans": [...] }
```

### 비밀번호 재발급
```bash
ssh earnlearning 'sudo earnlearning-db rotate seowon todoapp'
```

### 목록
```bash
ssh earnlearning 'sudo earnlearning-db list seowon'     # 특정 유저
ssh earnlearning 'sudo earnlearning-db list-all'         # 전체
```

### 이름 규칙
- `^[a-z][a-z0-9_]{2,31}$` — 소문자 시작, 소문자/숫자/밑줄, 3~32자
- 최종 DB명 `{username}_{projname}` 은 63자를 넘길 수 없다 (PG identifier 제한)
- 유저당 최대 5개 DB (`MAX_DB_PER_USER` 환경변수로 조정)

## 권한 격리 보장

각 학생 DB는 다음과 같이 격리되어 있다:
- `REVOKE CONNECT ... FROM PUBLIC` → 본인만 접속 가능
- `REVOKE ALL ON SCHEMA public FROM PUBLIC` → 타 유저가 스키마 접근 불가
- `ALTER SCHEMA public OWNER TO {user}` → public 스키마 소유권까지 학생에게 이전
- `NOSUPERUSER`, `NOCREATEDB`, `NOCREATEROLE` → 권한 상승 불가
- `CONNECTION LIMIT 10` → 한 학생이 커넥션을 독점 못 하게 제한

## 운영 명령어

### 서비스 상태 / 로그
```bash
sudo systemctl status postgresql pgbouncer
sudo journalctl -u postgresql -n 100
sudo journalctl -u pgbouncer -n 100
sudo tail -f /var/log/postgresql/postgresql-16-main.log
sudo tail -f /var/log/postgresql/pgbouncer.log
sudo tail -f /var/log/earnlearning-db.log
```

### 메모리/커넥션 모니터링
```bash
# 실시간 커넥션 수
sudo -u postgres psql -c "SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;"

# PgBouncer 통계
psql -h 127.0.0.1 -p 6432 -U postgres pgbouncer -c "SHOW POOLS;"
psql -h 127.0.0.1 -p 6432 -U postgres pgbouncer -c "SHOW STATS;"
```

### 재시작
```bash
sudo systemctl restart postgresql   # PG 재시작 (모든 세션 종료됨)
sudo systemctl restart pgbouncer    # pgbouncer 재시작
```

### 백업 (수동)
```bash
# 특정 DB 덤프
sudo -u postgres pg_dump seowon_todoapp > /tmp/seowon_todoapp.sql

# 전체 덤프
sudo -u postgres pg_dumpall > /tmp/all.sql
```

## 롤백 / 제거

```bash
ssh earnlearning
sudo systemctl stop pgbouncer postgresql
sudo systemctl disable pgbouncer postgresql
# 완전 제거 (데이터 포함, 위험):
# sudo apt-get purge postgresql-16 postgresql-contrib-16 pgbouncer
# sudo rm -rf /var/lib/postgresql /etc/postgresql /etc/pgbouncer
```

기존 blue/stage 스택은 영향 없음 (완전 독립 프로세스).

## 리소스 예산 (t3.small 기준)

| 항목 | 사용량 |
|------|--------|
| PG base | ~50 MB |
| shared_buffers | 128 MB |
| 커넥션 (max 60 × ~8MB) | ~480 MB (피크) |
| PgBouncer | ~20 MB |
| **추가 사용** | **~400~700 MB** |
| 기존 blue+stage | ~730 MB |
| **총합 피크** | **~1.4 GB** (2GB 중) |

피크 초과 시 swap이 완충. swap 사용이 지속되면 커넥션 한도를 낮추거나 인스턴스 업그레이드 검토.
