#!/usr/bin/env bash
# Create Mosquitto password file for local collector + ingestion users.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/passwords"

create_with_local() {
  if [[ -n "${MQTT_COLLECTOR_PASSWORD:-}" ]]; then
    mosquitto_passwd -b -c "${OUT}" collector "${MQTT_COLLECTOR_PASSWORD}"
  else
    echo "Creating password file at ${OUT} (interactive collector password)"
    mosquitto_passwd -c "${OUT}" collector
  fi
  mosquitto_passwd -b "${OUT}" ingestion "${MQTT_INGESTION_PASSWORD:-ingestion}"
}

create_with_docker() {
  local collector_pass="${MQTT_COLLECTOR_PASSWORD:-}"
  if [[ -z "${collector_pass}" ]]; then
    echo "Set MQTT_COLLECTOR_PASSWORD for non-interactive Docker password creation." >&2
    echo "Example: MQTT_COLLECTOR_PASSWORD=secret ./scripts/gen-passwords.sh" >&2
    exit 1
  fi
  docker run --rm -v "${ROOT}:/work" eclipse-mosquitto:2 \
    mosquitto_passwd -b -c /work/passwords collector "${collector_pass}"
  docker run --rm -v "${ROOT}:/work" eclipse-mosquitto:2 \
    mosquitto_passwd -b /work/passwords ingestion "${MQTT_INGESTION_PASSWORD:-ingestion}"
}

if command -v mosquitto_passwd >/dev/null 2>&1; then
  create_with_local
elif command -v docker >/dev/null 2>&1; then
  create_with_docker
else
  echo "Need mosquitto_passwd or docker. Install mosquitto clients, e.g.:" >&2
  echo "  brew install mosquitto" >&2
  exit 1
fi

chmod 600 "${OUT}"
echo "Created users: collector, ingestion at ${OUT}"
echo "Export MQTT_PASSWORD for the collector process (collector user password)."
