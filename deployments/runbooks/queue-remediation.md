# Queue remediation (SQLite outbox)

The collector SQLite database is transport durability. PostgreSQL on the
appliance remains the monitoring system of record. The hardened buffer path is
`/var/lib/snmp-collector/buffer.db`.

## Symptoms

- `sqlite_queue_depth` grows in heartbeats or the TUI transport view.
- MQTT is disconnected or publish attempts fail.
- `/readyz` fails while `/healthz` remains healthy.

## Local broker interruption

1. Confirm the local Mosquitto container is running and its TLS files are
   readable.
2. Leave the collector running; it continues polling and buffering.
3. Restore the broker and allow the flusher to drain the outbox in order.
4. Confirm the TUI transport view, queue depth, and dashboard update.

## Full buffer

If `max_entries` is reached, restore local broker delivery first. Do not delete
the database while the collector is running.

## Corrupt or unreadable buffer

1. Stop the affected collector.
2. Copy `buffer.db*` to a protected forensic location.
3. Remove the corrupt files from the collector state volume.
4. Start the collector; it creates a fresh outbox. In-flight unacknowledged
   events may be lost.
5. Confirm `/readyz`, a v2 smoke, or a live poll.

Preserve the old files until the incident review is complete.
