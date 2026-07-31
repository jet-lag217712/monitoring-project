#!/usr/bin/env bash
# First-boot and reconfigure entrypoint for the on-prem appliance deployment.
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
  RECONFIGURE_MODE="full"
  shift
fi

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      RECONFIGURE_MODE="${2:-full}"
      shift 2
      ;;
    *)
      break
      ;;
  esac
done

RECONFIGURE_MODE="${RECONFIGURE_MODE:-}"

ensure_appliance_rendered_secrets() {
  local compose_env="/run/equate/rendered/compose.env"
  if [[ -f "${compose_env}" ]]; then
    return 0
  fi
  local configure_script="${SCRIPT_DIR}/scripts/configure-vm.sh"
  if [[ ! -f "${configure_script}" ]]; then
    echo "missing ${compose_env} and ${configure_script}" >&2
    exit 1
  fi
  echo "First boot: generating per-installation secrets under /run/equate/rendered…" >&2
  bash "${configure_script}" --bootstrap-only
}

sync_appliance_db_role_passwords() {
  local script="${SCRIPT_DIR}/scripts/sync-db-role-passwords.sh"
  local compose_env="/run/equate/rendered/compose.env"
  if [[ ! -f "${compose_env}" || ! -f "${script}" ]]; then
    return 0
  fi
  echo "bootstrapper: syncing database role passwords…" >&2
  EQUATE_RELEASE_DIR="${SCRIPT_DIR}" COMPOSE_ENV="${compose_env}" bash "${script}"
}

compose_env_args() {
  if [[ -f /run/equate/rendered/compose.env ]]; then
    echo --env-file /run/equate/rendered/compose.env
  fi
}

if [[ ! -f .setup-complete ]]; then
  ensure_appliance_rendered_secrets
  echo "First boot: launching Equate appliance setup wizard…" >&2
  SETUP_CMD=()
  SETUP_EXTRA=()
  if [[ -n "${RECONFIGURE_MODE}" ]]; then
    export EQUATE_SETUP_RECONFIGURE="${RECONFIGURE_MODE}"
    SETUP_EXTRA=(-reconfigure "${RECONFIGURE_MODE}")
  fi
  if command -v collector >/dev/null 2>&1; then
    SETUP_CMD=(collector setup -dir "${SCRIPT_DIR}" -theme auto -profile appliance "${SETUP_EXTRA[@]}")
  elif COLLECTOR_MOD="$(resolve_collector_module)"; then
    SETUP_CMD=(go run -C "${COLLECTOR_MOD}" ./cmd/collector setup -dir "${SCRIPT_DIR}" -theme auto -profile appliance "${SETUP_EXTRA[@]}")
  else
    echo "collector binary or snmp-collector source not found; install collector or sync repo." >&2
    exit 1
  fi
  exec "${SETUP_CMD[@]}"
fi

if [[ ! -f .env ]]; then
  if [[ -f .env.example ]]; then
    cp -n .env.example .env
  fi
  echo "Missing or incomplete .env — run equate configure or ./bootstrapper.sh --reconfigure" >&2
  exit 1
fi

if [[ ! -f sites/manifest.yaml ]] || [[ ! -f docker-compose.sites.generated.yml ]]; then
  echo "Missing sites/manifest.yaml or docker-compose.sites.generated.yml — run equate configure" >&2
  exit 1
fi

# shellcheck disable=SC1091
set -a
source ./.env
set +a

mkdir -p certs
if [[ ! -f certs/ca.crt ]]; then
  echo "Missing certs/ca.crt (appliance Mosquitto CA)." >&2
  exit 1
fi

BUILD_CONTEXT="../../../services/snmp-collector"
if [[ -d ./src/services/snmp-collector ]]; then
  BUILD_CONTEXT="./src/services/snmp-collector"
fi

export MQTT_BROKER MQTT_PASSWORD
export SNMP_COMMUNITY SNMP_DISCOVERY_COMMUNITY

mapfile -t SITE_IDS < <(awk '/site_id:/ {gsub(/"/, "", $2); if ($2 != "") print $2}' sites/manifest.yaml)
mapfile -t SERVICES < <(awk '/service_name:/ {print $2}' sites/manifest.yaml)
mapfile -t ADMIN_PORTS < <(awk '/admin_port: / {print $2}' sites/manifest.yaml)

if [[ "${#SITE_IDS[@]}" -eq 0 ]] || [[ "${#SERVICES[@]}" -eq 0 ]]; then
  echo "sites/manifest.yaml is invalid — run equate configure" >&2
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
COMPOSE_ENV_ARGS=()
while IFS= read -r arg; do
  [[ -n "${arg}" ]] && COMPOSE_ENV_ARGS+=("${arg}")
done < <(compose_env_args)

docker compose "${COMPOSE_ENV_ARGS[@]}" -f docker-compose.yml -f docker-compose.sites.generated.yml up -d postgres mosquitto 2>/dev/null || true
sync_appliance_db_role_passwords

docker compose "${COMPOSE_ENV_ARGS[@]}" "${COMPOSE_FILES[@]}" build --build-arg BUILDKIT_INLINE_CACHE=1 "${SERVICES[@]}" 2>/dev/null || true
docker compose "${COMPOSE_ENV_ARGS[@]}" "${COMPOSE_FILES[@]}" up -d --build --remove-orphans

PROJECT_NAME="${COMPOSE_PROJECT_NAME:-equate-appliance}"
for site_id in "${SITE_IDS[@]}"; do
  vol_slug=$(printf '%s' "$site_id" | tr '[:upper:]' '[:lower:]')
  docker run --rm -v "${PROJECT_NAME}_collector-state-${vol_slug}:/var/lib/snmp-collector" busybox:1.36 \
    chown -R 65532:65532 /var/lib/snmp-collector >/dev/null 2>&1 || true
  docker run --rm -v "${SCRIPT_DIR}/sites/${site_id}/run:/run/snmp-collector" busybox:1.36 \
    rm -f /run/snmp-collector/control.sock >/dev/null 2>&1 || true
  docker run --rm -v "${SCRIPT_DIR}/sites/${site_id}/run:/run/snmp-collector" busybox:1.36 \
    chown 65532:65532 /run/snmp-collector >/dev/null 2>&1 || true
  docker run --rm -v "${SCRIPT_DIR}/sites/${site_id}/managed:/var/lib/snmp-collector/managed" busybox:1.36 \
    chown -R 65532:65532 /var/lib/snmp-collector/managed >/dev/null 2>&1 || true
done

docker compose "${COMPOSE_ENV_ARGS[@]}" "${COMPOSE_FILES[@]}" up -d "${SERVICES[@]}"
rm -f "${COMPOSE_OVERRIDE}"

if [[ -x "${SCRIPT_DIR}/scripts/sync-site-topology.sh" && -f sites/manifest.yaml ]]; then
  echo "bootstrapper: syncing site topology into PostgreSQL…" >&2
  EQUATE_DEPLOY_DIR="${SCRIPT_DIR}" \
    EQUATE_COMPOSE_ENV="/run/equate/rendered/compose.env" \
    bash "${SCRIPT_DIR}/scripts/sync-site-topology.sh" || {
      echo "bootstrapper: site topology sync failed" >&2
      exit 1
    }
fi

echo "bootstrapper: ${#SERVICES[@]} site collector(s) started."
for i in "${!SITE_IDS[@]}"; do
  port="${ADMIN_PORTS[$i]:-$((9090 + i))}"
  echo "  ${SITE_IDS[$i]} (${SERVICES[$i]}): curl -fsS http://127.0.0.1:${port}/healthz"
done
