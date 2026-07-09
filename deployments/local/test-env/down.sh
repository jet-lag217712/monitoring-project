#!/usr/bin/env bash
# Stop the local test-env containers.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
COMPOSE="${ROOT}/deployments/local/test-env/docker-compose.yaml"

cd "${ROOT}"

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not reachable. Start Docker Desktop, then retry." >&2
  exit 1
fi

docker compose -f "${COMPOSE}" down
echo "Local test-env containers stopped."
