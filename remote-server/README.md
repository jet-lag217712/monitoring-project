# Seven-device C7200 GNS3 lab

This is a self-contained remote-server customer-plane lab for exercising Equate SNMP collection. It contains seven monitored C7200 routers and two unmonitored GNS3 infrastructure nodes:

- `NAT` provides DHCP and outbound Internet access to `DO-CORE`.
- `TEST-LANS-SW` keeps each routed test-LAN interface physically up. Its six ports are isolated access VLANs; it does not route or provide hosts.

It is intentionally a lab artifact, not a mechanism for the dashboard or collector to configure devices. The collector only polls these routers from the customer OOB monitoring plane.

## Import and initial configuration

1. In GNS3, import or open [`seven-device-c7200.gns3`](seven-device-c7200.gns3).
2. Edit the router template image path if GNS3 cannot locate your licensed C7200 IOS image. The project uses a C7200 NPE-400 with 512 MB RAM.
3. Start the topology. For each C7200, paste the matching file in [`configurations/`](configurations/) at the privileged EXEC prompt, then run `write memory` when IOS asks for confirmation. The configuration files end with `end`; do **not** paste the Markdown fencing from this guide.
4. Verify that `DO-CORE GigabitEthernet0/0` received a DHCP address and that its DHCP default route is installed. If the GNS3 NAT appliance cannot provide a default route with the installed IOS image, use the DHCP-supplied gateway in a temporary static route, then retain `network 0.0.0.0` only while a default route exists.

All router configuration uses the deliberately non-secret lab login `gns3 / CHANGE-ME-BEFORE-DEPLOYMENT`. It exists only to make `login local` usable in the lab. Replace it and the SNMP community before using any configuration outside an isolated lab.

## Inventory and cabling

| Router | ASN | Loopback0 | Local test LAN | Uplink(s) |
| --- | ---: | --- | --- | --- |
| DO-CORE | 65000 | 10.255.0.1/32 | — | NAT, MacBook Cloud management handoff, Site A MDF, Site B MDF, Site C MDF |
| SITE-A-MDF | 65100 | 10.255.0.11/32 | 10.10.10.1/24 | Core, Site A IDF1, Site A IDF2 |
| SITE-A-IDF1 | 65101 | 10.255.0.12/32 | 10.10.11.1/24 | Site A MDF |
| SITE-A-IDF2 | 65102 | 10.255.0.13/32 | 10.10.12.1/24 | Site A MDF |
| SITE-B-MDF | 65200 | 10.255.0.21/32 | 10.20.20.1/24 | Core |
| SITE-C-MDF | 65300 | 10.255.0.31/32 | 10.30.30.1/24 | Core, Site C IDF1 |
| SITE-C-IDF1 | 65301 | 10.255.0.32/32 | 10.30.31.1/24 | Site C MDF |

The topology’s port labels identify every cable. Inter-router links use `/31` addressing:

| Link | Subnet | First endpoint | Second endpoint |
| --- | --- | --- | --- |
| DO-CORE ↔ SITE-A-MDF | 10.254.0.0/31 | Core .0 | MDF .1 |
| DO-CORE ↔ SITE-B-MDF | 10.254.0.2/31 | Core .2 | MDF .3 |
| DO-CORE ↔ SITE-C-MDF | 10.254.0.4/31 | Core .4 | MDF .5 |
| SITE-A-MDF ↔ SITE-A-IDF1 | 10.254.1.0/31 | MDF .1 | IDF1 .0 |
| SITE-A-MDF ↔ SITE-A-IDF2 | 10.254.1.2/31 | MDF .3 | IDF2 .2 |
| SITE-C-MDF ↔ SITE-C-IDF1 | 10.254.3.0/31 | MDF .1 | IDF1 .0 |
| MacBook ↔ DO-CORE via GNS3 Cloud | 192.168.103.0/24 | MacBook .1 | Core .2 |

## Hardware layout

