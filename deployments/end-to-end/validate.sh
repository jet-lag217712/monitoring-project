#!/usr/bin/env bash
# Validate end-to-end compose/config before startup.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

ROOT="$(repo_root)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"

cd "${ROOT}"
require_docker

ensure_env_file "${ENV_FILE}" "${SCRIPT_DIR}/.env.example"
load_env_file "${ENV_FILE}"

require_file "${COMPOSE}"
require_file "${SCRIPT_DIR}/configs/ingestion.yaml"
require_file "${SCRIPT_DIR}/configs/api.yaml"
require_file "${SCRIPT_DIR}/configs/collector.yaml"
require_file "${SCRIPT_DIR}/nginx-frontend.conf"
require_file "${ROOT}/services/snmp-collector/Dockerfile"
require_file "${ROOT}/services/ingestion-service/Dockerfile"
require_file "${ROOT}/services/backend-api/Dockerfile"
require_file "${ROOT}/frontend/Dockerfile"
require_file "${ROOT}/infrastructure/docker/mqtt-broker/Dockerfile"

mkdir -p "${SCRIPT_DIR}/run"

if grep -q 'REPLACE_WITH_DEVICE_IP' "${SCRIPT_DIR}/configs/collector.yaml"; then
  echo "NOTE: configs/collector.yaml still has REPLACE_WITH_DEVICE_IP (OK for synthetic smoke; required for acceptance)."
fi

echo "Rendering compose config..."
compose_cmd "${ENV_FILE}" "${COMPOSE}" config >/dev/null
echo "end-to-end validate: OK"
