#!/usr/bin/env bash
# Validate production vxrail skeleton (compose render with placeholder env).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../../lib/common.sh
source "${SCRIPT_DIR}/../../lib/common.sh"

ROOT="$(repo_root)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"

cd "${ROOT}"
require_docker

require_file "${COMPOSE}"
require_file "${SCRIPT_DIR}/configs/collector.yaml"
require_file "${ROOT}/services/snmp-collector/Dockerfile"
mkdir -p "${SCRIPT_DIR}/run"

export MQTT_BROKER=tls://example.com:8883
export MQTT_PASSWORD=placeholder
export SNMP_COMMUNITY=placeholder
export SNMP_DISCOVERY_COMMUNITY=placeholder

docker compose -f "${COMPOSE}" config >/dev/null
echo "production/vxrail validate: OK"
