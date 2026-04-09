#!/usr/bin/env bash
# PostgreSQL 16 + PgBouncer 설치 스크립트
# 대상: earnlearning EC2 (t3.small, Ubuntu 24.04)
# 실행: ssh earnlearning sudo bash < deploy/postgres/install.sh
#
# 멱등성: 재실행해도 안전. 기존 데이터 보존.
# 실패 시: set -e 로 즉시 중단
#
# 설계 포인트:
#   - PgBouncer는 auth_query 방식 사용 (userlist.txt 수동 관리 불필요)
#   - 동적 계정 추가/삭제 시 pgbouncer 재로드 없이 즉시 반영
#   - 학생용 DB는 public schema 격리, CREATE 권한 제한

set -euo pipefail

POSTGRES_VERSION=16
PG_CONF_DIR="/etc/postgresql/${POSTGRES_VERSION}/main"
PGBOUNCER_CONF_DIR="/etc/pgbouncer"
SECRETS_DIR="/root/earnlearning-secrets"
POSTGRES_ADMIN_PASSWORD_FILE="${SECRETS_DIR}/postgres_admin_password"
PGBOUNCER_AUTH_PASSWORD_FILE="${SECRETS_DIR}/pgbouncer_auth_password"

log() { echo "[install] $*"; }
err() { echo "[install][ERROR] $*" >&2; }

require_root() {
  if [[ $EUID -ne 0 ]]; then
    err "root로 실행해야 합니다 (sudo bash install.sh)"
    exit 1
  fi
}

gen_password() {
  openssl rand -base64 24 | tr -d '=+/' | cut -c1-24
}

step_swap() {
  log "1/7 swap 2GB 추가"
  if swapon --show | grep -q '/swapfile'; then
    log "  - 이미 활성화됨, skip"
    return
  fi
  if [[ ! -f /swapfile ]]; then
    fallocate -l 2G /swapfile
    chmod 600 /swapfile
    mkswap /swapfile
  fi
  swapon /swapfile
  if ! grep -q '/swapfile' /etc/fstab; then
    echo '/swapfile none swap sw 0 0' >> /etc/fstab
  fi
  sysctl -w vm.swappiness=10 >/dev/null
  grep -q '^vm.swappiness' /etc/sysctl.conf || echo 'vm.swappiness=10' >> /etc/sysctl.conf
  log "  - swap 활성화 완료"
}

step_install_packages() {
  log "2/7 PostgreSQL ${POSTGRES_VERSION} + PgBouncer 설치"
  if command -v psql >/dev/null && psql --version | grep -q " ${POSTGRES_VERSION}\."; then
    log "  - PostgreSQL ${POSTGRES_VERSION} 이미 설치, skip"
  else
    install -d /etc/apt/keyrings
    if [[ ! -f /etc/apt/keyrings/postgresql.gpg ]]; then
      curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
        | gpg --dearmor -o /etc/apt/keyrings/postgresql.gpg
    fi
    codename=$(lsb_release -cs)
    echo "deb [signed-by=/etc/apt/keyrings/postgresql.gpg] http://apt.postgresql.org/pub/repos/apt ${codename}-pgdg main" \
      > /etc/apt/sources.list.d/pgdg.list
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      postgresql-${POSTGRES_VERSION} \
      postgresql-contrib-${POSTGRES_VERSION} \
      pgbouncer
  fi
  systemctl enable postgresql pgbouncer >/dev/null 2>&1 || true
}

step_tune_postgres() {
  log "3/7 postgresql.conf 튜닝"
  local conf="${PG_CONF_DIR}/postgresql.conf"
  local marker_begin="# === earnlearning tuning ==="
  local marker_end="# === earnlearning tuning end ==="

  if grep -q "${marker_begin}" "${conf}"; then
    sed -i "/${marker_begin}/,/${marker_end}/d" "${conf}"
  fi

  cat >> "${conf}" <<EOF
${marker_begin}
# t3.small(2GB RAM) + 50명 학생용 (2026-04-10)
listen_addresses = '*'
port = 5432
max_connections = 60
shared_buffers = 128MB
effective_cache_size = 512MB
work_mem = 2MB
maintenance_work_mem = 32MB
wal_buffers = 4MB
min_wal_size = 80MB
max_wal_size = 1GB
checkpoint_completion_target = 0.9
random_page_cost = 1.1
statement_timeout = 30000                      # 30s, 무한루프 차단
idle_in_transaction_session_timeout = 60000
lock_timeout = 10000
log_min_duration_statement = 1000              # 1s 이상 쿼리 기록
log_connections = on
log_disconnections = on
log_line_prefix = '%t [%p] %q%u@%d '
password_encryption = scram-sha-256
${marker_end}
EOF
  log "  - 튜닝 완료"
}

