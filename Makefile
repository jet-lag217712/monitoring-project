# Appliance release bundle outputs and local build artifacts.
.DEFAULT_GOAL := help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
ARCH ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

.PHONY: help appliance-bundle appliance-bundle-all appliance-package appliance-stage appliance-configure appliance-upgrade appliance-prepare-ova appliance-ova-amd64-ci appliance-publish-azure

help:
	@echo "Appliance release targets:"
	@echo "  make appliance-bundle [ARCH=arm64|amd64] [VERSION=x.y.z]"
	@echo "  make appliance-bundle-all [VERSION=x.y.z]"
	@echo "  make appliance-package [ARCH=...] [VERSION=...]   # .eqa (+ sign if EQUATE_UPDATE_SIGNING_KEY set)"
	@echo "  make appliance-ova-amd64-ci [VERSION=x.y.z]"
	@echo "  make appliance-stage HOST=vm.local [USER=$$USER] ARCH=... VERSION=..."
	@echo "  make appliance-configure BUNDLE=/tmp/equate-staging/bundle VERSION=..."
	@echo "  make appliance-upgrade BUNDLE=/tmp/equate-staging/bundle VERSION=... [CANARY=1]"
	@echo "  make appliance-publish-azure STORAGE_ACCOUNT=... VERSION=... ARCH=... [CHANNEL=stable] [EDITION=standard]"
	@echo "  make appliance-prepare-ova"

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
