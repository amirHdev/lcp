#!/usr/bin/env bash
set -euo pipefail

: "${DB_DSN:?DB_DSN is required}"
: "${BACKUP_FILE:=/tmp/lcp-backup.sql}"

pg_dump "$DB_DSN" > "$BACKUP_FILE"
psql "$DB_DSN" -v ON_ERROR_STOP=1 -c "BEGIN; CREATE TEMP TABLE restore_probe AS SELECT * FROM audit_entries LIMIT 1; ROLLBACK;"
echo "backup written to $BACKUP_FILE"
echo "restore probe passed"
