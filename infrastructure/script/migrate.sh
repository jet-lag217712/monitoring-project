#!/usr/bin/env bash
# Apply or inspect database migrations with golang-migrate.
#
# Usage (from repo root):
#   export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable
#   ./infrastructure/script/migrate.sh up
#   ./infrastructure/script/migrate.sh version
#   ./infrastructure/script/migrate.sh down 1
#
# Requires: migrate CLI (brew install golang-migrate) or Docker.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
MIGRATIONS="${ROOT}/database/migrations"
ACTION="${1:-up}"
shift || true

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi

run_migrate() {
  if command -v migrate >/dev/null 2>&1; then
    migrate -path "${MIGRATIONS}" -database "${DATABASE_URL}" "$@"
    return
  fi

  if command -v docker >/dev/null 2>&1; then
    # Host networking so 127.0.0.1 in DATABASE_URL reaches local Postgres.
    docker run --rm --network host \
      -v "${MIGRATIONS}:/migrations:ro" \
      migrate/migrate:v4.18.1 \
      -path /migrations -database "${DATABASE_URL}" "$@"
    return
  fi

  echo "Neither 'migrate' nor 'docker' is available. Install golang-migrate or Docker." >&2
  exit 1
}

case "${ACTION}" in
  up)
    run_migrate up "$@"
    ;;
  down)
    run_migrate down "$@"
    ;;
  version)
    run_migrate version
    ;;
  force)
    if [[ $# -lt 1 ]]; then
      echo "usage: $0 force <version>" >&2
      exit 1
    fi
    run_migrate force "$@"
    ;;
  goto)
    if [[ $# -lt 1 ]]; then
      echo "usage: $0 goto <version>" >&2
      exit 1
    fi
    run_migrate goto "$@"
    ;;
  *)
    echo "usage: $0 {up|down|version|force|goto} [args...]" >&2
    exit 1
    ;;
esac
