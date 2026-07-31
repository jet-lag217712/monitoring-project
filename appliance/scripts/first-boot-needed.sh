#!/usr/bin/env bash
# ExecCondition helper: exit 0 when first-boot setup should run on the console.
set -euo pipefail

DEPLOY_FILE="/etc/equate/deploy-dir"
if [[ ! -f "${DEPLOY_FILE}" ]]; then
  exit 1
fi
DEPLOY_DIR="$(tr -d '[:space:]' < "${DEPLOY_FILE}")"
if [[ -z "${DEPLOY_DIR}" || ! -d "${DEPLOY_DIR}" ]]; then
  exit 1
fi
if [[ -f "${DEPLOY_DIR}/.setup-complete" ]]; then
  exit 1
fi
exit 0
