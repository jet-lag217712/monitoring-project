#!/usr/bin/env bash
# Validate in-place appliance upgrade helpers on a configured VM.
#
# Usage (on the VM as root, after a release is configured):
#   OLD_VERSION=0.2.3 NEW_VERSION=0.2.4 BUNDLE=/tmp/equate-staging/bundle \
#     bash appliance/scripts/e2e-appliance-upgrade.sh
#
# Optional:
#   CANARY=1            pass --canary to the upgrade
#   SKIP_ROLLBACK=1     skip rollback verification
set -euo pipefail

OLD_VERSION="${OLD_VERSION:-}"
NEW_VERSION="${NEW_VERSION:-}"
BUNDLE="${BUNDLE:-/tmp/equate-staging/bundle}"
CANARY="${CANARY:-0}"
SKIP_ROLLBACK="${SKIP_ROLLBACK:-0}"
DEPLOY_DIR="${DEPLOY_DIR:-/etc/equate/deploy-dir}"
COMPOSE_ENV="/run/equate/rendered/compose.env"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

if [[ -z "${OLD_VERSION}" || -z "${NEW_VERSION}" ]]; then
  echo "OLD_VERSION and NEW_VERSION are required" >&2
  exit 1
fi
if [[ ! -d "${BUNDLE}" ]]; then
  echo "bundle not found: ${BUNDLE}" >&2
  exit 1
fi

CURRENT_RELEASE="$(tr -d '[:space:]' < "${DEPLOY_DIR}")"
SITE_MANIFEST="${CURRENT_RELEASE}/sites/manifest.yaml"
if [[ ! -f "${SITE_MANIFEST}" ]]; then
  echo "configured sites manifest missing: ${SITE_MANIFEST}" >&2
  exit 1
fi

backup_dir="$(mktemp -d)"
trap 'rm -rf "${backup_dir}"' EXIT

echo "capturing pre-upgrade site artifacts from ${CURRENT_RELEASE}..."
cp -a "${CURRENT_RELEASE}/sites" "${backup_dir}/sites"
if [[ -f "${CURRENT_RELEASE}/.env" ]]; then
  cp -a "${CURRENT_RELEASE}/.env" "${backup_dir}/.env"
fi

CONFIGURE_SCRIPT="${BUNDLE}/../configure-vm.sh"
if [[ ! -f "${CONFIGURE_SCRIPT}" ]]; then
  CONFIGURE_SCRIPT="/tmp/equate-staging/configure-vm.sh"
fi
if [[ ! -f "${CONFIGURE_SCRIPT}" ]]; then
  echo "configure-vm.sh not found next to bundle" >&2
  exit 1
fi

upgrade_args=(--upgrade --bundle "${BUNDLE}" --version "${NEW_VERSION}")
if [[ "${CANARY}" == "1" ]]; then
  upgrade_args+=(--canary)
fi

echo "running upgrade ${OLD_VERSION} -> ${NEW_VERSION}..."
bash "${CONFIGURE_SCRIPT}" "${upgrade_args[@]}"

NEW_RELEASE="/opt/equate/releases/${NEW_VERSION}"
if [[ "$(tr -d '[:space:]' < "${DEPLOY_DIR}")" != "${NEW_RELEASE}" ]]; then
  echo "deploy-dir did not switch to ${NEW_RELEASE}" >&2
  exit 1
fi

if ! diff -qr "${backup_dir}/sites" "${NEW_RELEASE}/sites" >/dev/null; then
  echo "site artifacts changed during upgrade" >&2
  diff -qr "${backup_dir}/sites" "${NEW_RELEASE}/sites" || true
  exit 1
fi
echo "OK site artifacts preserved"

if [[ ! -f "${COMPOSE_ENV}" ]]; then
  echo "missing compose env after upgrade: ${COMPOSE_ENV}" >&2
  exit 1
fi
if ! grep -q "^POSTGRES_PASSWORD=" "${COMPOSE_ENV}"; then
  echo "compose env missing postgres secrets after upgrade" >&2
  exit 1
fi
echo "OK compose secrets preserved"

if command -v equate >/dev/null 2>&1; then
  equate status || true
fi

if [[ "${SKIP_ROLLBACK}" == "1" ]]; then
  echo "upgrade validation complete (rollback skipped)"
  exit 0
fi

echo "rolling back to ${OLD_VERSION}..."
if command -v equate >/dev/null 2>&1; then
  equate upgrade --rollback --yes
else
  bash "${CONFIGURE_SCRIPT}" --rollback
fi

if [[ "$(tr -d '[:space:]' < "${DEPLOY_DIR}")" != "/opt/equate/releases/${OLD_VERSION}" ]]; then
  echo "deploy-dir did not roll back to ${OLD_VERSION}" >&2
  exit 1
fi
echo "OK rollback restored previous release"

echo "e2e appliance upgrade validation complete"
