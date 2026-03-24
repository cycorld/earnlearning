#!/bin/bash
set -euo pipefail

# ─── EarnLearning 원커맨드 배포 (빌드서버 → GHCR → EC2) ─────
# 사용법:
#   ./deploy-remote.sh              # 빌드 → push → stage 배포
#   ./deploy-remote.sh promote      # prod blue-green 배포
#   ./deploy-remote.sh rollback     # prod 즉시 롤백
#   ./deploy-remote.sh status       # 서버 상태 확인

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 빌드 서버 (cycorld: x86_64 16코어 60GB — 네이티브 amd64 빌드)
BUILD_HOST="cycorld"
BUILD_REPO_DIR="/home/cycorld/Workspace/earnlearning"

# 배포 서버 (EC2 t3.small)
DEPLOY_HOST="earnlearning"
DEPLOY_DIR="/home/ubuntu/lms/deploy"

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
  log "=== 1단계: 빌드서버(cycorld)에서 빌드 + GHCR Push ==="

  # 빌드 서버에서 최신 코드 pull + 빌드 + push
  IMAGE_TAG=$(ssh "$BUILD_HOST" "cd ${BUILD_REPO_DIR} && git pull --ff-only >&2 && ./deploy/build-and-push.sh")

  log "IMAGE_TAG: ${IMAGE_TAG}"

  log "=== 2단계: Stage 배포 (EC2) ==="
  ssh "$DEPLOY_HOST" "cd ${DEPLOY_DIR} && IMAGE_TAG=${IMAGE_TAG} ./deploy.sh stage"

  log ""
  log "=== Stage 배포 완료! ($(elapsed)) ==="
  log "IMAGE_TAG: ${IMAGE_TAG}"
  log "확인: https://stage.earnlearning.com"
  log ""
  log "Prod 배포하려면: ./deploy-remote.sh promote"
}

# ─── Prod 배포 (promote) ─────────────────────────────────────

promote_to_prod() {
  # Stage에 배포된 이미지 태그를 서버에서 가져옴 (로컬 HEAD와 다를 수 있음)
  local image_tag="${IMAGE_TAG:-$(ssh "$DEPLOY_HOST" "cd /home/ubuntu/lms && git rev-parse --short HEAD")}"

  log "=== Prod Blue-Green 배포 ==="
  log "IMAGE_TAG: ${image_tag}"
  ssh "$DEPLOY_HOST" "cd ${DEPLOY_DIR} && IMAGE_TAG=${image_tag} ./deploy.sh prod"

  log "=== Prod 배포 완료! ($(elapsed)) ==="
  log "URL: https://earnlearning.com"
}

# ─── Rollback ────────────────────────────────────────────────

rollback_prod() {
  log "=== Prod 롤백 ==="
  ssh "$DEPLOY_HOST" "cd ${DEPLOY_DIR} && ./deploy.sh rollback"
  log "=== 롤백 완료! ($(elapsed)) ==="
}

# ─── Status ──────────────────────────────────────────────────

show_status() {
  ssh "$DEPLOY_HOST" "cd ${DEPLOY_DIR} && ./deploy.sh status"
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
    echo "  deploy     빌드(cycorld) → GHCR push → Stage 배포 (기본)"
    echo "  promote    Prod blue-green 배포"
    echo "  rollback   Prod 즉시 롤백"
    echo "  status     서버 상태 확인"
    exit 1
    ;;
esac
