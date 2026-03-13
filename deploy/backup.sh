#!/bin/bash
# EarnLearning Production DB Backup
# Usage: ./backup.sh [backup|restore|list]
set -euo pipefail

BACKUP_DIR="/home/ubuntu/backups/earnlearning"
S3_BUCKET="s3://earnlearning-backups"
CONTAINER="earnlearning-prod-backend-1"
DB_PATH="/data/db/earnlearning.db"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

backup() {
    echo "[$(date)] Starting backup..."

    # SQLite safe backup using .backup command (consistent snapshot)
    sudo docker exec "$CONTAINER" sqlite3 "$DB_PATH" ".backup /data/db/backup.db"
    sudo docker cp "$CONTAINER:/data/db/backup.db" "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db"
    sudo docker exec "$CONTAINER" rm /data/db/backup.db

    # Compress
    gzip "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db"
    echo "[$(date)] Local backup: $BACKUP_DIR/earnlearning_${TIMESTAMP}.db.gz"

    # Upload to S3
    if aws s3 cp "$BACKUP_DIR/earnlearning_${TIMESTAMP}.db.gz" "$S3_BUCKET/earnlearning_${TIMESTAMP}.db.gz" 2>/dev/null; then
        echo "[$(date)] S3 backup uploaded"
    else
        echo "[$(date)] S3 upload skipped (bucket not configured)"
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

    # Stop backend
    sudo docker compose -f /home/ubuntu/lms/deploy/docker-compose.prod.yml -p earnlearning-prod stop backend

    # Restore
    gunzip -k "$file"
    local dbfile="${file%.gz}"
    sudo docker cp "$dbfile" "$CONTAINER:/data/db/earnlearning.db"
    rm "$dbfile"

    # Restart
    sudo docker compose -f /home/ubuntu/lms/deploy/docker-compose.prod.yml -p earnlearning-prod start backend
    echo "[$(date)] Restore complete"
}

list() {
    echo "=== Local backups ==="
    ls -lh "$BACKUP_DIR"/earnlearning_*.db.gz 2>/dev/null || echo "  (none)"
    echo ""
    echo "=== S3 backups ==="
    aws s3 ls "$S3_BUCKET/" 2>/dev/null || echo "  (S3 bucket not configured)"
}

case "${1:-backup}" in
    backup)  backup ;;
    restore) restore "${2:-}" ;;
    list)    list ;;
    *)       echo "Usage: $0 [backup|restore|list]" ;;
esac