step_pg_hba() {
  log "4/7 pg_hba.conf 규칙 추가"
  local hba="${PG_CONF_DIR}/pg_hba.conf"
  local marker_begin="# === earnlearning rules ==="
  local marker_end="# === earnlearning rules end ==="

  if grep -q "${marker_begin}" "${hba}"; then
    sed -i "/${marker_begin}/,/${marker_end}/d" "${hba}"
  fi

  cat >> "${hba}" <<EOF
${marker_begin}
local   all             postgres                                peer
local   all             all                                     scram-sha-256
host    all             all             127.0.0.1/32            scram-sha-256
host    all             all             ::1/128                 scram-sha-256
# 외부 학생 접속 (PgBouncer 경유 권장, 직접도 허용)
host    all             all             0.0.0.0/0               scram-sha-256
host    all             all             ::/0                    scram-sha-256
${marker_end}
EOF
  log "  - pg_hba.conf 업데이트 완료"
}

step_secrets() {
  log "5/7 비밀번호 생성 및 설정"
  install -d -m 700 "${SECRETS_DIR}"

  if [[ ! -f "${POSTGRES_ADMIN_PASSWORD_FILE}" ]]; then
    gen_password > "${POSTGRES_ADMIN_PASSWORD_FILE}"
    chmod 600 "${POSTGRES_ADMIN_PASSWORD_FILE}"
    log "  - postgres 비밀번호 생성"
  fi
  if [[ ! -f "${PGBOUNCER_AUTH_PASSWORD_FILE}" ]]; then
    gen_password > "${PGBOUNCER_AUTH_PASSWORD_FILE}"
    chmod 600 "${PGBOUNCER_AUTH_PASSWORD_FILE}"
    log "  - pgbouncer_auth 비밀번호 생성"
  fi

  systemctl restart postgresql
  sleep 2

  local pg_pw pgb_pw
  pg_pw=$(cat "${POSTGRES_ADMIN_PASSWORD_FILE}")
  pgb_pw=$(cat "${PGBOUNCER_AUTH_PASSWORD_FILE}")

  sudo -u postgres psql -v ON_ERROR_STOP=1 <<SQL
ALTER USER postgres WITH PASSWORD '${pg_pw}';

-- pgbouncer_auth: auth_query 전용 (로그인 가능, 최소 권한)
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='pgbouncer_auth') THEN
    CREATE ROLE pgbouncer_auth LOGIN PASSWORD '${pgb_pw}';
  ELSE
    ALTER ROLE pgbouncer_auth WITH PASSWORD '${pgb_pw}';
  END IF;
END
\$\$;

-- auth_query 함수: pg_shadow 조회용 (SECURITY DEFINER)
CREATE OR REPLACE FUNCTION public.pgbouncer_get_auth(p_usename TEXT)
RETURNS TABLE(usename TEXT, passwd TEXT)
LANGUAGE plpgsql SECURITY DEFINER
AS \$\$
BEGIN
  RETURN QUERY
    SELECT s.usename::TEXT, s.passwd::TEXT
    FROM pg_catalog.pg_shadow s
    WHERE s.usename = p_usename;
END;
\$\$;

REVOKE ALL ON FUNCTION public.pgbouncer_get_auth(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.pgbouncer_get_auth(TEXT) TO pgbouncer_auth;

-- 격리: PUBLIC 은 postgres DB 에 접속 못 하게 차단 (학생 계정도 못 들어옴)
-- pgbouncer_auth 만 auth_query 용으로 접속 허용
REVOKE CONNECT ON DATABASE postgres FROM PUBLIC;
GRANT CONNECT ON DATABASE postgres TO pgbouncer_auth;
SQL
  log "  - postgres / pgbouncer_auth 설정 완료"
}

