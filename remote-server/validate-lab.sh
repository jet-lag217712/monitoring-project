#!/usr/bin/env bash
# Static artifact validation; does not require GNS3 or Dynamips.
set -euo pipefail

lab_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
configs=(DO-CORE SITE-A-MDF SITE-A-IDF1 SITE-A-IDF2 SITE-B-MDF SITE-C-MDF SITE-C-IDF1)

for device in "${configs[@]}"; do
  config="$lab_dir/configs/$device.cfg"
  test -f "$config"
  rg -qx "hostname $device" "$config"
  rg -q '^ip routing$' "$config"
  rg -q '^ip cef$' "$config"
  rg -q '^snmp-server community EquateMonitor RO 10$' "$config"
  rg -q '^ permit 10\.255\.0\.0 0\.0\.0\.255$' "$config"
  rg -q '^interface Loopback0$' "$config"
  rg -q '^ description OOB management loopback$' "$config"
done

rg -q '^ip nat inside source list NAT-INSIDE-10 interface GigabitEthernet0/0 overload$' "$lab_dir/configs/DO-CORE.cfg"
rg -q '^ip route 0\.0\.0\.0 0\.0\.0\.0 dhcp$' "$lab_dir/configs/DO-CORE.cfg"
rg -q '^ network 0\.0\.0\.0$' "$lab_dir/configs/DO-CORE.cfg"

python3 -m json.tool "$lab_dir/seven-device-c7200.gns3" >/dev/null
router_count=$(rg -c '"node_type": "dynamips"' "$lab_dir/seven-device-c7200.gns3")
test "$router_count" -eq 7

echo "GNS3 lab artifact checks passed."
