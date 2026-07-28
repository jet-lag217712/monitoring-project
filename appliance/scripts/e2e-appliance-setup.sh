#!/usr/bin/env bash
# Non-interactive E2E appliance setup for lab validation.
set -euo pipefail

RELEASE_DIR="${RELEASE_DIR:-/opt/equate/current}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-EquateAdmin123!}"
OPERATOR_USER="${OPERATOR_USER:-equateops}"
OPERATOR_PASS="${OPERATOR_PASS:-EquateOps123!}"
SNMP_COMMUNITY="${SNMP_COMMUNITY:-EquateMonitor}"
COMPOSE_ENV="/run/equate/rendered/compose.env"
MANAGE_USERS="${RELEASE_DIR}/scripts/manage-users.sh"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

echo "creating appliance users..."
if ! id "${ADMIN_USER}" &>/dev/null; then
  "${MANAGE_USERS}" create "${ADMIN_USER}" "${ADMIN_PASS}"
else
  echo "user ${ADMIN_USER} already exists"
fi
if ! id "${OPERATOR_USER}" &>/dev/null; then
  "${MANAGE_USERS}" create "${OPERATOR_USER}" "${OPERATOR_PASS}"
else
  echo "user ${OPERATOR_USER} already exists"
fi

echo "setting SNMP community..."
if [[ -f "${COMPOSE_ENV}" ]]; then
  sed -i "s/^SNMP_COMMUNITY=.*/SNMP_COMMUNITY=${SNMP_COMMUNITY}/" "${COMPOSE_ENV}"
  sed -i "s/^SNMP_DISCOVERY_COMMUNITY=.*/SNMP_DISCOVERY_COMMUNITY=${SNMP_COMMUNITY}/" "${COMPOSE_ENV}"
fi
if [[ -f /run/equate/rendered/collector.env ]]; then
  sed -i "s/^SNMP_COMMUNITY=.*/SNMP_COMMUNITY=${SNMP_COMMUNITY}/" /run/equate/rendered/collector.env
  sed -i "s/^SNMP_DISCOVERY_COMMUNITY=.*/SNMP_DISCOVERY_COMMUNITY=${SNMP_COMMUNITY}/" /run/equate/rendered/collector.env
fi

echo "generating site artifacts..."
if [[ -x "${RELEASE_DIR}/bin/equate-setup-gen" ]]; then
  "${RELEASE_DIR}/bin/equate-setup-gen" "${RELEASE_DIR}"
else
  echo "equate-setup-gen binary missing" >&2
  exit 1
fi

echo "fixing collector volume permissions..."
for vol in equate-appliance_collector-state-site-a-mdf equate-appliance_collector-state-site-b-mdf; do
  docker run --rm -v "${vol}:/var/lib/snmp-collector" busybox:1.36 chown -R 65532:65532 /var/lib/snmp-collector 2>/dev/null || true
done

echo "fixing site artifact permissions..."
for site_dir in "${RELEASE_DIR}"/sites/*/; do
  [[ -d "${site_dir}" ]] || continue
  chmod 755 "${site_dir}" "${site_dir}configs" "${site_dir}run" 2>/dev/null || true
  chown -R 65532:65532 "${site_dir}managed" 2>/dev/null || true
  chown 65532:65532 "${site_dir}configs/collector.yaml" 2>/dev/null || true
  chmod 640 "${site_dir}configs/collector.yaml" 2>/dev/null || true
done

echo "starting collector services..."
(
  cd "${RELEASE_DIR}"
  docker compose --env-file "${COMPOSE_ENV}" -f docker-compose.yml -f docker-compose.sites.generated.yml up -d --remove-orphans
)

sleep 10

run_discovery() {
  local site_id="$1"
  local admin_port="$2"
  local sock="${RELEASE_DIR}/sites/${site_id}/run/control.sock"
  echo "discovery for ${site_id} on port ${admin_port}..."
  for _ in $(seq 1 30); do
    if [[ -S "${sock}" ]]; then
      break
    fi
    sleep 2
  done
  if [[ ! -S "${sock}" ]]; then
    echo "WARN control socket missing for ${site_id}" >&2
    return 1
  fi
  echo '{"id":"d1","method":"discovery.scan.start","params":{"async":true}}' | \
    collector rpc -socket "${sock}" -timeout 10m
  for _ in $(seq 1 120); do
    progress="$(echo '{"id":"d2","method":"discovery.scan.progress","params":null}' | collector rpc -socket "${sock}" -timeout 30s)"
    if echo "${progress}" | grep -q '"complete":true'; then
      break
    fi
    sleep 5
  done
  candidates="$(echo '{"id":"d3","method":"discovery.candidates.list","params":null}' | collector rpc -socket "${sock}" -timeout 30s)"
  echo "${candidates}" | head -c 500
  echo
}

run_discovery site-a-mdf 19090 || true
run_discovery site-b-mdf 19091 || true

echo "E2E setup complete."
docker compose --env-file "${COMPOSE_ENV}" -f "${RELEASE_DIR}/docker-compose.yml" -f "${RELEASE_DIR}/docker-compose.sites.generated.yml" ps
