#!/usr/bin/env bash
# Build an AMD64 Equate Appliance OVA on Linux CI (QEMU + Debian cloud image).
#
# Prerequisites: docker, go, qemu-system-x86_64, qemu-img, genisoimage, ssh, python3
#
# Usage:
#   ./appliance/scripts/build-ova-amd64-ci.sh --version 0.2.0
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION=""
DISK_GB=64
MEMORY_MB=4096
VCPUS=2
SSH_PORT=2222
DEBIAN_CLOUD_URL="${DEBIAN_CLOUD_URL:-https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2}"
WORK="${WORK_DIR:-${ROOT}/build/appliance/ova-amd64-ci}"
OUT_DIR="${ROOT}/dist"

usage() {
  cat <<'EOF'
usage: build-ova-amd64-ci.sh --version <semver> [--work-dir path] [--out-dir dist]

Builds the amd64 release bundle, provisions a Debian 12 cloud VM under QEMU,
installs the appliance, exports streamOptimized VMDK, and packages an OVA.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="${2:-}"; shift 2 ;;
    --work-dir) WORK="${2:-}"; shift 2 ;;
    --out-dir) OUT_DIR="${2:-}"; shift 2 ;;
    --disk-gb) DISK_GB="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "${VERSION}" ]]; then
  usage >&2
  exit 1
fi

require_cmd() {
  local cmd
  for cmd in "$@"; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      echo "required command not found: ${cmd}" >&2
      exit 1
    fi
  done
}

require_cmd docker go qemu-system-x86_64 qemu-img genisoimage ssh scp python3 ssh-keygen

mkdir -p "${WORK}" "${OUT_DIR}"
BUNDLE_DIR="${ROOT}/dist/appliance-amd64-${VERSION}"

echo "==> building amd64 release bundle (${VERSION})"
"${SCRIPT_DIR}/build-release.sh" --arch amd64 --version "${VERSION}"

if [[ ! -d "${BUNDLE_DIR}" ]]; then
  echo "expected bundle directory missing: ${BUNDLE_DIR}" >&2
  exit 1
fi

CLOUD_IMG="${WORK}/debian-12-generic-amd64.qcow2"
DISK="${WORK}/appliance-disk.qcow2"
CI_SEED="${WORK}/ci-seed.iso"
SSH_KEY="${WORK}/ci-ssh-key"
SSH_PUB="${SSH_KEY}.pub"
QEMU_LOG="${WORK}/qemu.log"

if [[ ! -f "${CLOUD_IMG}" ]]; then
  echo "==> downloading Debian 12 cloud image"
  curl -fsSL "${DEBIAN_CLOUD_URL}" -o "${CLOUD_IMG}.partial"
  mv "${CLOUD_IMG}.partial" "${CLOUD_IMG}"
fi

cp -f "${CLOUD_IMG}" "${DISK}"
qemu-img resize "${DISK}" "${DISK_GB}G"

if [[ ! -f "${SSH_KEY}" ]]; then
  ssh-keygen -t ed25519 -N "" -f "${SSH_KEY}" -q
fi

CI_SEED_DIR="${WORK}/ci-seed"
rm -rf "${CI_SEED_DIR}"
mkdir -p "${CI_SEED_DIR}"

cat >"${CI_SEED_DIR}/meta-data" <<EOF
instance-id: equate-appliance-amd64-build
local-hostname: equate-appliance-build
EOF

PUB_KEY="$(cat "${SSH_PUB}")"
cat >"${CI_SEED_DIR}/user-data" <<EOF
#cloud-config
package_update: true
package_upgrade: false
packages:
  - open-vm-tools
  - qemu-guest-agent
users:
  - default
  - name: debian
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - ${PUB_KEY}
runcmd:
  - [ systemctl, enable, --now, qemu-guest-agent ]
EOF

genisoimage -output "${CI_SEED}" -volid cidata -joliet -rock "${CI_SEED_DIR}/user-data" "${CI_SEED_DIR}/meta-data"

SSH_OPTS=(
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o ConnectTimeout=10
  -o ServerAliveInterval=30
  -o ServerAliveCountMax=240
  -o "Port=${SSH_PORT}"
  -i "${SSH_KEY}"
)

QEMU_ACCEL=()
QEMU_CPU=(-cpu qemu64)
if [[ -r /dev/kvm ]] && [[ -w /dev/kvm ]]; then
  QEMU_ACCEL=(-enable-kvm)
  QEMU_CPU=(-cpu max)
