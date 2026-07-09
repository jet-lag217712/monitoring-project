#!/usr/bin/env bash
# Stop Phase 2 local containers.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
COMPOSE="${ROOT}/deployments/local/phase2/docker-compose.yaml"

cd "${ROOT}"

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not reachable. Start Docker Desktop, then retry." >&2
  exit 1
fi

docker compose -f "${COMPOSE}" down
echo "Phase 2 containers stopped."
