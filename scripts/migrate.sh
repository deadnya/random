#!/bin/sh
set -eu

CMD="${MIGRATE_COMMAND:-up}"
DB_HOST="${DB_HOST:-db}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:?DB_USER is required}"
DB_PASSWORD="${DB_PASSWORD:?DB_PASSWORD is required}"
DB_NAME="${DB_NAME:?DB_NAME is required}"
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

set +e
OUTPUT=$(migrate -path /migrations -database "$DB_URL" $CMD 2>&1)
STATUS=$?
set -e

echo "$OUTPUT"

if [ "$STATUS" -ne 0 ]; then
  echo "$OUTPUT" | grep -qi "no change" && exit 0
  exit "$STATUS"
fi
