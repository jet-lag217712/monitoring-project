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

if [[ "${FAIL}" -ne 0 ]]; then
  echo "prepare-ova failed: sensitive material remains" >&2
  exit 1
fi

echo "prepare-ova complete: VM is ready for power-off and manual OVA export."
