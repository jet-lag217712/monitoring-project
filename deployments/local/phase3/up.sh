#!/usr/bin/env bash
# Start Phase 3 local containers (snmpsim + Mosquitto + Postgres) via Docker Desktop.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"
COMPOSE="${ROOT}/deployments/local/phase3/docker-compose.yaml"

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
docker compose -f "${COMPOSE}" ps

cat <<EOF

Phase 3 containers are up:
  snmpsim   UDP 127.0.0.1:1161
  mosquitto TLS 127.0.0.1:8883
  postgres  127.0.0.1:5432 (user/db/password: ogsd)

Network name: ogsd-phase3_default

Export for ingestion / integration tests:
  export MQTT_PASSWORD=ingestion
  export MQTT_BROKER=tls://127.0.0.1:8883
  export MQTT_CA_FILE=infrastructure/docker/mqtt-broker/certs/ca.crt
  export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable

Run ingestion (host):
  cd services/ingestion-service
  go run ./cmd/ingestion -config configs/ingestion.example.yaml

Run collector (host):
  cd services/snmp-collector
  export MQTT_PASSWORD=secret
  go run ./cmd/collector -config configs/collector.mqtt.example.yaml

Stop everything:
  ./deployments/local/phase3/down.sh
EOF
