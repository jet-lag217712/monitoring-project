# Deployment layouts

Equate has one supported customer deployment: the local appliance under
[`production/appliance/`](production/appliance/). It packages the dashboard,
API, ingestion, PostgreSQL, Mosquitto, and one generated collector service per
configured site on a single VMware-compatible VM.

```text
SNMP devices → collectors → local MQTT/TLS → ingestion → PostgreSQL
                                                   ↓
                                           API → nginx → dashboard
```

Develop and release from the repository Makefile. Operators on the VM use the
`equate` CLI. See [appliance-6](../.ai/decisions/appliance-6.md).

## Which directory to use

| Directory | Use |
|---|---|
| [`production/appliance/`](production/appliance/) | Customer appliance runtime and first-boot setup |
| [`runbooks/`](runbooks/) | Operator procedures for the local appliance |
| [`update-channel/`](update-channel/) | Connected `.eqa` channel schema and examples |

GNS3 lab fixtures live in [`../remote-server/`](../remote-server/).

## Commands

```bash
make help
make test
make appliance-bundle ARCH=arm64 VERSION=<version>
make appliance-stage HOST=<appliance-vm> ARCH=arm64 VERSION=<version>
```

On the Equate-Appliance VM:

```bash
sudo equate configure
make appliance-verify
```

## Appliance configuration ownership

| Concern | Owner |
|---|---|
| First boot, site creation, and generated services | Appliance setup TUI |
| Device inventory, discovery review, thresholds, dependencies, filters | Collector TUI over the local Unix socket |
| Static collector identity and protected deployment values | Appliance release configuration |
| Durable telemetry and monitoring history | Local PostgreSQL on the appliance |
| Dashboard access | Local PAM-backed appliance users |

The TUI never publishes a management socket over TCP. Static YAML is mounted
read-only; managed inventory is written atomically with a secret-free audit
entry and becomes active only after validation and reload.

## Default service ports

Only the frontend ports are customer-facing:

| Port | Scope | Purpose |
|---|---|---|
| 80 | Appliance edge | HTTP redirect to HTTPS |
| 443 | Appliance edge | Dashboard and approved API routes |
| 8000 | Private | Backend API |
| 9090–9092 | Private | Collector, ingestion, and API administration |
| 8883 | Private | MQTT/TLS broker |
| 5432 | Private | PostgreSQL |

## Acceptance order

1. Validate the local Compose files (`make test`).
2. Build an architecture-matched offline release.
3. Prepare a clean Debian 12 VM and import the release.
4. Complete first boot in the setup TUI.
5. Configure at least two sites, review discovery candidates, and confirm
   telemetry in the local dashboard.
6. Reboot, run the verifier, and test rollback/restore before handoff.
