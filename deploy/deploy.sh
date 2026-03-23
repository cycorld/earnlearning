#!/bin/bash
set -euo pipefail

# ─── EarnLearning 서버 배포 스크립트 (Blue-Green) ─────────────
# 사용법:
#   ./deploy.sh stage              # Stage 배포 (pull + restart)
#   ./deploy.sh prod               # Prod blue-green 배포
#   ./deploy.sh rollback           # Prod 즉시 롤백
#   ./deploy.sh status             # 현재 상태 표시

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$DEPLOY_DIR")"
ACTIVE_SLOT_CONF="/etc/nginx/earnlearning-active-slot.conf"
STARTED_AT=$(date +%s)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${GREEN}[deploy]${NC} $*"; }
warn() { echo -e "${YELLOW}[deploy]${NC} $*"; }
err()  { echo -e "${RED}[deploy]${NC} $*" >&2; }
info() { echo -e "${CYAN}[deploy]${NC} $*"; }

elapsed() { echo "$(( $(date +%s) - STARTED_AT ))s"; }

# ─── Slot 관리 ────────────────────────────────────────────────

get_active_slot() {
  if [ ! -f "$ACTIVE_SLOT_CONF" ]; then
    echo "none"
    return
  fi
  local content
  content=$(cat "$ACTIVE_SLOT_CONF")
  if echo "$content" | grep -q "8180"; then
    echo "blue"
  elif echo "$content" | grep -q "8181"; then
    echo "green"
  else
    echo "none"
  fi
}

get_inactive_slot() {
  local active
  active=$(get_active_slot)
  case "$active" in
    blue)  echo "green" ;;
    green) echo "blue" ;;
    *)     echo "blue" ;;   # 초기 상태: blue로 시작
  esac
}

slot_port() {
  case "$1" in
    blue)  echo "8180" ;;
    green) echo "8181" ;;
    stage) echo "8182" ;;
  esac
}

slot_compose() {
  case "$1" in
    blue)  echo "docker-compose.blue.yml" ;;
    green) echo "docker-compose.green.yml" ;;
    stage) echo "docker-compose.stage.yml" ;;
  esac
}

slot_project() {
  case "$1" in
    blue)  echo "earnlearning-blue" ;;
    green) echo "earnlearning-green" ;;
    stage) echo "earnlearning-stage" ;;
  esac
}

# ─── Health Check ─────────────────────────────────────────────

healthcheck() {
  local port="$1"
  local max_attempts=15
  local attempt=0

  log "Health check on port ${port}..."
  while [ $attempt -lt $max_attempts ]; do
    attempt=$((attempt + 1))
    local status
    status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${port}/api/health" 2>/dev/null || echo "000")
    if [ "$status" = "200" ]; then
      log "Healthy! (attempt ${attempt}/${max_attempts})"
      return 0
    fi
    sleep 2
  done

  err "Health check failed after ${max_attempts} attempts"
  return 1
}

# ─── Stage 배포 ──────────────────────────────────────────────

deploy_stage() {
  log "=== Stage 배포 시작 ==="
  log "IMAGE_TAG=${IMAGE_TAG:-latest}"

  cd "$DEPLOY_DIR"

  # tasks 디렉토리를 위해 git pull
  cd "$PROJECT_DIR" && git pull --ff-only 2>/dev/null || true
  cd "$DEPLOY_DIR"

  sudo -E docker compose \
    -f "$(slot_compose stage)" \
    -p "$(slot_project stage)" \
    pull

  sudo -E docker compose \
    -f "$(slot_compose stage)" \
    -p "$(slot_project stage)" \
    up -d --force-recreate

  if healthcheck "$(slot_port stage)"; then
    log "Stage 배포 완료! ($(elapsed))"
    log "URL: https://stage.earnlearning.com"
  else
    warn "Stage 배포됨, 헬스체크 실패 — 로그를 확인하세요"
  fi

  cleanup
}

# ─── Prod Blue-Green 배포 ────────────────────────────────────

