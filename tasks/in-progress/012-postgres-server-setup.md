---
id: 012
title: PostgreSQL + PgBouncer 서버 설치 (학생 DB 제공용)
priority: high
type: chore
branch: chore/postgres-server-setup
created: 2026-04-10
---

## 개요

earnlearning EC2(t3.small, 2GB RAM)에 PostgreSQL 16과 PgBouncer를 설치하여
50명 학생 대상 개인 DB 호스팅 환경을 구축한다. 학생들은 각자 `{username}_{projname}`
형태의 DB를 LMS 프로필에서 프로비저닝받아 바이브코딩 프로젝트의 백엔드 저장소로 사용한다.

## 배경 / 제약

- **서버**: t3.small, 2GB RAM, swap 0B, 디스크 29GB (19GB 여유)
- **동시 사용 예상**: 피크 20~30명 코딩, 동시 커넥션 60~90개 가능
- **기존 서비스**: blue/stage 백엔드 스택(현재 ~726MB 사용), 절대 간섭 금지
- **데이터량**: 학생 프로젝트당 수 MB 수준(학습용), 용량 부담 적음

## 리소스 예산

| 항목 | 할당 |
|------|------|
| PG base | 50MB |
| shared_buffers | 128MB |
| work_mem | 2MB |
| max_connections | 60 |
| PgBouncer | 20MB |
| **합계** | **약 360~400MB 추가** |

여유 메모리 1.2GB 중 400MB 차감 → 여유 800MB 확보.
**swap 2GB 추가 필수** (OOM 대비).

## 작업 내역

### 1. 사전 준비
- [ ] 2GB swapfile 생성 (`/swapfile`), `/etc/fstab` 등록
- [ ] 기존 리소스 사용량 스냅샷 백업 (`free -h`, `docker stats`)

### 2. PostgreSQL 16 설치
- [ ] `apt install postgresql-16 postgresql-contrib-16`
- [ ] `postgresql.conf` 튜닝:
  - `shared_buffers = 128MB`
  - `effective_cache_size = 512MB`
  - `work_mem = 2MB`
  - `maintenance_work_mem = 32MB`
  - `max_connections = 60`
  - `statement_timeout = 30000` (30초, 무한루프 차단)
  - `idle_in_transaction_session_timeout = 60000`
  - `log_min_duration_statement = 1000` (1초 이상 쿼리 로깅)
- [ ] `pg_hba.conf`:
  - `local all postgres peer` (superuser 로컬만)
  - `host all all 0.0.0.0/0 scram-sha-256` (외부 접속 암호)
  - SSL 강제(`hostssl`)
- [ ] `listen_addresses = '*'`, 포트 5432
- [ ] 초기 superuser 비밀번호 설정, 저장: `~/secrets/postgres.env`

### 3. PgBouncer 설치
- [ ] `apt install pgbouncer`
- [ ] `pgbouncer.ini`:
  - `pool_mode = transaction`
  - `max_client_conn = 200`
  - `default_pool_size = 10`
  - `reserve_pool_size = 5`
  - `listen_port = 6432`
  - `auth_type = scram-sha-256`
  - `auth_file = /etc/pgbouncer/userlist.txt`
- [ ] systemd enable + start

### 4. 관리 스크립트 (`/usr/local/bin/earnlearning-db`)
```bash
earnlearning-db create {username} {projname}  # 계정+DB 생성, 비밀번호 stdout
earnlearning-db delete {username} {projname}  # DROP DATABASE + DROP USER
earnlearning-db rotate {username} {projname}  # 비밀번호 재발급
earnlearning-db list {username}                # 해당 유저의 DB 목록
earnlearning-db list-all                        # 전체 목록 + 크기
```
- [ ] DB명/유저명 sanitization (`^[a-z][a-z0-9_]{2,31}$`)
- [ ] 생성 시 자동 처리:
  - `CREATE USER ... WITH PASSWORD ... LOGIN`
  - `CREATE DATABASE ... OWNER ...`
  - `REVOKE CONNECT ON DATABASE ... FROM PUBLIC`
  - `GRANT CONNECT ON DATABASE ... TO {user}`
  - `pgbouncer userlist.txt` 자동 업데이트 + `pgbouncer -R` 재로드
- [ ] 삭제는 세션 강제 종료 후 DROP
- [ ] 로그: `/var/log/earnlearning-db.log`

### 5. 방화벽 / 포트 개방
- [ ] AWS Security Group: 6432(pgbouncer) 전세계 개방, 5432 폐쇄
  - 또는 5432는 `127.0.0.1`만, 6432만 외부
- [ ] EC2 내부 UFW: 6432 허용
- [ ] 접속 엔드포인트: `db.earnlearning.com` (Cloudflare A record, Proxy OFF — TCP라 Cloudflare 프록시 불가)

### 6. 검증
- [ ] 더미 계정으로 `create → psql 접속 → SELECT 1 → delete` 플로우 확인
- [ ] PgBouncer 경유 접속 확인 (port 6432)
- [ ] 동시 커넥션 스트레스 테스트 (`pgbench` 또는 간단한 loop)
- [ ] 메모리/swap 사용량 before/after 비교

### 7. 문서화
- [ ] `docs/POSTGRES_SETUP.md` — 설치 절차 + 운영 매뉴얼 (백업, 복구, 재시작)
- [ ] `docs/STUDENT_DB_GUIDE.md` — 학생용 접속 가이드 (DBeaver/psql/Node/Python 예제)

## 보안 체크리스트

- [ ] superuser 비밀번호는 서버에만 저장, 로컬/git 금지
- [ ] 학생 계정은 `SUPERUSER`, `CREATEDB`, `CREATEROLE` 권한 없음
- [ ] `REVOKE ALL ON SCHEMA public FROM public` 각 DB에 적용 (유저간 격리)
- [ ] SSL 인증서 (self-signed 일단 OK, Let's Encrypt는 추후)
- [ ] fail2ban 또는 `log_connections` 모니터링

## 롤백 계획

문제 발생 시:
```bash
systemctl stop pgbouncer postgresql
systemctl disable pgbouncer postgresql
# swap 유지 (해는 없음)
```
기존 blue/stage에는 영향 없음 (독립 프로세스).

## 완료 기준

- [ ] PG 16 + PgBouncer 정상 구동 6시간 이상
- [ ] 관리 스크립트로 계정 5개 생성/삭제 성공
- [ ] 외부(학생 로컬)에서 `psql -h db.earnlearning.com -p 6432 -U testuser testuser_test` 접속 성공
- [ ] 기존 LMS 서비스(blue/stage) 정상 동작 확인
- [ ] `docs/POSTGRES_SETUP.md` 작성 완료
