#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCHEMA_FILE="$SCRIPT_DIR/../schema.sql"

if [ -z "${DATABASE_URL:-}" ]; then
    echo "ERROR: DATABASE_URL is not set"
    echo "Export it or add it to .env, then run this script again."
    exit 1
fi

echo "Pushing schema to database..."
psql "$DATABASE_URL" < "$SCHEMA_FILE"
echo "Done."
