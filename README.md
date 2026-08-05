# Equate local monitoring appliance

Equate is a local, on-premises network monitoring appliance for VxRail- or
VMware-type installations. The complete monitoring stack runs on the appliance
VM; it has no remote-service dependency at runtime and is intended to operate
inside the customer network.

The supported production artifact is an offline OVA. An operator deploys the
OVA, completes first boot in the appliance setup TUI, configures sites and SNMP
inventory, and uses the local dashboard for monitoring. Connected appliances
can optionally receive signed `.eqa` release updates over HTTPS; air-gapped
sites continue to upgrade from a staged offline bundle.

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
3. Day-2 changes use `collector tui` through a local Unix socket, or the
   `equate` CLI for stack-level operations.
4. Static deployment YAML remains read-only. The TUI writes validated managed
   inventory, thresholds, upstream dependencies, interface filters, and
   discovery policy, then performs an explicit reload.

Discovery is operator-invoked, limited to configured CIDRs and rate bounds, and
never enrolls a device without review and confirmation.

## Operator interfaces

The appliance exposes two Bubble Tea TUIs and an `equate` CLI for stack-level
operations. Per-site collector control uses a Unix socket; stack management
uses Docker Compose under the active release directory.

Day-2 operator commands that need host privileges
(`equate configure`, `equate users`, `equate sites`, `equate view`,
`equate upgrade`, `equate reset`) are available to members of the
`equate-appliance` group via passwordless sudo rules installed to
`/etc/sudoers.d/equate-appliance`.

Deploy directory resolution:
`EQUATE_DEPLOY_DIR` → `/etc/equate/deploy-dir` → `/opt/equate/current`.

### Setup TUI (`collector setup`)

First boot and reconfiguration:

```bash
collector setup -dir <deploy-dir> -theme auto -profile appliance
```

Prefer `equate configure` on a deployed appliance; it launches the same wizard
with the correct deploy directory and profile.

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

### `equate` CLI overview

