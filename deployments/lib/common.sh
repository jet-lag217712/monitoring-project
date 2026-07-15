#!/usr/bin/env bash
# Shared helpers for deployment profiles. Source from other scripts; do not execute directly.
# shellcheck shell=bash

deployments_root() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  cd "${here}/.." && pwd
}

repo_root() {
  local dep
  dep="$(deployments_root)"
  cd "${dep}/.." && pwd
}

require_cmd() {
  local cmd
  for cmd in "$@"; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      echo "Required command not found: ${cmd}" >&2
      exit 1
    fi
  done
}

require_docker() {
  require_cmd docker
  if ! docker info >/dev/null 2>&1; then
    echo "Docker is not reachable. Start Docker, then retry." >&2
    exit 1
  fi
  if ! docker compose version >/dev/null 2>&1; then
    echo "Docker Compose plugin is required (docker compose)." >&2
    exit 1
  fi
}

load_env_file() {
  local env_file="$1"
  if [[ ! -f "${env_file}" ]]; then
    echo "Missing env file: ${env_file}" >&2
    exit 1
  fi
  set -a
  # shellcheck disable=SC1090
  source "${env_file}"
  set +a
}

ensure_env_file() {
  local env_file="$1"
  local example_file="$2"
  if [[ ! -f "${env_file}" ]]; then
    echo "Creating ${env_file} from $(basename "${example_file}")..."
    cp "${example_file}" "${env_file}"
  fi
}

wait_http() {
  local url="$1"
  local label="${2:-${url}}"
  local attempts="${3:-60}"
  local i
  for i in $(seq 1 "${attempts}"); do
    if curl -fsS --max-time 2 "${url}" >/dev/null 2>&1; then
      echo "${label} is ready."
      return 0
    fi
    sleep 1
  done
  echo "${label} did not become ready: ${url}" >&2
  return 1
}

wait_postgres() {
  local compose_file="$1"
  local env_file="$2"
  local user="${3:-ogsd}"
  local db="${4:-ogsd}"
  local attempts="${5:-60}"
  local i
  for i in $(seq 1 "${attempts}"); do
    if docker compose --env-file "${env_file}" -f "${compose_file}" exec -T postgres \
      pg_isready -U "${user}" -d "${db}" >/dev/null 2>&1; then
      echo "Postgres is ready."
      return 0
    fi
    sleep 1
  done
  echo "Postgres did not become ready in time." >&2
  return 1
}

compose_cmd() {
  local env_file="$1"
  local compose_file="$2"
  shift 2
  if [[ -f "${env_file}" ]]; then
    docker compose --env-file "${env_file}" -f "${compose_file}" "$@"
  else
    docker compose -f "${compose_file}" "$@"
  fi
}

ensure_mqtt_material() {
  local root="$1"
  local mqtt_dir="${root}/infrastructure/docker/mqtt-broker"

  if [[ ! -f "${mqtt_dir}/certs/ca.crt" || ! -f "${mqtt_dir}/certs/server.crt" ]]; then
    echo "Generating Mosquitto TLS certs..."
    MQTT_SERVER_CN="${MQTT_SERVER_CN:-localhost}" \
    MQTT_SERVER_DNS="${MQTT_SERVER_DNS:-mosquitto,localhost}" \
    MQTT_SERVER_IP="${MQTT_SERVER_IP:-}" \
      "${mqtt_dir}/scripts/gen-dev-certs.sh"
  else
    echo "Mosquitto TLS certs already present (delete ${mqtt_dir}/certs to regenerate with new SANs)."
  fi

  if [[ ! -f "${mqtt_dir}/passwords" ]]; then
    echo "Creating Mosquitto password file..."
    MQTT_COLLECTOR_PASSWORD="${MQTT_COLLECTOR_PASSWORD:-secret}" \
    MQTT_INGESTION_PASSWORD="${MQTT_INGESTION_PASSWORD:-ingestion}" \
      "${mqtt_dir}/scripts/gen-passwords.sh"
  fi
}

migrate_and_bootstrap_roles() {
  local root="$1"
  local admin_url="$2"
  local ingestion_password="$3"
  local api_password="$4"

  echo "Applying database migrations..."
  DATABASE_URL="${admin_url}" "${root}/infrastructure/script/migrate.sh" up

  echo "Bootstrapping application role passwords..."
  DATABASE_URL="${admin_url}" \
    OGSD_INGESTION_PASSWORD="${ingestion_password}" \
    OGSD_API_PASSWORD="${api_password}" \
    "${root}/infrastructure/script/bootstrap-db-roles.sh"
}

require_file() {
  local path="$1"
  if [[ ! -f "${path}" ]]; then
    echo "Required file missing: ${path}" >&2
    exit 1
  fi
}

require_nonempty() {
  local name="$1"
  local value="$2"
  if [[ -z "${value}" ]]; then
    echo "Required variable is empty: ${name}" >&2
    exit 1
  fi
}
