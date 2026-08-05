# Equate local monitoring appliance

Equate is a local, on-premises network monitoring appliance for VxRail- or
VMware-type installations. The complete monitoring stack runs on the appliance
VM; it has no remote-service dependency at runtime and is intended to operate
inside the customer network.

The supported production artifact is an offline OVA. An operator deploys the
OVA, completes first boot in the appliance setup TUI, configures sites and SNMP
inventory, and uses the local dashboard for monitoring.

## Runtime architecture

```text
SNMP devices
    │
    ▼
Per-site SNMP collectors ── local Unix TUI/control socket
    │
    ▼
SQLite outbox → local MQTT/TLS → ingestion → PostgreSQL
                                            │
                                            ▼
                                   Backend API → nginx → browser
```

All services are separate containers on the same appliance VM. PostgreSQL is
the persistent system of record. MQTT is a local delivery boundary, and the
collector's SQLite database is a durable transport outbox rather than a second
monitoring database.

The appliance exposes only the dashboard on TCP 80/443. PostgreSQL, MQTT,
service administration endpoints, metrics, and collector control sockets are
private to the VM or its container networks. Collectors reach only the
configured SNMP networks and the local broker.

## Configuration model

Configuration is an operator workflow, not a remote control plane:

1. First boot launches `collector setup` in the appliance TUI.
2. The setup workflow creates the local administrator, appliance users, site
   definitions, generated collector services, and per-installation secrets.
3. Day-2 changes use `collector tui` through a local Unix socket.
4. Static deployment YAML remains read-only. The TUI writes validated managed
   inventory, thresholds, upstream dependencies, interface filters, and
   discovery policy, then performs an explicit reload.

Discovery is operator-invoked, limited to configured CIDRs and rate bounds, and
never enrolls a device without review and confirmation.

## Operator interfaces

The appliance exposes two Bubble Tea TUIs and an `equate` CLI for stack-level
operations. Per-site collector control uses a Unix socket; stack management
uses Docker Compose under the active release directory.

### Setup TUI (`collector setup`)

First boot and reconfiguration:

```bash
collector setup -dir <deploy-dir> -theme auto -profile appliance
```

Common keys: `Enter` continue/submit, `Tab`/`↓` next field, `Shift+Tab`/`↑`
previous field, `Ctrl+C` quit (confirms unless on splash), `q` quit from splash
or done screen.

During discovery review: `↑`/`↓` navigate candidates, `Space` toggle,
`Enter` accept reviewed, `s` skip site, `r` retry on error.

### Operator TUI (`collector tui`)

Day-2 per-site monitoring and configuration:

```bash
collector tui -socket <control.sock> -theme auto
equate view <site-id>    # runs collector tui inside the site container
```

| Key | Action |
|-----|--------|
| `1`–`6` | Switch views: Inventory, Device, Discovery, Thresholds, Transport, Config |
| `Tab`, `→`, `l` / `Shift+Tab`, `←`, `h` | Next / previous view |
| `r` | Refresh current view (auto-refreshes every 5s) |
| `R` | Force `config.reload` |
| `↑`/`↓`, `j`/`k`, `PgUp`/`PgDn` | Scroll |
| `q`, `Ctrl+C` | Quit |

View-specific keys:

| Key | View | Action |
|-----|------|--------|
| `n` / `p` | Device | Next / previous device |
| `t` | Device, Thresholds | Edit temperature threshold (global or per-device) |
| `d` | Inventory, Device | Edit upstream dependencies |
| `S` | Discovery | Start discovery scan |
| `A` | Discovery | Accept successful candidates |
| `e` | Discovery | Edit CIDR allowlist policy |

Mutations use prepare → confirm → commit → reload. After editing a value,
`Enter` submits, `y` confirms commit, `n`/`Esc` cancels.

### `equate` CLI

Appliance-level stack management. Deploy directory resolution:
`EQUATE_DEPLOY_DIR` → `/etc/equate/deploy-dir` → `/opt/equate/current`.

