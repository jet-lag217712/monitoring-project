## collector - 7

### Primary Service

SNMP Collector

### Secondary Services

- Ingestion Service
- Local validation and appliance deployment

### Choice Made

Phase 7 hardens deployment and operations around a **V2-only** production
telemetry contract. No production deployment ever used v1 routes, so the
Phase 4 dual-publish migration window is closed.

#### Telemetry cutover

- Deployment collector configs set `publisher.telemetry_version: v2`.
- An empty `telemetry_version` in code defaults to `v2` (supersedes the
  collector-5 default of `both`).
- Values `v1` and `both` remain accepted by validation for emergency/lab
  override only; they are unsupported in deployment profiles and runbooks.
- Deployment ingestion configs subscribe to v2 topics only.
- V1 serialization and ingestion code paths are retained but not exercised by
  default deploy smoke or acceptance tooling.

#### Compose filesystem layout

Collector-bearing Compose profiles use:

- Static inventory YAML mounted read-only (`:ro`).
- A persistent state volume for SQLite outbox, managed inventory, and audit
  log under `/var/lib/snmp-collector`.
- A runtime bind/volume for `admin.control_socket` under
  `/run/snmp-collector` (mode `0600`). The socket is never published as a
  host TCP port.
- Image non-root UID (`65532` / distroless `nonroot`). Deployments must not
  force `user: "0:0"`; operators `chown` state volumes when recreating them.
- Optional host publish of admin HTTP (`:9090`) remains scrape/liveness only
  (`/metrics`, `/healthz`, `/readyz`).

#### Smoke and acceptance ownership

- Automated smoke publishes and validates **v2** MQTT envelopes through
  ingestion to the API.
- Full GNS3/VMware field acceptance is a **manual** checklist under
  `deployments/runbooks/field-acceptance-gns3.md` (operator-owned).
- Migrations run before collector enablement in profile `up` flows and
  production runbooks.

#### Explicitly out of scope

- External infrastructure provisioning or external monitoring dashboards.
- Automated Playwright/Cypress dashboard E2E in CI.
- Committed live lab SNMP captures.
- New phase-named deployment stacks.

### Status

Accepted — Phase 7 implementation decision. Supersedes the dual-publish
default and migration-window wording in collector-5 and contracts.md for
production deployments.
