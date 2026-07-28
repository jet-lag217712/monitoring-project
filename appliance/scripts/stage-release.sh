#!/usr/bin/env bash
# SCP helper for staging an appliance release bundle onto a target VM.
#
# Usage:
#   ./appliance/scripts/stage-release.sh --host appliance.local --arch arm64 --version 1.0.0
#   ./appliance/scripts/stage-release.sh --host 192.168.1.50 --user admin --arch amd64 --version 1.0.0
#
# The bundle must already exist under dist/appliance-<arch>-<version>/.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HOST=""
USER_NAME="${USER}"
ARCH=""
VERSION=""
REMOTE_DIR="/tmp/equate-staging"

usage() {
  cat <<'EOF'
usage: stage-release.sh --host <hostname> [--user <ssh-user>] --arch <arm64|amd64> --version <semver> [--remote-dir <path>]

Transfers dist/appliance-<arch>-<version>/ plus configure-vm.sh, bootstrap-appliance-rendered.sh,
and prepare-ova.sh to the target VM.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      HOST="${2:-}"
      shift 2
      ;;
    --user)
      USER_NAME="${2:-}"
      shift 2
      ;;
    --arch)
      ARCH="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --remote-dir)
      REMOTE_DIR="${2:-}"
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

if [[ -z "${HOST}" || -z "${ARCH}" || -z "${VERSION}" ]]; then
  usage >&2
  exit 1
fi

BUNDLE_DIR="${ROOT}/dist/appliance-${ARCH}-${VERSION}"
if [[ ! -d "${BUNDLE_DIR}" ]]; then
  echo "bundle not found: ${BUNDLE_DIR}" >&2
  echo "run: make appliance-bundle ARCH=${ARCH} VERSION=${VERSION}" >&2
  exit 1
fi

REMOTE="${USER_NAME}@${HOST}"
SCRIPTS_SRC="${ROOT}/appliance/scripts"
echo "staging ${BUNDLE_DIR} to ${REMOTE}:${REMOTE_DIR}/"

ssh "${REMOTE}" "mkdir -p '${REMOTE_DIR}/bundle'"
scp -r "${BUNDLE_DIR}/." "${REMOTE}:${REMOTE_DIR}/bundle/"
scp \
  "${SCRIPTS_SRC}/configure-vm.sh" \
  "${SCRIPTS_SRC}/bootstrap-appliance-rendered.sh" \
  "${SCRIPTS_SRC}/prepare-ova.sh" \
  "${REMOTE}:${REMOTE_DIR}/"

cat <<EOF

Staged release ${VERSION} (${ARCH}) on ${HOST}.

On the VM (as root):
  sudo bash ${REMOTE_DIR}/configure-vm.sh --bundle ${REMOTE_DIR}/bundle --version ${VERSION}
EOF
