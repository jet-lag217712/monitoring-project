#!/usr/bin/env bash
# Root-only helper for PAM-backed appliance users in ${EQUATE_APPLIANCE_GROUP}.
set -euo pipefail

SCRIPT_NAME="$(basename "$0")"
OPERATION="${1:-}"
USERNAME="${2:-}"
PASSWORD="${3:-}"
APPLIANCE_GROUP="${EQUATE_APPLIANCE_GROUP:-equate-appliance}"

usage() {
  cat <<EOF
Usage:
  ${SCRIPT_NAME} create <username> <password>
  ${SCRIPT_NAME} list
  ${SCRIPT_NAME} disable <username>
  ${SCRIPT_NAME} enable <username>
  ${SCRIPT_NAME} reset-password <username> <password>
EOF
}

require_root() {
  if [[ "$(id -u)" -ne 0 ]]; then
    echo "must run as root" >&2
    exit 1
  fi
}

ensure_group() {
  getent group "${APPLIANCE_GROUP}" >/dev/null 2>&1 || groupadd --system "${APPLIANCE_GROUP}"
}

valid_username() {
  local name="$1"
  [[ "${name}" =~ ^[a-z][-a-z0-9_]*$ ]] && ((${#name} <= 32))
}

require_username() {
  if [[ -z "${USERNAME}" ]]; then
    echo "username is required" >&2
    exit 2
  fi
  if ! valid_username "${USERNAME}"; then
    echo "invalid username" >&2
    exit 2
  fi
}

in_appliance_group() {
  id -nG "${USERNAME}" 2>/dev/null | tr ' ' '\n' | grep -qx "${APPLIANCE_GROUP}"
}

create_user() {
  require_username
  if [[ -z "${PASSWORD}" ]]; then
    echo "password is required" >&2
    exit 2
  fi
  ensure_group
  if id "${USERNAME}" &>/dev/null; then
    echo "user already exists" >&2
    exit 1
  fi
  useradd -m -s /bin/bash -G "${APPLIANCE_GROUP}" "${USERNAME}"
  echo "${USERNAME}:${PASSWORD}" | chpasswd
  echo "created ${USERNAME}"
}

list_users() {
  ensure_group
  local members
  members="$(getent group "${APPLIANCE_GROUP}" | awk -F: '{print $4}')"
  if [[ -z "${members}" ]]; then
    echo "No appliance users listed."
    return 0
  fi
  local user
  IFS=',' read -ra users <<< "${members}"
  for user in "${users[@]}"; do
    [[ -z "${user}" ]] && continue
    status="enabled"
    if passwd -S "${user}" 2>/dev/null | awk '{print $2}' | grep -Eq 'L|LK'; then
      status="disabled"
    fi
    printf '%s (%s)\n' "${user}" "${status}"
  done
}

set_lock() {
  require_username
  if ! id "${USERNAME}" &>/dev/null; then
    echo "user not found" >&2
    exit 1
  fi
  if ! in_appliance_group; then
    echo "user is not an appliance account" >&2
    exit 1
  fi
  if [[ "${1}" == "disable" ]]; then
    passwd -l "${USERNAME}" >/dev/null
    echo "disabled ${USERNAME}"
  else
    passwd -u "${USERNAME}" >/dev/null
    echo "enabled ${USERNAME}"
  fi
}

reset_password() {
  require_username
  if [[ -z "${PASSWORD}" ]]; then
    echo "password is required" >&2
    exit 2
  fi
  if ! id "${USERNAME}" &>/dev/null; then
    echo "user not found" >&2
    exit 1
  fi
  if ! in_appliance_group; then
    echo "user is not an appliance account" >&2
    exit 1
  fi
  echo "${USERNAME}:${PASSWORD}" | chpasswd
  echo "reset password for ${USERNAME}"
}

require_root

case "${OPERATION}" in
  create) create_user ;;
  list) list_users ;;
  disable) set_lock disable ;;
  enable) set_lock enable ;;
  reset-password) reset_password ;;
  *)
    usage >&2
    exit 2
    ;;
esac
