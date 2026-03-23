#!/bin/bash
# EarnLearning Production DB Backup
# Usage: ./backup.sh [backup|restore|list]
set -euo pipefail

BACKUP_DIR="/home/ubuntu/backups/earnlearning"
DB_PATH="/data/db/earnlearning.db"
DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
ACTIVE_SLOT_CONF="/etc/nginx/earnlearning-active-slot.conf"

# 현재 active slot의 backend 컨테이너 찾기
get_active_container() {
  local slot="blue"
  if [ -f "$ACTIVE_SLOT_CONF" ] && grep -q "8181" "$ACTIVE_SLOT_CONF"; then
    slot="green"
  fi
  echo "earnlearning-${slot}-backend-1"
}

CONTAINER="$(get_active_container)"

# Cloudflare R2 (S3-compatible)
R2_ENDPOINT="${R2_ENDPOINT:-https://<ACCOUNT_ID>.r2.cloudflarestorage.com}"
R2_BUCKET="${R2_BUCKET:-earnlearning-backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

backup() {
    echo "[$(date)] Starting backup..."

    # SQLite safe backup: copy DB file via docker cp (WAL mode checkpoint first)
    sudo docker cp "$CONTAINER:$DB_PATH" "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db"
    # Also copy WAL/SHM if they exist
    sudo docker cp "$CONTAINER:${DB_PATH}-wal" "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db-wal" 2>/dev/null || true
    sudo docker cp "$CONTAINER:${DB_PATH}-shm" "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db-shm" 2>/dev/null || true

    # Remove WAL/SHM after copy (they get merged on next open)
    rm -f "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db-wal" "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db-shm"

    # Compress
    gzip "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db"
    echo "[$(date)] Local backup: $BACKUP_DIR/earnlearning_${TIMESTAMP}.db.gz"

    # Upload to Cloudflare R2
    if aws --endpoint-url "$R2_ENDPOINT" --profile r2 s3 cp \
        "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db.gz" \
        "s3://${R2_BUCKET}/earnlearning_${TIMESTAMP}.db.gz" 2>/dev/null; then
        echo "[$(date)] R2 backup uploaded"
    else
        echo "[$(date)] R2 upload skipped (not configured)"
    fi

    # Keep only last 7 local backups
    ls -t "$BACKUP_DIR"/earnlearning_*.db.gz 2>/dev/null | tail -n +8 | xargs -r rm
    echo "[$(date)] Backup complete"
}

restore() {
    local file="${1:-}"
    if [ -z "$file" ]; then
        echo "Usage: $0 restore <backup_file.db.gz>"
        echo "Available backups:"
        ls -lh "$BACKUP_DIR"/earnlearning_*.db.gz 2>/dev/null || echo "  (none)"
        exit 1
    fi

    echo "⚠️  This will REPLACE the production database!"
    read -p "Type 'yes' to confirm: " confirm
    if [ "$confirm" != "yes" ]; then
        echo "Cancelled."
        exit 1
    fi

    # 현재 active slot 확인
    local slot="blue"
    if [ -f "$ACTIVE_SLOT_CONF" ] && grep -q "8181" "$ACTIVE_SLOT_CONF"; then
      slot="green"
    fi
    local compose_file="${DEPLOY_DIR}/docker-compose.${slot}.yml"
    local project="earnlearning-${slot}"

    # Stop backend
    sudo docker compose -f "$compose_file" -p "$project" stop backend

    # Restore
    gunzip -k "$file"
    local dbfile="${file%.gz}"
    sudo docker cp "$dbfile" "$CONTAINER:/data/db/earnlearning.db"
    rm "$dbfile"

    # Restart
    sudo docker compose -f "$compose_file" -p "$project" start backend
    echo "[$(date)] Restore complete"
}

list() {
    echo "=== Local backups ==="
    ls -lh "$BACKUP_DIR"/earnlearning_*.db.gz 2>/dev/null || echo "  (none)"
    echo ""
    echo "=== R2 backups ==="
    aws --endpoint-url "$R2_ENDPOINT" --profile r2 s3 ls "s3://${R2_BUCKET}/" 2>/dev/null || echo "  (R2 not configured)"
}

case "${1:-backup}" in
    backup)  backup ;;
    restore) restore "${2:-}" ;;
    list)    list ;;
    *)       echo "Usage: $0 [backup|restore|list]" ;;
esac
