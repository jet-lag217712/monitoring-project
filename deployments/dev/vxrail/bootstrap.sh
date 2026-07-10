#!/usr/bin/env bash
# Bootstrap dev VxRail collector (MQTT egress to Azure Mosquitto).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
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
  echo "Set MQTT_BROKER in ${ENV_FILE} to the Azure Mosquitto URL, e.g. tls://x.x.x.x:8883" >&2
  exit 1
fi

mkdir -p "${SCRIPT_DIR}/certs"
if [[ ! -f "${CA_DST}" ]]; then
  echo "Missing ${CA_DST}." >&2
  echo "Copy the CA that signed the Azure Mosquitto server cert into certs/ca.crt." >&2
  echo "See deployments/dev/cloud/README.md." >&2
  exit 1
fi

echo "Starting dev VxRail collector (MQTT_BROKER=${MQTT_BROKER})..."
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" up --build -d
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" ps

cat <<EOF

Dev VxRail plane is up (Compose project: ogsd-dev-vxrail).

  snmp-collector  admin :${COLLECTOR_ADMIN_PORT:-9090}
  MQTT egress     ${MQTT_BROKER}

Verify:
  curl -sf http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz

Stop:
  docker compose --env-file ${ENV_FILE} -f ${COMPOSE} down
EOF
