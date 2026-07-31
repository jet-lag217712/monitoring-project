#!/usr/bin/env bash
# Sync PostgreSQL application-role passwords from rendered appliance secrets.
# Migrations create ogsd_api / ogsd_ingestion without passwords; services read
# credentials from compose.env. This step is idempotent and must run after
# migrate and before backend-api / ingestion start.
set -euo pipefail

RUN_DIR="${EQUATE_RUN_DIR:-/run/equate}"
COMPOSE_ENV="${EQUATE_COMPOSE_ENV:-${RUN_DIR}/rendered/compose.env}"
POSTGRES_ENV="${EQUATE_POSTGRES_ENV:-${RUN_DIR}/rendered/postgres.env}"
RELEASE_DIR="${EQUATE_RELEASE_DIR:-}"

resolve_release_dir() {
  if [[ -n "${RELEASE_DIR}" ]]; then
    return 0
  fi
  if [[ -f /etc/equate/deploy-dir ]]; then
    RELEASE_DIR="$(tr -d '[:space:]' < /etc/equate/deploy-dir)"
    return 0
  fi
  if [[ -L /opt/equate/current ]]; then
    RELEASE_DIR="$(readlink -f /opt/equate/current)"
    return 0
  fi
  echo "sync-db-role-passwords: cannot resolve release directory" >&2
  return 1
}

require_nonempty() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "sync-db-role-passwords: required variable empty: ${name}" >&2
    exit 1
  fi
}

sql_escape() {
  printf "%s" "${1//\'/\'\'}"
}

compose_files=(-f docker-compose.yml)
if [[ -f "${RELEASE_DIR}/docker-compose.sites.generated.yml" ]]; then
  compose_files+=(-f docker-compose.sites.generated.yml)
fi

compose() {
  (
    cd "${RELEASE_DIR}"
    docker compose \
      --env-file "${COMPOSE_ENV}" \
      "${compose_files[@]}" \
      "$@"
  )
}

main() {
  resolve_release_dir

  if [[ ! -f "${COMPOSE_ENV}" ]]; then
    echo "sync-db-role-passwords: missing ${COMPOSE_ENV}" >&2
    exit 1
  fi
  if [[ ! -f "${POSTGRES_ENV}" ]]; then
    echo "sync-db-role-passwords: missing ${POSTGRES_ENV}" >&2
    exit 1
  fi
  if [[ ! -f "${RELEASE_DIR}/docker-compose.yml" ]]; then
    echo "sync-db-role-passwords: missing ${RELEASE_DIR}/docker-compose.yml" >&2
    exit 1
  fi

  set -a
  # shellcheck disable=SC1090
  source "${POSTGRES_ENV}"
  # shellcheck disable=SC1090
  source "${COMPOSE_ENV}"
  set +a

  require_nonempty POSTGRES_USER
  require_nonempty POSTGRES_DB
  require_nonempty OGSD_INGESTION_PASSWORD
  require_nonempty OGSD_API_PASSWORD

  echo "sync-db-role-passwords: ensuring postgres is running..."
  compose up -d postgres >/dev/null

  echo "sync-db-role-passwords: waiting for postgres..."
  for _ in $(seq 1 60); do
    if compose exec -T postgres pg_isready -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
  if ! compose exec -T postgres pg_isready -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" >/dev/null 2>&1; then
    echo "sync-db-role-passwords: postgres is not ready" >&2
    exit 1
  fi

  echo "sync-db-role-passwords: applying ogsd_ingestion and ogsd_api passwords..."
  compose exec -T postgres psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=1 \
    -c "ALTER ROLE ogsd_ingestion WITH PASSWORD '$(sql_escape "${OGSD_INGESTION_PASSWORD}")';" \
    -c "ALTER ROLE ogsd_api WITH PASSWORD '$(sql_escape "${OGSD_API_PASSWORD}")';"

  echo "sync-db-role-passwords: done"
}

main "$@"