fi

echo "==> starting QEMU (port ${SSH_PORT} -> guest :22)"
qemu-system-x86_64 \
  "${QEMU_ACCEL[@]}" \
  -m "${MEMORY_MB}" \
  -smp "${VCPUS}" \
  "${QEMU_CPU[@]}" \
  -drive "file=${DISK},if=virtio,format=qcow2" \
  -drive "file=${CI_SEED},if=virtio,media=cdrom" \
  -netdev "user,id=net,hostfwd=tcp::${SSH_PORT}-:22" \
  -device virtio-net-pci,netdev=net \
  -nographic \
  >"${QEMU_LOG}" 2>&1 &
QEMU_PID=$!

cleanup_qemu() {
  if kill -0 "${QEMU_PID}" 2>/dev/null; then
    kill "${QEMU_PID}" 2>/dev/null || true
    wait "${QEMU_PID}" 2>/dev/null || true
  fi
}
trap cleanup_qemu EXIT

echo "==> waiting for SSH on localhost:${SSH_PORT}"
READY=0
for _ in $(seq 1 120); do
  if ssh "${SSH_OPTS[@]}" debian@127.0.0.1 true 2>/dev/null; then
    READY=1
    break
  fi
  if ! kill -0 "${QEMU_PID}" 2>/dev/null; then
    echo "QEMU exited before SSH became ready; log:" >&2
    tail -n 80 "${QEMU_LOG}" >&2 || true
    exit 1
  fi
  sleep 5
done

if [[ "${READY}" -ne 1 ]]; then
  echo "timed out waiting for guest SSH" >&2
  tail -n 80 "${QEMU_LOG}" >&2 || true
  exit 1
fi

STAGING="/tmp/equate-staging"
echo "==> staging bundle and installer scripts on guest"
ssh "${SSH_OPTS[@]}" debian@127.0.0.1 "sudo rm -rf ${STAGING} /tmp/equate-ci-scripts && sudo mkdir -p ${STAGING}/bundle /tmp/equate-ci-scripts"
tar -C "${BUNDLE_DIR}" -cf - . | ssh "${SSH_OPTS[@]}" debian@127.0.0.1 "sudo tar -xf - -C ${STAGING}/bundle"
tar -cf - \
  -C "${SCRIPT_DIR}" configure-vm.sh prepare-ova.sh \
  -C "${ROOT}/appliance/ci" provision-guest.sh | \
  ssh "${SSH_OPTS[@]}" debian@127.0.0.1 "tar -xf - -C /tmp/equate-ci-scripts"
ssh "${SSH_OPTS[@]}" debian@127.0.0.1 "sudo install -m 0755 /tmp/equate-ci-scripts/configure-vm.sh /tmp/equate-ci-scripts/prepare-ova.sh /tmp/equate-ci-scripts/provision-guest.sh ${STAGING}/ && sudo rm -rf /tmp/equate-ci-scripts"

echo "==> provisioning appliance on guest (configure-vm + prepare-ova); this may take 30+ minutes"
ssh "${SSH_OPTS[@]}" debian@127.0.0.1 "sudo bash ${STAGING}/provision-guest.sh ${VERSION}"

echo "==> waiting for QEMU to exit after guest poweroff"
wait "${QEMU_PID}" || true
trap - EXIT

VMDK_WORK="${WORK}/export"
rm -rf "${VMDK_WORK}"
mkdir -p "${VMDK_WORK}"
VMDK="${VMDK_WORK}/Equate-Appliance-${VERSION}-amd64-disk1.vmdk"

echo "==> converting disk to streamOptimized VMDK"
qemu-img convert -f qcow2 -O vmdk -o subformat=streamOptimized "${DISK}" "${VMDK}"

echo "==> packaging OVA"
chmod +x "${SCRIPT_DIR}/package-ova.sh" "${SCRIPT_DIR}/generate-ovf.sh"
"${SCRIPT_DIR}/package-ova.sh" \
  --arch amd64 \
  --version "${VERSION}" \
  --vmdk "${VMDK}" \
  --out-dir "${OUT_DIR}" \
  --disk-gb "${DISK_GB}"

echo "==> AMD64 OVA build complete: ${OUT_DIR}/Equate-Appliance-${VERSION}-amd64.ova"
