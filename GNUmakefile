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

.PHONY: lint
lint:
	golangci-lint run

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
