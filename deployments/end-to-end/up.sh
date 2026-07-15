#!/usr/bin/env bash
# Start the full end-to-end stack (all services including SNMP collector).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

ROOT="$(repo_root)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"

cd "${ROOT}"
require_docker
require_cmd curl

ensure_env_file "${ENV_FILE}" "${SCRIPT_DIR}/.env.example"
load_env_file "${ENV_FILE}"

ADMIN_DATABASE_URL="${ADMIN_DATABASE_URL:-postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable}"
OGSD_INGESTION_PASSWORD="${OGSD_INGESTION_PASSWORD:-ingestion}"
OGSD_API_PASSWORD="${OGSD_API_PASSWORD:-api}"

ensure_mqtt_material "${ROOT}"

echo "Starting end-to-end Compose project..."
compose_cmd "${ENV_FILE}" "${COMPOSE}" up --build -d

echo "Waiting for Postgres..."
wait_postgres "${COMPOSE}" "${ENV_FILE}" "${POSTGRES_USER:-ogsd}" "${POSTGRES_DB:-ogsd}"

migrate_and_bootstrap_roles "${ROOT}" "${ADMIN_DATABASE_URL}" "${OGSD_INGESTION_PASSWORD}" "${OGSD_API_PASSWORD}"

echo "Waiting for application health endpoints..."
wait_http "http://127.0.0.1:${INGESTION_ADMIN_PORT:-9091}/healthz" "ingestion"
wait_http "http://127.0.0.1:${API_ADMIN_PORT:-9092}/healthz" "backend-api"
wait_http "http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz" "snmp-collector"
wait_http "http://127.0.0.1:${FRONTEND_HOST_PORT:-80}/" "frontend"
wait_http "http://127.0.0.1:${API_HOST_PORT:-8000}/api/test-config" "backend-api REST"

compose_cmd "${ENV_FILE}" "${COMPOSE}" ps

cat <<EOF

End-to-end stack is up (Compose project: ogsd-e2e).

  mosquitto       TLS 0.0.0.0:${MQTT_HOST_PORT:-8883}
  postgres        127.0.0.1:${POSTGRES_HOST_PORT:-5432}
  ingestion       http://127.0.0.1:${INGESTION_ADMIN_PORT:-9091}/healthz
  backend-api     http://127.0.0.1:${API_HOST_PORT:-8000}
  frontend        http://127.0.0.1:${FRONTEND_HOST_PORT:-80}/
  snmp-collector  http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz

Edit real device inventory in:
  ${SCRIPT_DIR}/configs/collector.yaml

Validate:
  ./deployments/end-to-end/validate.sh
  ./deployments/end-to-end/smoke.sh
  ./deployments/end-to-end/acceptance.sh   # requires reachable SNMP device

Stop:
  ./deployments/end-to-end/down.sh
EOF
