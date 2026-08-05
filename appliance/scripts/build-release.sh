#!/usr/bin/env bash
# Build pinned container images and assemble an immutable offline appliance bundle.
#
# Usage:
#   ./appliance/scripts/build-release.sh --arch arm64 --version 1.0.0
#   ./appliance/scripts/build-release.sh --arch amd64 --version 1.0.0
#
# Output:
#   dist/appliance-<arch>-<version>/
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ARCH=""
VERSION=""
POSTGRES_REF="postgres:16-alpine"
MIGRATE_REF="migrate/migrate:v4.18.1"

usage() {
  cat <<'EOF'
usage: build-release.sh --arch <arm64|amd64> --version <semver>

Builds appliance container images for the requested architecture, exports image
tarballs with pinned digests, and packages an offline release bundle under dist/.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch)
      ARCH="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "${ARCH}" in
  arm64|amd64) ;;
  *)
    echo "--arch must be arm64 or amd64" >&2
    exit 1
    ;;
esac

if [[ -z "${VERSION}" ]]; then
  echo "--version is required" >&2
  exit 1
fi

require_cmd() {
  local cmd
  for cmd in "$@"; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      echo "required command not found: ${cmd}" >&2
      exit 1
    fi
  done
}

require_cmd docker git go

# #region agent log
_debug_log() {
  local hypothesis_id="$1" location="$2" message="$3" data="$4"
  printf '{"sessionId":"11e234","hypothesisId":"%s","location":"%s","message":"%s","data":%s,"timestamp":%s}\n' \
    "$hypothesis_id" "$location" "$message" "$data" "$(($(date +%s) * 1000))" \
    >> "${ROOT}/.cursor/debug-11e234.log" 2>/dev/null || true
}
# #endregion

docker_config_uses_osxkeychain() {
  local config="${HOME}/.docker/config.json"
  [[ -f "${config}" ]] || return 1
  python3 - <<'PY' "${config}"
import json, sys
try:
    cfg = json.load(open(sys.argv[1]))
except (OSError, json.JSONDecodeError):
    sys.exit(1)
store = cfg.get("credsStore") or cfg.get("credStore")
sys.exit(0 if store == "osxkeychain" else 1)
PY
}

ensure_docker_credential_path() {
  local os_name dir cred resolved
  os_name="$(uname -s)"
  # #region agent log
  _debug_log "A" "build-release.sh:ensure_docker_credential_path" "credential check entry" "{\"os\":\"${os_name}\"}"
  # #endregion

  if [[ "${os_name}" != "Darwin" ]]; then
    # #region agent log
    _debug_log "A" "build-release.sh:ensure_docker_credential_path" "skipped on non-macOS" "{\"os\":\"${os_name}\"}"
    # #endregion
    return 0
  fi

  if ! docker_config_uses_osxkeychain; then
    # #region agent log
    _debug_log "B" "build-release.sh:ensure_docker_credential_path" "skipped; docker config does not use osxkeychain" "{}"
    # #endregion
    return 0
  fi

  resolved="$(command -v docker-credential-osxkeychain 2>/dev/null || true)"
  if [[ -n "${resolved}" ]] && [[ -x "${resolved}" ]]; then
    # #region agent log
    _debug_log "C" "build-release.sh:ensure_docker_credential_path" "credential helper already in PATH" "{\"path\":\"${resolved}\"}"
    # #endregion
    return 0
  fi
  local candidates=(
    "/Applications/Docker.app/Contents/Resources/bin"
    "${HOME}/.orbstack/bin"
    "/Applications/OrbStack.app/Contents/MacOS/xbin"
  )
  for dir in "${candidates[@]}"; do
    cred="${dir}/docker-credential-osxkeychain"
    if [[ -x "${cred}" ]]; then
      export PATH="${dir}:${PATH}"
      # #region agent log
      _debug_log "C" "build-release.sh:ensure_docker_credential_path" "credential helper added to PATH" "{\"path\":\"${cred}\"}"
      # #endregion
      return 0
    fi
  done
  # #region agent log
  _debug_log "D" "build-release.sh:ensure_docker_credential_path" "credential helper missing on macOS" "{}"
  # #endregion
  echo "docker-credential-osxkeychain not found in PATH (broken OrbStack symlink or missing Docker Desktop)" >&2
  echo "fix: reinstall Docker Desktop, restore OrbStack, or remove credsStore from ~/.docker/config.json" >&2
  exit 1
}

ensure_docker_credential_path

