#!/usr/bin/env bash
# Agent debug logging for session 47e701 (upgrade / topology handoff).
# Writes NDJSON to /var/lib/equate/debug-47e701.ndjson on the appliance.
set -euo pipefail

DEBUG_LOG="${EQUATE_DEBUG_LOG:-/var/lib/equate/debug-47e701.ndjson}"

debug_agent_log() {
  local hypothesis_id="$1"
  local location="$2"
  local message="$3"
  local data_json="${4:-{}}"
  local ts
  ts="$(date +%s%3N 2>/dev/null || python3 -c 'import time; print(int(time.time()*1000))')"
  install -d -m 0750 "$(dirname "${DEBUG_LOG}")" 2>/dev/null || true
  # #region agent log
  printf '%s\n' "{\"sessionId\":\"47e701\",\"timestamp\":${ts},\"hypothesisId\":\"${hypothesis_id}\",\"location\":\"${location}\",\"message\":\"${message}\",\"data\":${data_json}}" >>"${DEBUG_LOG}" 2>/dev/null || true
  # #endregion
}

query_db_topology_json() {
  local deploy_dir="$1"
  local compose_env="$2"
  if [[ ! -f "${compose_env}" || ! -f "${deploy_dir}/docker-compose.yml" ]]; then
    echo "null"
    return 0
  fi
  set -a
  # shellcheck disable=SC1090
  source "${compose_env}"
  set +a
  (
    cd "${deploy_dir}"
    docker compose \
      --env-file "${compose_env}" \
      -f docker-compose.yml \
      -f docker-compose.sites.generated.yml \
      exec -T postgres \
      psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -t -A -F '|' -c \
      "SELECT name, COALESCE(upstream_site_ids::text,'{}'), COALESCE(hub_device_ids::text,'{}') FROM sites ORDER BY name;" 2>/dev/null \
      | python3 -c 'import json,sys
rows=[]
for line in sys.stdin.read().splitlines():
    line=line.strip()
    if not line: continue
    parts=line.split("|",2)
    if len(parts)<3: continue
    rows.append({"name":parts[0],"upstream":parts[1],"hubs":parts[2]})
print(json.dumps(rows))'
  ) || echo "null"
}

manifest_topology_json() {
  local manifest="$1"
  if [[ ! -f "${manifest}" ]]; then
    echo "null"
    return 0
  fi
  python3 - "${manifest}" <<'PY'
import json, re, sys
path = sys.argv[1]
lines = open(path, encoding="utf-8").read().splitlines()
in_sites = False
sites = []
current = None
list_key = None

def unquote(v):
    v = v.strip()
    if len(v) >= 2 and v[0] == v[-1] and v[0] in "\"'":
        return v[1:-1]
    return v

for line in lines:
    if re.match(r"^sites:\s*$", line):
        in_sites = True
        continue
    if not in_sites:
        continue
    if re.match(r"^\w+:", line):
        break
    dash = re.match(r"^(\s*)-\s*(.*)$", line)
    if dash:
        rest = dash.group(2).strip()
        if list_key and ":" not in rest:
            current[list_key].append(unquote(rest))
            continue
        if current and current.get("site_id"):
            sites.append(current)
        current = {"upstream_site_ids": [], "hub_device_ids": []}
        list_key = None
        if ":" in rest:
            k, v = rest.split(":", 1)
            if k.strip() == "site_id":
                current["site_id"] = unquote(v)
        continue
    if current is None:
        continue
    item = re.match(r"^\s+-\s+(.*)$", line)
    if item and list_key:
        current[list_key].append(unquote(item.group(1)))
        continue
    field = re.match(r"^\s+(\w+):\s*(.*)$", line)
    if not field:
        continue
    key, val = field.group(1), field.group(2)
    if key == "site_id":
        current["site_id"] = unquote(val)
        list_key = None
    elif key in ("upstream_site_ids", "hub_device_ids"):
        if val.strip() in ("", "[]"):
            current[key] = []
            list_key = key
        elif val.strip().startswith("["):
            inner = val.strip()[1:-1].strip()
            current[key] = [unquote(x) for x in inner.split(",") if x.strip()] if inner else []
            list_key = None
        else:
            current[key] = [unquote(val)]
            list_key = None
    else:
        list_key = None
if current and current.get("site_id"):
    sites.append(current)
print(json.dumps(sites))
PY
}
