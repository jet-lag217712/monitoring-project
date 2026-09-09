# Field acceptance — GNS3 / VMware (manual)

Operator-owned checklist. Not automated in CI.

## Preconditions

- [ ] Equate-Appliance VM running `deployments/production/appliance/` (`equate configure` complete)
- [ ] SNMP reachability into [`remote-server/`](../../remote-server/) GNS3 lab
- [ ] `community_env` values set locally (never committed)
- [ ] Mosquitto CA trusted by collectors
- [ ] Optional: `sudo remote-server/setup-gns3-bridge.sh` on the appliance VM

## Connectivity and health

- [ ] Collector `/healthz` and `/readyz` OK
- [ ] Ingestion and API health OK
- [ ] `collector tui` connects via `./run/control.sock`
- [ ] Synthetic v2 smoke passes ([`remote-server/smoke_mqtt_v2_to_api.sh`](../../remote-server/smoke_mqtt_v2_to_api.sh))

## Polling and telemetry

- [ ] Real device appears in API with `status` and optional `status_reason`
- [ ] CPU/memory/temperature/power visible when vendor profile applies
- [ ] Interface counters update for selected interfaces
- [ ] Collector heartbeat updates local collector status

## Reachability / cascade

- [ ] Independent poll continues when an upstream fails
- [ ] Root Critical vs dependent Unknown (`upstream_unreachable`) distinguished in API/UI
- [ ] Recovery clears failure count and restores Healthy/Warning

## Control plane

- [ ] Threshold prepare/commit + reload persists managed inventory
- [ ] Invalid managed edit does not change active snapshot
- [ ] Discovery scan respects allowlist and rate limit; accept requires confirmation

## Failure drills

- [ ] MQTT stop/start: collector stays up; buffer drains after restore ([`remote-server/mqtt_outage_drill.sh`](../../remote-server/mqtt_outage_drill.sh))
- [ ] SQLite backup/restore or corrupt-DB recovery per queue remediation runbook
- [ ] Image/config rollback leaves managed state intact when intended

## Dashboard

- [ ] Unknown visual treatment (not Critical) for upstream-unreachable devices
- [ ] Site summary separates critical vs dependency-impacted counts

Record date, profile, git SHA, and any deviations when signing off.
