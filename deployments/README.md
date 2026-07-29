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

## Which directory to use

| Directory | Use |
|---|---|
| [`production/appliance/`](production/appliance/) | Customer appliance runtime and first-boot setup |
| [`end-to-end/`](end-to-end/) | Single-host source validation before packaging |
| [`development/`](development/) | Developer integration fixture only |
| [`runbooks/`](runbooks/) | Operator procedures for the local appliance |
| [`lib/`](lib/) | Smoke and failure-drill helpers |

The old split deployment directories remain in the repository only where their
source fixtures are needed by tests. They are not supported customer
installation paths and must not be presented as product architecture.

## Commands

```bash
# Source validation on one local host
./deployments/end-to-end/up.sh
./deployments/end-to-end/validate.sh
./deployments/end-to-end/smoke.sh
./deployments/end-to-end/acceptance.sh
./deployments/end-to-end/down.sh

# Aggregate repository checks
./deployments/test.sh --quick
./deployments/test.sh --with-smoke
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

1. Validate the local Compose files and migrations.
2. Build an architecture-matched offline release.
3. Prepare a clean Debian 12 VM and import the release.
4. Complete first boot in the setup TUI.
5. Configure at least two sites, review discovery candidates, and confirm
   telemetry in the local dashboard.
6. Reboot, run the verifier, and test rollback/restore before handoff.
