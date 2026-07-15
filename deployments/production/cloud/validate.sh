#!/usr/bin/env bash
# Validate production cloud skeleton (compose render with placeholder env).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../../lib/common.sh
source "${SCRIPT_DIR}/../../lib/common.sh"

ROOT="$(repo_root)"
COMPOSE="${SCRIPT_DIR}/docker-compose.yml"

cd "${ROOT}"
require_docker

require_file "${COMPOSE}"
require_file "${SCRIPT_DIR}/configs/ingestion.yaml"
require_file "${SCRIPT_DIR}/configs/api.yaml"
require_file "${SCRIPT_DIR}/nginx-frontend.conf"
require_file "${SCRIPT_DIR}/.env.example"

# Render with required placeholders so `:?` interpolations succeed without real secrets.
export POSTGRES_USER=ogsd
export POSTGRES_PASSWORD=placeholder
export POSTGRES_DB=ogsd
export OGSD_INGESTION_USER=ogsd_ingestion
export OGSD_INGESTION_PASSWORD=placeholder
export OGSD_API_USER=ogsd_api
export OGSD_API_PASSWORD=placeholder
export MQTT_INGESTION_PASSWORD=placeholder
export GOOGLE_CLIENT_ID=placeholder
export VITE_API_BASE_URL=https://example.com

docker compose -f "${COMPOSE}" config >/dev/null
echo "production/cloud validate: OK"
