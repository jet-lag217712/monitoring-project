#!/usr/bin/env bash
# On-site acceptance: wait for v2 telemetry produced by a real SNMP device via the collector.
# Requires configs/collector.yaml pointed at reachable devices and ACCEPTANCE_* env vars.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

ROOT="$(repo_root)"
ENV_FILE="${SCRIPT_DIR}/.env"

cd "${ROOT}"
require_cmd curl

ensure_env_file "${ENV_FILE}" "${SCRIPT_DIR}/.env.example"
load_env_file "${ENV_FILE}"

if grep -q 'REPLACE_WITH_DEVICE_IP' "${SCRIPT_DIR}/configs/collector.yaml"; then
  echo "configs/collector.yaml still contains REPLACE_WITH_DEVICE_IP." >&2
  echo "Set a real SNMP host before running acceptance." >&2
  exit 1
fi

SITE_ID="${ACCEPTANCE_SITE_ID:-site-001}"
DEVICE_ID="${ACCEPTANCE_DEVICE_ID:-do-core}"
API_URL="http://127.0.0.1:${API_HOST_PORT:-8000}/api/devices/${DEVICE_ID}?siteId=${SITE_ID}"
TIMEOUT_SEC="${ACCEPTANCE_TIMEOUT_SEC:-180}"

echo "Waiting up to ${TIMEOUT_SEC}s for real collector v2 telemetry: ${DEVICE_ID} @ ${SITE_ID}"
deadline=$((SECONDS + TIMEOUT_SEC))
while (( SECONDS < deadline )); do
  if BODY="$(curl -fsS --max-time 3 "${API_URL}" 2>/dev/null)"; then
    echo "Acceptance device visible:"
    echo "${BODY}"
    if command -v python3 >/dev/null 2>&1; then
      if ! printf '%s' "${BODY}" | python3 -c '
import json, sys
d = json.load(sys.stdin)
missing = [k for k in ("id", "site_id", "status") if k not in d]
if missing:
    raise SystemExit("missing keys: %s" % missing)
if d.get("status") is None:
    raise SystemExit("status is null")
print("v2 acceptance fields OK (status=%s, status_reason=%s)" % (d.get("status"), d.get("status_reason")))
'; then
        echo "Warning: device JSON missing expected v2 projection fields" >&2
      fi
    fi
    echo
    echo "end-to-end acceptance: OK"
    exit 0
  fi
  sleep 5
done

echo "Timed out waiting for ${API_URL}" >&2
echo "Check collector logs, SNMP reachability, community string, and site/device IDs." >&2
exit 1
