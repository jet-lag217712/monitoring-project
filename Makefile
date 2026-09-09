# Appliance development (Equate-Appliance VM) and production release targets.
.DEFAULT_GOAL := help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
ARCH ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

.PHONY: help test \
	appliance-bundle appliance-bundle-all appliance-package appliance-stage \
	appliance-configure appliance-upgrade appliance-prepare-ova appliance-ova-amd64-ci \
	appliance-publish-azure appliance-generate-keys appliance-setup-azure-channel \
	appliance-package-ova appliance-export-ova-arm64 appliance-verify appliance-verify-ova \
	lab-validate lab-appliance-setup lab-appliance-upgrade \
	db-migrate db-bootstrap-roles mqtt-dev-certs

help:
	@echo "Develop (Equate-Appliance VM from source):"
	@echo "  make appliance-bundle [ARCH=arm64|amd64] [VERSION=x.y.z]"
	@echo "  make appliance-stage HOST=vm.local [USER=$$USER] ARCH=... VERSION=..."
	@echo "  make appliance-configure BUNDLE=/tmp/equate-staging/bundle VERSION=..."
	@echo "  make appliance-upgrade BUNDLE=/tmp/equate-staging/bundle VERSION=... [CANARY=1]"
	@echo "  make appliance-verify"
	@echo "  make test [QUICK=1]"
	@echo
	@echo "Lab (GNS3 + appliance VM):"
	@echo "  make lab-validate"
	@echo "  make lab-appliance-setup"
	@echo "  make lab-appliance-upgrade OLD_VERSION=... NEW_VERSION=... BUNDLE=..."
	@echo
	@echo "Release:"
	@echo "  make appliance-bundle-all [VERSION=x.y.z]"
	@echo "  make appliance-package [ARCH=...] [VERSION=...]   # .eqa (+ sign if EQUATE_UPDATE_SIGNING_KEY set)"
	@echo "  make appliance-ova-amd64-ci [VERSION=x.y.z]"
	@echo "  make appliance-prepare-ova"
	@echo "  make appliance-publish-azure STORAGE_ACCOUNT=... VERSION=... ARCH=... [CHANNEL=stable] [EDITION=standard]"
	@echo "  make appliance-generate-keys [KEYS_DIR=...]"
	@echo "  make appliance-setup-azure-channel STORAGE_ACCOUNT=..."
	@echo
	@echo "Engineer helpers:"
	@echo "  make db-migrate DATABASE_URL=..."
	@echo "  make db-bootstrap-roles DATABASE_URL=..."
	@echo "  make mqtt-dev-certs"

test:
	@./deployments/test.sh $(if $(QUICK),--quick,)

appliance-bundle:
	@./appliance/scripts/build-release.sh --arch "$(ARCH)" --version "$(VERSION)"

appliance-bundle-all:
	@$(MAKE) appliance-bundle ARCH=arm64 VERSION="$(VERSION)"
	@$(MAKE) appliance-bundle ARCH=amd64 VERSION="$(VERSION)"

appliance-package:
	@./appliance/scripts/package-eqa.sh --arch "$(ARCH)" --version "$(VERSION)"

appliance-ova-amd64-ci:
	@./appliance/scripts/build-ova-amd64-ci.sh --version "$(VERSION)"

appliance-stage:
	@test -n "$(HOST)" || (echo "HOST is required" >&2; exit 1)
	@./appliance/scripts/stage-release.sh --host "$(HOST)" --user "$(USER)" --arch "$(ARCH)" --version "$(VERSION)"

appliance-configure:
	@test -n "$(BUNDLE)" || (echo "BUNDLE is required" >&2; exit 1)
	@sudo ./appliance/scripts/configure-vm.sh --bundle "$(BUNDLE)" --version "$(VERSION)"

appliance-upgrade:
	@test -n "$(BUNDLE)" || (echo "BUNDLE is required" >&2; exit 1)
	@sudo ./appliance/scripts/configure-vm.sh --upgrade --bundle "$(BUNDLE)" --version "$(VERSION)" $(if $(CANARY),--canary,)

appliance-publish-azure:
	@test -n "$(STORAGE_ACCOUNT)" || (echo "STORAGE_ACCOUNT is required" >&2; exit 1)
	@./appliance/scripts/publish-update-channel-azure.sh \
		--storage-account "$(STORAGE_ACCOUNT)" \
		--container "$(or $(CONTAINER),updates)" \
		--channel "$(or $(CHANNEL),stable)" \
		--edition "$(or $(EDITION),standard)" \
		--arch "$(ARCH)" \
		--version "$(VERSION)" \
		$(if $(CDN_BASE),--cdn-base "$(CDN_BASE)",)

appliance-prepare-ova:
	@sudo ./appliance/scripts/prepare-ova.sh

appliance-generate-keys:
	@./appliance/scripts/generate-update-keys.sh $(if $(KEYS_DIR),--out-dir "$(KEYS_DIR)",)

appliance-setup-azure-channel:
	@test -n "$(STORAGE_ACCOUNT)" || (echo "STORAGE_ACCOUNT is required" >&2; exit 1)
	@./appliance/scripts/setup-update-channel-azure.sh --storage-account "$(STORAGE_ACCOUNT)"

appliance-package-ova:
	@test -n "$(VMDK)" || (echo "VMDK is required" >&2; exit 1)
	@./appliance/scripts/package-ova.sh --arch "$(ARCH)" --version "$(VERSION)" --vmdk "$(VMDK)"

appliance-export-ova-arm64:
	@test -n "$(VMX)" || (echo "VMX is required" >&2; exit 1)
	@./appliance/scripts/export-arm64-ova.sh --vmx "$(VMX)" --version "$(VERSION)"

appliance-verify:
	@sudo ./appliance/scripts/verify-appliance.sh

appliance-verify-ova:
	@sudo ./appliance/scripts/verify-ova-import.sh $(if $(ARTIFACT),--artifact "$(ARTIFACT)",)

lab-validate:
	@./remote-server/validate-lab.sh

lab-appliance-setup:
	@sudo ./appliance/scripts/e2e-appliance-setup.sh

lab-appliance-upgrade:
	@test -n "$(OLD_VERSION)" || (echo "OLD_VERSION is required" >&2; exit 1)
	@test -n "$(NEW_VERSION)" || (echo "NEW_VERSION is required" >&2; exit 1)
	@test -n "$(BUNDLE)" || (echo "BUNDLE is required" >&2; exit 1)
	@sudo OLD_VERSION="$(OLD_VERSION)" NEW_VERSION="$(NEW_VERSION)" BUNDLE="$(BUNDLE)" \
		./appliance/scripts/e2e-appliance-upgrade.sh

db-migrate:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" >&2; exit 1)
	@./infrastructure/script/migrate.sh up

db-bootstrap-roles:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" >&2; exit 1)
	@./infrastructure/script/bootstrap-db-roles.sh

mqtt-dev-certs:
	@./infrastructure/docker/mqtt-broker/scripts/gen-dev-certs.sh
	@./infrastructure/docker/mqtt-broker/scripts/gen-passwords.sh