step_pgbouncer() {
  log "6/7 PgBouncer 설정"
  local ini="${PGBOUNCER_CONF_DIR}/pgbouncer.ini"
  local userlist="${PGBOUNCER_CONF_DIR}/userlist.txt"
  local pgb_pw
  pgb_pw=$(cat "${PGBOUNCER_AUTH_PASSWORD_FILE}")

  # pgbouncer_auth 는 userlist.txt 에 cleartext 로 저장한다.
  # SCRAM 해시로는 pgbouncer → PG 서버 측 로그인을 할 수 없기 때문
  # ("cannot do SCRAM authentication" 에러). pgbouncer_auth 는 최소 권한이고
  # userlist.txt 는 600/postgres 소유이므로 안전하다.
  cat > "${userlist}" <<EOF
"pgbouncer_auth" "${pgb_pw}"
EOF
  chown postgres:postgres "${userlist}"
  chmod 600 "${userlist}"

  cat > "${ini}" <<'EOF'
[databases]
; 와일드카드: 클라이언트가 지정한 DB명을 그대로 localhost:5432에 릴레이
; auth_dbname=postgres: auth_query를 postgres DB에서 실행 → pgbouncer_auth가
; 각 학생 DB에 CONNECT 권한을 가질 필요 없음 (격리 보장)
* = host=127.0.0.1 port=5432 auth_dbname=postgres

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
unix_socket_dir = /var/run/postgresql

auth_type = scram-sha-256
auth_file = /etc/pgbouncer/userlist.txt
auth_user = pgbouncer_auth
auth_query = SELECT usename, passwd FROM public.pgbouncer_get_auth($1)

pool_mode = transaction
max_client_conn = 200
default_pool_size = 10
reserve_pool_size = 5
reserve_pool_timeout = 3
server_idle_timeout = 300
server_lifetime = 3600
client_idle_timeout = 0
server_reset_query = DISCARD ALL
ignore_startup_parameters = extra_float_digits,search_path

admin_users = postgres
stats_users = postgres

logfile = /var/log/postgresql/pgbouncer.log
pidfile = /var/run/postgresql/pgbouncer.pid
tcp_keepalive = 1
EOF
  chown postgres:postgres "${ini}"
  chmod 640 "${ini}"

  systemctl restart pgbouncer
  sleep 1
  if systemctl is-active --quiet pgbouncer; then
    log "  - pgbouncer 시작 성공 (포트 6432)"
  else
    err "pgbouncer 시작 실패:"
    journalctl -u pgbouncer -n 30 --no-pager || true
    exit 1
  fi
}

step_management_script() {
  log "7/7 earnlearning-db 관리 스크립트 설치"
  local src
  # install.sh 과 같은 디렉토리의 earnlearning-db.sh 를 찾음
  # stdin 파이프로 실행된 경우 ${BASH_SOURCE[0]}가 없으므로 fallback
  if [[ -n "${BASH_SOURCE[0]:-}" && -f "$(dirname "${BASH_SOURCE[0]}")/earnlearning-db.sh" ]]; then
    src="$(dirname "${BASH_SOURCE[0]}")/earnlearning-db.sh"
    install -m 755 "${src}" /usr/local/bin/earnlearning-db
  elif [[ -f /tmp/earnlearning-db.sh ]]; then
    install -m 755 /tmp/earnlearning-db.sh /usr/local/bin/earnlearning-db
  else
    log "  - earnlearning-db.sh 파일 없음, 별도 배포 필요"
    log "    scp deploy/postgres/earnlearning-db.sh earnlearning:/tmp/"
    log "    ssh earnlearning sudo install -m 755 /tmp/earnlearning-db.sh /usr/local/bin/earnlearning-db"
  fi
  touch /var/log/earnlearning-db.log
  chmod 640 /var/log/earnlearning-db.log || true
}

summary() {
  log ""
  log "=========================================="
  log "설치 완료!"
  log "=========================================="
  log ""
  log "다음 단계:"
  log "  1) AWS Security Group에서 TCP 6432 개방"
  log "  2) db.earnlearning.com DNS A record 추가 (Cloudflare Proxy: OFF)"
  log "  3) 테스트:"
  log "     earnlearning-db create testuser myproj"
  log "     psql -h 127.0.0.1 -p 6432 -U testuser_myproj testuser_myproj"
  log ""
  log "비밀번호 파일:"
  log "  postgres:       ${POSTGRES_ADMIN_PASSWORD_FILE}"
  log "  pgbouncer_auth: ${PGBOUNCER_AUTH_PASSWORD_FILE}"
  log ""
  log "현재 리소스:"
  free -h | sed 's/^/  /'
}

main() {
  require_root
  log "PostgreSQL + PgBouncer 설치 시작"
  step_swap
  step_install_packages
  step_tune_postgres
  step_pg_hba
  step_secrets
  systemctl restart postgresql
  sleep 2
  step_pgbouncer
  step_management_script
  summary
}

main "$@"
