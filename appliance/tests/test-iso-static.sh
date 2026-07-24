#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

test -z "$(find appliance -name '*.pkr.hcl' -print -quit)"
grep -F 'make iso ARCH=amd64' docs/architecture/equate-appliance.md >/dev/null
grep -F 'Install Equate Appliance - Erases First Disk' appliance/scripts/build-iso >/dev/null
grep -F 'partman-auto/disk' appliance/scripts/build-iso >/dev/null
grep -F 'netcfg/enable boolean false' appliance/scripts/build-iso >/dev/null
grep -F 'docker-compose-linux-' appliance/scripts/build-iso >/dev/null
grep -F 'debian_dvd_mirror' appliance/debian-dvd.lock >/dev/null
grep -F 'xorriso -indev' appliance/scripts/build-iso >/dev/null
grep -F -- '-boot_image any replay' appliance/scripts/build-iso >/dev/null
test ! -e appliance/iso/Dockerfile
test ! -e appliance/iso/build-in-container
grep -F 'compose_arm64_sha256' appliance/compose-plugin.lock >/dev/null
grep -F 'equate-release-load.service' deployments/appliance/systemd/equate-stack.service >/dev/null
if grep -q 'ConditionPathExists=!/var/lib/equate/.initialized' deployments/appliance/systemd/equate-init.service; then
  echo 'runtime secret rendering must run on every boot' >&2
  exit 1
fi
grep -F 'CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH)' Makefile >/dev/null
grep -F 'VERSION ?= 2.0.0' Makefile >/dev/null
grep -F 'EQUATE_GOOGLE_CLIENT_ID is required for appliance releases' Makefile >/dev/null
grep -F 'docker buildx build' Makefile >/dev/null
grep -F 'postgres@sha256:' appliance/image-lock.mk >/dev/null
grep -F '@sha256:' frontend/Dockerfile >/dev/null
grep -F 'debian_version=12.10.0' appliance/debian-dvd.lock >/dev/null
grep -F 'AuthenticationMethods publickey' deployments/appliance/systemd/50-equate-ssh.conf >/dev/null
grep -F 'Subsystem sftp /bin/false' deployments/appliance/systemd/50-equate-ssh.conf >/dev/null
grep -F 'DHCP=yes' deployments/appliance/systemd/20-equate-dhcp.network >/dev/null
grep -F 'systemd-networkd.service' appliance/scripts/install-target >/dev/null
grep -F 'allowed_domains: []' deployments/appliance/configs/application.yaml >/dev/null
grep -F 'google_session' deployments/appliance/configs/application.yaml >/dev/null
grep -F 'EQUATE_TLS' services/snmp-collector/internal/appliance/tls_import.go >/dev/null
grep -F 'snmp.community' appliance/scripts/equate-init >/dev/null
grep -F '/run/equate/rendered/certificates' deployments/appliance/compose.yaml >/dev/null
[[ "$(rg -c '^      - "[0-9]+:[0-9]+"$' deployments/appliance/compose.yaml)" == '2' ]]
grep -F '      - "443:443"' deployments/appliance/compose.yaml >/dev/null
grep -F '      - "80:80"' deployments/appliance/compose.yaml >/dev/null
test ! -e deployments/appliance/configs/tls.key
test ! -e deployments/appliance/configs/tls.crt
if grep -q '^d-i pkgsel/include.*cloud-init' appliance/scripts/build-iso; then
  echo 'the ISO must not embed cloud-init' >&2
  exit 1
fi
if grep -q 'Administrator password' services/snmp-collector/internal/appliance/console.go; then
  echo 'appliance must not expose local-password setup' >&2
  exit 1
fi
if rg -q 'appliance\.yaml' deployments/appliance appliance/scripts services/snmp-collector/internal/appliance; then
  echo 'appliance must not retain a second general configuration surface' >&2
  exit 1
fi
