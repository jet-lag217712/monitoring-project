# Local OVA appliance

The supported Equate product is the full-stack on-premises appliance in
`deployments/production/appliance/`. It is one production profile, not a
collector-only product and not a split service installation.

## Platform

- Debian 12 minimal VM with UEFI, one virtual NIC, and DHCP or a site-approved
  static address.
- Initial target: 4 vCPU, 8 GB RAM, and 64 GB disk; validate capacity against
  the expected site/device count.
- UI/nginx, Backend API, PostgreSQL, Mosquitto, Ingestion, and one generated
  collector container per configured site.
- Immutable releases under `/opt/equate/releases/<version>`.
- Configuration under `/etc/equate`, mutable state under `/var/lib/equate`,
  and transient sockets/rendered secrets under `/run/equate`.

## Flow and isolation

```text
SNMPv2c devices → per-site collectors → SQLite outboxes → local MQTT
→ Ingestion → PostgreSQL → Backend API → nginx → browser
```

Only TCP 80/443 is published. PostgreSQL, MQTT, admin endpoints, metrics, and
collector control sockets remain on private networks, Unix sockets, or
loopback. Collectors reach only their configured SNMP networks and the local
broker.

## First boot and TUI

The appliance starts without shared/default credentials. First boot launches
the setup TUI, which creates the initial administrator and local appliance
users, generates per-installation secrets, creates sites, and starts generated
collector services.

The collector TUI is the day-2 configuration surface. It manages inventory,
reviewed discovery, temperature thresholds, upstream dependencies, interface
filters, transport status, and reload. Static YAML is read-only; managed files
are validated, atomically written, and audited without secrets.

## Release lifecycle

1. Build an architecture-matched offline bundle with pinned images, migrations,
   configuration templates, checksums, image digests, and SBOM.
2. Optionally package a signed `.eqa` and publish it to an Azure Blob update
   channel for connected sites (`docs/releases/appliance-updates.md`).
3. Prepare a clean VM and stage the bundle (or let `equate upgrade` download it).
4. Finalize the guest by removing build accounts, staging material, and
   clone-specific identity.
5. Export the VM to OVA and verify its manifest and checksums.
6. Re-import into a clean VM, complete first boot, and run acceptance.

The OVA must boot, configure, poll, display local telemetry, survive reboot,
and roll back invalid TUI changes without external service access. Connected
updates are optional; air-gapped installs continue to use offline staging.

## Local identity and access

Dashboard authentication uses PAM-backed local appliance users through a
permissioned host broker socket. The API is unprivileged, never mounts
`/etc/shadow`, uses generic login errors and rate limiting, and stores opaque
revocable sessions. Appliance users are not members of the Docker group.

Day-2 operator commands (`equate view`, `equate configure`, `equate users`,
`equate sites`, `equate upgrade`) are available to members of the
`equate-appliance` group via passwordless sudo rules installed to
`/etc/sudoers.d/equate-appliance`.

Generated database, MQTT, TLS, session, and SNMP credentials remain outside
release artifacts and are redacted from logs, diagnostics, manifests, and
telemetry.
