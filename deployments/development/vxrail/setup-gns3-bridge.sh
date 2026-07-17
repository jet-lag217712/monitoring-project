#!/usr/bin/env bash
# Create an IP-less Linux bridge for a GNS3 Cloud node attached to the collector VM.
# Requires root. Safe to re-run.
set -euo pipefail

BRIDGE="${GNS3_BRIDGE_NAME:-br-gns3-vxrail}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run as root: sudo $0" >&2
  exit 1
fi

if ! ip link show "${BRIDGE}" >/dev/null 2>&1; then
  ip link add name "${BRIDGE}" type bridge
  echo "Created bridge ${BRIDGE}"
else
  echo "Bridge ${BRIDGE} already exists"
fi

ip link set "${BRIDGE}" up
# Keep the bridge IP-less; GNS3 Cloud + macvlan collector use L2 only.
ip addr flush dev "${BRIDGE}" 2>/dev/null || true

echo "Bind the GNS3 Cloud adapter to ${BRIDGE}, then attach Cloud → lab uplink."
echo "Collector container should use a macvlan parent on this bridge (see development/README.md)."
