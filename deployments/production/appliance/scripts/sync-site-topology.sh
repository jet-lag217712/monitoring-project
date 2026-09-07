#!/usr/bin/env bash
# Sync sites/manifest.yaml into PostgreSQL: upsert configured sites/topology and
# prune orphan sites rows (and dependents) that are no longer in the manifest.
# Keep delete order aligned with services/snmp-collector/.../remove_site.go.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -f "${SCRIPT_DIR}/debug-agent-log.sh" ]]; then
  # shellcheck source=debug-agent-log.sh
  source "${SCRIPT_DIR}/debug-agent-log.sh"
elif [[ -f "${SCRIPT_DIR}/../../appliance/scripts/debug-agent-log.sh" ]]; then
  # shellcheck source=../../appliance/scripts/debug-agent-log.sh
  source "${SCRIPT_DIR}/../../appliance/scripts/debug-agent-log.sh"
fi

DEPLOY_DIR="${EQUATE_DEPLOY_DIR:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
MANIFEST="${EQUATE_MANIFEST:-${DEPLOY_DIR}/sites/manifest.yaml}"
COMPOSE_ENV="${EQUATE_COMPOSE_ENV:-/run/equate/rendered/compose.env}"

if [[ ! -f "${MANIFEST}" ]]; then
  echo "manifest not found: ${MANIFEST}" >&2
  exit 1
fi

generate_sql() {
  python3 - "${MANIFEST}" <<'PY'
import re
import sys
import uuid

manifest_path = sys.argv[1]


def unquote(value: str) -> str:
    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
        return value[1:-1]
    return value


def parse_inline_list(value: str) -> list[str]:
    value = value.strip()
    if not value or value == "[]":
        return []
    if value.startswith("[") and value.endswith("]"):
        inner = value[1:-1].strip()
        if not inner:
            return []
        return [unquote(part) for part in inner.split(",") if part.strip()]
    return [unquote(value)]


def parse_manifest(path: str) -> list[dict]:
    lines = open(path, encoding="utf-8").read().splitlines()
    in_sites = False
    sites: list[dict] = []
    current: dict | None = None
    list_key: str | None = None

    for line in lines:
        if re.match(r"^sites:\s*$", line):
            in_sites = True
            continue
        if not in_sites:
            continue
        if re.match(r"^\w+:", line):
            break

        dash_match = re.match(r"^(\s*)-\s*(.*)$", line)
        if dash_match:
            rest = dash_match.group(2).strip()
            if list_key and ":" not in rest:
                current[list_key].append(unquote(rest))
                continue
            if current and current.get("site_id"):
                sites.append(current)
            current = {"upstream_site_ids": [], "hub_device_ids": []}
            list_key = None
            if ":" in rest:
                key, value = rest.split(":", 1)
                key = key.strip()
                value = value.strip()
                if key == "site_id":
                    current["site_id"] = unquote(value)
            continue

        if current is None:
            continue

        list_item = re.match(r"^\s+-\s+(.*)$", line)
        if list_item and list_key:
            current[list_key].append(unquote(list_item.group(1)))
            continue

        field = re.match(r"^\s+(\w+):\s*(.*)$", line)
        if not field:
            continue

        key, value = field.group(1), field.group(2)
        if key == "site_id":
            current["site_id"] = unquote(value)
            list_key = None
        elif key in ("upstream_site_ids", "hub_device_ids"):
            if value:
                current[key] = parse_inline_list(value)
                list_key = None
            else:
                current[key] = []
                list_key = key
        else:
            list_key = None

    if current and current.get("site_id"):
        sites.append(current)
    return sites


sites = parse_manifest(manifest_path)
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

# Prune sites that left the manifest (configure shrink / partial delete).
# Empty keep list is refused above so we never wipe the database.
keep = []
seen = set()
for site in sites:
    site_id = site["site_id"]
    if site_id in seen:
        continue
    seen.add(site_id)
    keep.append("'" + site_id.replace("'", "''") + "'")
keep_list = ", ".join(keep)
orphan_sites = f"SELECT id FROM sites WHERE name NOT IN ({keep_list})"
orphan_devices = f"SELECT id FROM devices WHERE site_id IN ({orphan_sites})"
orphan_interfaces = f"SELECT id FROM interfaces WHERE device_id IN ({orphan_devices})"
print(f"DELETE FROM device_health_history WHERE site_id IN ({orphan_sites});")
print(f"DELETE FROM device_health_current WHERE site_id IN ({orphan_sites});")
print(f"DELETE FROM device_temperature_readings WHERE device_id IN ({orphan_devices});")
print(f"DELETE FROM device_temperature_components WHERE device_id IN ({orphan_devices});")
print(f"DELETE FROM device_power_readings WHERE device_id IN ({orphan_devices});")
print(f"DELETE FROM device_power_components WHERE device_id IN ({orphan_devices});")
print(f"DELETE FROM interface_samples WHERE interface_id IN ({orphan_interfaces});")
print(f"DELETE FROM metric_samples WHERE device_id IN ({orphan_devices});")
print(f"DELETE FROM alerts WHERE device_id IN ({orphan_devices});")
print(f"DELETE FROM interfaces WHERE device_id IN ({orphan_devices});")
print(f"DELETE FROM collector_status_current WHERE site_id IN ({orphan_sites});")
print(f"DELETE FROM collector_heartbeat_history WHERE site_id IN ({orphan_sites});")
print(f"DELETE FROM collectors WHERE site_id IN ({orphan_sites});")
print(f"DELETE FROM devices WHERE site_id IN ({orphan_sites});")
print(f"DELETE FROM sites WHERE name NOT IN ({keep_list});")
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

if declare -F debug_agent_log >/dev/null 2>&1; then
  db_after="$(query_db_topology_json "${DEPLOY_DIR}" "${COMPOSE_ENV}")"
  debug_agent_log "H2" "sync-site-topology.sh" "topology sync applied" "{\"manifest\":$(manifest_topology_json "${MANIFEST}"),\"db_after\":${db_after}}"
fi

echo "Site topology synced from ${MANIFEST}"
