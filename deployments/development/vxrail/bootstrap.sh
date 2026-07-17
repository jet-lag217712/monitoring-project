#!/usr/bin/env bash
# Build and start the development vxrail collector on the OrbStack VM.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "${SCRIPT_DIR}"

if [[ ! -f .env ]]; then
  cp -n .env.example .env
  echo "Created .env from .env.example — set MQTT_BROKER and SNMP_COMMUNITY_* before continuing." >&2
  exit 1
fi

# shellcheck disable=SC1091
set -a
source ./.env
set +a

mkdir -p run certs
if [[ ! -f certs/ca.crt ]]; then
  echo "Missing certs/ca.crt (sync from Mac Mosquitto CA)." >&2
  exit 1
fi

# Prefer synced collector source when present (VM layout after sync.sh).
BUILD_CONTEXT="../../../services/snmp-collector"
if [[ -d ./src/services/snmp-collector ]]; then
  BUILD_CONTEXT="./src/services/snmp-collector"
fi

export MQTT_BROKER MQTT_PASSWORD
export SNMP_COMMUNITY_DO_CORE SNMP_COMMUNITY_SITE_A_MDF SNMP_COMMUNITY_SITE_A_IDF1
export SNMP_COMMUNITY_SITE_A_IDF2 SNMP_COMMUNITY_SITE_B_MDF SNMP_COMMUNITY_SITE_C_MDF
export SNMP_COMMUNITY_SITE_C_IDF1

docker compose build --build-arg BUILDKIT_INLINE_CACHE=1 snmp-collector 2>/dev/null || true
# Rewrite build context for the VM without editing the tracked compose file permanently:
COMPOSE_OVERRIDE="$(mktemp)"
cat >"${COMPOSE_OVERRIDE}" <<EOF
services:
  snmp-collector:
    build: ${BUILD_CONTEXT}
EOF

docker compose -f docker-compose.yml -f "${COMPOSE_OVERRIDE}" up -d --build
rm -f "${COMPOSE_OVERRIDE}"

# Ensure nonroot can write state + control socket dir.
docker run --rm -v ogsd-development-vxrail_collector-state:/var/lib/snmp-collector busybox:1.36 \
  chown -R 65532:65532 /var/lib/snmp-collector >/dev/null 2>&1 || true
docker run --rm -v "${SCRIPT_DIR}/run:/run/snmp-collector" busybox:1.36 \
  chown -R 65532:65532 /run/snmp-collector >/dev/null 2>&1 || true
docker compose up -d snmp-collector

echo "bootstrap: collector started. Check: curl -fsS http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz"
