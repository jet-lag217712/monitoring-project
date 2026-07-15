#!/usr/bin/env bash
# Stop the development cloud-plane Compose project (volumes preserved unless -v).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

ROOT="$(repo_root)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"

cd "${ROOT}"
require_docker

DOWN_ARGS=(down)
if [[ "${DOWN_VOLUMES:-}" == "1" ]] || [[ "${1:-}" == "-v" ]]; then
  DOWN_ARGS+=(-v)
  echo "Removing volumes as well."
fi

compose_cmd "${ENV_FILE}" "${COMPOSE}" "${DOWN_ARGS[@]}"
echo "Development cloud-plane containers stopped."
