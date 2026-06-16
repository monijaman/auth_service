#!/usr/bin/env bash
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://auth_user:auth_pass@localhost:5432/auth_db?sslmode=disable}"
MIGRATIONS_DIR="$(dirname "$0")/../migrations"
DIRECTION="${1:-up}"

echo "Running migrations ($DIRECTION) against $DATABASE_URL"

migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" "$DIRECTION"

echo "Done."
