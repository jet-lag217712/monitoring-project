## collector - 6

### Primary Service

SNMP Collector

### Secondary Services

- None (local operator plane only)

### Choice Made

Phase 6 adds a local-only Unix status/control protocol and a Bubble Tea TUI
client. The TUI never talks to SNMP or MQTT directly; it is only a
client of the control socket.

#### Transport and auth

Control binds a Unix domain socket configured by `admin.control_socket`. An
empty path disables the control server (backward-compatible for tests). The
socket file is mode `0600`; a stale socket is removed on bind. Authentication is
OS filesystem access only. Public HTTP remains scrape/liveness only
(`/metrics`, `/healthz`, `/readyz`); no mutation HTTP endpoint is added.

#### Versioned NDJSON protocol

Requests and responses are newline-delimited JSON with an explicit protocol
version. Current version is `1`. Unsupported versions are rejected with
`UNSUPPORTED_VERSION`.

Request shape:

```json
{"version":1,"id":"request-id","method":"status.summary","params":{}}
```

Success response:

```json
{"version":1,"id":"request-id","ok":true,"result":{}}
```

Error response:

```json
{"version":1,"id":"request-id","ok":false,"error":{"code":"CONFIG_RELOAD_FAILED","message":"validation failed"}}
```

Stable error codes: `UNSUPPORTED_VERSION`, `INVALID_REQUEST`, `METHOD_NOT_FOUND`,
`CONFIRM_EXPIRED`, `REVISION_MISMATCH`, `VALIDATION_FAILED`,
`CONFIG_RELOAD_FAILED`, `NOT_FOUND`, `INTERNAL`.

Limits: maximum request frame 256 KiB; maximum response frame 1 MiB. Oversized
frames are rejected with `INVALID_REQUEST`. Default per-request server timeout
is 30 seconds; the handler context is cancelled when the timeout elapses.

Idempotency: status methods are idempotent. `*.prepare` always issues a new
confirm token. `*.commit` is single-use. `config.reload` is safe to retry and
may activate an unchanged snapshot. Discovery accept requires explicit
confirmation and is not silently repeated.

#### Mutation workflow

Prepare returns `confirm_token`, active `revision` (`ConfigRevision`), and
`expires_at` (default 60 seconds). Commit must supply both token and revision.
Expired tokens return `CONFIRM_EXPIRED`. Tokens bound to a superseded revision
return `REVISION_MISMATCH`. Every successful and failed mutation attempt is
appended to a secret-free audit log (JSON lines). Audit never records
communities, passwords, certificates, env values, or payload bodies.

#### Configuration ownership

```
Static inventory → Managed overlay → Runtime snapshot
```

Runtime state never writes back into static inventory or managed overlays.
Mutations persist only through `WriteManagedInventory` (or the full managed
document writer), then an explicit reload builds a new runtime snapshot.

Static-authoritative (not overlayable): host, port, SNMP version,
`community_env`, collector identity, MQTT, buffer, admin listener, discovery
CIDR allowlist.

Managed overlay (allowed): temperature thresholds (global and per-device),
upstream dependencies, interface filters, discovery rate
(`max_probes_per_second`, `probe_burst` only), and per-device
`alerts_enabled` (Administratively Ignored when false).

A managed device entry with the same ID as a static device is an overlay of
allowed fields onto the static device. Unique managed IDs append full devices.
The managed file remains the only file the TUI writes.

#### Operator status store

`internal/status` is an operator state cache, not a metrics database. It may
retain last poll timestamp/result, reachability, last error class, interface
and component summaries, buffer depth, reload status, and configuration
revision. It must not store historical metrics, OID dumps, time-series, or
interface counter history. MQTT telemetry remains responsible for history.

#### Discovery orchestration

Control/TUI only orchestrates existing operations:

`Scanner → Review → AcceptReviewed → WriteManagedInventory → Explicit Reload`

Never auto-enroll devices, never auto-create LLDP dependencies, and never
activate discovered devices without explicit confirmation.

#### Reload

`config.reload` uses the same `Manager.Reload()` path as `SIGHUP`. Invalid
reloads retain the prior snapshot and return `CONFIG_RELOAD_FAILED`.

### Status

Accepted — Phase 6 implementation decision.
