#!/usr/bin/env bash
# Verification for a re-imported Equate appliance OVA.
#
# Usage:
#   verify-ova-import.sh --artifact <path-to.ova>   # build-host artifact checks
#   verify-ova-import.sh [--configured]             # guest VM checks after import
set -euo pipefail

MODE=guest
CONFIGURED=0
ARTIFACT=""
VERIFY_APPLIANCE=/usr/local/lib/equate/verify-appliance.sh

failures=0

usage() {
  cat <<'EOF'
Usage:
  verify-ova-import.sh --artifact <file.ova>
  verify-ova-import.sh [--configured]

Options:
  --artifact <path>   Validate OVA/OVF artifact on the build host
  --configured        Also run post-install stack checks (after first-boot TUI)
  -h, --help          Show this help
EOF
}

note() {
  printf 'verify-ova-import: %s\n' "$*"
}

pass() {
  note "OK  $*"
}

fail() {
  note "FAIL $*"
  failures=$((failures + 1))
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --artifact)
      MODE=artifact
      ARTIFACT=${2:?artifact path required}
      shift 2
      ;;
    --configured)
      CONFIGURED=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

check_artifact() {
  local ova=$1
  local base

  [[ -f "${ova}" ]] || { fail "missing OVA: ${ova}"; return; }
  pass "OVA file exists: ${ova}"

  if command -v shasum >/dev/null 2>&1 && [[ -f "${ova}.sha256" ]]; then
    if shasum -a 256 -c "${ova}.sha256" >/dev/null 2>&1; then
      pass "OVA checksum matches ${ova}.sha256"
    else
      fail "OVA checksum mismatch"
    fi
  else
    note "WARN no companion ${ova}.sha256 found; skipping checksum verification"
  fi

  if command -v tar >/dev/null 2>&1; then
    if tar -tf "${ova}" >/dev/null 2>&1; then
      pass "OVA archive is readable"
    else
      fail "OVA archive is not readable"
    fi
  fi

  if command -v ovftool >/dev/null 2>&1; then
    if ovftool --verifyOnly "${ova}" >/dev/null 2>&1; then
      pass "ovftool --verifyOnly succeeded"
    else
      fail "ovftool --verifyOnly failed"
    fi
  else
    note "WARN ovftool not installed; skipping OVF structural verification"
  fi

  base="$(basename "${ova}")"
  case "${base}" in
    *-arm64.ova)
      pass "artifact name indicates arm64 architecture"
      ;;
    *-amd64.ova)
      pass "artifact name indicates amd64 architecture"
      ;;
    *)
      note "WARN artifact name does not encode architecture suffix"
      ;;
  esac
}

guest_listening_ports() {
  ss -H -tln 2>/dev/null | awk '{print $4}' | sed 's/.*://g' | sort -u
}

check_guest_network_surface() {
  local port
  local listeners
  listeners="$(guest_listening_ports || true)"

  for port in ${listeners}; do
    case "${port}" in
      80|443|22)
        ;;
      *)
        fail "unexpected listening TCP port on guest: ${port}"
        ;;
    esac
  done

  if grep -qx 80 <<<"${listeners}" && grep -qx 443 <<<"${listeners}"; then
    pass "guest publishes only expected HTTP/S ports (80/443; SSH ignored for engineering access)"
  else
    note "WARN guest is not yet listening on both 80 and 443 (first boot may be incomplete)"
  fi
}

check_no_default_credentials() {
  if getent group equate >/dev/null 2>&1; then
    local members
    members="$(getent group equate | awk -F: '{print $4}')"
    if [[ -z "${members}" ]]; then
      pass "no pre-created appliance users in group equate"
    elif [[ "${CONFIGURED}" -eq 1 ]]; then
      pass "appliance users exist after configured acceptance"
    else
      fail "appliance users already exist before first-boot TUI completion"
    fi
  else
    pass "equate group not present yet (expected before first-boot TUI)"
  fi

  if [[ -d /var/lib/equate/postgres ]] && find /var/lib/equate/postgres -mindepth 1 -print -quit | grep -q .; then
    if [[ "${CONFIGURED}" -eq 1 ]]; then
      pass "postgres data directory populated after configuration"
    else
      fail "postgres data present on fresh import before first-boot configuration"
    fi
  else
    pass "no populated postgres data directory on fresh import"
  fi
}

check_clone_artifacts_absent() {
  local path
  local issues=0

  for path in /root/equate-staging /root/.ssh/authorized_keys; do
    if [[ -e "${path}" ]]; then
      fail "clone-specific path still present: ${path}"
      issues=1
    fi
  done

  if [[ -e /var/lib/equate/.initialized ]]; then
    if [[ "${CONFIGURED}" -eq 1 ]]; then
      pass "appliance initialized marker present after configuration"
    else
      fail "appliance initialized marker present before first-boot configuration"
      issues=1
    fi
  fi

  if [[ "${issues}" -eq 0 ]]; then
    pass "no staging or clone marker paths found"
  fi
}

check_machine_identity() {
  if [[ -s /etc/machine-id ]]; then
    pass "/etc/machine-id present"
  else
    fail "/etc/machine-id missing or empty"
  fi

  if [[ -d /etc/ssh ]]; then
    local key_count
    key_count="$(find /etc/ssh -maxdepth 1 -name 'ssh_host_*_key' | wc -l | tr -d ' ')"
    if [[ "${key_count}" -gt 0 ]]; then
      pass "SSH host keys present (${key_count})"
    else
      note "WARN no SSH host keys yet (acceptable if SSH is disabled)"
    fi
  fi
}

check_first_boot_service() {
  if systemctl is-enabled equate-first-boot.service >/dev/null 2>&1 \
    || systemctl is-enabled equate-setup.service >/dev/null 2>&1; then
    pass "first-boot systemd unit is enabled"
  else
    note "WARN first-boot systemd unit not found (name may differ on this build)"
  fi
}

check_configured_stack() {
  if [[ -x "${VERIFY_APPLIANCE}" ]]; then
    note "running ${VERIFY_APPLIANCE}"
    "${VERIFY_APPLIANCE}"
    pass "post-install verifier succeeded"
  elif [[ -x ./verify-appliance.sh ]]; then
    note "running ./verify-appliance.sh"
    ./verify-appliance.sh
    pass "post-install verifier succeeded"
  else
    fail "verify-appliance.sh not installed"
  fi
}

main() {
  case "${MODE}" in
    artifact)
      note "artifact verification mode"
      check_artifact "${ARTIFACT}"
      ;;
    guest)
      note "guest verification mode"
      check_guest_network_surface
      check_no_default_credentials
      check_clone_artifacts_absent
      check_machine_identity
      check_first_boot_service
      if [[ "${CONFIGURED}" -eq 1 ]]; then
        check_configured_stack
      fi
      ;;
  esac

  if [[ "${failures}" -gt 0 ]]; then
    note "${failures} check(s) failed"
    exit 1
  fi

  note "all checks passed"
}

main "$@"
