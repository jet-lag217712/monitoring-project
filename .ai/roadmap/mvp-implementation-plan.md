# Equate local appliance implementation plan

This roadmap describes the local, on-premises Equate appliance. The production
target is an offline OVA for a VxRail- or VMware-type VM. Every product service
runs on that VM, and configuration is performed through local terminal/TUI
workflows.

## Product baseline

```text
SNMPv2c devices → collectors → SQLite outbox → local MQTT/TLS
→ ingestion → PostgreSQL → Backend API → nginx → React dashboard
```

The baseline includes:

- per-site collectors with bounded polling and vendor-neutral telemetry;
- local health and dependency evidence;
- durable SQLite buffering across broker interruptions;
- v2 MQTT event schemas with idempotent PostgreSQL ingestion;
- local PAM-backed dashboard sessions;
- first-boot and day-2 configuration TUI workflows;
- offline release bundles, OVA packaging, and re-import verification.

## Delivery phases

### Phase 1 — Local appliance foundation

- Keep the full stack in `deployments/production/appliance/`.
- Generate per-installation database, MQTT, TLS, and session secrets at first
  boot.
- Keep internal networks private and publish only nginx on 80/443.
- Store immutable releases under `/opt/equate/releases`, configuration under
  `/etc/equate`, mutable data under `/var/lib/equate`, and runtime sockets under
  `/run/equate`.

Exit: a clean VM can boot the stack without pre-created credentials or external
service access.

### Phase 2 — TUI configuration

- First boot uses `collector setup -profile appliance`.
- Day-2 `collector tui` covers sites, inventory, discovery, thresholds,
  dependencies, interfaces, transport, and reload.
- Mutations use prepare/confirm/commit and secret-free audit entries.
- Static YAML is read-only; managed inventory is validated and atomically
  written.

Exit: an operator can configure a multi-site appliance without hand-editing
runtime files or exposing a remote management endpoint.

### Phase 3 — Collector v2 telemetry

- Poll SNMPv2c devices through bounded worker pools.
- Normalize core identity/IF-MIB data and supported vendor profiles.
- Emit device, interface, health, and heartbeat events using the v2 schemas.
- Preserve direct Critical, temperature Warning, and dependency Unknown states.

Exit: live and fixture tests prove independent polling, component readings,
health transitions, recovery, and dependency evidence.

### Phase 4 — Durable local ingestion

- Validate v2 topics and envelopes before writes.
- Deduplicate by event ID and natural sample keys.
- ACK only after a transaction commits or a message is safely rejected.
- Retain observation-time ordering for current health and collector status.

Exit: broker interruption, duplicate delivery, database interruption, and
malformed messages have documented and tested behavior.

### Phase 5 — Dashboard and access

- Serve the React dashboard from local nginx.
- Use `appliance_local` authentication backed by the PAM broker.
- Expose site, device, interface, metric, component, health, and alert views.
- Keep Unknown dependency impact visually distinct from direct Critical.

Exit: two local appliance users can sign in and view live local telemetry.

### Phase 6 — Release and acceptance

- Build pinned image bundles for ARM64 and AMD64.
- Generate checksums, image digests, migrations, and SBOM data.
- Prepare, finalize, export, and re-import the OVA.
- Verify only 80/443 is reachable and state survives reboot.

Exit: the acceptance checklist passes on a clean VMware-compatible VM with no
Internet access after staging.

## Non-goals

- External control planes or remote configuration services.
- Remote collector management through HTTP.
- SNMPv3, SNMP writes, device console access, or automatic enrollment.
- A full physical-network graph; upstream dependencies remain operator-defined.
- Fabricated zero readings for unsupported vendor metrics.

## Required evidence

- `docs/releases/appliance-ova.md` for build and handoff.
- `docs/architecture/contracts.md` for the v2 wire contract.
- `deployments/runbooks/` for operator procedures.
- `services/snmp-collector/README.md` for TUI and collector operation.
