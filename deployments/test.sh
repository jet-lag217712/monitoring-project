#!/usr/bin/env bash
# Aggregate deployment validation runner.
#
# Usage:
#   ./deployments/test.sh              # unit + validate + (optional) builds
#   ./deployments/test.sh --quick      # shell + compose validate + go unit only
#   ./deployments/test.sh --with-smoke # also start development stack and smoke it
#   ./deployments/test.sh --with-integration  # require live stack; fail if missing
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=./lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

ROOT="$(repo_root)"
cd "${ROOT}"

QUICK=0
WITH_SMOKE=0
WITH_INTEGRATION=0
WITH_BUILDS=1

for arg in "$@"; do
  case "${arg}" in
    --quick) QUICK=1; WITH_BUILDS=0 ;;
    --with-smoke) WITH_SMOKE=1 ;;
    --with-integration) WITH_INTEGRATION=1 ;;
    --no-builds) WITH_BUILDS=0 ;;
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

section "Shell syntax"
require_cmd bash
SH_FILES=()
while IFS= read -r f; do
  SH_FILES+=("${f}")
done < <(find "${SCRIPT_DIR}" -type f -name '*.sh' | sort)
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

section "Profile validate"
"${SCRIPT_DIR}/end-to-end/validate.sh"
"${SCRIPT_DIR}/development/validate.sh"
"${SCRIPT_DIR}/development/vxrail/validate.sh"
"${SCRIPT_DIR}/production/cloud/validate.sh"
"${SCRIPT_DIR}/production/vxrail/validate.sh"

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

if [[ "${WITH_BUILDS}" -eq 1 ]]; then
  section "Docker image builds (development cloud + collector)"
  require_docker
  ensure_env_file "${SCRIPT_DIR}/development/.env" "${SCRIPT_DIR}/development/.env.example"
  compose_cmd "${SCRIPT_DIR}/development/.env" "${SCRIPT_DIR}/development/docker-compose.yml" build
  docker build -t ogsd-snmp-collector:test "${ROOT}/services/snmp-collector"
fi

if [[ "${WITH_SMOKE}" -eq 1 ]]; then
  section "Development stack smoke"
  "${SCRIPT_DIR}/development/up.sh"
  "${SCRIPT_DIR}/development/smoke.sh"
fi

if [[ "${WITH_INTEGRATION}" -eq 1 ]]; then
  section "Go integration tests (required — no skip)"
  MQTT_DIR="${ROOT}/infrastructure/docker/mqtt-broker"
  require_file "${MQTT_DIR}/certs/ca.crt"
  require_nonempty MQTT_PASSWORD_CHECK "${MQTT_PASSWORD:-${MQTT_INGESTION_PASSWORD:-ingestion}}"

  export MQTT_PASSWORD="${MQTT_PASSWORD:-ingestion}"
  export MQTT_BROKER="${MQTT_BROKER:-tls://127.0.0.1:8883}"
  export MQTT_CA_FILE="${MQTT_CA_FILE:-${MQTT_DIR}/certs/ca.crt}"
  export DATABASE_URL="${DATABASE_URL:-postgres://ogsd_ingestion:ingestion@127.0.0.1:5432/ogsd?sslmode=disable}"
  export DATABASE_ADMIN_URL="${DATABASE_ADMIN_URL:-postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable}"

  # Fail fast if broker/DB unreachable instead of letting tests Skip.
  require_cmd curl
  wait_http "http://127.0.0.1:9091/healthz" "ingestion" 10 || {
    echo "Stack not up. Run ./deployments/development/up.sh first." >&2
    exit 1
  }

  (
    cd "${ROOT}/services/ingestion-service" && go test -tags=integration ./tests/ -count=1 -v
  )
  (
    cd "${ROOT}/services/snmp-collector" && go test -tags=integration ./tests/ -count=1 -v
  )
  (
    cd "${ROOT}/services/backend-api" && go test -tags=integration ./tests/ -count=1 -v
  )
fi

echo
echo "deployments/test.sh: OK"
