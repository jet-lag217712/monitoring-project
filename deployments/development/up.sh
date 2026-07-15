#!/usr/bin/env bash
# Start the development cloud plane (Frontend, DB, Ingestion, Mosquitto, Backend API).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

ROOT="/Users/jeetlad/Projects/Equate/monitoring-dashboard"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"

cd "${ROOT}"
require_docker
require_cmd curl

ensure_env_file "${ENV_FILE}" "${SCRIPT_DIR}/.env.example"
load_env_file "${ENV_FILE}"

ADMIN_DATABASE_URL="${ADMIN_DATABASE_URL:-postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable}"
OGSD_INGESTION_PASSWORD="${OGSD_INGESTION_PASSWORD:-ingestion}"
OGSD_API_PASSWORD="${OGSD_API_PASSWORD:-api}"

if [[ -z "${MQTT_SERVER_IP:-}" ]]; then
  echo "WARNING: MQTT_SERVER_IP is empty. Set it to the Mac IP visible from the OrbStack VM before first cert gen."
fi

ensure_mqtt_material "${ROOT}"

echo "Starting development cloud-plane Compose project..."
compose_cmd "${ENV_FILE}" "${COMPOSE}" up --build -d

echo "Waiting for Postgres..."
wait_postgres "${COMPOSE}" "${ENV_FILE}" "${POSTGRES_USER:-ogsd}" "${POSTGRES_DB:-ogsd}"

migrate_and_bootstrap_roles "${ROOT}" "${ADMIN_DATABASE_URL}" "${OGSD_INGESTION_PASSWORD}" "${OGSD_API_PASSWORD}"

echo "Waiting for application health endpoints..."
wait_http "http://127.0.0.1:${INGESTION_ADMIN_PORT:-9091}/healthz" "ingestion"
wait_http "http://127.0.0.1:${API_ADMIN_PORT:-9092}/healthz" "backend-api"
wait_http "http://127.0.0.1:${FRONTEND_HOST_PORT:-80}/" "frontend"
wait_http "http://127.0.0.1:${API_HOST_PORT:-8000}/api/test-config" "backend-api REST"

compose_cmd "${ENV_FILE}" "${COMPOSE}" ps

INGESTION_DSN="postgres://ogsd_ingestion:${OGSD_INGESTION_PASSWORD}@127.0.0.1:${POSTGRES_HOST_PORT:-5432}/ogsd?sslmode=disable"

cat <<EOF

Development cloud plane is up (Compose project: ogsd-development).

  mosquitto     TLS 0.0.0.0:${MQTT_HOST_PORT:-8883}  (reachable from OrbStack VM)
  postgres      127.0.0.1:${POSTGRES_HOST_PORT:-5432}
  ingestion     http://127.0.0.1:${INGESTION_ADMIN_PORT:-9091}/healthz
  backend-api   http://127.0.0.1:${API_HOST_PORT:-8000}
  frontend      http://127.0.0.1:${FRONTEND_HOST_PORT:-80}/

Next — collector on the OrbStack Ubuntu VM:
  1. Set MQTT_SERVER_IP in ${ENV_FILE} (Mac IP) and regenerate certs if needed
  2. ./deployments/development/vxrail/sync.sh
  3. On the VM: sudo ./setup-gns3-bridge.sh && ./bootstrap.sh

Export for host-run integration tests:
  export MQTT_PASSWORD=ingestion
  export MQTT_BROKER=tls://127.0.0.1:8883
  export MQTT_CA_FILE=${MQTT_DIR}/certs/ca.crt
  export DATABASE_URL=${INGESTION_DSN}

Stop:
  ./deployments/development/down.sh
EOF
