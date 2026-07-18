#!/usr/bin/env bash
# Build and start the production vxrail collector on the on-site VM.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "${SCRIPT_DIR}"

resolve_collector_module() {
  local candidates=(
    "${SCRIPT_DIR}/src/services/snmp-collector"
    "${SCRIPT_DIR}/../../../services/snmp-collector"
  )
  for dir in "${candidates[@]}"; do
    if [[ -f "${dir}/go.mod" ]]; then
      printf '%s\n' "$(cd "${dir}" && pwd)"
      return 0
    fi
  done
  return 1
}

if [[ ! -f .setup-complete ]]; then
  echo "First boot: launching Equate collector setup wizard…" >&2
  SETUP_CMD=()
  if command -v collector >/dev/null 2>&1; then
    SETUP_CMD=(collector setup -dir "${SCRIPT_DIR}" -theme auto)
  elif COLLECTOR_MOD="$(resolve_collector_module)"; then
    SETUP_CMD=(go run -C "${COLLECTOR_MOD}" ./cmd/collector setup -dir "${SCRIPT_DIR}" -theme auto)
  else
    echo "collector binary or snmp-collector source not found; install collector or sync repo." >&2
    exit 1
  fi
  exec "${SETUP_CMD[@]}"
fi

if [[ ! -f .env ]]; then
  cp -n .env.example .env
  echo "Created .env from .env.example — set MQTT_BROKER and SNMP_COMMUNITY_* before continuing." >&2
  exit 1
fi

# shellcheck disable=SC1091
set -a
source ./.env
set +a

mkdir -p run certs managed
if [[ ! -f certs/ca.crt ]]; then
  echo "Missing certs/ca.crt (production Mosquitto CA)." >&2
  exit 1
fi

export MQTT_BROKER MQTT_PASSWORD
export SNMP_COMMUNITY SNMP_DISCOVERY_COMMUNITY

docker compose build --build-arg BUILDKIT_INLINE_CACHE=1 snmp-collector 2>/dev/null || true
docker compose up -d --build

docker run --rm -v ogsd-production-vxrail_collector-state:/var/lib/snmp-collector busybox:1.36 \
  chown -R 65532:65532 /var/lib/snmp-collector >/dev/null 2>&1 || true
docker run --rm -v "${SCRIPT_DIR}/run:/run/snmp-collector" busybox:1.36 \
  chown -R 65532:65532 /run/snmp-collector >/dev/null 2>&1 || true
docker run --rm -v "${SCRIPT_DIR}/managed:/var/lib/snmp-collector/managed" busybox:1.36 \
  chown -R 65532:65532 /var/lib/snmp-collector/managed >/dev/null 2>&1 || true
docker compose up -d snmp-collector

echo "bootstrap: collector started. Check: curl -fsS http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz"
