#!/bin/bash
set -euo pipefail
PATH=/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin:/opt/homebrew/bin

NETCFG="/Library/Preferences/VMware Fusion/networking"
VMX="/Users/jeetlad/Virtual Machines.localized/GNS3 VM.vmwarevm/GNS3 VM.vmx"
VMNETCLI="/Applications/VMware Fusion.app/Contents/Library/vmnet-cli"

cp "$NETCFG" "$NETCFG.bak.dupfix.$(date +%Y%m%d%H%M%S)"

python3 - <<'PY'
from pathlib import Path
p = Path("/Library/Preferences/VMware Fusion/networking")
t = p.read_text()
block = (
    "answer VNET_2_DHCP no\n"
    "answer VNET_2_HOSTONLY_NETMASK 255.255.255.0\n"
    "answer VNET_2_HOSTONLY_SUBNET 192.168.110.0\n"
    "answer VNET_2_VIRTUAL_ADAPTER yes\n"
)
if "VNET_2_HOSTONLY_SUBNET" not in t:
    if "add_bridge_mapping" in t:
        t = t.replace("add_bridge_mapping", block + "add_bridge_mapping", 1)
    else:
        t = t + "\n" + block
    p.write_text(t)
    print("VNET_2 added")
else:
    print("VNET_2 already present")
print(p.read_text())
PY

"$VMNETCLI" --configure || true
"$VMNETCLI" --stop || true
"$VMNETCLI" --start || true

cp "$VMX" "$VMX.bak.dupfix.$(date +%Y%m%d%H%M%S)"
if grep -q 'ethernet2.vnet' "$VMX"; then
  sed -i '' 's/ethernet2.vnet = ".*"/ethernet2.vnet = "vmnet2"/' "$VMX"
else
  echo 'ethernet2.vnet = "vmnet2"' >> "$VMX"
fi
if grep -q 'ethernet2.connectionType' "$VMX"; then
  sed -i '' 's/ethernet2.connectionType = ".*"/ethernet2.connectionType = "custom"/' "$VMX"
else
  echo 'ethernet2.connectionType = "custom"' >> "$VMX"
fi

echo "==== ethernet2 settings ===="
grep -E 'ethernet2\.' "$VMX"
echo "==== networking ===="
cat "$NETCFG"
echo "DONE_HOST_SIDE"
