# terraform-provider-netbox developer targets.
#
# Environment for acceptance tests / demo:
#   make demo-token           # writes .env.demo for https://demo.netbox.dev
#   set -a; . ./.env.demo; set +a
#   make testacc

PROVIDER_NAME ?= netbox
BINARY        ?= terraform-provider-$(PROVIDER_NAME)
GOFLAGS       ?=
TEST_TIMEOUT  ?= 120m
ACC_SHARD     ?= TestAcc

default: build

.PHONY: build
build:
	go build -v $(GOFLAGS) -o $(BINARY) .

.PHONY: install
install:
	go install -v $(GOFLAGS) .

.PHONY: fmt
fmt:
	gofmt -s -w .

# Hand-written packages only: the regenerated client (netbox/, 1,100+ files) and
# the generated resources (internal/provider/gen/) carry "Code generated" headers
# and linting them exhausts CI runner memory without finding anything.
LINT_PKGS ?= . ./internal/acctest/... ./internal/client/... ./internal/conv/... ./internal/customfields/... ./internal/gen/... ./internal/provider ./internal/provider/manual/... ./tools/...

.PHONY: lint
lint:
	golangci-lint run $(LINT_PKGS)

.PHONY: test
test:
	go test ./internal/... -count=1 -timeout 15m $(TESTARGS)

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./internal/... -v -run '$(ACC_SHARD)' -timeout $(TEST_TIMEOUT) $(TESTARGS)

.PHONY: gen
gen:
	go run ./internal/gen

.PHONY: gen-check
gen-check: gen
	git diff --exit-code

.PHONY: client-gen
client-gen:
	./tools/client-gen/generate.sh

.PHONY: docs
docs:
	go tool tfplugindocs generate --provider-name $(PROVIDER_NAME) --rendered-provider-name NetBox

.PHONY: docs-check
docs-check:
	go tool tfplugindocs validate --provider-name $(PROVIDER_NAME)

.PHONY: demo-token
demo-token:
	./scripts/demo-token.sh

.PHONY: docker-up
docker-up:
	docker compose -f docker/docker-compose.yml up -d --wait

.PHONY: docker-down
docker-down:
	docker compose -f docker/docker-compose.yml down -v

.PHONY: sweep
sweep:
	@echo "WARNING: this deletes tfacc-* objects from the NetBox named by NETBOX_SERVER_URL"
	go test ./internal/provider -sweep=all -v -timeout 30m $(SWEEPARGS)

# --- packaging / distribution (docs/INSTALL.md) -----------------------------
GORELEASER    ?= $(shell command -v goreleaser 2>/dev/null || echo "go run github.com/goreleaser/goreleaser/v2@v2.12.5")
TF_PLUGIN_DIR ?= $(HOME)/.terraform.d/plugins
DEV_VERSION   ?= 0.0.0-dev
HOST_OS       := $(shell go env GOOS)
HOST_ARCH     := $(shell go env GOARCH)
MIRROR_ARGS   ?=

# Unsigned snapshot release into dist/: one zip per OS/arch, SHA256SUMS, manifest.
# Each target needs ~4 GB while compiling the generated client; lower
# DIST_PARALLELISM on small machines (CI uses 1).
DIST_PARALLELISM ?= 2
.PHONY: dist
dist:
	$(GORELEASER) release --snapshot --clean --skip=sign,publish --parallelism $(DIST_PARALLELISM)

# Network-mirror tree (index.json, <version>.json, zips) under dist/mirror.
# Pass MIRROR_ARGS='-base-url https://.../' to point at hosted zips instead.
.PHONY: mirror
mirror:
	go run ./tools/mirror-index -dist dist -out dist/mirror $(MIRROR_ARGS)

# Installs dist/mirror through a local HTTPS network mirror and scripts/install.sh.
.PHONY: mirror-check
mirror-check:
	./scripts/mirror-smoke.sh

# Browse dist/mirror at http://localhost:8000/ (Terraform itself requires HTTPS).
.PHONY: serve-mirror
serve-mirror:
	python3 -m http.server -d dist/mirror 8000

# Builds the provider for this machine into Terraform's implied local mirror so
# `source = "elliot/netbox"` with version = "$(DEV_VERSION)" resolves to it
# without any CLI configuration or dev_overrides.
.PHONY: install-local
install-local:
	$(eval DEST := $(TF_PLUGIN_DIR)/registry.terraform.io/elliot/$(PROVIDER_NAME)/$(DEV_VERSION)/$(HOST_OS)_$(HOST_ARCH))
	mkdir -p "$(DEST)"
	go build $(GOFLAGS) -trimpath -ldflags "-s -w -X main.version=$(DEV_VERSION)" -o "$(DEST)/$(BINARY)_v$(DEV_VERSION)" .
	@echo "installed $(DEST)/$(BINARY)_v$(DEV_VERSION)"
	@echo 'use: required_providers { $(PROVIDER_NAME) = { source = "elliot/$(PROVIDER_NAME)", version = "$(DEV_VERSION)" } }'
