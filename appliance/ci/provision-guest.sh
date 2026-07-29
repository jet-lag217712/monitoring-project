#!/usr/bin/env bash
# Guest-side finalization for CI OVA builds (run as root on the build VM).
#
# Installs the staged bundle, scrubs customer-specific state, prepares for export,
# and powers off the VM.
set -euo pipefail

VERSION="${1:?usage: provision-guest.sh <version>}"
STAGING="${STAGING_DIR:-/tmp/equate-staging}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "provision-guest.sh must run as root" >&2
  exit 1
fi

wait_for_apt() {
  local waited=0
  local max_wait=300
  while pgrep -x apt-get >/dev/null 2>&1 \
    || pgrep -x apt >/dev/null 2>&1 \
    || pgrep -x dpkg >/dev/null 2>&1 \
    || pgrep -x unattended-upgrade >/dev/null 2>&1; do
    if (( waited >= max_wait )); then
      echo "timed out waiting for apt/dpkg after ${max_wait}s" >&2
      pgrep -a apt-get apt dpkg unattended-upgrade 2>/dev/null || true
      return 1
    fi
    sleep 2
    waited=$((waited + 2))
  done
}

ensure_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    return 0
  fi
  echo "installing Docker Engine with compose plugin..."
  # #region agent log
  echo "==> debug apt state before docker install (hypothesis A/C): cloud-init=$(cloud-init status 2>&1 || true); apt-get=$(pgrep -a apt-get 2>/dev/null || echo none); dpkg=$(pgrep -a dpkg 2>/dev/null || echo none)"
  # #endregion
  wait_for_apt
  export DEBIAN_FRONTEND=noninteractive
  apt-get -o DPkg::Lock::Timeout=300 update -qq
  apt-get -o DPkg::Lock::Timeout=300 install -y -qq curl ca-certificates
  curl -fsSL https://get.docker.com | sh
}

ensure_docker_compose

if [[ ! -f "${STAGING}/configure-vm.sh" ]]; then
  echo "missing ${STAGING}/configure-vm.sh" >&2
  exit 1
fi

bash "${STAGING}/configure-vm.sh" --bundle "${STAGING}/bundle" --version "${VERSION}"

RELEASE_DIR="$(cat /etc/equate/deploy-dir)"
COMPOSE_ENV="/run/equate/rendered/compose.env"
echo "scrubbing customer state before golden image export (release=${RELEASE_DIR})..."
if [[ -f "${COMPOSE_ENV}" ]]; then
  (
    cd "${RELEASE_DIR}"
    docker compose \
      --env-file "${COMPOSE_ENV}" \
      -f docker-compose.yml \
      -f docker-compose.sites.generated.yml \
      down --timeout 60 2>/dev/null || true
  )
fi
rm -rf /var/lib/equate/postgres
rm -rf /etc/equate/sites
find /var/lib/equate -maxdepth 1 -type d -name 'collector-state-*' -exec rm -rf {} + 2>/dev/null || true

bash "${STAGING}/prepare-ova.sh" --staging-dir "${STAGING}"

sync
echo "guest provision complete; powering off..."
poweroff
