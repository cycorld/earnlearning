#!/usr/bin/env bash
# earnlearning-db: 학생용 PostgreSQL 계정/DB 관리 스크립트
#
# 사용법:
#   earnlearning-db create <username> <projname>
#   earnlearning-db delete <username> <projname>
#   earnlearning-db rotate <username> <projname>
#   earnlearning-db list   <username>
#   earnlearning-db list-all
#
# 이름 규칙: ^[a-z][a-z0-9_]{2,31}$  (소문자 시작, 소문자/숫자/밑줄, 3~32자)
# 생성되는 PG 유저명/DB명: {username}_{projname}
#
# 권한:
#   - 생성된 유저는 본인 DB에만 CONNECT 가능
#   - 본인 DB의 public schema에서 모든 DDL/DML 가능
#   - 다른 DB/시스템 테이블 접근 불가

set -euo pipefail

SECRETS_DIR="/root/earnlearning-secrets"
POSTGRES_ADMIN_PASSWORD_FILE="${SECRETS_DIR}/postgres_admin_password"
LOG_FILE="/var/log/earnlearning-db.log"
MAX_DB_PER_USER="${MAX_DB_PER_USER:-5}"

usage() {
  sed -n '3,16p' "$0"
  exit 1
}

log() {
  local ts
  ts=$(date -Iseconds)
  echo "[${ts}] $*" >> "${LOG_FILE}"
  echo "$*"
}

require_root() {
  if [[ $EUID -ne 0 ]]; then
    echo "ERROR: root로 실행해야 합니다 (sudo earnlearning-db ...)" >&2
    exit 1
  fi
}

validate_name() {
  local name="$1" field="$2"
  if [[ ! "${name}" =~ ^[a-z][a-z0-9_]{2,31}$ ]]; then
    echo "ERROR: ${field} '${name}' 은 유효하지 않습니다" >&2
    echo "       규칙: 소문자 시작, 소문자/숫자/밑줄만, 3~32자" >&2
    exit 2
  fi
}

psql_admin() {
  # postgres superuser 로 SQL 실행 (peer auth)
  sudo -u postgres psql -v ON_ERROR_STOP=1 "$@"
}

gen_password() {
  openssl rand -base64 24 | tr -d '=+/' | cut -c1-24
}

# identifier 안전 인용 (quote_ident 와 유사)
quote_ident() {
  local s="${1//\"/\"\"}"
  echo "\"${s}\""
}
quote_literal() {
  local s="${1//\'/\'\'}"
  echo "'${s}'"
}

