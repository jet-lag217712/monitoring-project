# Queue remediation (SQLite outbox)

The collector SQLite buffer is **transport durability**, not cloud source of truth.
Path in hardened deployments: `/var/lib/snmp-collector/buffer.db`.

## Symptoms

- Growing `sqlite_queue_depth` in heartbeats / TUI transport view
- MQTT disconnected / publish failures
- Collector `/readyz` failing while `/healthz` remains OK

## MQTT outage

1. Confirm Mosquitto reachability and TLS trust.
2. Leave the collector running — it continues polling and buffering.
3. After broker recovery, the flusher drains the outbox (QoS 1).
4. Optional drill: `./deployments/lib/mqtt_outage_drill.sh deployments/end-to-end`

## Cap / full buffer

If `max_entries` is reached, new enqueues fail. Remediate by restoring MQTT first.
Do not delete the DB while the collector is running.

## Corrupt or unreadable DB

1. Stop the collector.
2. Copy `buffer.db*` aside for forensics (`buffer.db.bak-$(date -u +%Y%m%dT%H%M%SZ)`).
3. Remove the corrupt files from the state volume.
4. Start the collector — a fresh outbox is created (in-flight unacked telemetry may be lost; cloud SoR is PostgreSQL).
5. Confirm `/readyz` and a successful v2 smoke or real poll.
