# Equate Appliance Architecture

This document is the canonical production architecture for a self-contained
Equate ISO-installed appliance. It is governed by
[appliance-1](../decisions/appliance-1.md) and
[appliance-2](../decisions/appliance-2.md).

## Runtime

```text
Browser / restricted Equate CLI
              |
            HTTPS
              |
UI (nginx) -> API -> PostgreSQL
                    ^
                    |
          Equate Core: Poller -> SQLite/WAL spool -> In-process ingestion
                    |
             SNMPv2c managed devices
```

Compose has five services: `ui`, `api`, `postgres`, `equate-core`, and
`migrate`. Only UI publishes 443 and an optional HTTP-to-HTTPS 80 redirect.
The core service is the only container on the out-of-band Docker bridge, so it
is the only controlled path to monitored devices. PostgreSQL, API admin,
control sockets, and core health endpoints are not published.

## Ownership and paths

| Domain | Owner | Persistent location |
|---|---|---|
| DHCP, TLS import, SNMP setup, support/update/reset | Appliance SNMP TUI | local console/control socket |
| Authentication/polling/discovery/dashboard/alerts | API and Equate Core | `/etc/equate/application.yaml` |
| Encrypted SNMP communities and TLS private key | Appliance SNMP TUI | `/etc/equate/secrets/*.age` |
| Runtime rendered credentials and sockets | Systemd services | `/run/equate/` |
| PostgreSQL, spool, backups, updates, cache | Equate | `/var/lib/equate/` |
| Immutable Compose release bundles | Release installer | `/opt/equate/releases/<version>/` |

`equate-stack.service` resolves `current/compose.yaml` only at start/stop; it
never overwrites a release. Activation and rollback repoint `current`
atomically. A database-affecting rollback restores the verified pre-update
backup. Migrations must be additive and compatible with the immediately prior
release.

## Installation media

The appliance is delivered as a fully offline, UEFI-only Debian 12.10
installer ISO. It contains the Debian packages required by the appliance and
the complete initial release bundle. The boot menu defaults to cancellation;
an administrator must explicitly select `Install Equate Appliance - Erases
First Disk` before the first non-removable virtual disk is repartitioned. The
installer does not configure an Internet package mirror. OCI images are
checksum-verified and loaded only after the installed system boots and Docker
is available.

`Equate-Appliance-<version>-amd64.iso` is the ESXi/vSphere artifact.
`Equate-Appliance-<version>-arm64.iso` is solely for Apple Silicon Fusion
validation; it is not a substitute for the client AMD64 deployment image.

## Security boundary

The appliance-owned `systemd-networkd` profile acquires DHCP on Ethernet
interfaces before the stack starts. The UI is automatically available on ports
80 and 443; port 80 only redirects to HTTPS. The initial TLS endpoint uses a
transient self-signed certificate until the local SNMP TUI imports a valid,
matching leaf certificate and key from read-only media labeled `EQUATE_TLS`.
The leaf must contain exactly one concrete `*.equatecloud.tech` DNS SAN. That
SAN becomes the appliance hostname; the customer supplies matching internal
DNS and trusts its issuing client CA.

The manager and Equate Core expose local, typed Unix-socket APIs only. There is
no public management HTTP API or general appliance configuration menu. SSH is
disabled by default; when enabled it accepts public keys only and runs
`ForceCommand equate`, so it cannot offer a Linux shell, SFTP, or port
forwarding. Browser sessions are `__Host-` secure HTTPS cookies. The dashboard
uses Google Identity Services: it exchanges an ID token for a server session
only after validating its client ID, verified email, and an exact Google
Workspace hosted-domain allow-list entered in the SNMP TUI. Consumer/no-`hd`
identities are denied. The Google client ID is a non-secret release setting;
Equate operations register each exact appliance HTTPS origin before use.

Support bundles and exports are written to VMware-mounted media and redact
secrets, credentials, passwords, tokens, private keys, and SNMP communities.

## Explicit exclusions

SNMPv3, automatic discovery enrollment, ACME, SMTP OAuth, multi-role RBAC,
general Linux shells/SFTP, cloud telemetry synchronization, cloud-init,
local-password bootstrap, and legacy deployment migration are not appliance
v2 features.
