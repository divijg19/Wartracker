#!/usr/bin/env bash
# Simple SQLite backup script — uses sqlite3 online backup if available.
set -euo pipefail
DB_PATH="${1:-./data/guild_data.db}"
DEST_DIR="${2:-./backups}"
mkdir -p "$DEST_DIR"
TS=$(date -u +"%Y%m%dT%H%M%SZ")
DEST="$DEST_DIR/guild_data.$TS.db"

if command -v sqlite3 >/dev/null 2>&1; then
  sqlite3 "$DB_PATH" ".backup '$DEST'"
  echo "Backup created: $DEST"
else
  cp "$DB_PATH" "$DEST"
  echo "sqlite3 not found; raw copy created: $DEST"
fi
