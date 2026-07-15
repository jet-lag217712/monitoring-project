#!/usr/bin/env bash
# Publish a schema-valid device metric over MQTT/TLS and wait until the API sees it.
# Required env: MQTT_PASSWORD, MQTT_CA_FILE, COMPOSE_NETWORK (or MQTT_HOST),
#               API_BASE (e.g. http://127.0.0.1:8000), SMOKE_SITE_ID, SMOKE_DEVICE_ID
# Optional: MQTT_HOST (default mosquitto), MQTT_PORT (default 8883), SMOKE_VALUE
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
SMOKE_VALUE="${SMOKE_VALUE:-42}"
TOPIC="site/${SMOKE_SITE_ID}/device/${SMOKE_DEVICE_ID}/metric/device"
TS="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
PAYLOAD="$(printf '{"timestamp":"%s","metric":"uptime_seconds","value":%s,"ip_address":"10.255.255.254"}' "${TS}" "${SMOKE_VALUE}")"

DOCKER_ARGS=(run --rm -v "${MQTT_CA_FILE}:/certs/ca.crt:ro")
if [[ -n "${COMPOSE_NETWORK:-}" ]]; then
  DOCKER_ARGS+=(--network "${COMPOSE_NETWORK}")
else
  # Publish to host-mapped Mosquitto (works on Docker Desktop / OrbStack).
  DOCKER_ARGS+=(--add-host=host.docker.internal:host-gateway)
  MQTT_HOST="${MQTT_HOST:-host.docker.internal}"
fi

echo "Publishing synthetic telemetry to ${TOPIC} via ${MQTT_HOST}:${MQTT_PORT}..."
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
  if curl -fsS --max-time 3 "${API_URL}" >/dev/null 2>&1; then
    echo "Smoke telemetry visible at ${API_URL}"
    curl -fsS --max-time 3 "${API_URL}"
    echo
    exit 0
  fi
  sleep 1
done

echo "Timed out waiting for device ${SMOKE_DEVICE_ID} at ${API_URL}" >&2
exit 1
