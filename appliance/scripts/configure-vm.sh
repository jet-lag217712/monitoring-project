#!/usr/bin/env bash
# Install a staged appliance release bundle on a Debian 12 VM.
#
# Usage (on the VM as root):
#   bash configure-vm.sh --bundle /tmp/equate-staging/bundle --version 1.0.0
#   bash configure-vm.sh --upgrade --bundle /tmp/equate-staging/bundle --version 1.0.1
#   bash configure-vm.sh --upgrade --bundle /tmp/equate-staging/bundle --version 1.0.1 --canary
#   bash configure-vm.sh --rollback
#   bash configure-vm.sh --bootstrap-only
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

BUNDLE_DIR=""
VERSION=""
BOOTSTRAP_ONLY=0
UPGRADE_MODE=0
ROLLBACK_MODE=0
UPGRADE_CANARY=0
EQUATE_ROOT="/opt/equate"
ETC_DIR="/etc/equate"
VAR_DIR="/var/lib/equate"
RUN_DIR="/run/equate"
APPLIANCE_GROUP="equate-appliance"
# eclipse-mosquitto:2 runs as this UID and must read mounted TLS/password files.
MOSQUITTO_UID=1883

usage() {
  cat <<'EOF'
usage:
  configure-vm.sh --bundle <path> --version <semver>
  configure-vm.sh --upgrade --bundle <path> --version <semver> [--canary]
  configure-vm.sh --rollback
  configure-vm.sh --bootstrap-only [--version <semver>]

Installs the immutable release under /opt/equate/releases/<version> (bundle mode),
upgrades an existing configured appliance in place (--upgrade), rolls back the
last upgrade (--rollback), or regenerates ephemeral secrets for an already-installed
release (bootstrap-only). Fresh install and bootstrap-only render secrets under
/run/equate and start the production Compose stack.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bundle)
      BUNDLE_DIR="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --bootstrap-only)
      BOOTSTRAP_ONLY=1
      shift
      ;;
    --upgrade)
      UPGRADE_MODE=1
      shift
      ;;
    --rollback)
      ROLLBACK_MODE=1
      shift
      ;;
    --canary)
      UPGRADE_CANARY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

source_bootstrap_script() {
  local candidates=(
    "${SCRIPT_DIR}/bootstrap-appliance-rendered.sh"
    "${SCRIPT_DIR}/scripts/bootstrap-appliance-rendered.sh"
  )
  if [[ -n "${BUNDLE_DIR}" ]]; then
    candidates+=("${BUNDLE_DIR}/scripts/bootstrap-appliance-rendered.sh")
  fi
  if [[ -f "${ETC_DIR}/deploy-dir" ]]; then
    local release_dir
    release_dir="$(tr -d '[:space:]' < "${ETC_DIR}/deploy-dir")"
    candidates+=("${release_dir}/scripts/bootstrap-appliance-rendered.sh")
  fi
  local path
  for path in "${candidates[@]}"; do
    if [[ -f "${path}" ]]; then
      # shellcheck source=bootstrap-appliance-rendered.sh
      source "${path}"
      return 0
    fi
  done
  echo "bootstrap-appliance-rendered.sh not found. Copy it next to configure-vm.sh or into the bundle scripts/ directory." >&2
  echo "searched:" >&2
  for path in "${candidates[@]}"; do
    echo "  ${path}" >&2
  done
  exit 1
}

source_bootstrap_script

if [[ "$(id -u)" -ne 0 ]]; then
  echo "configure-vm.sh must run as root" >&2
  exit 1
fi

if [[ "${ROLLBACK_MODE}" -eq 1 ]]; then
  if [[ "${BOOTSTRAP_ONLY}" -eq 1 || "${UPGRADE_MODE}" -eq 1 || -n "${BUNDLE_DIR}" || -n "${VERSION}" ]]; then
    echo "--rollback cannot be combined with install or upgrade flags" >&2
    exit 1
  fi
elif [[ "${BOOTSTRAP_ONLY}" -eq 1 ]]; then
  if [[ -n "${BUNDLE_DIR}" || "${UPGRADE_MODE}" -eq 1 ]]; then
    echo "--bootstrap-only cannot be combined with --bundle or --upgrade" >&2
    exit 1
  fi
elif [[ "${UPGRADE_MODE}" -eq 1 ]]; then
  if [[ -z "${BUNDLE_DIR}" || -z "${VERSION}" ]]; then
    echo "--upgrade requires --bundle and --version" >&2
    usage >&2
    exit 1
  fi
  if [[ ! -f "${BUNDLE_DIR}/release.env" ]]; then
    echo "bundle missing release.env: ${BUNDLE_DIR}" >&2
    exit 1
  fi
else
  if [[ -z "${BUNDLE_DIR}" || -z "${VERSION}" ]]; then
    usage >&2
    exit 1
  fi
  if [[ ! -f "${BUNDLE_DIR}/release.env" ]]; then
    echo "bundle missing release.env: ${BUNDLE_DIR}" >&2
    exit 1
  fi
fi

# Callers may delete the release directory while cd'd into it; use a stable cwd.
cd /

require_cmd() {
  local cmd
  for cmd in "$@"; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      echo "required command not found: ${cmd}" >&2
      exit 1
    fi
  done
}

require_cmd docker openssl install

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose plugin is required" >&2
  exit 1
fi

install_host_packages() {
  if command -v apt-get >/dev/null 2>&1; then
    echo "installing host packages (pamtester, python3)..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq pamtester python3 curl
  else
    require_cmd pamtester python3 curl
  fi
}

