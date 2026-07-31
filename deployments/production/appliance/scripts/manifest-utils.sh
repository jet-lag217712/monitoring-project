#!/usr/bin/env bash
# Shared helpers for reading generated sites/manifest.yaml on the appliance.
set -euo pipefail

# Go's yaml.Marshal indents list items with 4 spaces and fields with 6; do not
# anchor patterns to a fixed column count.
read_manifest_site_ids() {
  local manifest="$1"
  awk '/site_id:/ {
    val = $2
    gsub(/"/, "", val)
    if (val != "") print val
  }' "${manifest}"
}

read_manifest_service_names() {
  local manifest="$1"
  awk '/service_name:/ {print $2}' "${manifest}"
}

manifest_has_sites() {
  local manifest="$1"
  read_manifest_site_ids "${manifest}" | grep -q .
}

load_release_dotenv() {
  local release_dir="$1"
  local dotenv="${release_dir}/.env"
  if [[ ! -f "${dotenv}" ]]; then
    return 0
  fi
  # shellcheck disable=SC1091
  set -a
  source "${dotenv}"
  set +a
}
