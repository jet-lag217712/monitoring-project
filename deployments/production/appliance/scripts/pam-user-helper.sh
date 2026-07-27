#!/usr/bin/env bash
# Stub PAM user helper for appliance setup wizard development.
# Production releases install a root-owned allowlisted helper at this path.
set -euo pipefail

ACTION="${1:?action required}"
STUB_DB="${EQUATE_PAM_STUB_DB:-/var/lib/equate/pam-users.stub}"

usage() {
  echo "usage: pam-user-helper.sh <create|list|disable|reset> [flags]" >&2
  exit 2
}

read_flag() {
  local name="$1"
  local value="${2:-}"
  if [[ -z "${value}" ]]; then
    echo "missing value for ${name}" >&2
    exit 2
  fi
  printf '%s' "${value}"
}

username=""
password=""
shift || usage

while [[ $# -gt 0 ]]; do
  case "$1" in
    --username)
      username="$(read_flag "$1" "${2:-}")"
      shift 2
      ;;
    --password)
      password="$(read_flag "$1" "${2:-}")"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

mkdir -p "$(dirname "${STUB_DB}")"
touch "${STUB_DB}"

case "${ACTION}" in
  create)
    [[ -n "${username}" && -n "${password}" ]] || { echo "create requires username and password" >&2; exit 2; }
    if grep -q "^${username}:" "${STUB_DB}"; then
      echo "user already exists: ${username}" >&2
      exit 1
    fi
    printf '%s:enabled\n' "${username}" >>"${STUB_DB}"
    echo "created ${username}"
    ;;
  list)
    if [[ ! -s "${STUB_DB}" ]]; then
      echo "no users"
      exit 0
    fi
    awk -F: '{printf "%s (%s)\n", $1, $2}' "${STUB_DB}"
    ;;
  disable)
    [[ -n "${username}" ]] || { echo "disable requires --username" >&2; exit 2; }
    if ! grep -q "^${username}:" "${STUB_DB}"; then
      echo "user not found: ${username}" >&2
      exit 1
    fi
    sed -i.bak "s/^${username}:.*/${username}:disabled/" "${STUB_DB}" && rm -f "${STUB_DB}.bak"
    echo "disabled ${username}"
    ;;
  reset)
    [[ -n "${username}" && -n "${password}" ]] || { echo "reset requires username and password" >&2; exit 2; }
    if ! grep -q "^${username}:" "${STUB_DB}"; then
      echo "user not found: ${username}" >&2
      exit 1
    fi
    sed -i.bak "s/^${username}:.*/${username}:enabled/" "${STUB_DB}" && rm -f "${STUB_DB}.bak"
    echo "reset password for ${username}"
    ;;
  *)
    usage
    ;;
esac
