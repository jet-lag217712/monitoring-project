#!/usr/bin/env bash
# OVA-safe finalization: remove staging material and fail closed on leftover secrets.
#
# Usage (on the VM as root, before export):
#   bash prepare-ova.sh [--staging-dir /tmp/equate-staging]
set -euo pipefail

STAGING_DIR="/tmp/equate-staging"
FAIL=0

usage() {
  cat <<'EOF'
usage: prepare-ova.sh [--staging-dir <path>]

Removes build/staging artifacts, scrubs clone-specific credentials, and verifies
that no sensitive material remains before manual OVA export.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --staging-dir)
      STAGING_DIR="${2:-}"
      shift 2
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

if [[ "$(id -u)" -ne 0 ]]; then
  echo "prepare-ova.sh must run as root" >&2
  exit 1
fi

report() {
  echo "prepare-ova: $*" >&2
  FAIL=1
}

echo "removing staging directories..."
rm -rf "${STAGING_DIR}" /root/equate-staging /var/tmp/equate-staging

echo "scrubbing rendered secrets and logs..."
rm -rf /run/equate/rendered
find /var/lib/equate -type f \( -name '*.log' -o -name '*.journal' \) -delete 2>/dev/null || true
: > /root/.bash_history
: > /root/.lesshst
history -c 2>/dev/null || true

echo "removing SSH host keys and machine identity..."
rm -f /etc/ssh/ssh_host_*
truncate -s 0 /etc/machine-id
rm -f /var/lib/dbus/machine-id

echo "removing build SSH credentials..."
rm -f /root/.ssh/authorized_keys /root/.ssh/id_rsa /root/.ssh/id_ed25519 /root/.ssh/id_ecdsa
rmdir /root/.ssh 2>/dev/null || true
for build_home in /home/debian /home/packer; do
  if [[ -d "${build_home}/.ssh" ]]; then
    rm -rf "${build_home}/.ssh"
  fi
done

echo "removing CI build accounts..."
for build_user in debian packer; do
  if id "${build_user}" >/dev/null 2>&1; then
    # #region agent log
    # Hypothesis A: CI SSH keeps the account "in use"; plain userdel fails and was swallowed.
    _procs="$(ps -u "${build_user}" -o pid=,comm= 2>/dev/null | tr '\n' ';' | head -c 500 || true)"
    echo "==> debug prepare-ova userdel pre (hypothesis A): user=${build_user}; procs=${_procs}"
    # #endregion
    # -f is required when finalize runs under an active SSH login as the build user
    # (AMD64 CI: ssh debian@guest 'sudo …/prepare-ova.sh'). Without -f, userdel exits
    # with "currently used by process" and the account remains in the golden image.
    set +e
    _out="$(userdel -f -r "${build_user}" 2>&1)"
    _rc=$?
    if [[ "${_rc}" -ne 0 ]]; then
      _out2="$(userdel -f "${build_user}" 2>&1)"
      _rc2=$?
      _out="${_out}; fallback: ${_out2}"
      _rc="${_rc2}"
    fi
    set -e
    # #region agent log
    _exists="$(id "${build_user}" >/dev/null 2>&1 && echo yes || echo no)"
    echo "==> debug prepare-ova userdel post (hypothesis A): user=${build_user}; rc=${_rc}; exists=${_exists}; out=${_out}"
    # #endregion
    if id "${build_user}" >/dev/null 2>&1; then
      report "failed to remove CI build account ${build_user}: ${_out}"
    fi
  fi
done

echo "resetting hostname to equate-appliance..."
hostnamectl set-hostname equate-appliance 2>/dev/null || echo "equate-appliance" > /etc/hostname
if [[ -f /etc/hosts ]]; then
  sed -i 's/equate-appliance-build/equate-appliance/g' /etc/hosts 2>/dev/null || true
fi

DEPLOY_DIR=""
if [[ -f /etc/equate/deploy-dir ]]; then
  DEPLOY_DIR="$(tr -d '[:space:]' < /etc/equate/deploy-dir)"
fi
if [[ -n "${DEPLOY_DIR}" && -d "${DEPLOY_DIR}" ]]; then
  echo "scrubbing deploy setup state under ${DEPLOY_DIR}..."
  rm -f "${DEPLOY_DIR}/.setup-complete" "${DEPLOY_DIR}/.env"
  rm -f "${DEPLOY_DIR}/docker-compose.sites.generated.yml"
  rm -rf "${DEPLOY_DIR}/sites"
fi

echo "removing DHCP leases..."
rm -f /var/lib/dhcp/dhcpd.leases /var/lib/NetworkManager/*.lease 2>/dev/null || true

SENSITIVE_PATHS=(
  /run/equate/rendered
  /tmp/equate-staging
  /root/.ssh/authorized_keys
  /root/.ssh/id_rsa
  /root/.ssh/id_ed25519
)

for path in "${SENSITIVE_PATHS[@]}"; do
  if [[ -e "${path}" ]]; then
    report "sensitive path still present: ${path}"
  fi
done

if grep -R -E -q 'packer_password|CHANGE_ME|POSTGRES_PASSWORD=' /tmp /root /home 2>/dev/null; then
  report "possible plaintext secret remains under /tmp, /root, or /home"
fi

if id debian >/dev/null 2>&1; then
  report "CI build account debian still present"
fi
if [[ "$(hostname)" == "equate-appliance-build" ]]; then
  report "hostname still set to CI build name"
fi
if [[ -n "${DEPLOY_DIR}" && -f "${DEPLOY_DIR}/.setup-complete" ]]; then
  report "deploy dir still marked setup-complete"
fi

if [[ "${FAIL}" -ne 0 ]]; then
  echo "prepare-ova failed: sensitive material remains" >&2
  exit 1
fi

echo "prepare-ova complete: VM is ready for power-off and manual OVA export."
