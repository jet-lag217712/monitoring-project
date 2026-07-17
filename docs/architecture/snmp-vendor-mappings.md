# SNMP Collector v2 Vendor Mapping Evidence

## Purpose and status

This document records the evidence used to design Cisco and Arista v2 profile
mappings before profile code is written. It is a Phase 0 design artifact, not a
claim that any mapping is implemented or supported in production.

The core SNMPv2-MIB and IF-MIB profile remains the fallback for every device.
Unknown or unsupported model families omit unavailable values; they never emit
fabricated zero values.

## Evidence matrix

| Vendor | Capability | Candidate source | Mapping requirement | Evidence status |
|---|---|---|---|---|
| Cisco | CPU utilization | CISCO-PROCESS-MIB | Select a model/OS-supported CPU table, define index selection and percent conversion, and preserve the source timestamp. | Official documentation reviewed; lab fixture required before Phase 2. |
| Cisco | Memory utilization | Cisco memory-pool MIB family | Define used/available calculation, pool selection, and behavior when a pool is absent. | Official MIB family must be matched to approved model fixtures. |
| Cisco | Temperature | CISCO-ENVMON-MIB and/or ENTITY-SENSOR-MIB | Preserve each sensor component, Celsius conversion, component status, and primary-temperature selection. | Environmental MIB coverage documented; model fixture required. |
| Cisco | Power supply | CISCO-ENVMON-MIB and platform-specific extensions | Preserve supply identity/index/status and any numeric electrical reading without reducing status to a fabricated scalar. | Candidate tables documented; model fixture required. |
| Arista | CPU utilization | EOS-supported process/system MIB documented for the approved model family | Define model-family OID, index, unit, and unsupported fallback. | EOS MIB support is documented; exact model fixture required. |
| Arista | Memory utilization | EOS-supported memory/system MIB documented for the approved model family | Define used/available calculation and pool semantics. | Exact model and EOS release fixture required. |
| Arista | Temperature | ENTITY-MIB plus ENTITY-SENSOR-MIB | Join physical entity description to sensor value, preserve component index/status, and convert the documented sensor scale to Celsius. | Official sensor examples reviewed; lab fixture required. |
| Arista | Power supply | EOS-supported ENTITY/ environment sensor tables | Preserve power component identity/status and numeric value/unit when available. | Exact model fixture required. |

## Official references

- [Cisco process CPU MIB example](https://www.cisco.com/c/en/us/support/docs/availability/high-availability/15112-HAS-baseline.html)
- [Cisco environmental MIB specifications](https://www.cisco.com/en/US/docs/wireless/asr_901/mib/reference/asrmib3_ps12890_TSD_Products_Technical_Reference_Chapter.html)
- [Arista 7100 sensor examples](https://www.arista.com/docs/Manuals/QuickStart-Managing7100Series.pdf)
- [Arista EOS SNMP documentation](https://www.arista.com/en/um-eos/eos-snmp)

## Fixture gate

Before Phase 2 profile code is accepted, each approved Cisco and Arista model
family must have a sanitized fixture containing:

- `sysObjectID`, `sysName`, and `sysDescr`;
- representative CPU and memory responses;
- all supported temperature/power component rows;
- raw type, index, value, unit, and status evidence;
- the device model and EOS/IOS release without credentials or addresses that
  identify a live customer environment.

Fixtures belong in tests or simulator data only after review. Communities,
TLS material, credentials, environment variables, process arguments, and raw
customer payloads are prohibited.
