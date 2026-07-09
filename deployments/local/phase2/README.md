# Phase 2 local stack (Docker Desktop)

One compose file runs the two containers the collector needs:

| Service   | Port              | Role                          |
|-----------|-------------------|-------------------------------|
| snmpsim   | `1161/udp`        | Simulated SNMP device         |
| mosquitto | `8883` (MQTT/TLS) | Telemetry broker              |

The collector itself still runs on the host with `go run`.

## Prerequisites

1. **Docker Desktop** running (not Colima).
2. CLI context: `docker context use desktop-linux` (or whatever Desktop shows in `docker context ls`).

## Start

From the **repo root**:

```bash
./deployments/local/phase2/up.sh
```

First run generates TLS certs and a password file if missing (default collector password: `secret`).

## Stop

```bash
./deployments/local/phase2/down.sh
```

## Collector

```bash
cd services/snmp-collector
export MQTT_PASSWORD=secret
go run ./cmd/collector -config configs/collector.mqtt.example.yaml
```

## Watch MQTT messages

```bash
cd infrastructure/docker/mqtt-broker
docker run --rm -it --network ogsd-phase2_default \
  -v "$PWD/certs:/certs:ro" eclipse-mosquitto:2 \
  mosquitto_sub -h mosquitto -p 8883 --cafile /certs/ca.crt \
  -u ingestion -P ingestion -t 'site/+/device/+/metric/#' -v
```

## Status / logs

```bash
docker compose -f deployments/local/phase2/docker-compose.yaml ps
docker compose -f deployments/local/phase2/docker-compose.yaml logs -f
```
