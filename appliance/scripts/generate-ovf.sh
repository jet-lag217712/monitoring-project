#!/usr/bin/env bash
# Generate a minimal OVF descriptor for VMware import.
#
# Usage:
#   generate-ovf.sh --arch amd64|arm64 --version 0.2.0 --name Equate-Appliance-0.2.0-amd64 \
#     --vmdk Equate-Appliance-0.2.0-amd64-disk1.vmdk --out Equate-Appliance-0.2.0-amd64.ovf
set -euo pipefail

ARCH=""
VERSION=""
NAME=""
VMDK=""
OUT=""
DISK_GB=64
MEMORY_MB=4096
VCPUS=2

usage() {
  cat <<'EOF'
usage: generate-ovf.sh --arch <amd64|arm64> --version <semver> --name <ovf-base> \
  --vmdk <disk.vmdk> --out <path.ovf> [--disk-gb N] [--memory-mb N] [--vcpus N]
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch) ARCH="${2:-}"; shift 2 ;;
    --version) VERSION="${2:-}"; shift 2 ;;
    --name) NAME="${2:-}"; shift 2 ;;
    --vmdk) VMDK="${2:-}"; shift 2 ;;
    --out) OUT="${2:-}"; shift 2 ;;
    --disk-gb) DISK_GB="${2:-}"; shift 2 ;;
    --memory-mb) MEMORY_MB="${2:-}"; shift 2 ;;
    --vcpus) VCPUS="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "${ARCH}" || -z "${VERSION}" || -z "${NAME}" || -z "${VMDK}" || -z "${OUT}" ]]; then
  usage >&2
  exit 1
fi

if [[ ! -f "${VMDK}" ]]; then
  echo "vmdk not found: ${VMDK}" >&2
  exit 1
fi

case "${ARCH}" in
  amd64)
    OS_TYPE="debian12_64Guest"
    GUEST_OS_LABEL="Debian 12 (64-bit)"
    ;;
  arm64)
    OS_TYPE="armOtherLinux64Guest"
    GUEST_OS_LABEL="Debian 13 ARM64"
    ;;
  *)
    echo "--arch must be amd64 or arm64" >&2
    exit 1
    ;;
esac

VMDK_BYTES="$(qemu-img info --output json "${VMDK}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["virtual-size"])')"
POPULATED="$(stat -c '%s' "${VMDK}" 2>/dev/null || stat -f '%z' "${VMDK}")"
VMDK_FILE="$(basename "${VMDK}")"

cat >"${OUT}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<Envelope xmlns="http://schemas.dmtf.org/ovf/envelope/1"
  xmlns:ovf="http://schemas.dmtf.org/ovf/envelope/1"
  xmlns:rasd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData"
  xmlns:vssd="http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData"
  xmlns:vmw="http://www.vmware.com/schema/ovf">
  <References>
    <File ovf:href="${VMDK_FILE}" ovf:id="file1" ovf:size="${POPULATED}"/>
  </References>
  <DiskSection>
    <Info>Virtual disk information</Info>
    <Disk ovf:capacity="${DISK_GB}" ovf:capacityAllocationUnits="byte * 2^30"
      ovf:diskId="vmdisk1" ovf:fileRef="file1"
      ovf:format="http://www.vmware.com/interfaces/specifications/vmdk.html#streamOptimized"
      ovf:populatedSize="${POPULATED}"/>
  </DiskSection>
  <NetworkSection>
    <Info>The list of logical networks</Info>
    <Network ovf:name="VM Network">
      <Description>VM Network</Description>
    </Network>
  </NetworkSection>
  <VirtualSystem ovf:id="vm">
    <Info>Equate Appliance ${VERSION} (${ARCH})</Info>
    <Name>${NAME}</Name>
    <OperatingSystemSection ovf:id="1" vmw:osType="${OS_TYPE}">
      <Info>${GUEST_OS_LABEL}</Info>
    </OperatingSystemSection>
    <VirtualHardwareSection>
      <Info>Virtual hardware requirements</Info>
      <System>
        <vssd:ElementName>Virtual Hardware Family</vssd:ElementName>
        <vssd:InstanceID>0</vssd:InstanceID>
        <vssd:VirtualSystemIdentifier>${NAME}</vssd:VirtualSystemIdentifier>
        <vssd:VirtualSystemType>vmx-21</vssd:VirtualSystemType>
      </System>
      <Item>
        <rasd:ElementName>${VCPUS} virtual CPU(s)</rasd:ElementName>
        <rasd:InstanceID>1</rasd:InstanceID>
        <rasd:ResourceType>3</rasd:ResourceType>
        <rasd:VirtualQuantity>${VCPUS}</rasd:VirtualQuantity>
      </Item>
      <Item>
        <rasd:ElementName>${MEMORY_MB}MB of memory</rasd:ElementName>
        <rasd:InstanceID>2</rasd:InstanceID>
        <rasd:ResourceType>4</rasd:ResourceType>
        <rasd:VirtualQuantity>${MEMORY_MB}</rasd:VirtualQuantity>
      </Item>
      <Item>
        <rasd:AddressOnParent>0</rasd:AddressOnParent>
        <rasd:ElementName>disk0</rasd:ElementName>
        <rasd:HostResource>ovf:/disk/vmdisk1</rasd:HostResource>
        <rasd:InstanceID>3</rasd:InstanceID>
        <rasd:Parent>4</rasd:Parent>
        <rasd:ResourceType>17</rasd:ResourceType>
      </Item>
      <Item>
        <rasd:Address>0</rasd:Address>
        <rasd:Description>SCSI Controller</rasd:Description>
        <rasd:ElementName>scsi0</rasd:ElementName>
        <rasd:InstanceID>4</rasd:InstanceID>
        <rasd:ResourceType>6</rasd:ResourceType>
      </Item>
      <Item>
        <rasd:AddressOnParent>0</rasd:AddressOnParent>
        <rasd:AutomaticAllocation>true</rasd:AutomaticAllocation>
        <rasd:Connection>VM Network</rasd:Connection>
        <rasd:Description>E1000e ethernet adapter</rasd:Description>
        <rasd:ElementName>ethernet0</rasd:ElementName>
        <rasd:InstanceID>5</rasd:InstanceID>
        <rasd:ResourceSubType>E1000e</rasd:ResourceSubType>
        <rasd:ResourceType>10</rasd:ResourceType>
      </Item>
      <vmw:Config ovf:required="false" vmw:key="firmware" vmw:value="efi"/>
    </VirtualHardwareSection>
  </VirtualSystem>
</Envelope>
EOF

echo "generated OVF: ${OUT} (virtual-size=${VMDK_BYTES} bytes)"