cmd_create() {
  local username="$1" projname="$2"
  validate_name "${username}" "username"
  validate_name "${projname}" "projname"

  local dbname="${username}_${projname}"
  if (( ${#dbname} > 63 )); then
    echo "ERROR: 생성될 DB명 '${dbname}' 이 63자를 초과합니다" >&2
    exit 2
  fi

  # 사용자당 DB 개수 제한
  local count
  count=$(psql_admin -Atc "
    SELECT count(*) FROM pg_database d
    JOIN pg_roles r ON d.datdba = r.oid
    WHERE d.datname LIKE '${username}\\_%' ESCAPE '\\'
      AND r.rolname LIKE '${username}\\_%' ESCAPE '\\'
  ")
  if (( count >= MAX_DB_PER_USER )); then
    echo "ERROR: 사용자 '${username}' 의 DB 개수가 한도(${MAX_DB_PER_USER})를 초과했습니다" >&2
    exit 3
  fi

  # 이미 존재하면 에러
  local exists
  exists=$(psql_admin -Atc "SELECT 1 FROM pg_database WHERE datname='${dbname}'")
  if [[ "${exists}" == "1" ]]; then
    echo "ERROR: DB '${dbname}' 이미 존재합니다" >&2
    exit 4
  fi

  local password
  password=$(gen_password)
  local q_db q_user
  q_db=$(quote_ident "${dbname}")
  q_user=$(quote_ident "${dbname}")

  # 유저 생성 → DB 생성 → 권한 설정
  psql_admin <<SQL
-- 1. 유저 생성 (superuser 권한 없음)
CREATE ROLE ${q_user} WITH
  LOGIN
  NOSUPERUSER
  NOCREATEDB
  NOCREATEROLE
  NOREPLICATION
  INHERIT
  CONNECTION LIMIT 10
  PASSWORD $(quote_literal "${password}");

-- 2. DB 생성 (유저를 owner로)
CREATE DATABASE ${q_db} OWNER ${q_user} ENCODING 'UTF8' LC_COLLATE 'C.UTF-8' LC_CTYPE 'C.UTF-8' TEMPLATE template0;

-- 3. DB-level 권한: PUBLIC 에서 CONNECT 제거, 본인만 허용
REVOKE CONNECT ON DATABASE ${q_db} FROM PUBLIC;
GRANT CONNECT ON DATABASE ${q_db} TO ${q_user};
GRANT ALL PRIVILEGES ON DATABASE ${q_db} TO ${q_user};
SQL

  # public schema 에서 PUBLIC 권한 제거 (격리)
  psql_admin -d "${dbname}" <<SQL
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO ${q_user};
ALTER SCHEMA public OWNER TO ${q_user};
SQL

  log "CREATE user=${username} proj=${projname} db=${dbname}"

  cat <<EOF
{
  "db_name": "${dbname}",
  "username": "${dbname}",
  "password": "${password}",
  "host": "db.earnlearning.com",
  "port": 6432,
  "psql": "PGPASSWORD='${password}' psql -h db.earnlearning.com -p 6432 -U ${dbname} ${dbname}",
  "url": "postgresql://${dbname}:${password}@db.earnlearning.com:6432/${dbname}"
}
EOF
}

cmd_delete() {
  local username="$1" projname="$2"
  validate_name "${username}" "username"
  validate_name "${projname}" "projname"
  local dbname="${username}_${projname}"
  local q_db q_user
  q_db=$(quote_ident "${dbname}")
  q_user=$(quote_ident "${dbname}")

  # 활성 세션 종료 → DB DROP → ROLE DROP
  psql_admin <<SQL
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = '${dbname}' AND pid <> pg_backend_pid();

DROP DATABASE IF EXISTS ${q_db};
DROP ROLE IF EXISTS ${q_user};
SQL

  log "DELETE user=${username} proj=${projname} db=${dbname}"
  echo "{\"deleted\": \"${dbname}\"}"
}

cmd_rotate() {
  local username="$1" projname="$2"
  validate_name "${username}" "username"
  validate_name "${projname}" "projname"
  local dbname="${username}_${projname}"
  local q_user
  q_user=$(quote_ident "${dbname}")

  local exists
  exists=$(psql_admin -Atc "SELECT 1 FROM pg_roles WHERE rolname='${dbname}'")
  if [[ "${exists}" != "1" ]]; then
    echo "ERROR: role '${dbname}' 이 존재하지 않습니다" >&2
    exit 4
  fi

  local password
  password=$(gen_password)
  psql_admin <<SQL
ALTER ROLE ${q_user} WITH PASSWORD $(quote_literal "${password}");
SQL

  log "ROTATE user=${username} proj=${projname} db=${dbname}"
  cat <<EOF
{
  "db_name": "${dbname}",
  "username": "${dbname}",
  "password": "${password}",
  "host": "db.earnlearning.com",
  "port": 6432
}
EOF
}

cmd_list() {
  local username="$1"
  validate_name "${username}" "username"
  psql_admin -Atc "
    SELECT json_agg(json_build_object(
      'db_name', d.datname,
      'size_bytes', pg_database_size(d.datname),
      'created', (pg_stat_file('base/' || d.oid)).modification
    ))
    FROM pg_database d
    WHERE d.datname LIKE '${username}\\_%' ESCAPE '\\'
  "
}

cmd_list_all() {
  psql_admin -c "
    SELECT
      d.datname AS db_name,
      r.rolname AS owner,
      pg_size_pretty(pg_database_size(d.datname)) AS size,
      r.rolconnlimit AS conn_limit
    FROM pg_database d
    JOIN pg_roles r ON d.datdba = r.oid
    WHERE d.datname NOT IN ('postgres','template0','template1')
    ORDER BY d.datname;
  "
}

main() {
  require_root
  local cmd="${1:-}"
  shift || true
  case "${cmd}" in
    create)   [[ $# -eq 2 ]] || usage; cmd_create "$1" "$2" ;;
    delete)   [[ $# -eq 2 ]] || usage; cmd_delete "$1" "$2" ;;
    rotate)   [[ $# -eq 2 ]] || usage; cmd_rotate "$1" "$2" ;;
    list)     [[ $# -eq 1 ]] || usage; cmd_list "$1" ;;
    list-all) cmd_list_all ;;
    *) usage ;;
  esac
}

main "$@"
