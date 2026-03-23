#!/bin/bash
set -euo pipefail

# ─── EarnLearning 원커맨드 배포 (로컬 → GHCR → EC2) ──────────
# 사용법:
#   ./deploy-remote.sh              # 빌드 → push → stage 배포
#   ./deploy-remote.sh promote      # prod blue-green 배포
#   ./deploy-remote.sh rollback     # prod 즉시 롤백
#   ./deploy-remote.sh status       # 서버 상태 확인

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
SSH_HOST="earnlearning"
REMOTE_DEPLOY_DIR="/home/ubuntu/lms/deploy"
STARTED_AT=$(date +%s)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[remote]${NC} $*"; }
warn() { echo -e "${YELLOW}[remote]${NC} $*"; }
err()  { echo -e "${RED}[remote]${NC} $*" >&2; }

elapsed() { echo "$(( $(date +%s) - STARTED_AT ))s"; }

# ─── 빌드 + Push + Stage 배포 ────────────────────────────────

deploy_to_stage() {
  log "=== 1단계: 로컬 빌드 + GHCR Push ==="

  # 빌드 + push (마지막 줄이 IMAGE_TAG 출력)
  IMAGE_TAG=$("$PROJECT_DIR/deploy/build-and-push.sh")

  log "=== 2단계: Stage 배포 (SSH) ==="
  ssh "$SSH_HOST" "cd ${REMOTE_DEPLOY_DIR} && IMAGE_TAG=${IMAGE_TAG} ./deploy.sh stage"

  log ""
  log "=== Stage 배포 완료! ($(elapsed)) ==="
  log "IMAGE_TAG: ${IMAGE_TAG}"
  log "확인: https://stage.earnlearning.com"
  log ""
  log "Prod 배포하려면: ./deploy-remote.sh promote"
}

# ─── Prod 배포 (promote) ─────────────────────────────────────

promote_to_prod() {
  # 최신 IMAGE_TAG 확인 (GHCR에서 가져올 태그)
  local image_tag="${IMAGE_TAG:-$(cd "$PROJECT_DIR" && git rev-parse --short HEAD)}"

  log "=== Prod Blue-Green 배포 ==="
  log "IMAGE_TAG: ${image_tag}"
  ssh "$SSH_HOST" "cd ${REMOTE_DEPLOY_DIR} && IMAGE_TAG=${image_tag} ./deploy.sh prod"

  log "=== Prod 배포 완료! ($(elapsed)) ==="
  log "URL: https://earnlearning.com"
}

# ─── Rollback ────────────────────────────────────────────────

rollback_prod() {
  log "=== Prod 롤백 ==="
  ssh "$SSH_HOST" "cd ${REMOTE_DEPLOY_DIR} && ./deploy.sh rollback"
  log "=== 롤백 완료! ($(elapsed)) ==="
}

# ─── Status ──────────────────────────────────────────────────

show_status() {
  ssh "$SSH_HOST" "cd ${REMOTE_DEPLOY_DIR} && ./deploy.sh status"
}

# ─── Main ─────────────────────────────────────────────────────

case "${1:-deploy}" in
  deploy)
    deploy_to_stage
    ;;
  promote)
    promote_to_prod
    ;;
  rollback)
    rollback_prod
    ;;
  status)
    show_status
    ;;
  *)
    echo "사용법: $0 {deploy|promote|rollback|status}"
    echo ""
    echo "  deploy     빌드 → GHCR push → Stage 배포 (기본)"
    echo "  promote    Prod blue-green 배포"
    echo "  rollback   Prod 즉시 롤백"
    echo "  status     서버 상태 확인"
    exit 1
    ;;
esac
