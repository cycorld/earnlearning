#!/bin/bash
set -euo pipefail

# ─── EarnLearning 배포 스크립트 ────────────────────────────────
# 사용법:
#   ./deploy.sh stage          # 스테이지 배포 (빠름)
#   ./deploy.sh prod           # 프로덕션 배포 (정석)
#   ./deploy.sh promote        # 스테이지 → 프로덕션 프로모트 (가장 빠름)
#   ./deploy.sh prod --full    # 프로덕션 풀 빌드 (캐시 무시)

DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$DEPLOY_DIR")"
STARTED_AT=$(date +%s)

# Build info (from CI env or git)
export BUILD_NUMBER="${BUILD_NUMBER:-$(cd "$PROJECT_DIR" && git rev-list --count HEAD 2>/dev/null || echo 'dev')}"
export COMMIT_SHA="${COMMIT_SHA:-$(cd "$PROJECT_DIR" && git rev-parse HEAD 2>/dev/null || echo 'unknown')}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[deploy]${NC} $*"; }
warn() { echo -e "${YELLOW}[deploy]${NC} $*"; }
err()  { echo -e "${RED}[deploy]${NC} $*" >&2; }

elapsed() {
  local now=$(date +%s)
  echo "$(( now - STARTED_AT ))s"
}

# ─── Git Pull ──────────────────────────────────────────────────
pull_latest() {
  log "Pulling latest code..."
  cd "$PROJECT_DIR" && git pull --ff-only
}

# ─── Stage 배포 (빠른 모드) ────────────────────────────────────
deploy_stage() {
  log "=== Stage 배포 시작 ==="
  log "Build #${BUILD_NUMBER} / ${COMMIT_SHA:0:7}"
  pull_latest

  cd "$DEPLOY_DIR"
  BUILD_NUMBER="$BUILD_NUMBER" COMMIT_SHA="$COMMIT_SHA" \
  sudo -E docker compose \
    -f docker-compose.stage.yml \
    -p earnlearning-stage \
    --env-file .env.stage \
    up -d --build --force-recreate

  log "Stage 배포 완료! ($(elapsed))"
  log "URL: https://stage.earnlearning.com"
}

# ─── Prod 배포 (정석 모드) ─────────────────────────────────────
deploy_prod() {
  local full_build="${1:-}"
  log "=== Production 배포 시작 ==="
  pull_latest

  cd "$DEPLOY_DIR"

  local extra_args=""
  if [ "$full_build" = "--full" ]; then
    extra_args="--no-cache"
    warn "Full build (no cache)"
  fi

  BUILD_NUMBER="$BUILD_NUMBER" COMMIT_SHA="$COMMIT_SHA" \
  sudo -E docker compose \
    -f docker-compose.prod.yml \
    -p earnlearning-prod \
    --env-file .env.prod \
    up -d --build --force-recreate $extra_args

  # 헬스 체크
  log "Health check..."
  sleep 3
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/auth/login 2>/dev/null || echo "000")
  if [ "$status" = "405" ] || [ "$status" = "200" ]; then
    log "Backend healthy (HTTP $status)"
  else
    warn "Backend returned HTTP $status — check logs!"
  fi

  log "Production 배포 완료! ($(elapsed))"
  log "URL: https://earnlearning.com"
}

# ─── Promote: Stage → Prod (가장 빠름) ─────────────────────────
# Stage에서 이미 빌드된 이미지를 그대로 Prod에 사용.
# 빌드 단계를 완전히 건너뜀 → ~5초
promote_stage_to_prod() {
  log "=== Stage → Production 프로모트 ==="

  # 1. Stage 이미지가 존재하는지 확인
  if ! sudo docker image inspect earnlearning-stage-backend:latest > /dev/null 2>&1; then
    err "Stage backend 이미지가 없습니다. 먼저 stage를 배포하세요."
    exit 1
  fi
  if ! sudo docker image inspect earnlearning-stage-frontend:latest > /dev/null 2>&1; then
    err "Stage frontend 이미지가 없습니다. 먼저 stage를 배포하세요."
    exit 1
  fi

  # 2. Stage 이미지를 Prod 태그로 복사
  log "Tagging stage images as prod..."
  sudo docker tag earnlearning-stage-backend:latest earnlearning-prod-backend:latest
  sudo docker tag earnlearning-stage-frontend:latest earnlearning-prod-frontend:latest

  # 3. Prod 컨테이너만 재시작 (빌드 없이)
  cd "$DEPLOY_DIR"
  sudo docker compose \
    -f docker-compose.prod.yml \
    -p earnlearning-prod \
    --env-file .env.prod \
    up -d --force-recreate --no-build

  # 4. 헬스 체크
  log "Health check..."
  sleep 3
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/auth/login 2>/dev/null || echo "000")
  if [ "$status" = "405" ] || [ "$status" = "200" ]; then
    log "Backend healthy (HTTP $status)"
  else
    warn "Backend returned HTTP $status — check logs!"
  fi

  log "Promote 완료! Stage → Production ($(elapsed))"
  log "URL: https://earnlearning.com"
}

# ─── 사용 후 정리 ─────────────────────────────────────────────
cleanup() {
  log "Cleaning up dangling images..."
  sudo docker image prune -f > /dev/null 2>&1
}

# ─── Main ──────────────────────────────────────────────────────
case "${1:-}" in
  stage)
    deploy_stage
    cleanup
    ;;
  prod)
    deploy_prod "${2:-}"
    cleanup
    ;;
  promote)
    promote_stage_to_prod
    cleanup
    ;;
  *)
    echo "사용법: $0 {stage|prod|promote}"
    echo ""
    echo "  stage           Stage 배포 (빌드 + 배포)"
    echo "  prod            Production 배포 (빌드 + 배포 + 헬스체크)"
    echo "  prod --full     Production 풀 빌드 (캐시 무시)"
    echo "  promote         Stage 이미지 → Production 프로모트 (빌드 스킵, ~5초)"
    exit 1
    ;;
esac
