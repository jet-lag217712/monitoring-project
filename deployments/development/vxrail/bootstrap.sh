#!/usr/bin/env bash
# Build and start the development vxrail multi-site collectors on the OrbStack VM.
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

if [[ "${1:-}" == "--reconfigure" ]]; then
  rm -f .setup-complete
  shift
fi

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
  echo "Created .env from .env.example — set MQTT_BROKER and SNMP_COMMUNITY before continuing." >&2
  exit 1
fi

if [[ ! -f sites/manifest.yaml ]] || [[ ! -f docker-compose.sites.generated.yml ]]; then
  echo "Missing sites/manifest.yaml or docker-compose.sites.generated.yml — run ./bootstrap.sh --reconfigure" >&2
  exit 1
fi

# shellcheck disable=SC1091
set -a
source ./.env
set +a

mkdir -p certs
if [[ ! -f certs/ca.crt ]]; then
  echo "Missing certs/ca.crt (sync from Mac Mosquitto CA)." >&2
  exit 1
fi

BUILD_CONTEXT="../../../services/snmp-collector"
if [[ -d ./src/services/snmp-collector ]]; then
  BUILD_CONTEXT="./src/services/snmp-collector"
fi

export MQTT_BROKER MQTT_PASSWORD
export SNMP_COMMUNITY SNMP_DISCOVERY_COMMUNITY

mapfile -t SITE_IDS < <(awk '/^    site_id: / {gsub(/"/, "", $2); print $2}' sites/manifest.yaml)
mapfile -t SERVICES < <(awk '/^    service_name: / {print $2}' sites/manifest.yaml)
mapfile -t ADMIN_PORTS < <(awk '/    admin_port: / {print $2}' sites/manifest.yaml)

if [[ "${#SITE_IDS[@]}" -eq 0 ]] || [[ "${#SERVICES[@]}" -eq 0 ]]; then
  echo "sites/manifest.yaml or docker-compose.sites.generated.yml is invalid — run ./bootstrap.sh --reconfigure" >&2
  exit 1
fi

for site_id in "${SITE_IDS[@]}"; do
  mkdir -p "sites/${site_id}/run" "sites/${site_id}/managed" "sites/${site_id}/configs"
done

COMPOSE_OVERRIDE="$(mktemp)"
{
  echo "services:"
  for service in "${SERVICES[@]}"; do
    echo "  ${service}:"
    echo "    build: ${BUILD_CONTEXT}"
  done
} >"${COMPOSE_OVERRIDE}"

COMPOSE_FILES=(-f docker-compose.yml -f docker-compose.sites.generated.yml -f "${COMPOSE_OVERRIDE}")

docker compose "${COMPOSE_FILES[@]}" build --build-arg BUILDKIT_INLINE_CACHE=1 "${SERVICES[@]}" 2>/dev/null || true
docker compose "${COMPOSE_FILES[@]}" up -d --build --remove-orphans "${SERVICES[@]}"

PROJECT_NAME="${COMPOSE_PROJECT_NAME:-ogsd-development-vxrail}"
for site_id in "${SITE_IDS[@]}"; do
  docker run --rm -v "${PROJECT_NAME}_collector-state-${site_id}:/var/lib/snmp-collector" busybox:1.36 \
    chown -R 65532:65532 /var/lib/snmp-collector >/dev/null 2>&1 || true
  docker run --rm -v "${SCRIPT_DIR}/sites/${site_id}/run:/run/snmp-collector" busybox:1.36 \
    rm -f /run/snmp-collector/control.sock >/dev/null 2>&1 || true
  docker run --rm -v "${SCRIPT_DIR}/sites/${site_id}/run:/run/snmp-collector" busybox:1.36 \
    chown 65532:65532 /run/snmp-collector >/dev/null 2>&1 || true
  docker run --rm -v "${SCRIPT_DIR}/sites/${site_id}/managed:/var/lib/snmp-collector/managed" busybox:1.36 \
    chown -R 65532:65532 /var/lib/snmp-collector/managed >/dev/null 2>&1 || true
done

docker compose "${COMPOSE_FILES[@]}" up -d "${SERVICES[@]}"
rm -f "${COMPOSE_OVERRIDE}"

echo "bootstrap: ${#SERVICES[@]} site collector(s) started."
for i in "${!SITE_IDS[@]}"; do
  port="${ADMIN_PORTS[$i]:-$((9090 + i))}"
  echo "  ${SITE_IDS[$i]} (${SERVICES[$i]}): curl -fsS http://127.0.0.1:${port}/healthz"
done
