# On-Prem OVA Appliance

## Authority and scope

The supported full-stack on-premises appliance lives under
`deployments/production/appliance/`. This document and
[`appliance-1`](../decisions/appliance-1.md) are authoritative for that profile.
The appliance is a production subprofile, not a fourth top-level deployment
profile and not a replacement for the hybrid production architecture.

The appliance is the only exception to physical separation of the Customer OOB
Monitoring Plane and UI/UX Plane. Both planes run on one VM, but all existing
service contracts and ownership boundaries remain in force.

## Platform

- Debian 12 minimal VM with UEFI, DHCP, one virtual NIC, and no required
  runtime cloud dependency
- Initial sizing target: 4 vCPU, 8 GB RAM, and 64 GB disk; validate capacity
  with representative site/device load before the AMD64 client release
- Existing UI/nginx, Backend API, PostgreSQL, Mosquitto, and Ingestion
  containers plus one generated collector container per configured site
- Immutable releases under `/opt/equate/releases/<version>`, configuration
  under `/etc/equate`, mutable state under `/var/lib/equate`, and transient
  sockets/rendered secrets under `/run/equate`

The application flow remains:

```text
SNMPv2c devices → per-site collectors → SQLite outboxes → local MQTT
→ Ingestion → PostgreSQL → Backend API → UI/nginx → browser
```

MQTT remains a delivery boundary, and PostgreSQL remains the appliance-wide
monitoring system of record. The appliance must not replace this flow with the
reverted ISO or `equate-core` in-process design.

Collector containers reach SNMP devices through the host VM route table. Host
`snmpwalk` succeeding while collector discovery fails usually means the
collector container network cannot route to the device subnet (not a missing host
SNMP package). The collector embeds its own SNMP client; `snmp`/`snmpwalk` on
the Debian VM are optional operator debugging tools only.

## Isolation and network surface

- Only TCP 80/443 may be published from the appliance VM. Port 80 should
  redirect to HTTPS once TLS is configured.
- PostgreSQL, MQTT, Ingestion/API administration endpoints, metrics, and
  collector control sockets stay on private container networks, Unix sockets,
  or loopback.
- Each collector owns its inventory, reviewed discovery workflow, SQLite
  outbox, and local control socket. Discovery never auto-enrolls candidates.
- Collectors may reach only their configured SNMP device networks and local
  Mosquitto. Browser traffic reaches nginx, which proxies only approved API
  routes.
- No service may create an inbound cloud-management path or expose secrets,
  raw Docker access, or collector mutation controls through the dashboard.

## Local identity and access

The first-boot TUI requires creation of the initial administrator; the OVA
ships without a shared/default password. The TUI also creates, lists, disables,
and resets additional PAM-backed appliance users in a dedicated OS group.
Those users authenticate to both the restricted VM appliance interface and the
local dashboard.

A root-owned, allowlisted helper performs privileged account/appliance
operations. Appliance users are not members of the Docker group. A host-side
PAM broker is reachable by the API only through a permissioned Unix socket; the
API remains unprivileged, never mounts `/etc/shadow`, and never logs passwords.
Dashboard authentication uses generic failures, rate limiting, active-account
checks, opaque revocable sessions, secure cookies, and CSRF protection.

Internal database, MQTT, machine, and TLS credentials are generated per
installation and are neither requested from the operator nor included in the
release bundle. SNMP communities and all generated credentials must remain out
of logs, diagnostics, manifests, and source control.

## Release and operational lifecycle

1. Build a checksummed, architecture-matched offline release bundle containing
   pinned binaries/images, migrations, configuration, image digests, source
   revision, and an SBOM.
2. Create a clean Debian 12 VM, transfer the bundle and installer over SCP,
   configure the VM, and validate the Compose stack.
3. Finalize in release mode: remove build accounts/staging material and all
   clone-specific credentials, machine identity, SSH host keys, DHCP leases,
   logs, and history. Finalization fails closed if sensitive material remains.
4. Power off, export manually to OVA, verify its OVF/VMDK/manifest and
   checksums, then re-import into a new VM for first-boot acceptance.
5. Qualify ARM64 first on VMware Fusion. Repeat the same parameterized process
   on an x86 VMware-capable host for the AMD64 VMware client release.

An EC2 VM is not assumed to export directly as a valid VMware OVA. GitHub
release publication is deferred; accepted outputs are the OVA, checksum, and
release manifest.

Operators must back up appliance configuration and PostgreSQL/collector state,
retain the matching release manifest and image digests, monitor disk/outbox
growth, and test restore and rollback before production use.

## Acceptance minimum

A fresh re-imported OVA must expose only 80/443, contain no default/customer
credentials, complete first boot without Internet access after staging, create
two PAM users usable for both VM and dashboard login, configure at least two
named sites, require explicit discovery review, display local telemetry, retain
state across reboot, and roll back a failed reconfiguration.
