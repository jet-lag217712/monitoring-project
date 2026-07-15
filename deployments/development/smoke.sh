#!/usr/bin/env bash
# Smoke-test the development cloud plane (health + synthetic MQTT → API).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

ROOT="$(repo_root)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"

cd "${ROOT}"
require_docker
require_cmd curl

ensure_env_file "${ENV_FILE}" "${SCRIPT_DIR}/.env.example"
load_env_file "${ENV_FILE}"
require_file "${MQTT_DIR}/certs/ca.crt"

wait_http "http://127.0.0.1:${INGESTION_ADMIN_PORT:-9091}/healthz" "ingestion"
wait_http "http://127.0.0.1:${API_ADMIN_PORT:-9092}/healthz" "backend-api"
wait_http "http://127.0.0.1:${FRONTEND_HOST_PORT:-80}/" "frontend"
wait_http "http://127.0.0.1:${API_HOST_PORT:-8000}/api/test-config" "backend-api REST"

NETWORK="$(compose_cmd "${ENV_FILE}" "${COMPOSE}" ps -q mosquitto | xargs -I{} docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}' {})"
require_nonempty COMPOSE_NETWORK "${NETWORK}"

MQTT_PASSWORD="${MQTT_COLLECTOR_PASSWORD:-secret}" \
MQTT_CA_FILE="${MQTT_DIR}/certs/ca.crt" \
COMPOSE_NETWORK="${NETWORK}" \
API_BASE="http://127.0.0.1:${API_HOST_PORT:-8000}" \
SMOKE_SITE_ID="${SMOKE_SITE_ID:-site-dev}" \
SMOKE_DEVICE_ID="${SMOKE_DEVICE_ID:-dev-smoke-device}" \
  "${SCRIPT_DIR}/../lib/smoke_mqtt_to_api.sh"

echo "development smoke: OK"
