#!/usr/bin/env bash
# Package a VMDK + generated OVF into an OVA tarball with checksum sidecar.
#
# Usage:
#   package-ova.sh --arch amd64 --version 0.2.0 --vmdk /path/disk.vmdk --out-dir dist/
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ARCH=""
VERSION=""
VMDK=""
OUT_DIR="${ROOT}/dist"
DISK_GB=64

usage() {
  cat <<'EOF'
usage: package-ova.sh --arch <amd64|arm64> --version <semver> --vmdk <path> [--out-dir dist]
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch) ARCH="${2:-}"; shift 2 ;;
    --version) VERSION="${2:-}"; shift 2 ;;
    --vmdk) VMDK="${2:-}"; shift 2 ;;
    --out-dir) OUT_DIR="${2:-}"; shift 2 ;;
    --disk-gb) DISK_GB="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "${ARCH}" || -z "${VERSION}" || -z "${VMDK}" ]]; then
  usage >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BASE="Equate-Appliance-${VERSION}-${ARCH}"
WORK="$(mktemp -d)"
trap 'rm -rf "${WORK}"' EXIT

DISK_NAME="${BASE}-disk1.vmdk"
OVF_NAME="${BASE}.ovf"
OVA="${OUT_DIR}/${BASE}.ova"

mkdir -p "${OUT_DIR}"

VMDK_ABS="$(readlink -f "${VMDK}")"
VMDK_DIR="$(dirname "${VMDK_ABS}")"
VMDK_BASE="$(basename "${VMDK_ABS}")"

if [[ "${VMDK_BASE}" != "${DISK_NAME}" ]]; then
  echo "vmdk basename must be ${DISK_NAME}, got: ${VMDK_BASE}" >&2
  exit 1
fi

"${SCRIPT_DIR}/generate-ovf.sh" \
  --arch "${ARCH}" \
  --version "${VERSION}" \
  --name "${BASE}" \
  --vmdk "${VMDK_ABS}" \
  --out "${WORK}/${OVF_NAME}" \
  --disk-gb "${DISK_GB}"

tar --format=posix -cf "${OVA}" \
  -C "${WORK}" "${OVF_NAME}" \
  -C "${VMDK_DIR}" "${DISK_NAME}"

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${OVA}" >"${OVA}.sha256"
else
  shasum -a 256 "${OVA}" >"${OVA}.sha256"
fi

ls -lh "${OVA}"
echo "checksum: ${OVA}.sha256"
