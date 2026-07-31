#!/usr/bin/env bash
# Post-setup hooks for the on-prem appliance (site permissions, topology sync, compose reconcile).
# Also used after in-place upgrades — mirrors the tail of equate configure --sites.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR="${EQUATE_DEPLOY_DIR:-$(cd "${SCRIPT_DIR}/.." && pwd)}"
COMPOSE_ENV="${EQUATE_COMPOSE_ENV:-/run/equate/rendered/compose.env}"

if [[ -f "${SCRIPT_DIR}/manifest-utils.sh" ]]; then
  # shellcheck source=manifest-utils.sh
  source "${SCRIPT_DIR}/manifest-utils.sh"
fi
if [[ -f "${SCRIPT_DIR}/debug-agent-log.sh" ]]; then
  # shellcheck source=debug-agent-log.sh
  source "${SCRIPT_DIR}/debug-agent-log.sh"
fi

cd "${DEPLOY_DIR}"

if [[ -f "${COMPOSE_ENV}" ]]; then
  # shellcheck disable=SC1091
  set -a
  source "${COMPOSE_ENV}"
  set +a
fi
load_release_dotenv "${DEPLOY_DIR}"

BUSYBOX_IMAGE="busybox:1.36"
if [[ -f "${DEPLOY_DIR}/release.env" ]]; then
  BUSYBOX_IMAGE="$(grep -E '^EQUATE_BUSYBOX_IMAGE=' "${DEPLOY_DIR}/release.env" | cut -d= -f2- | tr -d '[:space:]')"
  BUSYBOX_IMAGE="${BUSYBOX_IMAGE:-busybox:1.36}"
fi

reconcile_site_permissions() {
  local manifest="${DEPLOY_DIR}/sites/manifest.yaml"
  if [[ ! -f "${manifest}" ]]; then
    return 0
  fi
  mapfile -t SITE_IDS < <(read_manifest_site_ids "${manifest}")
  if [[ "${#SITE_IDS[@]}" -eq 0 ]]; then
    if declare -F debug_agent_log >/dev/null 2>&1; then
      debug_agent_log "H8" "post-configure.sh:reconcile_site_permissions" "no site ids parsed from manifest" "{\"manifest\":\"${manifest}\"}"
    fi
    echo "post-configure: warning: no site_id entries found in ${manifest}" >&2
    return 0
  fi
  local project_name="${COMPOSE_PROJECT_NAME:-equate-appliance}"
  local site_id vol_slug
  echo "reconciling site collector permissions for ${#SITE_IDS[@]} site(s)..."
  for site_id in "${SITE_IDS[@]}"; do
    vol_slug="$(printf '%s' "${site_id}" | tr '[:upper:]' '[:lower:]')"
    docker run --rm -v "${project_name}_collector-state-${vol_slug}:/var/lib/snmp-collector" "${BUSYBOX_IMAGE}" \
      chown -R 65532:65532 /var/lib/snmp-collector >/dev/null 2>&1 || true
    docker run --rm -v "${DEPLOY_DIR}/sites/${site_id}/run:/run/snmp-collector" "${BUSYBOX_IMAGE}" \
      rm -f /run/snmp-collector/control.sock >/dev/null 2>&1 || true
    docker run --rm -v "${DEPLOY_DIR}/sites/${site_id}/run:/run/snmp-collector" "${BUSYBOX_IMAGE}" \
      chown 65532:65532 /run/snmp-collector >/dev/null 2>&1 || true
    docker run --rm -v "${DEPLOY_DIR}/sites/${site_id}/managed:/var/lib/snmp-collector/managed" "${BUSYBOX_IMAGE}" \
      chown -R 65532:65532 /var/lib/snmp-collector/managed >/dev/null 2>&1 || true
  done
}

restart_site_collectors() {
  local manifest="${DEPLOY_DIR}/sites/manifest.yaml"
  if [[ ! -f "${manifest}" ]]; then
    return 0
  fi
  mapfile -t COLLECTOR_SERVICES < <(read_manifest_service_names "${manifest}")
  if [[ "${#COLLECTOR_SERVICES[@]}" -eq 0 ]]; then
    echo "post-configure: warning: no service_name entries found in ${manifest}" >&2
    return 0
  fi
  echo "restarting ${#COLLECTOR_SERVICES[@]} site collector(s)..."
  compose up -d --force-recreate --remove-orphans "${COLLECTOR_SERVICES[@]}"
}

compose() {
  docker compose \
    --env-file "${COMPOSE_ENV}" \
    -f docker-compose.yml \
    -f docker-compose.sites.generated.yml \
    "$@"
}

if declare -F debug_agent_log >/dev/null 2>&1; then
  db_before="$(query_db_topology_json "${DEPLOY_DIR}" "${COMPOSE_ENV}")"
  debug_agent_log "H6" "post-configure.sh" "post-configure start" "{\"deploy_dir\":\"${DEPLOY_DIR}\",\"db_before\":${db_before}}"
fi

reconcile_site_permissions

if [[ -x "${SCRIPT_DIR}/sync-site-topology.sh" && -f sites/manifest.yaml ]]; then
  EQUATE_DEPLOY_DIR="${DEPLOY_DIR}" \
    EQUATE_COMPOSE_ENV="${COMPOSE_ENV}" \
    bash "${SCRIPT_DIR}/sync-site-topology.sh"
fi

restart_site_collectors

echo "reconciling appliance stack..."
compose up -d --remove-orphans

if [[ -f sites/manifest.yaml ]]; then
  date -u +"%Y-%m-%dT%H:%M:%S.%NZ" > "${DEPLOY_DIR}/.setup-complete"
  chmod 0600 "${DEPLOY_DIR}/.setup-complete"
fi

if declare -F debug_agent_log >/dev/null 2>&1; then
  db_after="$(query_db_topology_json "${DEPLOY_DIR}" "${COMPOSE_ENV}")"
  manifest_sites="$(read_manifest_site_ids sites/manifest.yaml | paste -sd, - || true)"
  debug_agent_log "H6" "post-configure.sh" "post-configure complete" "{\"db_after\":${db_after},\"site_ids\":\"${manifest_sites}\"}"
fi
