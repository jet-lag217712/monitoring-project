#!/usr/bin/env bash
# Publish a signed .eqa and channel manifest to Azure Blob Storage.
#
# Prerequisites:
#   - az CLI logged in with write access to the storage account
#   - Signed package already built:
#       make appliance-package ARCH=amd64 VERSION=1.0.3
#       EQUATE_UPDATE_SIGNING_KEY=... make appliance-package ...
#
# Usage:
#   ./appliance/scripts/publish-update-channel-azure.sh \
#     --storage-account equateupdates \
#     --container updates \
#     --channel stable \
#     --edition standard \
#     --arch amd64 \
#     --version 1.0.3 \
#     [--cdn-base https://equateupdates.blob.core.windows.net/updates]
#
# Layout written under the container:
#   v1/channel/<channel>/manifest.json
#   v1/channel/<channel>/<version>/Equate-<version>-<arch>.eqa
#   v1/channel/<channel>/<version>/Equate-<version>-<arch>.eqa.sha256
#   v1/channel/<channel>/<version>/Equate-<version>-<arch>.eqa.sig
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
STORAGE_ACCOUNT=""
CONTAINER="updates"
CHANNEL="stable"
EDITION="standard"
ARCH=""
VERSION=""
CDN_BASE=""
DIST_DIR="${ROOT}/dist"

usage() {
  sed -n '2,30p' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --storage-account) STORAGE_ACCOUNT="${2:-}"; shift 2 ;;
    --container) CONTAINER="${2:-}"; shift 2 ;;
    --channel) CHANNEL="${2:-}"; shift 2 ;;
    --edition) EDITION="${2:-}"; shift 2 ;;
    --arch) ARCH="${2:-}"; shift 2 ;;
    --version) VERSION="${2:-}"; shift 2 ;;
    --cdn-base) CDN_BASE="${2:-}"; shift 2 ;;
    --dist-dir) DIST_DIR="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "${STORAGE_ACCOUNT}" || -z "${ARCH}" || -z "${VERSION}" ]]; then
  usage >&2
  exit 1
fi
case "${EDITION}" in
  standard|noauth) ;;
  *) echo "--edition must be standard or noauth" >&2; exit 1 ;;
esac
if ! command -v az >/dev/null 2>&1; then
  echo "az CLI is required" >&2
  exit 1
fi

EQA="${DIST_DIR}/Equate-${VERSION}-${ARCH}.eqa"
SHA_FILE="${EQA}.sha256"
SIG_FILE="${EQA}.sig"
if [[ ! -f "${EQA}" || ! -f "${SHA_FILE}" || ! -f "${SIG_FILE}" ]]; then
  echo "missing signed package artifacts under ${DIST_DIR}" >&2
  echo "need: $(basename "${EQA}"), .sha256, .sig" >&2
  exit 1
fi

SHA256="$(awk '{print $1; exit}' "${SHA_FILE}")"
SIG="$(tr -d '[:space:]' < "${SIG_FILE}")"
SIZE_BYTES="$(wc -c < "${EQA}" | tr -d '[:space:]')"

if [[ -z "${CDN_BASE}" ]]; then
  CDN_BASE="https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}"
fi
CDN_BASE="${CDN_BASE%/}"

PREFIX="v1/channel/${CHANNEL}"
ARTIFACT_BLOB="${PREFIX}/${VERSION}/Equate-${VERSION}-${ARCH}.eqa"
ARTIFACT_URL="${CDN_BASE}/${ARTIFACT_BLOB}"

echo "uploading ${EQA} -> ${STORAGE_ACCOUNT}/${CONTAINER}/${ARTIFACT_BLOB}"
az storage blob upload \
  --account-name "${STORAGE_ACCOUNT}" \
  --container-name "${CONTAINER}" \
  --name "${ARTIFACT_BLOB}" \
  --file "${EQA}" \
  --overwrite true \
  --auth-mode login >/dev/null

az storage blob upload \
  --account-name "${STORAGE_ACCOUNT}" \
  --container-name "${CONTAINER}" \
  --name "${ARTIFACT_BLOB}.sha256" \
  --file "${SHA_FILE}" \
  --overwrite true \
  --auth-mode login >/dev/null

az storage blob upload \
  --account-name "${STORAGE_ACCOUNT}" \
  --container-name "${CONTAINER}" \
  --name "${ARTIFACT_BLOB}.sig" \
  --file "${SIG_FILE}" \
  --overwrite true \
  --auth-mode login >/dev/null

MANIFEST_TMP="$(mktemp)"
trap 'rm -f "${MANIFEST_TMP}"' EXIT

EXISTING_MANIFEST="${PREFIX}/manifest.json"
# Merge into existing manifest when present so other arches/versions persist.
EXISTING_JSON=""
if az storage blob download \
  --account-name "${STORAGE_ACCOUNT}" \
  --container-name "${CONTAINER}" \
  --name "${EXISTING_MANIFEST}" \
  --file "${MANIFEST_TMP}.old" \
  --auth-mode login >/dev/null 2>&1; then
  EXISTING_JSON="$(cat "${MANIFEST_TMP}.old")"
  rm -f "${MANIFEST_TMP}.old"
fi

python3 - <<PY >"${MANIFEST_TMP}"
import json, datetime, sys
existing = '''${EXISTING_JSON}'''.strip()
edition = "${EDITION}"
channel = "${CHANNEL}"
version = "${VERSION}"
arch = "${ARCH}"
artifact = "Equate-${VERSION}-${ARCH}.eqa"
url = "${ARTIFACT_URL}"
sha = "${SHA256}"
sig = "${SIG}"
size = int("${SIZE_BYTES}")

if existing:
    doc = json.loads(existing)
else:
    doc = {"channel": channel, "edition": edition, "latest": version, "releases": {}}

if doc.get("edition") and doc["edition"] != edition:
    raise SystemExit(f"refusing to publish edition {edition} into channel with edition {doc['edition']}")

doc["channel"] = channel
doc["edition"] = edition
doc["latest"] = version
releases = doc.setdefault("releases", {})
rel = releases.setdefault(version, {"published_at": datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ"), "architectures": {}})
rel.setdefault("architectures", {})
rel["architectures"][arch] = {
    "artifact": artifact,
    "url": url,
    "sha256": sha,
    "size_bytes": size,
    "signature": sig,
}
releases[version] = rel
json.dump(doc, sys.stdout, indent=2, sort_keys=True)
print()
PY

echo "uploading channel manifest ${EXISTING_MANIFEST}"
az storage blob upload \
  --account-name "${STORAGE_ACCOUNT}" \
  --container-name "${CONTAINER}" \
  --name "${EXISTING_MANIFEST}" \
  --file "${MANIFEST_TMP}" \
  --content-type application/json \
  --overwrite true \
  --auth-mode login >/dev/null

cat <<EOF

Published ${VERSION} (${ARCH}, edition=${EDITION}) to Azure Blob Storage.

  Manifest: ${CDN_BASE}/${EXISTING_MANIFEST}
  Artifact: ${ARTIFACT_URL}

On the appliance (/etc/equate/update-channel.conf):
  channel_url=${CDN_BASE}/${EXISTING_MANIFEST}
  edition=${EDITION}

Then:
  sudo equate upgrade --check
  sudo equate upgrade
EOF
