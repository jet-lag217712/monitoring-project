#!/usr/bin/env bash
# Static artifact validation; does not require GNS3 or Dynamips.
set -euo pipefail

lab_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
configs=(DO-CORE SITE-A-MDF SITE-A-IDF1 SITE-A-IDF2 SITE-B-MDF SITE-C-MDF SITE-C-IDF1)

for device in "${configs[@]}"; do
  config="$lab_dir/configurations/$device.cfg"
  test -f "$config"
  rg -qx "hostname $device" "$config"
  rg -q '^ip routing$' "$config"
  rg -q '^ip cef$' "$config"
  rg -q '^snmp-server community EquateMonitor RO 10$' "$config"
  rg -q '^ permit 10\.255\.0\.0 0\.0\.0\.255$' "$config"
  rg -q '^ permit host 192\.168\.103\.1$' "$config"
  rg -q '^interface Loopback0$' "$config"
  rg -q '^ description OOB management loopback$' "$config"
done

rg -q '^ip nat inside source list NAT-INSIDE-10 interface GigabitEthernet0/0 overload$' "$lab_dir/configurations/DO-CORE.cfg"
rg -q '^ip route 0\.0\.0\.0 0\.0\.0\.0 dhcp$' "$lab_dir/configurations/DO-CORE.cfg"
rg -q '^ network 0\.0\.0\.0$' "$lab_dir/configurations/DO-CORE.cfg"
rg -U -q '^interface GigabitEthernet5/0\n description MacBook GNS3 Cloud management handoff\n ip address 192\.168\.103\.2 255\.255\.255\.0\n no shutdown\n!$' "$lab_dir/configurations/DO-CORE.cfg"
! rg -q '^interface GigabitEthernet6/0$' "$lab_dir/configurations/DO-CORE.cfg"

rg -qx '^ip route 0\.0\.0\.0 0\.0\.0\.0 10\.254\.0\.0$' "$lab_dir/configurations/SITE-A-MDF.cfg"
rg -qx '^ip route 0\.0\.0\.0 0\.0\.0\.0 10\.254\.1\.1$' "$lab_dir/configurations/SITE-A-IDF1.cfg"
rg -qx '^ip route 0\.0\.0\.0 0\.0\.0\.0 10\.254\.1\.3$' "$lab_dir/configurations/SITE-A-IDF2.cfg"
rg -qx '^ip route 0\.0\.0\.0 0\.0\.0\.0 10\.254\.0\.2$' "$lab_dir/configurations/SITE-B-MDF.cfg"
rg -qx '^ip route 0\.0\.0\.0 0\.0\.0\.0 10\.254\.0\.4$' "$lab_dir/configurations/SITE-C-MDF.cfg"
rg -qx '^ip route 0\.0\.0\.0 0\.0\.0\.0 10\.254\.3\.1$' "$lab_dir/configurations/SITE-C-IDF1.cfg"

for device in "${configs[@]}"; do
  route_count=$(rg -c '^ip route 0\.0\.0\.0 0\.0\.0\.0 ' "$lab_dir/configurations/$device.cfg")
  test "$route_count" -eq 1
done

python3 -m json.tool "$lab_dir/seven-device-c7200.gns3" >/dev/null
router_count=$(rg -c '"node_type": "dynamips"' "$lab_dir/seven-device-c7200.gns3")
test "$router_count" -eq 7

echo "GNS3 lab artifact checks passed."
