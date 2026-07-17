#!/usr/bin/env bash
# Documented MQTT outage drill for end-to-end / development stacks.
# Stops Mosquitto briefly, confirms collector stays up, restarts broker, and
# optionally re-runs v2 smoke. Not a CI gate — operator-invoked.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "${SCRIPT_DIR}/common.sh"

PROFILE_DIR="${1:-}"
if [[ -z "${PROFILE_DIR}" || ! -d "${PROFILE_DIR}" ]]; then
  echo "Usage: $0 <deployments/end-to-end|deployments/development>" >&2
  exit 2
fi

ROOT="$(repo_root)"
COMPOSE="${PROFILE_DIR}/docker-compose.yml"
ENV_FILE="${PROFILE_DIR}/.env"

cd "${ROOT}"
require_docker
require_cmd curl
ensure_env_file "${ENV_FILE}" "${PROFILE_DIR}/.env.example"
load_env_file "${ENV_FILE}"

COLLECTOR_URL="http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz"
INGESTION_URL="http://127.0.0.1:${INGESTION_ADMIN_PORT:-9091}/healthz"

echo "Pre-check collector health..."
wait_http "${COLLECTOR_URL}" "snmp-collector"

echo "Stopping Mosquitto to simulate MQTT/TLS outage..."
compose_cmd "${ENV_FILE}" "${COMPOSE}" stop mosquitto

echo "Collector must remain live while buffering..."
sleep 2
curl -fsS --max-time 3 "${COLLECTOR_URL}" >/dev/null
echo "Collector healthz OK during outage"

echo "Restarting Mosquitto..."
compose_cmd "${ENV_FILE}" "${COMPOSE}" start mosquitto
wait_http "${INGESTION_URL}" "ingestion"

if [[ -x "${PROFILE_DIR}/smoke.sh" ]]; then
  echo "Re-running profile smoke after broker recovery..."
  "${PROFILE_DIR}/smoke.sh"
fi

echo "mqtt_outage_drill: OK"
