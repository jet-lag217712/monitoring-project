#!/usr/bin/env bash
# Stop the local laptop E2E Compose project (volumes preserved unless -v).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"
ENV_FILE="${SCRIPT_DIR}/.env"

cd "${ROOT}"

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not reachable. Start Docker, then retry." >&2
  exit 1
fi

DOWN_ARGS=(down)
if [[ "${DOWN_VOLUMES:-}" == "1" ]] || [[ "${1:-}" == "-v" ]]; then
  DOWN_ARGS+=( -v )
  echo "Removing volumes as well."
fi

if [[ -f "${ENV_FILE}" ]]; then
  docker compose --env-file "${ENV_FILE}" -f "${COMPOSE}" "${DOWN_ARGS[@]}"
else
  docker compose -f "${COMPOSE}" "${DOWN_ARGS[@]}"
fi

echo "Local E2E containers stopped."
