#!/usr/bin/env bash
# Post-install health checks for the Equate on-prem appliance VM.
# Run after configure-vm completes or after reconfiguration.
set -euo pipefail

EQUATE_CURRENT=/opt/equate/releases/current
EQUATE_RUN=/run/equate
EQUATE_RENDERED=${EQUATE_RUN}/rendered
AUTH_SOCK=${EQUATE_RUN}/auth.sock
COMPOSE_PROJECT=equate

REQUIRED_SERVICES=(
  postgres
  mosquitto
  ingestion
  api
  ui
)

failures=0

note() {
  printf 'verify-appliance: %s\n' "$*"
}

pass() {
  note "OK  $*"
}

fail() {
  note "FAIL $*"
  failures=$((failures + 1))
}

require_cmd() {
  local cmd
  for cmd in "$@"; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      fail "required command not found: ${cmd}"
      return 1
    fi
  done
}

compose_file() {
  printf '%s/compose.yaml' "${EQUATE_CURRENT}"
}

compose_env_file() {
  if [[ -f "${EQUATE_RENDERED}/compose.env" ]]; then
    printf '%s\n' "${EQUATE_RENDERED}/compose.env"
  elif [[ -f "${EQUATE_RENDERED}/database-admin.env" ]]; then
    printf '%s\n' "${EQUATE_RENDERED}/database-admin.env"
  else
    return 1
  fi
}

compose_cmd() {
  local env_file
  env_file="$(compose_env_file)" || {
    fail "missing rendered compose env under ${EQUATE_RENDERED}"
    return 1
  }
  docker compose \
    --project-name "${COMPOSE_PROJECT}" \
    --env-file "${env_file}" \
    --file "$(compose_file)" \
    "$@"
}

service_container_id() {
  local service=$1
  compose_cmd ps -q "${service}" 2>/dev/null | head -n 1
}

service_running() {
  local service=$1
  local id state
  id="$(service_container_id "${service}")"
  [[ -n "${id}" ]] || return 1
  state="$(docker inspect -f '{{.State.Status}}' "${id}")"
  [[ "${state}" == "running" ]]
}

service_http() {
  local service=$1
  local port=$2
  local path=$3
  local id ip

  id="$(service_container_id "${service}")"
  [[ -n "${id}" ]] || return 1
  ip="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{if .IPAddress}}{{.IPAddress}}{{end}}{{end}}' "${id}" | awk 'NF { print; exit }')"
  [[ -n "${ip}" ]] || return 1
  curl -fsS --max-time 5 "http://${ip}:${port}${path}" >/dev/null
}

check_release_layout() {
  if [[ -L "${EQUATE_CURRENT}" && -f "$(compose_file)" ]]; then
    pass "active release at ${EQUATE_CURRENT}"
  else
    fail "missing active release symlink or compose.yaml"
  fi

  if [[ -d "${EQUATE_RENDERED}" ]]; then
    pass "rendered runtime directory ${EQUATE_RENDERED}"
  else
    fail "missing ${EQUATE_RENDERED}"
  fi
}

check_compose_up() {
  local service
  local ps_output

  if ! ps_output="$(compose_cmd ps --status running 2>/dev/null)"; then
    fail "docker compose ps failed"
    return
  fi

  for service in "${REQUIRED_SERVICES[@]}"; do
    if service_running "${service}"; then
      pass "compose service running: ${service}"
    else
      fail "compose service not running: ${service}"
    fi
  done

  if grep -q 'collector' <<<"${ps_output}"; then
    pass "at least one collector service is running"
  else
    note "WARN no collector services running yet (expected before site setup)"
  fi
}

check_postgres() {
  if compose_cmd exec -T postgres pg_isready -U "${POSTGRES_USER:-equate}" -d "${POSTGRES_DB:-equate}" >/dev/null 2>&1; then
    pass "postgres accepts connections"
    return
  fi
  fail "postgres is not ready"
}

check_mqtt() {
  if service_running mosquitto; then
    pass "mosquitto container is running"
  else
    fail "mosquitto container is not running"
    return
  fi

  local id
  id="$(service_container_id mosquitto)"
  if docker inspect -f '{{range $p, $cfg := .NetworkSettings.Ports}}{{if eq $p "8883/tcp"}}{{if $cfg}}open{{end}}{{end}}{{end}}' "${id}" 2>/dev/null | grep -q open; then
    fail "mosquitto port 8883 is published externally"
  else
    pass "mosquitto is not published externally"
  fi
}

check_api_health() {
  if service_http api 9092 /healthz; then
    pass "backend API admin /healthz"
  else
    fail "backend API admin /healthz unreachable"
  fi

  if service_http ingestion 9091 /healthz; then
    pass "ingestion admin /healthz"
  else
    fail "ingestion admin /healthz unreachable"
  fi
}

check_auth_socket() {
  if [[ -S "${AUTH_SOCK}" ]]; then
    pass "auth broker socket present at ${AUTH_SOCK}"
  else
    fail "auth broker socket missing: ${AUTH_SOCK}"
    return
  fi

  local mode owner
  mode="$(stat -c '%a' "${AUTH_SOCK}" 2>/dev/null || stat -f '%OLp' "${AUTH_SOCK}")"
  owner="$(stat -c '%U:%G' "${AUTH_SOCK}" 2>/dev/null || stat -f '%Su:%Sg' "${AUTH_SOCK}")"
  note "auth socket mode=${mode} owner=${owner}"
}

check_nginx() {
  local code

  code="$(curl -k -s -o /dev/null -w '%{http_code}' --max-time 5 https://127.0.0.1/ || true)"
  if [[ "${code}" =~ ^[23] ]]; then
    pass "nginx responds on https://127.0.0.1/ (HTTP ${code})"
  else
    fail "nginx HTTPS probe failed (HTTP ${code:-none})"
  fi

  code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://127.0.0.1/ || true)"
  if [[ "${code}" == "301" || "${code}" == "302" || "${code}" == "308" ]]; then
    pass "nginx redirects HTTP to HTTPS (HTTP ${code})"
  else
    note "WARN nginx HTTP redirect returned HTTP ${code:-none}"
  fi
}

check_published_ports() {
  local listeners
  listeners="$(ss -H -tln 2>/dev/null | awk '{print $4}' | sed 's/.*://g' | sort -u || true)"
  local allowed=(80 443)
  local port

  for port in ${listeners}; do
    local ok=0
    for allowed_port in "${allowed[@]}"; do
      if [[ "${port}" == "${allowed_port}" ]]; then
        ok=1
        break
      fi
    done
    if [[ "${ok}" -eq 0 && "${port}" != "22" ]]; then
      fail "unexpected listening TCP port: ${port}"
    fi
  done

  pass "only expected service ports are published (80/443; SSH ignored if present)"
}

main() {
  note "starting post-install verification"
  require_cmd docker curl ss

  if ! docker info >/dev/null 2>&1; then
    fail "docker daemon is not reachable"
    exit 1
  fi

  check_release_layout
  check_compose_up
  check_postgres
  check_mqtt
  check_api_health
  check_auth_socket
  check_nginx
  check_published_ports

  if [[ "${failures}" -gt 0 ]]; then
    note "${failures} check(s) failed"
    exit 1
  fi

  note "all checks passed"
}

main "$@"
