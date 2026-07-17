#!/usr/bin/env bash
# Validate development vxrail skeleton (compose render with placeholder env).
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
export SNMP_COMMUNITY_DO_CORE=placeholder
export SNMP_COMMUNITY_SITE_A_MDF=placeholder
export SNMP_COMMUNITY_SITE_A_IDF1=placeholder
export SNMP_COMMUNITY_SITE_A_IDF2=placeholder
export SNMP_COMMUNITY_SITE_B_MDF=placeholder
export SNMP_COMMUNITY_SITE_C_MDF=placeholder
export SNMP_COMMUNITY_SITE_C_IDF1=placeholder

docker compose -f "${COMPOSE}" config >/dev/null
echo "development/vxrail validate: OK"
