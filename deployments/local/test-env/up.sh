#!/usr/bin/env bash
# Start the local test-env stack (snmpsim + Mosquitto + Postgres) via Docker Desktop,
# then apply migrations and bootstrap application role passwords.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"
COMPOSE="${ROOT}/deployments/local/test-env/docker-compose.yaml"

# Superuser DSN for migrations / role bootstrap (local only).
ADMIN_DATABASE_URL="${ADMIN_DATABASE_URL:-postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable}"
OGSD_INGESTION_PASSWORD="${OGSD_INGESTION_PASSWORD:-ingestion}"
OGSD_API_PASSWORD="${OGSD_API_PASSWORD:-api}"

cd "${ROOT}"

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not reachable. Start Docker Desktop, then retry." >&2
  exit 1
fi

if [[ ! -f "${MQTT_DIR}/certs/ca.crt" || ! -f "${MQTT_DIR}/certs/server.crt" ]]; then
  echo "Generating Mosquitto TLS certs..."
  "${MQTT_DIR}/scripts/gen-dev-certs.sh"
fi

if [[ ! -f "${MQTT_DIR}/passwords" ]]; then
  echo "Creating Mosquitto password file..."
  MQTT_COLLECTOR_PASSWORD="${MQTT_COLLECTOR_PASSWORD:-secret}" \
  MQTT_INGESTION_PASSWORD="${MQTT_INGESTION_PASSWORD:-ingestion}" \
    "${MQTT_DIR}/scripts/gen-passwords.sh"
  echo "Collector password: ${MQTT_COLLECTOR_PASSWORD:-secret}"
  echo "Ingestion password: ${MQTT_INGESTION_PASSWORD:-ingestion}"
fi

docker compose -f "${COMPOSE}" up --build -d

echo "Waiting for Postgres to become healthy..."
for _ in $(seq 1 60); do
  if docker compose -f "${COMPOSE}" exec -T postgres pg_isready -U ogsd -d ogsd >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
if ! docker compose -f "${COMPOSE}" exec -T postgres pg_isready -U ogsd -d ogsd >/dev/null 2>&1; then
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

docker compose -f "${COMPOSE}" ps

INGESTION_DSN="postgres://ogsd_ingestion:${OGSD_INGESTION_PASSWORD}@127.0.0.1:5432/ogsd?sslmode=disable"

cat <<EOF

Local test-env containers are up:
  snmpsim   UDP 127.0.0.1:1161
  mosquitto TLS 127.0.0.1:8883
  postgres  127.0.0.1:5432

Network name: ogsd-test-env_default

Roles:
  ogsd              superuser (migrations / bootstrap only)
  ogsd_ingestion    ingestion writes (password: ${OGSD_INGESTION_PASSWORD})
  ogsd_api          API reads (password: ${OGSD_API_PASSWORD})

Export for ingestion / integration tests:
  export MQTT_PASSWORD=ingestion
  export MQTT_BROKER=tls://127.0.0.1:8883
  # Absolute path so it works from any cwd (e.g. services/ingestion-service).
  export MQTT_CA_FILE=${ROOT}/infrastructure/docker/mqtt-broker/certs/ca.crt
  export DATABASE_URL=${INGESTION_DSN}

Run ingestion (host):
  cd services/ingestion-service
  go run ./cmd/ingestion -config configs/ingestion.example.yaml

Run collector (host):
  cd services/snmp-collector
  export MQTT_PASSWORD=secret
  go run ./cmd/collector -config configs/collector.mqtt.example.yaml

Stop everything:
  ./deployments/local/test-env/down.sh

Schema changes on an existing volume: migrations re-run safely via migrate.
If you previously used initdb-mounted schema and hit conflicts, reset once:
  docker compose -f deployments/local/test-env/docker-compose.yaml down -v
  ./deployments/local/test-env/up.sh
EOF
