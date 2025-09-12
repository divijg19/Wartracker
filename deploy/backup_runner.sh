#!/usr/bin/env bash
set -euo pipefail
DB_PATH="${DB_PATH:-/data/guild_data.db}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"
mkdir -p "$BACKUP_DIR"

while true; do
  TS=$(date -u +"%Y%m%dT%H%M%SZ")
  DEST="$BACKUP_DIR/guild_data.$TS.db"
  if command -v sqlite3 >/dev/null 2>&1; then
    sqlite3 "$DB_PATH" ".backup '$DEST'"
    echo "Backup created: $DEST"
  else
    cp "$DB_PATH" "$DEST"
    echo "Backup copy created: $DEST"
  fi
  # Sleep 24h
  sleep 86400
done
