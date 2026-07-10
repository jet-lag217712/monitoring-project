#!/usr/bin/env bash
# Prepare local-physical VxRail (CA + env) and print host go run command.
# Usage:
#   ./bootstrap.sh                 # prepare + print go run instructions
#   ./bootstrap.sh --prepare-only  # CA/env only
#   ./bootstrap.sh --compose       # Linux host-network compose (not for Docker Desktop Mac)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"
CA_DST="${SCRIPT_DIR}/certs/ca.crt"
MODE="${1:-}"

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

MQTT_BROKER="${MQTT_BROKER:-tls://127.0.0.1:8883}"
MQTT_PASSWORD="${MQTT_PASSWORD:-secret}"

mkdir -p "${SCRIPT_DIR}/certs"
if [[ ! -f "${CA_DST}" ]]; then
  if [[ -f "${MQTT_DIR}/certs/ca.crt" ]]; then
    echo "Copying CA from local Mosquitto certs..."
    cp "${MQTT_DIR}/certs/ca.crt" "${CA_DST}"
  else
    echo "Missing Mosquitto CA. Start the cloud plane first:" >&2
    echo "  ./deployments/local/up.sh" >&2
    exit 1
  fi
fi

if grep -q 'REPLACE_WITH_' "${SCRIPT_DIR}/configs/collector.host.yaml" 2>/dev/null; then
  echo "WARNING: configs/collector.host.yaml still has REPLACE_WITH_* placeholders." >&2
  echo "Edit device hosts/communities before expecting successful SNMP polls." >&2
fi

if [[ "${MODE}" == "--compose" ]]; then
  if ! docker info >/dev/null 2>&1; then
    echo "Docker is not reachable." >&2
    exit 1
  fi
  echo "Starting Compose collector (network_mode: host — intended for Linux)..."
  docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" up --build -d
  docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" ps
  echo "Admin: http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz"
  exit 0
fi

cat <<EOF

Local-physical VxRail is prepared.

  MQTT_BROKER=${MQTT_BROKER}
  CA=${CA_DST}

Edit device inventory if needed:
  ${SCRIPT_DIR}/configs/collector.host.yaml

Start collector on the Mac host (recommended):

  cd ${ROOT}/services/snmp-collector
  export MQTT_PASSWORD=${MQTT_PASSWORD}
  export MQTT_BROKER=${MQTT_BROKER}
  go run ./cmd/collector -config ${SCRIPT_DIR}/configs/collector.host.yaml

Verify:
  curl -sf http://127.0.0.1:9090/healthz

Cloud plane (if not already up):
  ./deployments/local/up.sh

EOF

if [[ "${MODE}" == "--prepare-only" ]]; then
  exit 0
fi