deploy_prod() {
  local active
  active=$(get_active_slot)
  local target
  target=$(get_inactive_slot)
  local target_port
  target_port=$(slot_port "$target")

  log "=== Production Blue-Green 배포 ==="
  log "IMAGE_TAG=${IMAGE_TAG:-latest}"
  log "현재 active: ${active} → 배포 대상: ${target} (port ${target_port})"

  cd "$DEPLOY_DIR"

  # tasks 디렉토리를 위해 git pull
  cd "$PROJECT_DIR" && git pull --ff-only 2>/dev/null || true
  cd "$DEPLOY_DIR"

  # 1. 비활성 slot에 pull + 시작
  log "Pulling images for ${target}..."
  sudo -E docker compose \
    -f "$(slot_compose "$target")" \
    -p "$(slot_project "$target")" \
    pull

  log "Starting ${target} slot..."
  sudo -E docker compose \
    -f "$(slot_compose "$target")" \
    -p "$(slot_project "$target")" \
    up -d --force-recreate

  # 2. Health check
  if ! healthcheck "$target_port"; then
    err "Health check 실패! ${target} slot을 정지합니다."
    sudo docker compose \
      -f "$(slot_compose "$target")" \
      -p "$(slot_project "$target")" \
      down
    exit 1
  fi

  # 3. Nginx upstream 전환
  log "Nginx upstream 전환: ${active} → ${target}"
  echo "server 127.0.0.1:${target_port};" | sudo tee "$ACTIVE_SLOT_CONF" > /dev/null
  sudo nginx -t && sudo nginx -s reload

  log "Active slot: ${target}"

  # 4. 이전 slot 정리 (active가 있었을 때만)
  if [ "$active" != "none" ]; then
    log "이전 slot(${active}) 정리 중..."
    sleep 5  # graceful drain
    sudo docker compose \
      -f "$(slot_compose "$active")" \
      -p "$(slot_project "$active")" \
      down 2>/dev/null || true
  fi

  cleanup

  log "=== Production 배포 완료! ($(elapsed)) ==="
  log "URL: https://earnlearning.com"
}

# ─── Rollback ────────────────────────────────────────────────

rollback() {
  local active
  active=$(get_active_slot)
  local previous
  previous=$(get_inactive_slot)
  local previous_port
  previous_port=$(slot_port "$previous")

  log "=== Rollback: ${active} → ${previous} ==="

  # 이전 slot이 실행 중인지 확인
  local running
  running=$(sudo docker compose \
    -f "$DEPLOY_DIR/$(slot_compose "$previous")" \
    -p "$(slot_project "$previous")" \
    ps --format json 2>/dev/null | grep -c "running" || echo "0")

  if [ "$running" = "0" ]; then
    err "이전 slot(${previous})이 실행 중이 아닙니다."
    err "이전 slot을 먼저 시작해야 합니다: IMAGE_TAG=<tag> ./deploy.sh prod"
    exit 1
  fi

  # Nginx 전환
  echo "server 127.0.0.1:${previous_port};" | sudo tee "$ACTIVE_SLOT_CONF" > /dev/null
  sudo nginx -t && sudo nginx -s reload

  log "Rollback 완료! Active: ${previous} ($(elapsed))"
  log "URL: https://earnlearning.com"
}

# ─── Status ──────────────────────────────────────────────────

show_status() {
  local active
  active=$(get_active_slot)
  info "=== EarnLearning 배포 상태 ==="
  info "Active slot: ${active}"
  info ""

  for slot in blue green stage; do
    local compose_file="$DEPLOY_DIR/$(slot_compose "$slot")"
    local project="$(slot_project "$slot")"
    local port="$(slot_port "$slot")"
    local status_icon="⚪"

    if [ "$slot" = "$active" ]; then
      status_icon="🟢"
    fi

    local ps_output
    ps_output=$(sudo docker compose -f "$compose_file" -p "$project" ps --format "table {{.Name}}\t{{.Status}}" 2>/dev/null || echo "  (not running)")

    info "${status_icon} ${slot} (port ${port}):"
    echo "$ps_output" | sed 's/^/  /'
    info ""
  done
}

# ─── Cleanup ─────────────────────────────────────────────────

cleanup() {
  log "Cleaning up dangling images..."
  sudo docker image prune -f > /dev/null 2>&1 || true
}

# ─── Main ─────────────────────────────────────────────────────

export IMAGE_TAG="${IMAGE_TAG:-latest}"

case "${1:-}" in
  stage)
    deploy_stage
    ;;
  prod)
    deploy_prod
    ;;
  rollback)
    rollback
    ;;
  status)
    show_status
    ;;
  *)
    echo "사용법: $0 {stage|prod|rollback|status}"
    echo ""
    echo "  stage      Stage slot에 pull + 배포"
    echo "  prod       비활성 slot에 blue-green 배포"
    echo "  rollback   이전 slot으로 즉시 롤백"
    echo "  status     현재 active slot 및 상태 표시"
    echo ""
    echo "환경변수:"
    echo "  IMAGE_TAG  이미지 태그 (기본: latest)"
    exit 1
    ;;
esac