if ! docker buildx version >/dev/null 2>&1; then
  echo "docker buildx is required" >&2
  exit 1
fi

PLATFORM="linux/${ARCH}"
TAG_SUFFIX="${VERSION}-${ARCH}"
BUNDLE_DIR="${ROOT}/dist/appliance-${ARCH}-${VERSION}"
IMAGES_DIR="${BUNDLE_DIR}/images"
DEPLOY_SRC="${ROOT}/deployments/production/appliance"
SCRIPTS_SRC="${ROOT}/appliance/scripts"

rm -rf "${BUNDLE_DIR}"
mkdir -p "${IMAGES_DIR}"

build_local_image() {
  local name="$1"
  local context="$2"
  shift 2
  local tag="equate-${name}:${TAG_SUFFIX}"
  echo "building ${tag} (${PLATFORM})..." >&2
  if ! docker buildx build \
    --platform "${PLATFORM}" \
    --tag "${tag}" \
    --load \
    "$@" \
    "${context}"; then
    echo "docker build failed for ${tag}" >&2
    return 1
  fi
  printf '%s\n' "${tag}"
}

echo "building application images..."
MOSQUITTO_TAG="$(build_local_image mosquitto "${ROOT}/infrastructure/docker/mqtt-broker")"
INGESTION_TAG="$(build_local_image ingestion "${ROOT}/services/ingestion-service")"
BACKEND_API_TAG="$(build_local_image backend-api "${ROOT}/services/backend-api")"
FRONTEND_TAG="$(build_local_image frontend "${ROOT}/frontend" \
  --build-arg "VITE_API_BASE_URL=/api" \
  --build-arg "VITE_AUTH_MODE=appliance_local" \
  --build-arg "VITE_GOOGLE_CLIENT_ID=" \
  --build-arg "VITE_DEMO_ENABLED=false" \
  --build-arg "VITE_APP_VERSION=${VERSION}")"
COLLECTOR_TAG="$(build_local_image snmp-collector "${ROOT}/services/snmp-collector")"

echo "building host CLI binaries (${ARCH})..."
mkdir -p "${BUNDLE_DIR}/bin"
GIT_SHORT="$(git -C "${ROOT}" rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILT_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
LDFLAGS="-s -w -X main.buildVersion=${VERSION} -X main.buildGitCommit=${GIT_SHORT} -X main.buildTime=${BUILT_AT}"
(
  cd "${ROOT}/services/snmp-collector"
  if ! CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" \
    go build -ldflags "${LDFLAGS}" -o "${BUNDLE_DIR}/bin/collector" ./cmd/collector; then
    echo "failed to build collector CLI for linux/${ARCH}" >&2
    exit 1
  fi
  if ! CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" \
    go build -ldflags "${LDFLAGS}" -o "${BUNDLE_DIR}/bin/equate" ./cmd/equate; then
    echo "failed to build equate CLI for linux/${ARCH}" >&2
    exit 1
  fi
)

echo "pulling pinned third-party images (${PLATFORM})..."
BUSYBOX_REF="busybox:1.36"
docker pull --platform "${PLATFORM}" "${POSTGRES_REF}"
docker pull --platform "${PLATFORM}" "${MIGRATE_REF}"
docker pull --platform "${PLATFORM}" "${BUSYBOX_REF}"

pin_image() {
  local ref="$1"
  local pinned
  if ! pinned="$(docker image inspect --format '{{if gt (len .RepoDigests) 0}}{{index .RepoDigests 0}}{{else}}{{.Id}}{{end}}' "${ref}" 2>/dev/null)"; then
    echo "unable to resolve digest for ${ref}" >&2
    exit 1
  fi
  if [[ -z "${pinned}" ]]; then
    echo "unable to resolve digest for ${ref}" >&2
    exit 1
  fi
  printf '%s' "${pinned}"
}

PINNED_POSTGRES="$(pin_image "${POSTGRES_REF}")"
PINNED_MIGRATE="$(pin_image "${MIGRATE_REF}")"
PINNED_BUSYBOX="$(pin_image "${BUSYBOX_REF}")"
RELEASE_POSTGRES="${POSTGRES_REF}"
RELEASE_MIGRATE="${MIGRATE_REF}"
RELEASE_BUSYBOX="${BUSYBOX_REF}"
# Local images are saved and referenced by build tags so docker load restores usable names.
RELEASE_MOSQUITTO="${MOSQUITTO_TAG}"
RELEASE_INGESTION="${INGESTION_TAG}"
RELEASE_BACKEND_API="${BACKEND_API_TAG}"
RELEASE_FRONTEND="${FRONTEND_TAG}"
RELEASE_SNMP_COLLECTOR="${COLLECTOR_TAG}"
DIGEST_MOSQUITTO="$(pin_image "${MOSQUITTO_TAG}")"
DIGEST_INGESTION="$(pin_image "${INGESTION_TAG}")"
DIGEST_BACKEND_API="$(pin_image "${BACKEND_API_TAG}")"
DIGEST_FRONTEND="$(pin_image "${FRONTEND_TAG}")"
DIGEST_SNMP_COLLECTOR="$(pin_image "${COLLECTOR_TAG}")"

