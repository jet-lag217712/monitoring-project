#!/usr/bin/env bash
# Appliance-focused repository checks.
#
# Usage:
#   ./deployments/test.sh         # shell syntax, appliance compose, go unit, frontend build
#   ./deployments/test.sh --quick # skip frontend build
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

QUICK=0
for arg in "$@"; do
  case "${arg}" in
    --quick) QUICK=1 ;;
    *)
      echo "Unknown argument: ${arg}" >&2
      exit 2
      ;;
  esac
done

section() {
  echo
  echo "=== $* ==="
}

require_cmd() {
  local cmd
  for cmd in "$@"; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      echo "Required command not found: ${cmd}" >&2
      exit 1
    fi
  done
}

section "Shell syntax"
require_cmd bash
SH_FILES=()
while IFS= read -r f; do
  SH_FILES+=("${f}")
done < <(find "${ROOT}/appliance" "${ROOT}/deployments" "${ROOT}/infrastructure" "${ROOT}/remote-server" \
  -type f -name '*.sh' | sort)
for f in "${SH_FILES[@]}"; do
  bash -n "${f}"
done
echo "bash -n: ${#SH_FILES[@]} scripts OK"

if command -v shellcheck >/dev/null 2>&1; then
  shellcheck -x "${SH_FILES[@]}"
  echo "shellcheck: OK"
else
  echo "shellcheck not installed; skipped"
fi

section "Appliance compose validate"
require_cmd docker
if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose plugin is required (docker compose)." >&2
  exit 1
fi
COMPOSE="${ROOT}/deployments/production/appliance/docker-compose.yml"
export POSTGRES_USER=ogsd
export POSTGRES_PASSWORD=placeholder
export POSTGRES_DB=ogsd
export OGSD_INGESTION_USER=ogsd_ingestion
export OGSD_INGESTION_PASSWORD=placeholder
export OGSD_API_USER=ogsd_api
export OGSD_API_PASSWORD=placeholder
export MQTT_BROKER=tls://mosquitto:8883
export MQTT_PASSWORD=placeholder
export MQTT_INGESTION_PASSWORD=placeholder
export SNMP_COMMUNITY=placeholder
export SNMP_DISCOVERY_COMMUNITY=placeholder
docker compose -f "${COMPOSE}" config >/dev/null
echo "production/appliance compose: OK"

section "Go unit tests"
(
  cd "${ROOT}/services/snmp-collector" && go test ./... -count=1
)
(
  cd "${ROOT}/services/ingestion-service" && go test ./... -count=1
)
(
  cd "${ROOT}/services/backend-api" && go test ./... -count=1
)

if [[ "${QUICK}" -eq 0 ]]; then
  section "Frontend build"
  require_cmd npm
  (
    cd "${ROOT}/frontend"
    if [[ ! -d node_modules ]]; then
      npm ci
    fi
    npm run build
  )
fi

echo
echo "deployments/test.sh: OK"
