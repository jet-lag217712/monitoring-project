# Appliance release bundle outputs and local build artifacts.
.DEFAULT_GOAL := help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
ARCH ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

.PHONY: help appliance-bundle appliance-bundle-all appliance-stage appliance-configure appliance-prepare-ova appliance-ova-amd64-ci

help:
	@echo "Appliance release targets:"
	@echo "  make appliance-bundle [ARCH=arm64|amd64] [VERSION=x.y.z]"
	@echo "  make appliance-bundle-all [VERSION=x.y.z]"
	@echo "  make appliance-ova-amd64-ci [VERSION=x.y.z]"
	@echo "  make appliance-stage HOST=vm.local [USER=$$USER] ARCH=... VERSION=..."
	@echo "  make appliance-configure BUNDLE=/tmp/equate-staging/bundle VERSION=..."
	@echo "  make appliance-prepare-ova"

appliance-bundle:
	@./appliance/scripts/build-release.sh --arch "$(ARCH)" --version "$(VERSION)"

appliance-bundle-all:
	@$(MAKE) appliance-bundle ARCH=arm64 VERSION="$(VERSION)"
	@$(MAKE) appliance-bundle ARCH=amd64 VERSION="$(VERSION)"

appliance-ova-amd64-ci:
	@./appliance/scripts/build-ova-amd64-ci.sh --version "$(VERSION)"

appliance-stage:
	@test -n "$(HOST)" || (echo "HOST is required" >&2; exit 1)
	@./appliance/scripts/stage-release.sh --host "$(HOST)" --user "$(USER)" --arch "$(ARCH)" --version "$(VERSION)"

appliance-configure:
	@test -n "$(BUNDLE)" || (echo "BUNDLE is required" >&2; exit 1)
	@sudo ./appliance/scripts/configure-vm.sh --bundle "$(BUNDLE)" --version "$(VERSION)"

appliance-prepare-ova:
	@sudo ./appliance/scripts/prepare-ova.sh