Core and MDF nodes use `C7200-IO-GE-E` in slot 0 and `PA-GE` in slots 1 through 6. `DO-CORE` reserves `GigabitEthernet0/0` for the NAT handoff and `GigabitEthernet5/0` for a GNS3 Cloud connection to the MacBook management network. Bind the GNS3 Cloud node to the VM's host-only `eth2` adapter. VMware assigns the MacBook side `192.168.103.1/24`; connect the Cloud node to `GigabitEthernet5/0`, which uses `192.168.103.2/24`, and route `10.255.0.0/24` from macOS through `192.168.103.2`. IDF nodes use `C7200-IO-GE-E` in slot 0 and `PA-2FE-TX` in slots 1 through 6; `GigabitEthernet0/0` is the MDF uplink and `FastEthernet1/0` is the local routed LAN. All remaining unused physical ports are explicitly described and shut down.

## Routing and NAT behaviour

Every physical router-to-router link is a direct eBGP session. Each router originates only its own Loopback0 and local test LAN. Transit `/31` links are deliberately not advertised. `DO-CORE` has the DHCP-provided default route to NAT and originates it with `network 0.0.0.0`; BGP originates it only while that route exists in the RIB. Each non-core router also has an explicit static default route through its directly connected upstream neighbor toward `DO-CORE`, which is the active forwarding path. `DO-CORE` overloads the 10/8 space out of its DHCP interface; the MacBook Cloud management handoff is routed directly and is not a NAT inside interface.

The test-LAN ports are connected to isolated VLANs of the internal Ethernet switch solely to keep their routed interfaces up. You can validate end-to-end routing directly from router interfaces, for example:

```text
SITE-A-MDF# ping 10.30.31.1 source 10.10.10.1
SITE-C-IDF1# ping 10.20.20.1 source 10.30.31.1
SITE-B-MDF# ping 1.1.1.1 source 10.20.20.1
```

## SNMP contract

All routers use SNMPv2c community `EquateMonitor`, read-only, limited by standard ACL 10 to `10.255.0.0/24` plus the MacBook management address `192.168.103.1/32`. Each device enables cold-start, warm-start, link-up, and link-down trap categories. No `snmp-server host` is configured because the trap receiver address is environment-specific; add it only when the monitoring host supports trap reception.

| Collection | OID / method |
| --- | --- |
| Name, description, uptime | `sysName.0`, `sysDescr.0`, `sysUpTime.0` |
| IPv4 addresses | walk `ipAdEntAddr` |
| Interface count / active count | `ifNumber.0`; count rows with both `ifAdminStatus` and `ifOperStatus` up |
| CPU | walk `CISCO-PROCESS-MIB::cpmCPUTotal5min` |
| Memory | `ciscoMemoryPoolUsed` and `ciscoMemoryPoolFree`; calculate used / (used + free) |
| Temperature | walk `ciscoEnvMonTemperatureStatusValue`; record `unsupported` if no C7200/GNS3 sensor is exposed |
| Device state | Loopback0 `ifAdminStatus`; reachability plus Loopback0 `ifOperStatus` |
| Interface identity/state | `ifName`, `ifAlias`, `ifAdminStatus`, `ifOperStatus`, indexed by `ifIndex` |
| Interface performance | `ifHighSpeed` (`ifSpeed` fallback), `dot3StatsDuplexStatus`, HC octets (32-bit fallback), HC unicast packets, errors, `ifLastChange` |
| Interface utilization | calculate sampled octet delta × 8 / interface speed; it is not one SNMP scalar |

The current C7200 image may not implement every optional table, particularly environmental sensors, high-capacity counters, or duplex status on every interface. A collector must mark those values unsupported or use the documented fallback, never synthesize a value.

## Verification

Run these commands on each router after convergence:

```text
show ip bgp summary
show ip route 0.0.0.0
show ip bgp
show ip interface brief
show snmp community
```

Expected BGP peer counts are three on `DO-CORE`, three on `SITE-A-MDF`, two on `SITE-C-MDF`, and one on each remaining router. All sessions must be `Established`; `DO-CORE` must show the DHCP default route and every non-core router must show its static default route toward its upstream neighbor.

From an approved monitoring address (`10.255.0.x` or the MacBook address `192.168.103.1`), use `snmpwalk -v2c -c EquateMonitor <loopback> <MIB root>` for SNMPv2-MIB, IF-MIB, IP-MIB, CISCO-PROCESS-MIB, CISCO-ENVMON-MIB, and CISCO-MEMORY-POOL-MIB. Confirm an unapproved source is denied. Finally, shut and restore a test interface and verify that `ifOperStatus`, counters, and `ifLastChange` update.
