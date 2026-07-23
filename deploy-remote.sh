#!/bin/bash
set -euo pipefail

# ─── EarnLearning 원커맨드 배포 (로컬 빌드 → GHCR → EC2) ─────
# 이 스크립트는 빌드서버(cycorld) 위에서 직접 실행된다.
# 사용법:
#   ./deploy-remote.sh              # 로컬 빌드(이 서버=cycorld) → GHCR push → Stage 배포
#   ./deploy-remote.sh promote      # Stage에 떠 있는 이미지 태그를 그대로 Prod blue-green 배포
#   ./deploy-remote.sh rollback     # prod 즉시 롤백
#   ./deploy-remote.sh status       # 서버 상태 확인

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 배포 서버 (EC2 t3.small)
DEPLOY_HOST="earnlearning"
DEPLOY_DIR="/home/ubuntu/lms/deploy"
STAGE_BACKEND_CONTAINER="earnlearning-stage-backend-1"

STARTED_AT=$(date +%s)

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[remote]${NC} $*"; }
warn() { echo -e "${YELLOW}[remote]${NC} $*"; }
err()  { echo -e "${RED}[remote]${NC} $*" >&2; }

elapsed() { echo "$(( $(date +%s) - STARTED_AT ))s"; }

# ─── main 브랜치 & 클린 상태 확인 ────────────────────────────
# 빌드 전에 반드시 호출. 배포는 origin/main 과 일치하는 clean main 에서만.
# 이 스크립트는 브랜치를 변경/pull/merge 하지 않는다 (fetch 만 수행).

ensure_clean_main() {
  local branch
  branch="$(git -C "$PROJECT_DIR" rev-parse --abbrev-ref HEAD)"
  if [ "$branch" != "main" ]; then
    err "현재 브랜치가 main이 아닙니다: ${branch}"
    err "배포는 main 브랜치에서만 가능합니다. 개발 브랜치를 자동으로 checkout/pull 하지 않습니다."
    err "직접 'git checkout main && git pull --ff-only' 후 다시 실행하세요."
    exit 1
  fi

  if [ -n "$(git -C "$PROJECT_DIR" status --porcelain)" ]; then
    err "작업 트리가 clean 하지 않습니다. 커밋되지 않은 변경사항이 있습니다:"
    git -C "$PROJECT_DIR" status --short >&2
    err "변경사항을 커밋하거나 stash 한 후 다시 실행하세요."
    exit 1
  fi

  git -C "$PROJECT_DIR" fetch origin main

  local local_sha
  local_sha="$(git -C "$PROJECT_DIR" rev-parse HEAD)"
  local remote_sha
  remote_sha="$(git -C "$PROJECT_DIR" rev-parse origin/main)"
  if [ "$local_sha" != "$remote_sha" ]; then
    err "로컬 main이 origin/main과 일치하지 않습니다."
    err "  local  : ${local_sha}"
    err "  origin : ${remote_sha}"
    err "직접 'git pull --ff-only' 로 동기화한 후 다시 실행하세요."
    exit 1
  fi
}

# ─── 빌드 + Push + Stage 배포 ────────────────────────────────

deploy_to_stage() {
  log "=== 사전 확인: clean main & origin 동기화 ==="
  ensure_clean_main

  log "=== 1단계: 로컬(cycorld)에서 빌드 + GHCR Push ==="
  # build-and-push.sh 는 stdout 으로 태그만, 로그는 stderr 로 출력한다.
  IMAGE_TAG="$("$PROJECT_DIR/deploy/build-and-push.sh")"

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
  # Stage에 실제로 떠 있는 이미지 태그를 그대로 Prod로 승격한다.
  # (EC2 git HEAD 는 Stage가 실행 중인 이미지와 다를 수 있으므로 읽지 않는다.)
  local image_tag="${IMAGE_TAG:-}"

  if [ -z "$image_tag" ]; then
    local stage_image
    stage_image="$(ssh "$DEPLOY_HOST" "sudo docker inspect --format '{{.Config.Image}}' ${STAGE_BACKEND_CONTAINER}")"
    image_tag="${stage_image##*:}"
  fi

  if [ -z "$image_tag" ] || [ "$image_tag" = "latest" ]; then
    err "Stage 이미지 태그를 확인할 수 없습니다 (tag='${image_tag}')"
    err "명시적으로 태그를 지정해 다시 실행하세요: IMAGE_TAG=<sha> ./deploy-remote.sh promote"
    exit 1
  fi

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
    echo "  deploy     로컬 빌드(이 서버=cycorld) → GHCR push → Stage 배포 (기본)"
    echo "  promote    Stage에 떠 있는 이미지 태그를 그대로 Prod blue-green 배포"
    echo "  rollback   Prod 즉시 롤백"
    echo "  status     서버 상태 확인"
    exit 1
    ;;
esac
