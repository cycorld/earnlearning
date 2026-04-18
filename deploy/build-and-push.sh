#!/bin/bash
set -euo pipefail

# ─── EarnLearning 빌드 + GHCR Push ───────────────────────────
# 사용법: ./deploy/build-and-push.sh [IMAGE_TAG]
# x86_64 빌드서버에서 네이티브 빌드 → GHCR Push

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

# ─── Backend 빌드 ────────────────────────────────────────────
build_backend() {
  log "Building backend... tag=${IMAGE_TAG}"
  docker build \
    --build-arg BUILD_NUMBER="$BUILD_NUMBER" \
    --build-arg COMMIT_SHA="$COMMIT_SHA" \
    -t "${BACKEND_IMAGE}:${IMAGE_TAG}" \
    -t "${BACKEND_IMAGE}:latest" \
    -f "$PROJECT_DIR/backend/Dockerfile" \
    "$PROJECT_DIR" >&2
  log "Pushing backend..."
  docker push "${BACKEND_IMAGE}:${IMAGE_TAG}" >&2
  docker push "${BACKEND_IMAGE}:latest" >&2
  log "Backend pushed: ${BACKEND_IMAGE}:${IMAGE_TAG}"
}

# ─── Frontend 빌드 ───────────────────────────────────────────
build_frontend() {
  log "Building frontend... tag=${IMAGE_TAG}"
  docker build \
    --build-arg BUILD_NUMBER="$BUILD_NUMBER" \
    --build-arg COMMIT_SHA="$COMMIT_SHA" \
    -t "${FRONTEND_IMAGE}:${IMAGE_TAG}" \
    -t "${FRONTEND_IMAGE}:latest" \
    -f "$PROJECT_DIR/frontend/Dockerfile" \
    "$PROJECT_DIR" >&2
  log "Pushing frontend..."
  docker push "${FRONTEND_IMAGE}:${IMAGE_TAG}" >&2
  docker push "${FRONTEND_IMAGE}:latest" >&2
  log "Frontend pushed: ${FRONTEND_IMAGE}:${IMAGE_TAG}"
}

# ─── Main ─────────────────────────────────────────────────────
log "=== Build & Push 시작 ==="
log "Tag: ${IMAGE_TAG} | Build #${BUILD_NUMBER} | ${COMMIT_SHA:0:7}"

build_backend
build_frontend

log "=== Build & Push 완료! ($(elapsed)) ==="
log "Backend:  ${BACKEND_IMAGE}:${IMAGE_TAG}"
log "Frontend: ${FRONTEND_IMAGE}:${IMAGE_TAG}"

echo "$IMAGE_TAG"
