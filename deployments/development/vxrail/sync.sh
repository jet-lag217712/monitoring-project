#!/usr/bin/env bash
# Push development/vxrail runtime files to the OrbStack Ubuntu VM over SSH (tar stream).
# Usage: ./sync.sh [--dry-run]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../../lib/common.sh
source "${SCRIPT_DIR}/../../lib/common.sh"

ROOT="$(repo_root)"
ENV_FILE="${SCRIPT_DIR}/.env"
DRY_RUN=0
if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=1
fi

cd "${ROOT}"
ensure_env_file "${ENV_FILE}" "${SCRIPT_DIR}/.env.example"
load_env_file "${ENV_FILE}"

require_cmd ssh
require_cmd tar

VXRAIL_SSH_HOST="${VXRAIL_SSH_HOST:?set VXRAIL_SSH_HOST in vxrail/.env}"
VXRAIL_SSH_USER="${VXRAIL_SSH_USER:-gns3}"
VXRAIL_REMOTE_DIR="${VXRAIL_REMOTE_DIR:-/home/gns3/ogsd-vxrail}"
REMOTE="${VXRAIL_SSH_USER}@${VXRAIL_SSH_HOST}"

MQTT_CA="${ROOT}/infrastructure/docker/mqtt-broker/certs/ca.crt"
require_file "${MQTT_CA}"
require_file "${SCRIPT_DIR}/docker-compose.yml"
require_file "${SCRIPT_DIR}/configs/collector.yaml"
require_file "${SCRIPT_DIR}/bootstrap.sh"

echo "Sync target: ${REMOTE}:${VXRAIL_REMOTE_DIR}"
if [[ "${DRY_RUN}" -eq 1 ]]; then
  echo "Would sync: docker-compose.yml, configs/, .env.example, bootstrap.sh, setup-gns3-bridge.sh, services/snmp-collector/, ca.crt"
  exit 0
fi

ssh "${REMOTE}" "mkdir -p '${VXRAIL_REMOTE_DIR}/certs' '${VXRAIL_REMOTE_DIR}/src/services'"

# Profile files
tar -C "${SCRIPT_DIR}" -cf - \
  docker-compose.yml \
  validate.sh \
  bootstrap.sh \
  setup-gns3-bridge.sh \
  .env.example \
  configs \
  run \
  README.md \
  | ssh "${REMOTE}" "tar -C '${VXRAIL_REMOTE_DIR}' -xf -"

# Collector source for local image builds on the VM
tar -C "${ROOT}/services" -cf - snmp-collector \
  | ssh "${REMOTE}" "tar -C '${VXRAIL_REMOTE_DIR}/src/services' -xf -"

# Public CA only
scp -q "${MQTT_CA}" "${REMOTE}:${VXRAIL_REMOTE_DIR}/certs/ca.crt"

# Compose build context expects ../../../services/snmp-collector from vxrail/;
# on the VM we rewrite to the synced src tree via a small override note in bootstrap.
echo "sync complete. On the VM: cd ${VXRAIL_REMOTE_DIR} && cp -n .env.example .env && ./bootstrap.sh"
