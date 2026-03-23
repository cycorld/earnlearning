#!/bin/bash
set -euo pipefail

# ─── EarnLearning 로컬 빌드 + GHCR Push ──────────────────────
# 사용법: ./deploy/build-and-push.sh [IMAGE_TAG]
# Mac에서 amd64 이미지를 빌드하여 GHCR에 Push

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

REGISTRY="ghcr.io/cycorld"
BACKEND_IMAGE="${REGISTRY}/earnlearning-backend"
FRONTEND_IMAGE="${REGISTRY}/earnlearning-frontend"

# Image tag: 인자 또는 git short SHA
IMAGE_TAG="${1:-$(cd "$PROJECT_DIR" && git rev-parse --short HEAD)}"

# Build info
BUILD_NUMBER="$(cd "$PROJECT_DIR" && git rev-list --count HEAD 2>/dev/null || echo 'dev')"
COMMIT_SHA="$(cd "$PROJECT_DIR" && git rev-parse HEAD 2>/dev/null || echo 'unknown')"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[build]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[build]${NC} $*" >&2; }
err()  { echo -e "${RED}[build]${NC} $*" >&2; }

STARTED_AT=$(date +%s)
elapsed() { echo "$(( $(date +%s) - STARTED_AT ))s"; }

# ─── Buildx 준비 ─────────────────────────────────────────────
ensure_builder() {
  if ! docker buildx inspect earnlearning-builder > /dev/null 2>&1; then
    log "Creating buildx builder..."
    docker buildx create --name earnlearning-builder --use
  else
    docker buildx use earnlearning-builder
  fi
}

# ─── Backend 빌드 ────────────────────────────────────────────
build_backend() {
  log "Building backend (amd64)... tag=${IMAGE_TAG}"
  docker buildx build \
    --platform linux/amd64 \
    --build-arg BUILD_NUMBER="$BUILD_NUMBER" \
    --build-arg COMMIT_SHA="$COMMIT_SHA" \
    -t "${BACKEND_IMAGE}:${IMAGE_TAG}" \
    -t "${BACKEND_IMAGE}:latest" \
    --push \
    -f "$PROJECT_DIR/backend/Dockerfile" \
    "$PROJECT_DIR/backend"
  log "Backend pushed: ${BACKEND_IMAGE}:${IMAGE_TAG}"
}

# ─── Frontend 빌드 ───────────────────────────────────────────
build_frontend() {
  log "Building frontend (amd64)... tag=${IMAGE_TAG}"
  docker buildx build \
    --platform linux/amd64 \
    --build-arg BUILD_NUMBER="$BUILD_NUMBER" \
    --build-arg COMMIT_SHA="$COMMIT_SHA" \
    -t "${FRONTEND_IMAGE}:${IMAGE_TAG}" \
    -t "${FRONTEND_IMAGE}:latest" \
    --push \
    -f "$PROJECT_DIR/frontend/Dockerfile" \
    "$PROJECT_DIR"
  log "Frontend pushed: ${FRONTEND_IMAGE}:${IMAGE_TAG}"
}

# ─── Main ─────────────────────────────────────────────────────
log "=== Build & Push 시작 ==="
log "Tag: ${IMAGE_TAG} | Build #${BUILD_NUMBER} | ${COMMIT_SHA:0:7}"

ensure_builder
build_backend
build_frontend

log "=== Build & Push 완료! ($(elapsed)) ==="
log "Backend:  ${BACKEND_IMAGE}:${IMAGE_TAG}"
log "Frontend: ${FRONTEND_IMAGE}:${IMAGE_TAG}"

echo "$IMAGE_TAG"
