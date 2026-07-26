#!/bin/bash
set -uo pipefail

# ─── deploy-remote.sh 회귀 테스트 ─────────────────────────────
# 실제 git/ssh/docker/network 를 건드리지 않는 스크립트 레벨 테스트.
# 각 시나리오마다 mktemp 샌드박스를 만들고 mock git/ssh/build-and-push.sh 로
# deploy-remote.sh 의 분기 동작(로컬 빌드, ssh earnlearning, 태그 전달 등)을 검증한다.
#
# 회귀 방지 핵심:
#   - deploy 시 'ssh cycorld' (빌드서버 재접속) 이 절대 없어야 한다.
#   - promote 시 EC2 git HEAD('rev-parse --short HEAD') 를 읽지 않아야 한다.

TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR_REAL="$(dirname "$TESTS_DIR")"
REPO_ROOT="$(dirname "$DEPLOY_DIR_REAL")"
REAL_SCRIPT="$REPO_ROOT/deploy-remote.sh"

PASS=0
FAIL=0

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass() { PASS=$((PASS + 1)); echo -e "  ${GREEN}PASS${NC} $*"; }
fail() { FAIL=$((FAIL + 1)); echo -e "  ${RED}FAIL${NC} $*"; }

# assert_eq <expected> <actual> <message>
assert_eq() {
  if [ "$1" = "$2" ]; then
    pass "$3"
  else
    fail "$3 (expected='$1' actual='$2')"
  fi
}

# assert_contains <haystack> <needle> <message>
assert_contains() {
  if printf '%s' "$1" | grep -qF -- "$2"; then
    pass "$3"
  else
    fail "$3 (not found: '$2')"
  fi
}

# assert_not_contains <haystack> <needle> <message>
assert_not_contains() {
  if printf '%s' "$1" | grep -qF -- "$2"; then
    fail "$3 (unexpectedly found: '$2')"
  else
    pass "$3"
  fi
}


# ─── 샌드박스 구성 ────────────────────────────────────────────
# 인자로 넘긴 샌드박스 디렉토리에 스크립트 + mock 을 설치한다.
build_sandbox() {
  local sandbox="$1"
  mkdir -p "$sandbox/deploy" "$sandbox/mock-bin"

  cp "$REAL_SCRIPT" "$sandbox/deploy-remote.sh"
  chmod +x "$sandbox/deploy-remote.sh"

  # mock build-and-push.sh: 호출 로그 남기고 stdout 으로 태그만 출력
  cat > "$sandbox/deploy/build-and-push.sh" <<'EOF'
#!/bin/bash
echo "build-and-push $*" >> "$MOCK_LOG"
echo "abc1234"
EOF
  chmod +x "$sandbox/deploy/build-and-push.sh"

  # mock git
  cat > "$sandbox/mock-bin/git" <<'EOF'
#!/bin/bash
echo "git $*" >> "$MOCK_LOG"
# -C <dir> 접두어 처리
if [ "${1:-}" = "-C" ]; then
  shift 2
fi
case "$*" in
  "rev-parse --abbrev-ref HEAD")
    echo "${MOCK_BRANCH:-main}"
    ;;
  "status --porcelain"|"status --short")
    printf '%s' "${MOCK_DIRTY:-}"
    ;;
  fetch*)
    exit 0
    ;;
  "rev-parse origin/main")
    echo "${MOCK_REMOTE_SHA:-aaaa111}"
    ;;
  "rev-parse HEAD")
    echo "${MOCK_LOCAL_SHA:-aaaa111}"
    ;;
  *)
    exit 0
    ;;
esac
EOF
  chmod +x "$sandbox/mock-bin/git"

  # mock ssh
  cat > "$sandbox/mock-bin/ssh" <<'EOF'
#!/bin/bash
echo "ssh $*" >> "$MOCK_LOG"
# 원격 명령 문자열은 마지막 인자
cmd="${!#}"
if printf '%s' "$cmd" | grep -q "docker inspect"; then
  echo "${MOCK_STAGE_IMAGE:-ghcr.io/cycorld/earnlearning-backend:abc1234}"
fi
exit 0
EOF
  chmod +x "$sandbox/mock-bin/ssh"
}

# run_scenario 는 전역 RC / OUT / LOGC 를 채운다.
#   추가 env 는 호출 전에 export 하는 대신, 서브셸에서 인라인으로 넘긴다.
# 사용: RC OUT LOGC 를 읽으려면 run_case 를 통해 호출.

# ─── 시나리오 실행 헬퍼 ───────────────────────────────────────
# run_case <command> <env assignments...>  (env 는 "KEY=VAL" 형태)
# 결과: 전역 RC(exit code), OUT(stdout+stderr), LOGC(MOCK_LOG 내용)
run_case() {
  local cmd="$1"; shift
  local sandbox
  sandbox="$(mktemp -d)"
  build_sandbox "$sandbox"
  local mock_log="$sandbox/mock.log"
  : > "$mock_log"

  local env_kv=()
  local kv
  for kv in "$@"; do
    env_kv+=("$kv")
  done

  # 스크립트 최상단은 errexit(-e) 미설정이라 실패해도 계속 진행된다.
  OUT="$(env "${env_kv[@]}" \
    PATH="$sandbox/mock-bin:$PATH" \
    MOCK_LOG="$mock_log" \
    bash "$sandbox/deploy-remote.sh" "$cmd" 2>&1)"
  RC=$?

  LOGC="$(cat "$mock_log")"
  rm -rf "$sandbox"
}

echo "=== deploy-remote.sh 회귀 테스트 ==="

