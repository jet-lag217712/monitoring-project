#!/usr/bin/env bash
# Set passwords for application DB roles after migrations.
#
# Usage:
#   export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable
#   export OGSD_INGESTION_PASSWORD=ingestion
#   export OGSD_API_PASSWORD=api
#   export OGSD_ADMIN_PASSWORD=admin   # optional; local only
#   ./infrastructure/script/bootstrap-db-roles.sh
#
# Requires: psql (or docker with postgres client).
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

INGESTION_PASSWORD="${OGSD_INGESTION_PASSWORD:-ingestion}"
API_PASSWORD="${OGSD_API_PASSWORD:-api}"
ADMIN_PASSWORD="${OGSD_ADMIN_PASSWORD:-}"

run_psql() {
  if command -v psql >/dev/null 2>&1; then
    psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 "$@"
    return
  fi

  if command -v docker >/dev/null 2>&1; then
    # Parse URL roughly for docker exec against test-env container when psql missing.
    docker run --rm --network host postgres:16-alpine \
      psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 "$@"
    return
  fi

  echo "Neither 'psql' nor 'docker' is available." >&2
  exit 1
}

# Escape single quotes for SQL string literals.
sql_quote() {
  printf "%s" "${1//\'/\'\'}"
}

SQL=$(cat <<EOF
ALTER ROLE ogsd_ingestion WITH PASSWORD '$(sql_quote "${INGESTION_PASSWORD}")';
ALTER ROLE ogsd_api WITH PASSWORD '$(sql_quote "${API_PASSWORD}")';
EOF
)

if [[ -n "${ADMIN_PASSWORD}" ]]; then
  SQL+=$'\n'"ALTER ROLE ogsd_admin WITH PASSWORD '$(sql_quote "${ADMIN_PASSWORD}")';"
fi

run_psql -c "${SQL}"

echo "Role passwords updated for ogsd_ingestion and ogsd_api${ADMIN_PASSWORD:+ and ogsd_admin}."
