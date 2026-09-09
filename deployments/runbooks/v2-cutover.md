# V2 cutover

## Decision

Production and deployment profiles publish and consume **MQTT v2 only**
([`.ai/decisions/collector-7.md`](../../.ai/decisions/collector-7.md)).
No production workload depended on v1 routes, so the Phase 4 dual-publish
window is closed.

Deployment collector YAML sets:

```yaml
publisher:
  telemetry_version: v2
```

Empty `telemetry_version` in code also defaults to `v2`.

Deployment ingestion YAML subscribes to:

```yaml
topics:
  - "site/+/device/+/telemetry/v2/#"
  - "site/+/collector/+/telemetry/v2/heartbeat"
```

## Emergency override

For lab diagnosis only, set `publisher.telemetry_version: both` (or `v1`) and
temporarily add the legacy ingestion topic `site/+/device/+/metric/#`.
Revert to v2-only before any shared or production environment.

## Smoke

Use [`../../remote-server/smoke_mqtt_v2_to_api.sh`](../../remote-server/smoke_mqtt_v2_to_api.sh).
