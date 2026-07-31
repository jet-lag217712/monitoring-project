#!/usr/bin/env bash
# One-time Azure Blob setup for a public-read Equate update channel.
#
# Prerequisites: az CLI logged in; storage account already created.
#
# Usage:
#   ./appliance/scripts/setup-update-channel-azure.sh \
#     --storage-account <account> \
#     [--resource-group <rg>] \
#     [--container updates]
set -euo pipefail

STORAGE_ACCOUNT=""
RESOURCE_GROUP=""
CONTAINER="updates"

usage() {
  sed -n '2,14p' "$0"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --storage-account) STORAGE_ACCOUNT="${2:-}"; shift 2 ;;
    --resource-group) RESOURCE_GROUP="${2:-}"; shift 2 ;;
    --container) CONTAINER="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "${STORAGE_ACCOUNT}" ]]; then
  usage >&2
  exit 1
fi
if ! command -v az >/dev/null 2>&1; then
  echo "az CLI is required" >&2
  exit 1
fi

if [[ -z "${RESOURCE_GROUP}" ]]; then
  RESOURCE_GROUP="$(az storage account show --name "${STORAGE_ACCOUNT}" --query resourceGroup -o tsv)"
fi

echo "Enabling allowBlobPublicAccess on ${STORAGE_ACCOUNT} (${RESOURCE_GROUP})..."
az storage account update \
  --name "${STORAGE_ACCOUNT}" \
  --resource-group "${RESOURCE_GROUP}" \
  --allow-blob-public-access true >/dev/null

echo "Creating container '${CONTAINER}' with public blob read (anonymous)..."
# --public-access blob = anonymous read of blobs; listing the container is not public.
az storage container create \
  --account-name "${STORAGE_ACCOUNT}" \
  --name "${CONTAINER}" \
  --public-access blob \
  --auth-mode login >/dev/null 2>&1 || \
az storage container set-permission \
  --account-name "${STORAGE_ACCOUNT}" \
  --name "${CONTAINER}" \
  --public-access blob \
  --auth-mode login >/dev/null

BASE="https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}"
cat <<EOF

Azure update channel ready (public blob read).

  Storage account: ${STORAGE_ACCOUNT}
  Container:       ${CONTAINER}
  Public base URL: ${BASE}

Next:
  1. Create an Entra app + federated credential for GitHub Actions OIDC
     (see docs/releases/appliance-updates.md).
  2. Grant that identity 'Storage Blob Data Contributor' on the storage account.
  3. Set GitHub secrets/variables and run workflow
     'Publish appliance update channel (.eqa → Azure)'.

Appliance channel_url (after first publish):
  ${BASE}/v1/channel/stable/manifest.json
EOF