# ── 1. deploy happy path ──────────────────────────────────────
echo "[1] deploy happy path (clean/main/synced)"
run_case deploy
assert_eq 0 "$RC" "exit 0"
assert_contains "$LOGC" "build-and-push" "로컬 빌드 실행됨"
assert_contains "$LOGC" "fetch origin main" "origin/main fetch 수행됨"
# stage ssh 라인이 earnlearning 호스트 + 빌드 태그 + deploy.sh stage 를 포함
STAGE_LINE="$(printf '%s\n' "$LOGC" | grep '^ssh ' | grep 'deploy.sh stage' || true)"
assert_contains "$STAGE_LINE" "earnlearning" "stage ssh 호스트=earnlearning"
assert_contains "$STAGE_LINE" "IMAGE_TAG=abc1234" "stage 가 빌드 태그(abc1234) 수신"
assert_contains "$STAGE_LINE" "./deploy.sh stage" "stage 배포 명령 호출"
# 회귀: 빌드서버로 되돌아가는 ssh cycorld 없음
assert_not_contains "$LOGC" "ssh cycorld" "회귀: 'ssh cycorld' 없음"

# ── 2. deploy dirty worktree ──────────────────────────────────
echo "[2] deploy dirty worktree"
run_case deploy "MOCK_DIRTY= M backend/main.go"
assert_eq 1 "$RC" "exit nonzero"
assert_not_contains "$LOGC" "build-and-push" "빌드 전에 실패 (빌드 안 함)"
assert_not_contains "$LOGC" "deploy.sh stage" "stage ssh 호출 없음"

# ── 3. deploy on feature branch ───────────────────────────────
echo "[3] deploy on non-main branch"
run_case deploy "MOCK_BRANCH=feat/something"
assert_eq 1 "$RC" "exit nonzero"
assert_not_contains "$LOGC" "build-and-push" "빌드 안 함"
assert_not_contains "$LOGC" "deploy.sh stage" "stage ssh 호출 없음"

# ── 4. deploy out of sync ─────────────────────────────────────
echo "[4] deploy local != origin/main"
run_case deploy "MOCK_LOCAL_SHA=aaaa111" "MOCK_REMOTE_SHA=bbbb222"
assert_eq 1 "$RC" "exit nonzero"
assert_not_contains "$LOGC" "build-and-push" "빌드 안 함"
assert_not_contains "$LOGC" "deploy.sh stage" "stage ssh 호출 없음"

# ── 5. promote (태그 자동 감지) ───────────────────────────────
echo "[5] promote (no IMAGE_TAG env)"
run_case promote
assert_eq 0 "$RC" "exit 0"
INSPECT_LINE="$(printf '%s\n' "$LOGC" | grep '^ssh ' | grep 'docker inspect' || true)"
assert_contains "$INSPECT_LINE" "docker inspect" "docker inspect 로 stage 태그 조회"
assert_contains "$INSPECT_LINE" "earnlearning-stage-backend-1" "stage backend 컨테이너 대상"
PROD_LINE="$(printf '%s\n' "$LOGC" | grep '^ssh ' | grep 'deploy.sh prod' || true)"
assert_contains "$PROD_LINE" "IMAGE_TAG=abc1234" "prod 가 stage 태그(abc1234) 수신"
assert_contains "$PROD_LINE" "./deploy.sh prod" "prod 배포 명령 호출"
assert_not_contains "$LOGC" "build-and-push" "promote 는 빌드 안 함"
# 회귀: EC2 git HEAD 를 읽는 ssh 라인 없음
SSH_LINES="$(printf '%s\n' "$LOGC" | grep '^ssh ' || true)"
assert_not_contains "$SSH_LINES" "rev-parse --short HEAD" "회귀: EC2 git HEAD 조회 없음"

# ── 6. promote 가 'latest' 태그 거부 ──────────────────────────
echo "[6] promote refuses ambiguous 'latest'"
run_case promote "MOCK_STAGE_IMAGE=ghcr.io/cycorld/earnlearning-backend:latest"
assert_eq 1 "$RC" "exit nonzero"
assert_not_contains "$LOGC" "deploy.sh prod" "prod 배포 호출 없음 (latest 거부)"

# ── 7. promote 명시적 IMAGE_TAG env ───────────────────────────
echo "[7] promote with explicit IMAGE_TAG env"
run_case promote "IMAGE_TAG=ffff999"
assert_eq 0 "$RC" "exit 0"
PROD_LINE="$(printf '%s\n' "$LOGC" | grep '^ssh ' | grep 'deploy.sh prod' || true)"
assert_contains "$PROD_LINE" "IMAGE_TAG=ffff999" "prod 가 명시 태그(ffff999) 수신"

# ── 8. rollback / status ──────────────────────────────────────
echo "[8] rollback"
run_case rollback
assert_eq 0 "$RC" "exit 0"
ROLLBACK_LINE="$(printf '%s\n' "$LOGC" | grep '^ssh ' | grep 'deploy.sh rollback' || true)"
assert_contains "$ROLLBACK_LINE" "earnlearning" "rollback ssh 호스트=earnlearning"
assert_contains "$ROLLBACK_LINE" "./deploy.sh rollback" "rollback 명령 호출"

echo "[9] status"
run_case status
assert_eq 0 "$RC" "exit 0"
STATUS_LINE="$(printf '%s\n' "$LOGC" | grep '^ssh ' | grep 'deploy.sh status' || true)"
assert_contains "$STATUS_LINE" "earnlearning" "status ssh 호스트=earnlearning"
assert_contains "$STATUS_LINE" "./deploy.sh status" "status 명령 호출"

# ─── 요약 ─────────────────────────────────────────────────────
echo ""
echo "=== 결과: ${PASS} passed, ${FAIL} failed ==="
if [ "$FAIL" -ne 0 ]; then
  exit 1
fi
