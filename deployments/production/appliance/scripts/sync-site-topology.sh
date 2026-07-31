#!/usr/bin/env bash
# Sync sites/manifest.yaml upstream_site_ids and hub_device_ids into PostgreSQL.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR="${EQUATE_DEPLOY_DIR:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
MANIFEST="${EQUATE_MANIFEST:-${DEPLOY_DIR}/sites/manifest.yaml}"
COMPOSE_ENV="${EQUATE_COMPOSE_ENV:-/run/equate/rendered/compose.env}"

if [[ ! -f "${MANIFEST}" ]]; then
  echo "manifest not found: ${MANIFEST}" >&2
  exit 1
fi

generate_sql() {
  python3 - "${MANIFEST}" <<'PY'
import sys
import uuid
import yaml

manifest_path = sys.argv[1]
with open(manifest_path, "r", encoding="utf-8") as fh:
    doc = yaml.safe_load(fh)

sites = doc.get("sites") or []
if not sites:
    raise SystemExit("manifest has no sites")

namespace = uuid.uuid5(uuid.UUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8"), "equate-ogsd")

def site_uuid(site_id: str) -> str:
    return str(uuid.uuid5(namespace, f"site:{site_id}"))

def sql_array(values):
    if not values:
        return "ARRAY[]::text[]"
    escaped = []
    for value in values:
        value = str(value).replace("'", "''")
        escaped.append(f"'{value}'")
    return "ARRAY[" + ",".join(escaped) + "]::text[]"

for site in sites:
    site_id = site["site_id"]
    upstreams = site.get("upstream_site_ids") or []
    hubs = site.get("hub_device_ids") or []
    sid = site_uuid(site_id)
    print(
        "INSERT INTO sites (id, name, upstream_site_ids, hub_device_ids) "
        f"VALUES ('{sid}', '{site_id.replace(chr(39), chr(39)+chr(39))}', {sql_array(upstreams)}, {sql_array(hubs)}) "
        "ON CONFLICT (id) DO UPDATE SET "
        "name = EXCLUDED.name, "
        "upstream_site_ids = EXCLUDED.upstream_site_ids, "
        "hub_device_ids = EXCLUDED.hub_device_ids;"
    )
PY
}

run_psql() {
  if [[ -f "${COMPOSE_ENV}" && -f "${DEPLOY_DIR}/docker-compose.yml" ]]; then
    # Appliance postgres is not published on the host; use the compose network.
    set -a
    # shellcheck disable=SC1090
    source "${COMPOSE_ENV}"
    set +a
    (
      cd "${DEPLOY_DIR}"
      docker compose \
        --env-file "${COMPOSE_ENV}" \
        -f docker-compose.yml \
        -f docker-compose.sites.generated.yml \
        exec -T postgres \
        psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=1
    )
    return
  fi
  if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "sync-site-topology: DATABASE_URL or compose env (${COMPOSE_ENV}) is required" >&2
    exit 1
  fi
  psql "${DATABASE_URL}" -v ON_ERROR_STOP=1
}

generate_sql | run_psql

echo "Site topology synced from ${MANIFEST}"
