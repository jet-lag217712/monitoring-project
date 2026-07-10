#!/usr/bin/env bash
# Bootstrap local VxRail collector on the Debian VM (MQTT egress to Mac Mosquitto).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"
CA_DST="${SCRIPT_DIR}/certs/ca.crt"

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

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not reachable. Start Docker, then retry." >&2
  exit 1
fi

if [[ -z "${MQTT_BROKER:-}" || "${MQTT_BROKER}" == *"REPLACE"* ]]; then
  echo "Set MQTT_BROKER in ${ENV_FILE} to the Mac Mosquitto URL, e.g. tls://192.168.x.x:8883" >&2
  exit 1
fi

mkdir -p "${SCRIPT_DIR}/certs"
if [[ ! -f "${CA_DST}" ]]; then
  if [[ -f "${MQTT_DIR}/certs/ca.crt" ]]; then
    echo "Copying CA from Mac mqtt-broker certs (same repo checkout)..."
    cp "${MQTT_DIR}/certs/ca.crt" "${CA_DST}"
  else
    echo "Missing ${CA_DST}." >&2
    echo "Copy ca.crt from the Mac host:" >&2
    echo "  infrastructure/docker/mqtt-broker/certs/ca.crt → deployments/local/vxrail/certs/ca.crt" >&2
    exit 1
  fi
fi

echo "Starting local VxRail collector (MQTT_BROKER=${MQTT_BROKER})..."
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" up --build -d
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" ps

cat <<EOF

Local VxRail plane is up (Compose project: ogsd-local-vxrail).

  snmp-collector  admin :${COLLECTOR_ADMIN_PORT:-9090}
  MQTT egress     ${MQTT_BROKER}

Verify:
  curl -sf http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz

Stop:
  docker compose --env-file ${ENV_FILE} -f ${COMPOSE} down
EOF
