#!/usr/bin/env bash
# Package an offline appliance bundle as a signed .eqa artifact.
#
# Usage:
#   ./appliance/scripts/package-eqa.sh --arch amd64 --version 1.0.3
#   EQUATE_UPDATE_SIGNING_KEY=/secure/equate-updates.priv \
#     ./appliance/scripts/package-eqa.sh --arch amd64 --version 1.0.3
#
# Requires: dist/appliance-<arch>-<version>/ from build-release.sh
# Optional: EQUATE_UPDATE_SIGNING_KEY (hex private key). Without it, .eqa + .sha256
# are produced but signing is skipped (CI must sign before publish).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ARCH=""
VERSION=""
OUT_DIR="${ROOT}/dist"
SIGNING_KEY="${EQUATE_UPDATE_SIGNING_KEY:-}"

usage() {
  cat <<'EOF'
usage: package-eqa.sh --arch <arm64|amd64> --version <semver> [--out-dir dist]

Creates:
  dist/Equate-<version>-<arch>.eqa
  dist/Equate-<version>-<arch>.eqa.sha256
  dist/Equate-<version>-<arch>.eqa.sig   (when EQUATE_UPDATE_SIGNING_KEY is set)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch)
      ARCH="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --out-dir)
      OUT_DIR="${2:-}"
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

case "${ARCH}" in
  arm64|amd64) ;;
  *)
    echo "--arch must be arm64 or amd64" >&2
    exit 1
    ;;
esac
if [[ -z "${VERSION}" ]]; then
  echo "--version is required" >&2
  exit 1
fi

BUNDLE_DIR="${ROOT}/dist/appliance-${ARCH}-${VERSION}"
if [[ ! -d "${BUNDLE_DIR}" ]]; then
  echo "bundle not found: ${BUNDLE_DIR}" >&2
  echo "run: make appliance-bundle ARCH=${ARCH} VERSION=${VERSION}" >&2
  exit 1
fi
if [[ ! -f "${BUNDLE_DIR}/release.env" ]]; then
  echo "bundle missing release.env: ${BUNDLE_DIR}" >&2
  exit 1
fi

mkdir -p "${OUT_DIR}"
EQA="${OUT_DIR}/Equate-${VERSION}-${ARCH}.eqa"
echo "packaging ${BUNDLE_DIR} -> ${EQA}"
# Tar the bundle contents (not the parent directory name) so extract lands release.env at staging root.
tar -C "${BUNDLE_DIR}" -czf "${EQA}" .

checksum_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    shasum -a 256 "$@"
  fi
}

(
  cd "${OUT_DIR}"
  checksum_file "Equate-${VERSION}-${ARCH}.eqa"
) >"${EQA}.sha256"
echo "wrote ${EQA}.sha256"

if [[ -n "${SIGNING_KEY}" ]]; then
  if [[ ! -f "${SIGNING_KEY}" ]]; then
    echo "EQUATE_UPDATE_SIGNING_KEY not found: ${SIGNING_KEY}" >&2
    exit 1
  fi
  echo "signing ${EQA}"
  (
    cd "${ROOT}/services/snmp-collector"
    go run ./cmd/equate-sign --key "${SIGNING_KEY}" --file "${EQA}"
  )
  echo "wrote ${EQA}.sig"
else
  echo "warning: EQUATE_UPDATE_SIGNING_KEY unset; produced unsigned .eqa (sha256 only)" >&2
fi

echo "eqa package ready: ${EQA}"
