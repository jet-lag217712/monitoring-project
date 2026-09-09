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

sync_db_role_passwords_from_release() {
  local release_dir="$1"
  local script="${release_dir}/scripts/sync-db-role-passwords.sh"
  local compose_env="${RUN_DIR}/rendered/compose.env"
  if [[ ! -f "${script}" || ! -f "${compose_env}" ]]; then
    return 0
  fi
  echo "syncing database role passwords from release ${release_dir}..."
  EQUATE_RELEASE_DIR="${release_dir}" COMPOSE_ENV="${compose_env}" bash "${script}"
  local compose_files=(-f docker-compose.yml)
  if [[ -f "${release_dir}/docker-compose.sites.generated.yml" ]]; then
    compose_files+=(-f docker-compose.sites.generated.yml)
  fi
  (
    cd "${release_dir}"
    docker compose \
      --env-file "${compose_env}" \
      "${compose_files[@]}" \
      restart backend-api ingestion
  )
}

sync_site_topology_from_release() {
  local release_dir="$1"
  local script="${release_dir}/scripts/sync-site-topology.sh"
  local manifest="${release_dir}/sites/manifest.yaml"
  if [[ ! -f "${script}" || ! -f "${manifest}" ]]; then
    return 0
  fi
  echo "syncing site topology from ${manifest}..."
  local sync_rc=0
  EQUATE_DEPLOY_DIR="${release_dir}" \
    EQUATE_COMPOSE_ENV="${RUN_DIR}/rendered/compose.env" \
    bash "${script}" || sync_rc=$?
  return "${sync_rc}"
}

run_post_configure_from_release() {
  local release_dir="$1"
  local script="${release_dir}/scripts/post-configure.sh"
  if [[ ! -f "${release_dir}/sites/manifest.yaml" ]]; then
    return 0
  fi
  if [[ ! -x "${script}" ]]; then
    echo "post-configure.sh missing; falling back to topology sync only" >&2
    sync_site_topology_from_release "${release_dir}"
    return $?
  fi
  echo "running post-configure handoff from ${release_dir}..."
  EQUATE_DEPLOY_DIR="${release_dir}" \
    EQUATE_COMPOSE_ENV="${RUN_DIR}/rendered/compose.env" \
    bash "${script}"
}

source_bootstrap_script() {
  local candidates=()
  # Prefer the staged bundle during install/upgrade so new bootstrap logic applies
  # before /opt/equate/current is repointed at the target release.
  if [[ -n "${BUNDLE_DIR}" ]]; then
    candidates+=(
      "${BUNDLE_DIR}/scripts/bootstrap-appliance-rendered.sh"
      "${BUNDLE_DIR}/bootstrap-appliance-rendered.sh"
    )
  fi
  candidates+=(
    "${SCRIPT_DIR}/bootstrap-appliance-rendered.sh"
    "${SCRIPT_DIR}/scripts/bootstrap-appliance-rendered.sh"
  )
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
    echo "installing host packages (pamtester, python3, python3-yaml)..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq pamtester python3 python3-yaml curl
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
  if [[ -f "${RELEASE_DIR}/scripts/sync-db-role-passwords.sh" ]]; then
    install -m 0755 "${RELEASE_DIR}/scripts/sync-db-role-passwords.sh" /opt/equate/scripts/sync-db-role-passwords.sh
  fi
  install_first_boot_console
  install_appliance_sudoers
}

install_first_boot_console() {
  local lib_dir="/usr/local/lib/equate"
  install -d -m 0755 "${lib_dir}"
  for script in first-boot-needed.sh first-boot-console.sh verify-appliance.sh verify-ova-import.sh; do
    if [[ -f "${RELEASE_DIR}/scripts/${script}" ]]; then
      install -m 0755 "${RELEASE_DIR}/scripts/${script}" "${lib_dir}/${script}"
    elif [[ -f "${SCRIPT_DIR}/${script}" ]]; then
      install -m 0755 "${SCRIPT_DIR}/${script}" "${lib_dir}/${script}"
    fi
  done
  if [[ -f "${RELEASE_DIR}/scripts/equate-first-boot.service" ]]; then
    install -m 0644 "${RELEASE_DIR}/scripts/equate-first-boot.service" /etc/systemd/system/equate-first-boot.service
  elif [[ -f "${SCRIPT_DIR}/equate-first-boot.service" ]]; then
    install -m 0644 "${SCRIPT_DIR}/equate-first-boot.service" /etc/systemd/system/equate-first-boot.service
  fi
  if [[ -f "${RELEASE_DIR}/scripts/getty-tty1-override.conf" ]]; then
    install -d -m 0755 /etc/systemd/system/getty@tty1.service.d
    install -m 0644 "${RELEASE_DIR}/scripts/getty-tty1-override.conf" \
      /etc/systemd/system/getty@tty1.service.d/equate-first-boot.conf
  elif [[ -f "${SCRIPT_DIR}/getty-tty1-override.conf" ]]; then
    install -d -m 0755 /etc/systemd/system/getty@tty1.service.d
    install -m 0644 "${SCRIPT_DIR}/getty-tty1-override.conf" \
      /etc/systemd/system/getty@tty1.service.d/equate-first-boot.conf
  fi
  systemctl daemon-reload
  systemctl enable equate-first-boot.service 2>/dev/null || true
}

install_appliance_sudoers() {
  local src=""
  if [[ -f "${RELEASE_DIR}/scripts/equate-appliance.sudoers" ]]; then
    src="${RELEASE_DIR}/scripts/equate-appliance.sudoers"
  elif [[ -f "${SCRIPT_DIR}/equate-appliance.sudoers" ]]; then
    src="${SCRIPT_DIR}/equate-appliance.sudoers"
  fi
  if [[ -z "${src}" ]]; then
    return 0
  fi
  install -m 0440 "${src}" /etc/sudoers.d/equate-appliance
  visudo -c -f /etc/sudoers.d/equate-appliance
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
  sync_db_role_passwords_from_release "${RELEASE_DIR}"
  run_post_configure_from_release "${RELEASE_DIR}"

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