save_image() {
  local key="$1"
  local ref="$2"
  local out="${IMAGES_DIR}/${key}.tar"
  echo "exporting ${ref} -> ${out}"
  docker save -o "${out}" "${ref}"
}

save_image postgres "${RELEASE_POSTGRES}"
save_image migrate "${RELEASE_MIGRATE}"
save_image busybox "${RELEASE_BUSYBOX}"
save_image mosquitto "${RELEASE_MOSQUITTO}"
save_image ingestion "${RELEASE_INGESTION}"
save_image backend-api "${RELEASE_BACKEND_API}"
save_image frontend "${RELEASE_FRONTEND}"
save_image snmp-collector "${RELEASE_SNMP_COLLECTOR}"

echo "copying release assets..."
cp "${DEPLOY_SRC}/docker-compose.release.yml" "${BUNDLE_DIR}/docker-compose.yml"
cp "${DEPLOY_SRC}/nginx-frontend.conf" "${BUNDLE_DIR}/nginx-frontend.conf"
cp "${DEPLOY_SRC}/bootstrapper.sh" "${BUNDLE_DIR}/bootstrapper.sh"
cp "${DEPLOY_SRC}/.env.example" "${BUNDLE_DIR}/.env.example"
chmod 0755 "${BUNDLE_DIR}/bootstrapper.sh"
mkdir -p "${BUNDLE_DIR}/configs"
cp "${DEPLOY_SRC}/configs/"*.yaml "${BUNDLE_DIR}/configs/"
cp -R "${ROOT}/database/migrations" "${BUNDLE_DIR}/migrations"
mkdir -p "${BUNDLE_DIR}/scripts"
cp "${SCRIPTS_SRC}/auth-broker.sh" "${BUNDLE_DIR}/scripts/auth-broker.sh"
cp "${SCRIPTS_SRC}/configure-vm.sh" "${BUNDLE_DIR}/scripts/configure-vm.sh"
cp "${SCRIPTS_SRC}/bootstrap-appliance-rendered.sh" "${BUNDLE_DIR}/scripts/bootstrap-appliance-rendered.sh"
cp "${SCRIPTS_SRC}/sync-db-role-passwords.sh" "${BUNDLE_DIR}/scripts/sync-db-role-passwords.sh"
cp "${SCRIPTS_SRC}/prepare-ova.sh" "${BUNDLE_DIR}/scripts/prepare-ova.sh"
cp "${SCRIPTS_SRC}/equate-auth-broker.service" "${BUNDLE_DIR}/scripts/equate-auth-broker.service"
for fb in first-boot-needed.sh first-boot-console.sh equate-first-boot.service getty-tty1-override.conf equate-appliance.sudoers; do
  if [[ -f "${SCRIPTS_SRC}/${fb}" ]]; then
    cp "${SCRIPTS_SRC}/${fb}" "${BUNDLE_DIR}/scripts/${fb}"
  fi
done
if [[ -f "${DEPLOY_SRC}/scripts/manage-users.sh" ]]; then
  cp "${DEPLOY_SRC}/scripts/manage-users.sh" "${BUNDLE_DIR}/scripts/manage-users.sh"
fi
if [[ -f "${DEPLOY_SRC}/scripts/sync-site-topology.sh" ]]; then
  cp "${DEPLOY_SRC}/scripts/sync-site-topology.sh" "${BUNDLE_DIR}/scripts/sync-site-topology.sh"
fi
if [[ -f "${ROOT}/appliance/scripts/debug-agent-log.sh" ]]; then
  cp "${ROOT}/appliance/scripts/debug-agent-log.sh" "${BUNDLE_DIR}/scripts/debug-agent-log.sh"
fi
if [[ -f "${DEPLOY_SRC}/scripts/post-configure.sh" ]]; then
  cp "${DEPLOY_SRC}/scripts/post-configure.sh" "${BUNDLE_DIR}/scripts/post-configure.sh"
