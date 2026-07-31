#!/usr/bin/env bash
# Sync sites/manifest.yaml upstream_site_ids and hub_device_ids into PostgreSQL.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
MANIFEST="${DEPLOY_DIR}/sites/manifest.yaml"

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 1
fi
if [[ ! -f "${MANIFEST}" ]]; then
  echo "manifest not found: ${MANIFEST}" >&2
  exit 1
fi

python3 - "${MANIFEST}" <<'PY' | psql "${DATABASE_URL}" -v ON_ERROR_STOP=1
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

echo "Site topology synced from ${MANIFEST}"