resolve_release_dir() {
  local deploy_file="${ETC_DIR}/deploy-dir"
  local current_link="${EQUATE_ROOT}/current"
  if [[ -f "${deploy_file}" ]]; then
    RELEASE_DIR="$(tr -d '[:space:]' < "${deploy_file}")"
  elif [[ -e "${current_link}" ]]; then
    RELEASE_DIR="$(readlink -f "${current_link}")"
  else
    echo "cannot resolve installed release (missing ${deploy_file})" >&2
    exit 1
  fi
  if [[ ! -f "${RELEASE_DIR}/release.env" ]]; then
    echo "installed release missing release.env: ${RELEASE_DIR}" >&2
    exit 1
  fi
  if [[ -z "${VERSION}" ]]; then
    VERSION="$(grep -E '^EQUATE_VERSION=' "${RELEASE_DIR}/release.env" | cut -d= -f2- | tr -d '[:space:]')"
  fi
  if [[ -z "${VERSION}" ]]; then
    echo "could not determine release version" >&2
    exit 1
  fi
}

finalize_release_install() {
  if [[ -x "${RELEASE_DIR}/bin/equate" ]]; then
    install -m 0755 "${RELEASE_DIR}/bin/equate" /usr/local/bin/equate
    install -m 0755 "${RELEASE_DIR}/bin/collector" /usr/local/bin/collector
  fi
  install -d -m 0755 /etc/equate /opt/equate/scripts "${RELEASE_DIR}/scripts"
  printf '%s\n' "${RELEASE_DIR}" > /etc/equate/deploy-dir
  chmod 0644 /etc/equate/deploy-dir
  if [[ -f "${RELEASE_DIR}/scripts/manage-users.sh" ]]; then
    install -m 0755 "${RELEASE_DIR}/scripts/manage-users.sh" /opt/equate/scripts/manage-users.sh
  fi
}

install_host_packages

CURRENT_LINK="${EQUATE_ROOT}/current"

if [[ "${ROLLBACK_MODE}" -eq 1 ]]; then
  rollback_appliance_release
  ln -sfn "${RELEASE_DIR}" "${CURRENT_LINK}"
  finalize_release_install
  cat <<EOF

Appliance rolled back to release ${VERSION}.

  Release: ${RELEASE_DIR}
  Config:  ${ETC_DIR}
  State:   ${VAR_DIR}
  Secrets: ${RUN_DIR}/rendered
EOF
elif [[ "${BOOTSTRAP_ONLY}" -eq 1 ]]; then
  resolve_release_dir
  echo "bootstrapping rendered secrets for installed release ${VERSION} at ${RELEASE_DIR}"
  LOAD_IMAGES=0
  bootstrap_appliance_rendered_and_stack
  finalize_release_install
elif [[ "${UPGRADE_MODE}" -eq 1 ]]; then
  NEW_VERSION="${VERSION}"
  resolve_release_dir
  OLD_RELEASE_DIR="${RELEASE_DIR}"
  OLD_VERSION="$(grep -E '^EQUATE_VERSION=' "${OLD_RELEASE_DIR}/release.env" | cut -d= -f2- | tr -d '[:space:]')"
  VERSION="${NEW_VERSION}"
  RELEASE_DIR="${EQUATE_ROOT}/releases/${VERSION}"

  echo "staging upgrade release ${VERSION} to ${RELEASE_DIR}"
  install -d -m 0755 "${EQUATE_ROOT}/releases"
  rm -rf "${RELEASE_DIR}"
  cp -a "${BUNDLE_DIR}" "${RELEASE_DIR}"

  export UPGRADE_CANARY
  upgrade_appliance_release "${OLD_RELEASE_DIR}" "${OLD_VERSION}"

  ln -sfn "${RELEASE_DIR}" "${CURRENT_LINK}"
  finalize_release_install

  cat <<EOF

Appliance upgraded ${OLD_VERSION} -> ${VERSION}.

  Release: ${RELEASE_DIR}
  Previous release kept at: ${OLD_RELEASE_DIR}
  Config:  ${ETC_DIR}
  State:   ${VAR_DIR}
  Secrets: ${RUN_DIR}/rendered

Rollback if needed:
  sudo equate upgrade --rollback
EOF
else
  RELEASE_DIR="${EQUATE_ROOT}/releases/${VERSION}"

  echo "installing release ${VERSION} to ${RELEASE_DIR}"
  install -d -m 0755 "${EQUATE_ROOT}/releases"
  rm -rf "${RELEASE_DIR}"
  cp -a "${BUNDLE_DIR}" "${RELEASE_DIR}"
  ln -sfn "${RELEASE_DIR}" "${CURRENT_LINK}"

  LOAD_IMAGES=1
  bootstrap_appliance_rendered_and_stack
  finalize_release_install

  cat <<EOF

Appliance release ${VERSION} installed.

  Release: ${RELEASE_DIR}
  Config:  ${ETC_DIR}
  State:   ${VAR_DIR}
  Secrets: ${RUN_DIR}/rendered (ephemeral; back up /etc/equate and ${VAR_DIR})

Next steps:
  1. Run first-boot setup: sudo equate configure
  2. Create appliance users and configure sites in the setup wizard.
  3. Open https://<vm-ip>/ and sign in with a local appliance user.
  4. Replace self-signed TLS under ${RUN_DIR}/rendered/certificates when ready.
EOF
fi