fi
if [[ -f "${DEPLOY_SRC}/scripts/manifest-utils.sh" ]]; then
  cp "${DEPLOY_SRC}/scripts/manifest-utils.sh" "${BUNDLE_DIR}/scripts/manifest-utils.sh"
fi
chmod 0755 "${BUNDLE_DIR}/scripts/"*.sh

cat >"${BUNDLE_DIR}/docker-compose.sites.generated.yml" <<'EOF'
# Generated during appliance setup — do not edit by hand.
services: {}
volumes: {}
EOF

cat >"${BUNDLE_DIR}/release.env" <<EOF
COMPOSE_PROJECT_NAME=equate-appliance
EQUATE_VERSION=${VERSION}
EQUATE_ARCH=${ARCH}
EQUATE_PULL_POLICY=never
EQUATE_POSTGRES_IMAGE=${RELEASE_POSTGRES}
EQUATE_MIGRATE_IMAGE=${RELEASE_MIGRATE}
EQUATE_MOSQUITTO_IMAGE=${RELEASE_MOSQUITTO}
EQUATE_INGESTION_IMAGE=${RELEASE_INGESTION}
EQUATE_BACKEND_API_IMAGE=${RELEASE_BACKEND_API}
EQUATE_FRONTEND_IMAGE=${RELEASE_FRONTEND}
EQUATE_COLLECTOR_IMAGE=${RELEASE_SNMP_COLLECTOR}
EQUATE_BUSYBOX_IMAGE=${RELEASE_BUSYBOX}
EQUATE_POSTGRES_DATA=/var/lib/equate/postgres
EQUATE_MIGRATIONS_DIR=./migrations
EQUATE_MQTT_CERTS=/run/equate/rendered/mqtt/certs
EQUATE_MQTT_PASSWORDS=/run/equate/rendered/mqtt/passwords
EQUATE_MQTT_CA=/run/equate/rendered/mqtt/certs/ca.crt
EQUATE_AUTH_SOCKET_DIR=/run/equate
EQUATE_UI_CERTS=/run/equate/rendered/certificates
EOF

GIT_REV="$(git -C "${ROOT}" rev-parse HEAD 2>/dev/null || echo unknown)"
BUILT_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cat >"${BUNDLE_DIR}/image-digests.txt" <<EOF
postgres=${PINNED_POSTGRES}
migrate=${PINNED_MIGRATE}
mosquitto=${DIGEST_MOSQUITTO}
ingestion=${DIGEST_INGESTION}
backend-api=${DIGEST_BACKEND_API}
frontend=${DIGEST_FRONTEND}
snmp-collector=${DIGEST_SNMP_COLLECTOR}
busybox=${PINNED_BUSYBOX}
EOF

python3 - <<PY >"${BUNDLE_DIR}/manifest.json"
import json
print(json.dumps({
    "format": 2,
    "version": "${VERSION}",
    "architecture": "${ARCH}",
    "built_at": "${BUILT_AT}",
    "source_revision": "${GIT_REV}",
    "images": {
        "postgres": "${PINNED_POSTGRES}",
        "migrate": "${PINNED_MIGRATE}",
        "mosquitto": "${DIGEST_MOSQUITTO}",
        "ingestion": "${DIGEST_INGESTION}",
        "backend-api": "${DIGEST_BACKEND_API}",
        "frontend": "${DIGEST_FRONTEND}",
        "snmp-collector": "${DIGEST_SNMP_COLLECTOR}",
        "busybox": "${PINNED_BUSYBOX}",
    },
}, indent=2, sort_keys=True))
PY

if command -v syft >/dev/null 2>&1; then
  syft packages "dir:${BUNDLE_DIR}" -o spdx-json >"${BUNDLE_DIR}/sbom.spdx.json"
else
  cat >"${BUNDLE_DIR}/sbom.spdx.json" <<EOF
{"spdxVersion":"SPDX-2.3","name":"equate-appliance-${VERSION}-${ARCH}","packages":[]}
EOF
fi

checksum_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    shasum -a 256 "$@"
  fi
}

(
  cd "${BUNDLE_DIR}"
  find . -type f ! -name checksums.txt | sort | while read -r path; do
    checksum_file "${path#./}"
  done
) >"${BUNDLE_DIR}/checksums.txt"

echo "${VERSION}" >"${BUNDLE_DIR}/VERSION"
echo "release bundle ready: ${BUNDLE_DIR}"
