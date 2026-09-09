#!/usr/bin/env bash
# MQTT outage drill for a configured Equate appliance.
# Stops Mosquitto briefly, confirms collectors stay running, then restarts the broker.
# Usage: sudo ./mqtt_outage_drill.sh [DEPLOY_DIR]
set -euo pipefail

DEPLOY_DIR="${1:-${EQUATE_DEPLOY_DIR:-/opt/equate/current}}"
COMPOSE_ENV="${EQUATE_COMPOSE_ENV:-/run/equate/rendered/compose.env}"

if [[ ! -d "${DEPLOY_DIR}" || ! -f "${DEPLOY_DIR}/docker-compose.yml" ]]; then
  echo "Usage: $0 [appliance-deploy-dir]" >&2
  echo "Missing compose project at ${DEPLOY_DIR}" >&2
  exit 2
fi
if [[ ! -f "${COMPOSE_ENV}" ]]; then
  echo "Missing compose env: ${COMPOSE_ENV}" >&2
  exit 2
fi

compose() {
  docker compose \
    --env-file "${COMPOSE_ENV}" \
    -f "${DEPLOY_DIR}/docker-compose.yml" \
    -f "${DEPLOY_DIR}/docker-compose.sites.generated.yml" \
    "$@"
}

echo "Pre-check appliance stack..."
compose ps

echo "Stopping Mosquitto to simulate MQTT/TLS outage..."
compose stop mosquitto

echo "Collectors must remain running while buffering..."
sleep 2
compose ps --status running | grep -q snmp-collector || {
  echo "no running snmp-collector service after broker stop" >&2
  exit 1
}
echo "Collectors still running during outage"

echo "Restarting Mosquitto..."
compose start mosquitto
sleep 2
compose ps mosquitto

echo "mqtt_outage_drill: OK"
