VERSION ?= 2.0.0
ARCH ?= amd64
EQUATE_GOOGLE_CLIENT_ID ?=
DIST_DIR := dist
BUILD_DIR := build/appliance
RELEASE_DIR := $(BUILD_DIR)/release
ISO_NAME := Equate-Appliance-$(VERSION)-$(ARCH).iso
include appliance/image-lock.mk

.PHONY: iso iso-all appliance-release appliance-check iso-check test-appliance-iso update-package clean-appliance

# Produces a self-contained, UEFI-only Debian installer ISO. The build host
# needs Docker Desktop/Buildx with the requested Linux platform available.
iso: appliance-release appliance-check iso-check
	./appliance/scripts/build-iso $(BUILD_DIR) $(DIST_DIR) $(VERSION) $(ARCH)
	shasum -a 256 $(DIST_DIR)/$(ISO_NAME) > $(DIST_DIR)/$(ISO_NAME).sha256
	./appliance/scripts/verify-iso $(DIST_DIR)/$(ISO_NAME) $(ARCH) $(VERSION)

# The two images are intentionally built serially because each owns a complete
# architecture-specific release bundle under build/appliance/release.
iso-all:
	$(MAKE) iso ARCH=amd64 VERSION=$(VERSION)
	$(MAKE) iso ARCH=arm64 VERSION=$(VERSION)

appliance-release:
	@test "$(ARCH)" = "amd64" || test "$(ARCH)" = "arm64"
	@test -n "$(EQUATE_GOOGLE_CLIENT_ID)" || { echo "EQUATE_GOOGLE_CLIENT_ID is required for appliance releases" >&2; exit 2; }
	rm -rf $(RELEASE_DIR) $(BUILD_DIR)/equate
	rm -f $(BUILD_DIR)/collector
	mkdir -p $(RELEASE_DIR)/images $(RELEASE_DIR)/migrations $(BUILD_DIR)
	cd services/snmp-collector && CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o ../../$(BUILD_DIR)/equate ./cmd/equate
	cd services/snmp-collector && CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o ../../$(BUILD_DIR)/collector ./cmd/collector
	./appliance/scripts/verify-linux-binary $(BUILD_DIR)/equate $(ARCH)
	./appliance/scripts/verify-linux-binary $(BUILD_DIR)/collector $(ARCH)
	docker buildx build --platform=linux/$(ARCH) --load -t equate-api:$(VERSION)-$(ARCH) ./services/backend-api
	docker buildx build --platform=linux/$(ARCH) --load -t equate-ui:$(VERSION)-$(ARCH) ./frontend
	docker buildx build --platform=linux/$(ARCH) --load -t equate-core:$(VERSION)-$(ARCH) -f services/snmp-collector/Dockerfile.core .
	docker pull --platform=linux/$(ARCH) $(POSTGRES_IMAGE)
	docker pull --platform=linux/$(ARCH) $(MIGRATE_IMAGE)
	docker save -o $(RELEASE_DIR)/images/equate-api.tar equate-api:$(VERSION)-$(ARCH)
	docker save -o $(RELEASE_DIR)/images/equate-ui.tar equate-ui:$(VERSION)-$(ARCH)
	docker save -o $(RELEASE_DIR)/images/equate-core.tar equate-core:$(VERSION)-$(ARCH)
	docker save -o $(RELEASE_DIR)/images/postgres.tar $(POSTGRES_IMAGE)
	docker save -o $(RELEASE_DIR)/images/migrate.tar $(MIGRATE_IMAGE)
	cp deployments/appliance/compose.yaml $(RELEASE_DIR)/compose.yaml
	cp deployments/appliance/nginx.conf $(RELEASE_DIR)/nginx.conf
	sed 's|__EQUATE_GOOGLE_CLIENT_ID__|$(EQUATE_GOOGLE_CLIENT_ID)|g' deployments/appliance/configs/application.yaml > $(RELEASE_DIR)/application.yaml
	cp database/migrations/* $(RELEASE_DIR)/migrations/
	printf '%s\n' $(VERSION) > $(RELEASE_DIR)/VERSION
	printf 'EQUATE_API_IMAGE=equate-api:%s-%s\nEQUATE_UI_IMAGE=equate-ui:%s-%s\nEQUATE_CORE_IMAGE=equate-core:%s-%s\nEQUATE_POSTGRES_IMAGE=%s\nEQUATE_MIGRATE_IMAGE=%s\n' $(VERSION) $(ARCH) $(VERSION) $(ARCH) $(VERSION) $(ARCH) '$(POSTGRES_IMAGE)' '$(MIGRATE_IMAGE)' > $(RELEASE_DIR)/release.env
	printf '{"version":"%s","architecture":"%s","format":1,"inputs":{"postgres":"%s","migrate":"%s","debian_installer":"12.10.0","debian_snapshot":"20250320T000000Z"}}\n' $(VERSION) $(ARCH) '$(POSTGRES_IMAGE)' '$(MIGRATE_IMAGE)' > $(RELEASE_DIR)/manifest.json
	cd $(RELEASE_DIR) && find . -type f ! -name checksums.txt -print0 | sort -z | xargs -0 shasum -a 256 > checksums.txt

# The Ed25519 key path is supplied only by the release environment. All online,
# mirror, and virtual-media updates use this same signed artifact.
update-package: appliance-release
	test -n "$$EQUATE_SIGNING_KEY"
	mkdir -p $(DIST_DIR)
	./appliance/scripts/build-update-package $(RELEASE_DIR) $(DIST_DIR)/Equate-$(VERSION).eqa "$$EQUATE_SIGNING_KEY"

appliance-check:
	command -v docker >/dev/null
	docker info >/dev/null

iso-check:
	command -v docker >/dev/null
	command -v xorriso >/dev/null
	command -v curl >/dev/null

test-appliance-iso:
	bash appliance/tests/test-iso-static.sh
	bash appliance/tests/test-release-loader.sh

clean-appliance:
	rm -rf $(BUILD_DIR) $(DIST_DIR)
