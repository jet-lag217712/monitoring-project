#!/usr/bin/env bash
# Publish a schema-valid v2 device_telemetry event over MQTT/TLS and wait until the API sees it.
# Required env: MQTT_PASSWORD, MQTT_CA_FILE, COMPOSE_NETWORK (or MQTT_HOST),
#               API_BASE (e.g. http://127.0.0.1:8000), SMOKE_SITE_ID, SMOKE_DEVICE_ID
# Optional: MQTT_HOST (default mosquitto), MQTT_PORT (default 8883),
#           SMOKE_COLLECTOR_ID, SMOKE_UPTIME_SECONDS
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

require_cmd curl docker date

require_nonempty MQTT_PASSWORD "${MQTT_PASSWORD:-}"
require_nonempty MQTT_CA_FILE "${MQTT_CA_FILE:-}"
require_nonempty API_BASE "${API_BASE:-}"
require_nonempty SMOKE_SITE_ID "${SMOKE_SITE_ID:-}"
require_nonempty SMOKE_DEVICE_ID "${SMOKE_DEVICE_ID:-}"
require_file "${MQTT_CA_FILE}"

MQTT_HOST="${MQTT_HOST:-mosquitto}"
MQTT_PORT="${MQTT_PORT:-8883}"
SMOKE_COLLECTOR_ID="${SMOKE_COLLECTOR_ID:-collector-smoke}"
SMOKE_UPTIME_SECONDS="${SMOKE_UPTIME_SECONDS:-42}"
TS="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

if command -v uuidgen >/dev/null 2>&1; then
  EVENT_ID="$(uuidgen | tr '[:upper:]' '[:lower:]')"
else
  EVENT_ID="$(python3 - <<'PY'
import uuid
print(uuid.uuid4())
PY
)"
fi

TOPIC="site/${SMOKE_SITE_ID}/device/${SMOKE_DEVICE_ID}/telemetry/v2/device"
PAYLOAD="$(python3 - <<PY
import json
print(json.dumps({
  "schema_version": "2.0",
  "event_id": "${EVENT_ID}",
  "event_type": "device_telemetry",
  "site_id": "${SMOKE_SITE_ID}",
  "collector_id": "${SMOKE_COLLECTOR_ID}",
  "device_id": "${SMOKE_DEVICE_ID}",
  "observed_at": "${TS}",
  "emitted_at": "${TS}",
  "config_revision": "smoke-revision",
  "payload": {
    "identity": {
      "hostname": "${SMOKE_DEVICE_ID}",
      "sys_object_id": "1.3.6.1.4.1.9.1.9999",
      "sys_name": "${SMOKE_DEVICE_ID}",
      "sys_descr": "Synthetic smoke device",
      "vendor": "cisco",
      "model": "smoke-model",
      "serial": "smoke-serial",
      "snmp_version": "2c"
    },
    "profile": {
      "name": "core",
      "capabilities": []
    },
    "readings": {
      "uptime_seconds": ${SMOKE_UPTIME_SECONDS}
    }
  }
}))
PY
)"

DOCKER_ARGS=(run --rm -v "${MQTT_CA_FILE}:/certs/ca.crt:ro")
if [[ -n "${COMPOSE_NETWORK:-}" ]]; then
  DOCKER_ARGS+=(--network "${COMPOSE_NETWORK}")
else
  DOCKER_ARGS+=(--add-host=host.docker.internal:host-gateway)
  MQTT_HOST="${MQTT_HOST:-host.docker.internal}"
fi

echo "Publishing synthetic v2 telemetry to ${TOPIC} via ${MQTT_HOST}:${MQTT_PORT}..."
docker "${DOCKER_ARGS[@]}" eclipse-mosquitto:2 \
  mosquitto_pub \
  -h "${MQTT_HOST}" \
  -p "${MQTT_PORT}" \
  --cafile /certs/ca.crt \
  -u collector \
  -P "${MQTT_PASSWORD}" \
  -t "${TOPIC}" \
  -m "${PAYLOAD}" \
  -q 1

API_URL="${API_BASE}/api/devices/${SMOKE_DEVICE_ID}?siteId=${SMOKE_SITE_ID}"
echo "Waiting for API to expose ${SMOKE_DEVICE_ID}..."
for _ in $(seq 1 45); do
  if BODY="$(curl -fsS --max-time 3 "${API_URL}" 2>/dev/null)"; then
    echo "Smoke v2 telemetry visible at ${API_URL}"
    echo "${BODY}"
    echo
    exit 0
  fi
  sleep 1
done

echo "Timed out waiting for device ${SMOKE_DEVICE_ID} at ${API_URL}" >&2
exit 1
