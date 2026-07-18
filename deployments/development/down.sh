#!/usr/bin/env bash
# Stop the development cloud-plane Compose project.
# Testing profile: wipes all Compose volumes by default (postgres-data, mosquitto-data).
# Use --keep-data or DOWN_KEEP_DATA=1 to stop containers without erasing state.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../lib/common.sh
source "${SCRIPT_DIR}/../lib/common.sh"

ROOT="$(repo_root)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"

usage() {
  echo "Usage: $0 [--keep-data]" >&2
  echo "  Default: stop containers and wipe all development volumes (clean test reset)." >&2
  echo "  --keep-data: stop containers only; preserve postgres-data and mosquitto-data." >&2
  exit 1
}

KEEP_DATA=0
for arg in "$@"; do
  case "${arg}" in
    --keep-data) KEEP_DATA=1 ;;
    -v) ;; # backward compatible no-op; wipe is already the default
    -h|--help) usage ;;
    *) echo "Unknown option: ${arg}" >&2; usage ;;
  esac
done
if [[ "${DOWN_KEEP_DATA:-}" == "1" ]]; then
  KEEP_DATA=1
fi

cd "${ROOT}"
require_docker

DOWN_ARGS=(down)
if [[ "${KEEP_DATA}" -eq 0 ]]; then
  DOWN_ARGS+=(-v)
  echo "Wiping development volumes (postgres-data, mosquitto-data)."
else
  echo "Stopping containers; volumes preserved (--keep-data)."
fi

compose_cmd "${ENV_FILE}" "${COMPOSE}" "${DOWN_ARGS[@]}"
echo "Development cloud-plane containers stopped."
