#!/bin/bash
# EarnLearning Log Viewer
# Usage: ./logs.sh [prod|stage] [options]
#   ./logs.sh              # prod 최근 로그
#   ./logs.sh prod         # prod 최근 로그
#   ./logs.sh stage        # stage 최근 로그
#   ./logs.sh prod -f      # prod 실시간 로그
#   ./logs.sh prod errors  # prod 에러만 (4xx, 5xx)
#   ./logs.sh prod all     # prod 전체 컨테이너 로그

set -euo pipefail

ENV="${1:-prod}"
ACTION="${2:-}"
LINES="${3:-50}"

ACTIVE_SLOT_CONF="/etc/nginx/earnlearning-active-slot.conf"

get_prod_project() {
  local slot="blue"
  if [ -f "$ACTIVE_SLOT_CONF" ] && grep -q "8181" "$ACTIVE_SLOT_CONF"; then
    slot="green"
  fi
  echo "earnlearning-${slot}"
}

case "$ENV" in
  prod)  PROJECT="$(get_prod_project)" ;;
  stage) PROJECT="earnlearning-stage" ;;
  *)     echo "Usage: $0 [prod|stage] [options]"; exit 1 ;;
esac

BACKEND="${PROJECT}-backend-1"
NGINX="${PROJECT}-nginx-1"
FRONTEND="${PROJECT}-frontend-1"

case "$ACTION" in
  -f|follow)
    echo "=== ${ENV} 실시간 로그 (Ctrl+C to stop) ==="
    sudo docker logs -f --tail 20 "$BACKEND" 2>&1
    ;;
  errors|err)
    echo "=== ${ENV} 에러 로그 (4xx/5xx) ==="
    sudo docker logs "$BACKEND" 2>&1 | grep -E " [45][0-9]{2} " | tail -"$LINES"
    ;;
  403)
    echo "=== ${ENV} 403 Forbidden ==="
    sudo docker logs "$BACKEND" 2>&1 | grep " 403 " | tail -"$LINES"
    ;;
  500)
    echo "=== ${ENV} 500 Server Error ==="
    sudo docker logs "$BACKEND" 2>&1 | grep -E " 5[0-9]{2} " | tail -"$LINES"
    ;;
  nginx)
    echo "=== ${ENV} Nginx 로그 ==="
    sudo docker logs "$NGINX" 2>&1 | tail -"$LINES"
    ;;
  all)
    echo "=== ${ENV} Backend ==="
    sudo docker logs "$BACKEND" 2>&1 | tail -20
    echo ""
    echo "=== ${ENV} Nginx ==="
    sudo docker logs "$NGINX" 2>&1 | tail -20
    echo ""
    echo "=== ${ENV} Frontend ==="
    sudo docker logs "$FRONTEND" 2>&1 | tail -10
    ;;
  *)
    echo "=== ${ENV} Backend 최근 로그 ==="
    sudo docker logs "$BACKEND" 2>&1 | tail -"$LINES"
    ;;
esac
