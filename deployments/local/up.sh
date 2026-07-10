#!/usr/bin/env bash
# Start the Mac cloud-plane stack: certs, passwords, compose, migrations, roles.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"

cd "${ROOT}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Creating ${ENV_FILE} from .env.example..."
  cp "${SCRIPT_DIR}/.env.example" "${ENV_FILE}"
fi

# shellcheck disable=SC1090
set -a
# shellcheck source=/dev/null
source "${ENV_FILE}"
set +a

ADMIN_DATABASE_URL="${ADMIN_DATABASE_URL:-postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable}"
OGSD_INGESTION_PASSWORD="${OGSD_INGESTION_PASSWORD:-ingestion}"
OGSD_API_PASSWORD="${OGSD_API_PASSWORD:-api}"
MQTT_COLLECTOR_PASSWORD="${MQTT_COLLECTOR_PASSWORD:-secret}"
MQTT_INGESTION_PASSWORD="${MQTT_INGESTION_PASSWORD:-ingestion}"./

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not reachable. Start Docker Desktop, then retry." >&2
  exit 1
fi

if [[ ! -f "${MQTT_DIR}/certs/ca.crt" || ! -f "${MQTT_DIR}/certs/server.crt" ]]; then
  echo "Generating Mosquitto TLS certs..."
  MQTT_SERVER_CN="${MQTT_SERVER_CN:-localhost}" \
  MQTT_SERVER_DNS="${MQTT_SERVER_DNS:-mosquitto}" \
  MQTT_SERVER_IP="${MQTT_SERVER_IP:-}" \
    "${MQTT_DIR}/scripts/gen-dev-certs.sh"
else
  echo "Mosquitto TLS certs already present (delete ${MQTT_DIR}/certs to regenerate with new SANs)."
fi

if [[ ! -f "${MQTT_DIR}/passwords" ]]; then
  echo "Creating Mosquitto password file..."
  MQTT_COLLECTOR_PASSWORD="${MQTT_COLLECTOR_PASSWORD}" \
  MQTT_INGESTION_PASSWORD="${MQTT_INGESTION_PASSWORD}" \
    "${MQTT_DIR}/scripts/gen-passwords.sh"
  echo "Collector password: ${MQTT_COLLECTOR_PASSWORD}"
  echo "Ingestion password: ${MQTT_INGESTION_PASSWORD}"
fi

echo "Starting local cloud-plane Compose project..."
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" up --build -d

echo "Waiting for Postgres to become healthy..."
for _ in $(seq 1 60); do
  if docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" exec -T postgres \
    pg_isready -U "${POSTGRES_USER:-ogsd}" -d "${POSTGRES_DB:-ogsd}" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if ! docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" exec -T postgres \
  pg_isready -U "${POSTGRES_USER:-ogsd}" -d "${POSTGRES_DB:-ogsd}" >/dev/null 2>&1; then
  echo "Postgres did not become ready in time." >&2
  exit 1
fi

echo "Applying database migrations..."
DATABASE_URL="${ADMIN_DATABASE_URL}" "${ROOT}/infrastructure/script/migrate.sh" up

echo "Bootstrapping application role passwords..."
DATABASE_URL="${ADMIN_DATABASE_URL}" \
  OGSD_INGESTION_PASSWORD="${OGSD_INGESTION_PASSWORD}" \
  OGSD_API_PASSWORD="${OGSD_API_PASSWORD}" \
  "${ROOT}/infrastructure/script/bootstrap-db-roles.sh"

docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" ps

INGESTION_DSN="postgres://ogsd_ingestion:${OGSD_INGESTION_PASSWORD}@127.0.0.1:5432/ogsd?sslmode=disable"

cat <<EOF

Local cloud plane is up (Compose project: ogsd-local).

  mosquitto     TLS 0.0.0.0:${MQTT_HOST_PORT:-8883}  (reachable from Debian VM)
  postgres      127.0.0.1:${POSTGRES_HOST_PORT:-5432}
  ingestion     http://127.0.0.1:${INGESTION_ADMIN_PORT:-9091}/healthz
  backend-api   http://127.0.0.1:${API_HOST_PORT:-8000}
  frontend      http://127.0.0.1:${FRONTEND_HOST_PORT:-80}/

Next — on the Debian VM (GNS3 + collector):
  1. Copy ${MQTT_DIR}/certs/ca.crt → deployments/local/vxrail/certs/ca.crt
  2. Set MQTT_BROKER=tls://<mac-host-ip>:8883 in deployments/local/vxrail/.env
  3. ./deployments/local/vxrail/bootstrap.sh

Export for host-run integration tests:
  export MQTT_PASSWORD=ingestion
  export MQTT_BROKER=tls://127.0.0.1:8883
  export MQTT_CA_FILE=${MQTT_DIR}/certs/ca.crt
  export DATABASE_URL=${INGESTION_DSN}

Azure path (later): see deployments/dev/

Stop:
  ./deployments/local/down.sh
EOF