| Command | Purpose |
|---------|---------|
| [`equate configure`](#equate-configure) | First-boot or reconfigure setup wizard; site/user modes; global temperature |
| [`equate sites`](#equate-sites) | List configured sites or delete one site |
| [`equate upgrade`](#equate-upgrade) | In-place release upgrade (online `.eqa` channel or offline bundle) and rollback |
| `equate users` | Manage local PAM-backed appliance users |
| `equate view <site-id>` | Open the per-site collector operator TUI |
| `equate status` | Summarize Compose stack and collector/core health |
| `equate reset` | Stop containers and clear setup state (root required) |
| `equate version` | Show release version and build metadata |

---

## `equate configure`

### What it does

`equate configure` runs the appliance setup / reconfigure wizard against the
active deploy directory. It ensures rendered secrets and database role
passwords are in place, then launches the setup TUI (or a scoped
reconfiguration mode).

Without flags it walks the full first-boot style flow: administrators and
users, site definitions, SNMP communities, discovery bounds, and generated
collector services. Scoped flags limit the wizard to sites or users only.
`--temperature` bypasses the wizard and applies a global temperature warning
threshold to every site collector.

### Purpose

Use configure when you need to:

- Complete first boot after OVA import (or after `equate reset`)
- Re-run setup to change site count, CIDRs, communities, or probe bounds
- Add or adjust appliance users through the wizard (`--users`)
- Change only site layout without revisiting user screens (`--sites`)
- Set one Celsius warning threshold across all sites without opening each
  collector TUI

It is the supported day-2 entry point for host-level setup changes. Per-device
inventory, discovery acceptance, and interface filters remain in
`equate view` / `collector tui`.

### How to use it

```bash
# Full setup / reconfigure wizard
sudo equate configure

# Sites only (site IDs, CIDRs, probe rate/burst, communities as applicable)
sudo equate configure --sites

# Users only (appliance PAM users)
sudo equate configure --users

# Apply global temperature warning (°C) to every site collector
sudo equate configure --temperature 55
```

| Invocation | Behavior |
|------------|----------|
| `equate configure` | Full wizard (`--mode full`) |
| `equate configure --sites` | Sites-only reconfigure |
| `equate configure --users` | Users-only reconfigure |
| `equate configure --temperature <celsius>` | Non-interactive global threshold apply |

Notes:

- `--temperature` cannot be combined with `--sites` or `--users`.
- Temperature apply fails if any site collector control socket is unreachable.
- After a normal `equate reset`, run `sudo equate configure` again before
  returning the appliance to service.

---

## `equate sites`

### What it does

`equate sites` manages the site list recorded in the appliance manifest.

- Bare `equate sites` or `equate sites list` prints each site ID, Compose
  service name, collector admin URL, and SNMP CIDR.
- `equate sites delete <site-id>` permanently removes one site: stops and
  removes its collector container and volume, rewrites the manifest and
  generated Compose file, deletes host artifacts under `sites/<site-id>`,
  deletes related PostgreSQL rows, reconciles the stack, and re-syncs site
  topology.

### Purpose

Use sites when you need to:

- Inventory which collectors are defined on the appliance
- Confirm site IDs before `equate view <site-id>`
- Retire a monitoring site without running the full configure wizard (which
  risks accidental CIDR or inventory changes)

Listing is read-only. Delete is destructive and requires confirmation (or
`--yes`).

### How to use it

```bash
# List sites (default)
equate sites
equate sites list

# Delete a site (prompts: type the site-id to confirm)
sudo equate sites delete campus-a

# Non-interactive delete
sudo equate sites delete campus-a --yes
```

List output columns:

| Column | Meaning |
|--------|---------|
| `SITE_ID` | Stable site identifier used by `equate view` and the dashboard |
| `SERVICE` | Generated Docker Compose service name for the collector |
| `ADMIN_URL` | Local collector admin/health base URL |
| `CIDR` | Configured SNMP discovery / poll network |

Delete confirmation: type the exact `site-id` when prompted, unless `--yes` is
set. Deleting the last remaining sites leaves the core stack running without
collectors; add sites again with `sudo equate configure --sites`.

---

## `equate upgrade`

### What it does

`equate upgrade` performs an in-place appliance release upgrade or rollback.
It always requires root.

There are three apply paths:

1. **Connected (online) channel** — when `/etc/equate/update-channel.conf` is
   present, `equate upgrade` fetches the channel manifest over HTTPS, selects
   the matching edition/architecture artifact, downloads a signed `.eqa`
   package, verifies SHA-256 and an Ed25519 signature against the public key
   embedded in `equate`, extracts to `/tmp/equate-staging/bundle`, and applies
   the release through `configure-vm.sh --upgrade`.
2. **Offline staged bundle** — when you pass `--bundle` and `--version`, or
   when no channel config exists but a bundle is already staged at
   `/tmp/equate-staging/bundle`, upgrade applies that directory directly.
3. **Rollback** — `--rollback` reverts to the previous release under
   `/opt/equate/releases`.

`.eqa` packages are gzip-compressed tars of the same offline release bundle
produced by `make appliance-bundle`. Integrity is enforced by checksum and
signature verification inside `equate`; the update host is not a trust
anchor.

### Purpose

Use upgrade to:

- Patch or advance a **connected** appliance from the published update channel
  without SCP staging
- Advance an **air-gapped** appliance from a manually staged release bundle
- Verify whether a newer channel version exists (`--check`) before applying
- Roll collectors out gradually with health checks (`--canary`)
- Recover to the previous release if an upgrade misbehaves (`--rollback`)

Connected updates are optional. Air-gapped installs keep working with offline
staging only. Standard and NoAuth editions use separate channels and never
cross-update.

For release-engineering publish details (Azure Blob layout, signing keys,
GitHub Actions), see
[`docs/releases/appliance-updates.md`](docs/releases/appliance-updates.md).

### How to use it — connected (online `.eqa`) path

#### 1. Configure the update channel

Create `/etc/equate/update-channel.conf` on the appliance:

```ini
channel_url=https://<storage-account>.blob.core.windows.net/updates/v1/channel/stable/manifest.json
edition=standard
```

| Key | Required | Meaning |
|-----|----------|---------|
| `channel_url` | Yes | HTTPS URL of the channel `manifest.json` |
| `edition` | Yes | Appliance edition (`standard` or `noauth`); must match the channel |

Example for the public stable channel:

```bash
sudo tee /etc/equate/update-channel.conf >/dev/null <<'EOF'
channel_url=https://equateupdate.blob.core.windows.net/updates/v1/channel/stable/manifest.json
edition=standard
EOF
```

#### 2. Check for an update

```bash
sudo equate upgrade --check
```

Reports the installed version and the channel latest. Exit without downloading
when already up to date, or when no newer version is available. Fails if the
channel config is missing.

#### 3. Apply the update

```bash
sudo equate upgrade
```

Operator flow:

1. Read installed version and channel latest
2. Prompt for confirmation (`Type UPGRADE to continue`, unless `--yes`)
3. Download the `.eqa` to `/var/lib/equate/downloads`
4. Verify SHA-256 and Ed25519 signature against the embedded public key
5. Extract to `/tmp/equate-staging/bundle`
6. Run `configure-vm.sh --upgrade` (preserves sites, secrets, and migrations)

```bash
# Skip confirmation (automation / remote ops)
sudo equate upgrade --yes

# Canary: roll collectors one site at a time with health checks
sudo equate upgrade --canary
sudo equate upgrade --canary --yes
```

#### 4. Rollback if needed

```bash
sudo equate upgrade --rollback
# or
sudo equate upgrade --rollback --yes
```

Prompts for `ROLLBACK` unless `--yes` is set.

### How to use it — offline bundle path

Stage a release bundle onto the appliance (for example with
`make appliance-stage` or manual SCP), then:

```bash
sudo equate upgrade --bundle /tmp/equate-staging/bundle --version <semver>
sudo equate upgrade --bundle /tmp/equate-staging/bundle --version <semver> --canary
sudo equate upgrade --bundle /tmp/equate-staging/bundle --version <semver> --yes
```

Both `--bundle` and `--version` are required for explicit offline mode.

If `/etc/equate/update-channel.conf` is **absent** and a staged bundle exists
at `/tmp/equate-staging/bundle` with a readable `release.env`, bare
`sudo equate upgrade` will use that staged bundle when its version is newer
than the installed release.

### Upgrade flags reference

| Flag | Behavior |
|------|----------|
| *(none)* | Connected channel upgrade if configured; else staged `/tmp/equate-staging/bundle` if present |
| `--check` | Report installed vs channel latest; do not download or apply |
| `--bundle <dir>` | Offline bundle directory (requires `--version`) |
| `--version <semver>` | Target version for offline apply (requires `--bundle`) |
| `--canary` | Roll out collectors one site at a time with health checks |
| `--rollback` | Revert to the previous release |
| `--yes` | Skip `UPGRADE` / `ROLLBACK` confirmation |
| `--url` / `--sha256` / `--signature` | Direct `.eqa` download for testing (all three required together) |
| `--channel-config <path>` | Override path to `update-channel.conf` (default `/etc/equate/update-channel.conf`) |
| `--allow-insecure-http` | Allow `http://` channel/artifact URLs (**local testing only**) |

### Security notes (updates)

- Production channel and artifact URLs must be HTTPS.
- SHA-256 and Ed25519 signature must verify before extract/apply.
- The trust anchor is the public key baked into `equate` (also shipped under
  `appliance/keys/`), not a key fetched from the update host.
- Edition mismatch fails closed.
- Apply and rollback remain delegated to `configure-vm.sh`.

---

## Other `equate` commands

### `equate users`

Manage local PAM-backed dashboard users (create, list, delete, disable,
enable, reset-password):

```bash
equate users list
sudo equate users create <username>          # prompts for password twice
sudo equate users reset-password <username>
sudo equate users disable <username>
sudo equate users enable <username>
sudo equate users delete <username>          # type DELETE to confirm
```

### `equate view`

Open the day-2 collector operator TUI for one site:

```bash
equate view <site-id>
```

Site IDs come from `equate sites list`.

### `equate status`

Print Compose service status, per-site collector `/healthz`, and core
endpoint health (frontend, backend-api, ingestion):

```bash
equate status
```

### `equate reset`

Stop containers and clear setup state so the appliance can be reconfigured:

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

### `equate version`

```bash
equate version
```

Prints `equate <version> (<git-commit>) built <timestamp>`.

---

## Repository guide

| Area | Purpose |
|---|---|
| [`deployments/production/appliance/`](deployments/production/appliance/) | Supported local appliance Compose runtime and setup inputs |
| [`deployments/update-channel/`](deployments/update-channel/) | Channel manifest schema and `update-channel.conf` examples |
| [`appliance/scripts/`](appliance/scripts/) | Offline release, VM preparation, OVA packaging, `.eqa` publish |
| [`docs/releases/appliance-ova.md`](docs/releases/appliance-ova.md) | OVA build, first boot, acceptance, and handoff runbook |
| [`docs/releases/appliance-updates.md`](docs/releases/appliance-updates.md) | Connected `.eqa` updates, signing, and Azure publish |
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
checksums, image digests, and an SBOM. Package a signed `.eqa` for the update
channel with `make appliance-package` (see
[`docs/releases/appliance-updates.md`](docs/releases/appliance-updates.md)).
The offline path stages the bundle and verifies by re-importing the resulting
OVA into a clean VMware-compatible VM.

## Design commitments

- Local-first and on-premises: the appliance remains useful without Internet
  access after its release bundle is staged.
- Optional connected updates: signed `.eqa` packages over HTTPS; air-gap keeps
  offline staging.
- TUI-first operations: setup and day-2 collector configuration are performed
  through local terminal workflows with explicit confirmation.
- Reviewed discovery: scanning and enrollment are separate actions.
- Durable telemetry: polling continues during broker interruption and drains
  the SQLite outbox after recovery.
- Honest health: direct failures, temperature warnings, and dependency-impact
  Unknown states remain distinct.
- Least privilege: local PAM authentication, private service networks, secret
  redaction, and no public collector mutation endpoint.
