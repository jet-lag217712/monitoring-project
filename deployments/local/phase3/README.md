# Phase 3 local stack (Docker Desktop)

One compose file runs the containers ingestion needs for local E2E:

| Service   | Port              | Role                          |
|-----------|-------------------|-------------------------------|
| snmpsim   | `1161/udp`        | Simulated SNMP device         |
| mosquitto | `8883` (MQTT/TLS) | Telemetry broker              |
| postgres  | `5432`            | Schema 001–009 + metric seed  |

Ingestion and the collector still run on the host with `go run`.

**Compose network:** `ogsd-phase3_default`

## Prerequisites

1. **Docker Desktop** running.
2. CLI context: `docker context use desktop-linux` (or whatever Desktop shows in `docker context ls`).

## C0. Start dependencies

From the **repo root**:

```bash
./deployments/local/phase3/up.sh
```

**Check:** `docker compose -f deployments/local/phase3/docker-compose.yaml ps` shows mosquitto + postgres Up (postgres healthy).

First run generates TLS certs and a password file if missing (collector `secret`, ingestion `ingestion`).

## Environment

```bash
export MQTT_PASSWORD=ingestion
export MQTT_BROKER=tls://127.0.0.1:8883
export MQTT_CA_FILE=infrastructure/docker/mqtt-broker/certs/ca.crt
export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable
```

## C1. Verify schema + seed

```bash
psql "$DATABASE_URL" -c "\dt"
psql "$DATABASE_URL" -c "SELECT name FROM metric_types;"
```

**Check:** tables include `sites`, `devices`, `interfaces`, `metric_samples`, `interface_samples`; `uptime_seconds` is present.

## C2. Start ingestion (host)

```bash
cd services/ingestion-service
export MQTT_PASSWORD=ingestion
export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable
go run ./cmd/ingestion -config configs/ingestion.example.yaml
```

**Check:** logs show MQTT connected; `curl -sf http://127.0.0.1:9091/healthz` → 200; `curl -sf http://127.0.0.1:9091/metrics | grep ingestion_mqtt_connected` shows `1`.

## C3. Start collector (host)

```bash
# separate terminal — snmpsim is already up from phase3 compose
cd services/snmp-collector
export MQTT_PASSWORD=secret
go run ./cmd/collector -config configs/collector.mqtt.example.yaml
```

**Check:** collector metrics show polls succeeding; MQTT publish totals increase.

## C4. Verify rows landed

```bash
psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM metric_samples;"
psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM interface_samples;"
psql "$DATABASE_URL" -c "SELECT hostname, status, last_seen FROM devices;"
```

**Check:** both sample counts ≥ 1 after ~30s of polling; devices `status='online'` and `last_seen` recent.

## C5. Duplicate / idempotency (manual)

Publish the **same** device metric twice (fixed timestamp), then:

```bash
psql "$DATABASE_URL" -c "SELECT device_id, metric_type_id, collected_at, COUNT(*) FROM metric_samples GROUP BY 1,2,3 HAVING COUNT(*) > 1;"
```

**Check:** query returns **0 rows**. Ingestion logs show `deduplicated` (or `ingestion_messages_deduplicated_total` increased).

## C6. Reject path (manual)

Publish invalid JSON to `site/site-001/device/dev-001/metric/device`.

**Check:** no new DB rows; ingestion log `rejected`; `ingestion_messages_rejected_total` increased; process still running.

## C7. DB failure / no-ACK (manual)

1. Note current `metric_samples` count.
2. Stop Postgres: `docker compose -f deployments/local/phase3/docker-compose.yaml stop postgres`
3. Publish one new unique device metric (new timestamp).
4. **Check:** ingestion logs `database_error`; `ingestion_db_write_failure_total` up; message **not** treated as accepted.
5. Start Postgres again: `docker compose -f deployments/local/phase3/docker-compose.yaml start postgres`
6. Wait for MQTT session redelivery (or restart ingestion with same `client_id` and `CleanStart=false`).
7. **Check:** sample count increased by **exactly 1** for that metric; no duplicates.

## C8. Dockerfile

```bash
cd services/ingestion-service
docker build -t equate/ingestion-service:dev .
```

**Check:** build exit 0.

## C9. Stop

```bash
./deployments/local/phase3/down.sh
```

## Integration tests (Layer B)

With the phase3 stack up and env exported:

```bash
cd services/ingestion-service
go test -tags=integration ./tests/ -count=1 -v
```

**Pass:** all tests PASS with zero skips.
