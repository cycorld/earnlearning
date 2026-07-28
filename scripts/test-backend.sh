#!/usr/bin/env bash
set -euo pipefail

readonly GO_TOOLCHAIN="go1.24.13"
readonly GO_TAGS="sqlite_fts5"
readonly ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT/backend"

case "${1:-all}" in
  smoke)
    exec env GOTOOLCHAIN="$GO_TOOLCHAIN" go test -tags "$GO_TAGS" ./tests/integration/ -run TestSmoke -timeout 60s
    ;;
  integration)
    exec env GOTOOLCHAIN="$GO_TOOLCHAIN" go test -tags "$GO_TAGS" ./tests/integration/ -timeout 180s
    ;;
  all)
    exec env GOTOOLCHAIN="$GO_TOOLCHAIN" go test -tags "$GO_TAGS" ./... -timeout 180s
    ;;
  *)
    echo "usage: $0 [smoke|integration|all]" >&2
    exit 2
    ;;
esac