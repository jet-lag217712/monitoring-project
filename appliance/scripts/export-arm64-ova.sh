#!/usr/bin/env bash
# Export a powered-off ARM64 Fusion VM to OVA and document Fusion import steps.
#
# Usage:
#   ./appliance/scripts/export-arm64-ova.sh --vmx "/path/to/VM.vmx" --version 0.2.0
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VMX=""
VERSION="0.2.0"
OUT_DIR="${ROOT}/dist"

usage() {
  cat <<'EOF'
usage: export-arm64-ova.sh --vmx <path-to.vmx> [--version <semver>]

Exports a powered-off VMware Fusion ARM64 VM to dist/Equate-Appliance-<version>-arm64.ova
using the Fusion-bundled ovftool. After import on Apple Silicon, set guest OS to
"Debian 13 64-bit Arm" (arm-debian13-64) before the first power-on if Fusion
labels it "Other".
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --vmx) VMX="${2:-}"; shift 2 ;;
    --version) VERSION="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "${VMX}" || ! -f "${VMX}" ]]; then
  echo "--vmx must point to an existing .vmx file" >&2
  exit 1
fi

OVFTOOL="/Applications/VMware Fusion.app/Contents/Library/VMware OVF Tool/ovftool"
if [[ ! -x "${OVFTOOL}" ]]; then
  echo "ovftool not found at ${OVFTOOL}" >&2
  exit 1
fi

OVA="${OUT_DIR}/Equate-Appliance-${VERSION}-arm64.ova"
mkdir -p "${OUT_DIR}"

echo "exporting ${VMX} -> ${OVA}"
"${OVFTOOL}" --acceptAllEulas "${VMX}" "${OVA}"

if command -v shasum >/dev/null 2>&1; then
  shasum -a 256 "${OVA}" >"${OVA}.sha256"
elif command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${OVA}" >"${OVA}.sha256"
fi

cat >"${OUT_DIR}/Equate-Appliance-${VERSION}-arm64.import-notes.txt" <<EOF
Equate Appliance ${VERSION} ARM64 OVA — VMware Fusion import notes

1. Import the OVA in Fusion (File → Import).
2. BEFORE first boot, open VM Settings → General and set Guest OS to
   "Debian 13 64-bit Arm" (or edit the .vmx: guestOS = "arm-debian13-64").
   ovftool may label the guest as "Other", which Fusion treats as x86.
3. Power on. The console launches the first-boot setup wizard automatically.
4. Open https://<vm-ip>/ and sign in with a local appliance user.

Checksum: see ${OVA}.sha256
EOF

ls -lh "${OVA}"
echo "import notes: ${OUT_DIR}/Equate-Appliance-${VERSION}-arm64.import-notes.txt"