| Command | Purpose |
|---------|---------|
| `equate configure` | Run first-boot or reconfigure setup wizard |
| `equate configure --temperature <celsius>` | Set global temperature warning on all site collectors |
| `equate reset` | Stop containers and clear setup state (root required) |
| `equate upgrade` | In-place release upgrade or rollback (root required) |
| `equate view <site-id>` | Open per-site collector operator TUI |
| `equate sites` / `equate sites list` | List configured sites from manifest |
| `equate sites delete <site-id> [--yes]` | Remove a site (collector, artifacts, DB rows) |
| `equate status` | Summarize stack and collector health |
| `equate version` | Show release version |

#### `equate reset`

```bash
sudo equate reset [--yes] [--volumes] [--full] [--hard]
```

| Flag | Effect |
|------|--------|
| *(none)* | Stop containers, clear setup artifacts, restart core stack |
| `--yes` | Skip interactive confirmation (otherwise type `RESET`) |
| `--volumes` | Remove named Docker volumes (collector state, postgres, mosquitto) |
| `--full` | Also wipe Postgres data under `/var/lib/equate/postgres` |
| `--hard` | Full wipe (`--volumes` + `--full` + mosquitto data); leaves stack stopped |

Always removes from the deploy directory: `.setup-complete`, `.env`,
`docker-compose.sites.generated.yml`, and `sites/`. After a normal reset, run
`sudo equate configure`. After `--hard`, bring the stack up manually, then
configure.

#### `equate upgrade`

```bash
sudo equate upgrade --bundle <dir> --version <semver> [--canary] [--yes]
sudo equate upgrade --rollback [--yes]
```

`--canary` rolls out collectors one site at a time with health checks.
`--rollback` reverts to the previous release (prompts for `ROLLBACK` unless
`--yes`).

## Repository guide

| Area | Purpose |
|---|---|
| [`deployments/production/appliance/`](deployments/production/appliance/) | Supported local appliance Compose runtime and setup inputs |
| [`appliance/scripts/`](appliance/scripts/) | Offline release, VM preparation, OVA packaging, and verification |
| [`docs/releases/appliance-ova.md`](docs/releases/appliance-ova.md) | OVA build, first boot, acceptance, and handoff runbook |
| [`docs/architecture/`](docs/architecture/) | Service boundaries, data flow, contracts, and storage |
| [`deployments/runbooks/`](deployments/runbooks/) | Installation, TUI operations, rotation, recovery, and rollback |
| [`.ai/`](.ai/) | Canonical project context, decisions, standards, and roadmap |

`deployments/end-to-end/` and the development directories are validation
fixtures for engineers. They are not alternative customer deployment models.

## Local source validation

For a source checkout with Docker Compose and reachable SNMP test devices:

```bash
./deployments/end-to-end/up.sh
./deployments/end-to-end/validate.sh
./deployments/end-to-end/smoke.sh
./deployments/end-to-end/acceptance.sh
./deployments/end-to-end/down.sh
```

For the supported offline release workflow, use the OVA runbook and the release
scripts rather than copying service containers or hand-editing a customer VM.

## Appliance release

```bash
make appliance-bundle ARCH=arm64 VERSION=<version>
make appliance-bundle ARCH=amd64 VERSION=<version>
```

The release contains pinned images, migrations, configuration templates,
checksums, image digests, and an SBOM. The release is staged offline and
verified by re-importing the resulting OVA into a clean VMware-compatible VM.

## Design commitments

- Local-first and on-premises: the appliance remains useful without Internet
  access after its release bundle is staged.
- TUI-first operations: setup and day-2 collector configuration are performed
  through local terminal workflows with explicit confirmation.
- Reviewed discovery: scanning and enrollment are separate actions.
- Durable telemetry: polling continues during broker interruption and drains
  the SQLite outbox after recovery.
- Honest health: direct failures, temperature warnings, and dependency-impact
  Unknown states remain distinct.
- Least privilege: local PAM authentication, private service networks, secret
  redaction, and no public collector mutation endpoint.
