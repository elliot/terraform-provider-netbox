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

# The generated client (netbox/, ~1,100 files) compiles as one package and the Go
# compiler needs ~5.9 GB for it at the default GOGC=100 (measured, cold cache).
# GOGC=25 cuts that to ~4 GB at no wall-time cost, and build-client below compiles
# it once, alone, so other packages are not compiled next to it. Override with
# e.g. `make test GOGC=100`; on small machines also pass GOFLAGS=-p=1.
export GOGC ?= 25

default: build

# Compile the generated client on its own first; the result lands in the build
# cache, so the build/test/docs steps below never recompile it concurrently with
# other packages.
.PHONY: build-client
build-client:
	go build $(GOFLAGS) ./netbox/

.PHONY: build
build: build-client
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
LINT_PKGS ?= . ./internal/acctest/... ./internal/client/... ./internal/conv/... ./internal/customfields/... ./internal/gen/... ./internal/provider ./internal/provider/manual/... ./internal/upgrade/... ./tools/...

# golangci-lint is pinned here (CI reads this value through `make lint-version`) and
# installed with the repo's own Go toolchain into a local, gitignored bin dir. A
# golangci-lint built with an older Go than go.mod's refuses to load the config.
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2
LOCAL_BIN             ?= $(CURDIR)/.bin
GOLANGCI_LINT         ?= $(LOCAL_BIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

.PHONY: lint-version
lint-version:
	@echo $(GOLANGCI_LINT_VERSION)

.PHONY: lint-install
lint-install: $(GOLANGCI_LINT)

$(GOLANGCI_LINT):
	GOBIN=$(LOCAL_BIN)/tmp go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv $(LOCAL_BIN)/tmp/golangci-lint $@
	rmdir $(LOCAL_BIN)/tmp

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run $(LINT_PKGS)

.PHONY: test
test: build-client
	go test ./internal/... ./tools/... -count=1 -timeout 15m $(TESTARGS)

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
docs: build-client
	go tool tfplugindocs generate --provider-name $(PROVIDER_NAME) --rendered-provider-name NetBox

.PHONY: docs-check
docs-check: build-client
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
# renovate: datasource=github-releases depName=goreleaser/goreleaser
GORELEASER_VERSION ?= v2.18.3
GORELEASER    ?= $(shell command -v goreleaser 2>/dev/null || echo "go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)")
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

# --- releases (docs/DEVELOPMENT.md#releasing) --------------------------------
# Adds the next version's section to CHANGELOG.md from the commits since the
# last tag; VERSION=0.2.0 overrides the version git-cliff works out. The
# Prepare release workflow does the same on a release/vX.Y.Z branch.
VERSION ?=
.PHONY: changelog
changelog:
	./scripts/prepare-release.sh $(VERSION)

# Prints the GitHub Release notes for VERSION (its CHANGELOG.md section).
.PHONY: release-notes
release-notes:
	@test -n "$(VERSION)" || { echo "usage: make release-notes VERSION=0.2.0" >&2; exit 1; }
	@go run ./tools/changelog notes -version $(VERSION)

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
