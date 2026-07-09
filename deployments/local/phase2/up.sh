#!/usr/bin/env bash
# Start Phase 2 local containers (snmpsim + Mosquitto) via Docker Desktop.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"
COMPOSE="${ROOT}/deployments/local/phase2/docker-compose.yaml"

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
  echo "Export: export MQTT_PASSWORD=${MQTT_COLLECTOR_PASSWORD:-secret}"
fi

docker compose -f "${COMPOSE}" up --build -d
docker compose -f "${COMPOSE}" ps

cat <<EOF

Phase 2 containers are up:
  snmpsim   UDP 127.0.0.1:1161
  mosquitto TLS 127.0.0.1:8883

Run the collector (host):
  cd services/snmp-collector
  export MQTT_PASSWORD=\${MQTT_PASSWORD:-secret}
  go run ./cmd/collector -config configs/collector.mqtt.example.yaml

Subscribe to telemetry:
  cd infrastructure/docker/mqtt-broker
  docker run --rm -it --network ogsd-phase2_default \\
    -v "\$PWD/certs:/certs:ro" eclipse-mosquitto:2 \\
    mosquitto_sub -h mosquitto -p 8883 --cafile /certs/ca.crt \\
    -u ingestion -P ingestion -t 'site/+/device/+/metric/#' -v

Stop everything:
  ./deployments/local/phase2/down.sh
EOF
