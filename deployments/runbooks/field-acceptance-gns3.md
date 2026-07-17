# Field acceptance — GNS3 / VMware (manual)

Operator-owned checklist. Not automated in CI.

## Preconditions

- [ ] Profile chosen (`development` + vxrail, or `end-to-end`)
- [ ] Migrations applied before collector enablement
- [ ] `community_env` values set locally (never committed)
- [ ] Mosquitto CA trusted by collector
- [ ] State volume + `./run` owned by UID `65532`

## Connectivity and health

- [ ] Collector `/healthz` and `/readyz` OK
- [ ] Ingestion and API health OK
- [ ] `collector tui` connects via `./run/control.sock`
- [ ] Synthetic v2 smoke passes (`smoke.sh`)

## Polling and telemetry

- [ ] Real device appears in API with `status` and optional `status_reason`
- [ ] CPU/memory/temperature/power visible when vendor profile applies
- [ ] Interface counters update for selected interfaces
- [ ] Collector heartbeat updates cloud collector status

## Reachability / cascade

- [ ] Independent poll continues when an upstream fails
- [ ] Root Critical vs dependent Unknown (`upstream_unreachable`) distinguished in API/UI
- [ ] Recovery clears failure count and restores Healthy/Warning

## Control plane

- [ ] Threshold prepare/commit + reload persists managed inventory
- [ ] Invalid managed edit does not change active snapshot
- [ ] Discovery scan respects allowlist and rate limit; accept requires confirmation

## Failure drills

- [ ] MQTT stop/start: collector stays up; buffer drains after restore (`mqtt_outage_drill.sh`)
- [ ] SQLite backup/restore or corrupt-DB recovery per queue remediation runbook
- [ ] Image/config rollback leaves managed state intact when intended

## Dashboard

- [ ] Unknown visual treatment (not Critical) for upstream-unreachable devices
- [ ] Site summary separates critical vs dependency-impacted counts

Record date, profile, git SHA, and any deviations when signing off.
