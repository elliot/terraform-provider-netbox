# Development guide

## Layout

| Path | What |
|---|---|
| `main.go` | provider server entry point (`registry.terraform.io/elliot/netbox`) |
| `internal/provider/` | provider schema/configuration, version check, registry of resources, import/identity helpers |
| `internal/client/` | HTTP transport chain: retry/backoff, pacing (rate limit, delays, serialization), auth headers, logging |
| `internal/conv/` | conversions between framework values and the API client used by generated code |
| `internal/acctest/` | shared acceptance-test helpers (provider factories, fixtures, sweepers) |
| `internal/gen/` | the code generator (`go run ./internal/gen`) |
| `internal/provider/gen/<app>/` | **generated** resources, data sources and tests, one package per NetBox app |
| `internal/provider/manual/` | hand-written resources (allocations, primary IPs) |
| `generator/overrides/*.yaml` | hand-maintained generator tuning and test fixtures |
| `netbox/` | **generated** API client (go-netbox compatible) plus hand-written `client_extra.go` |
| `spec/` | pinned OpenAPI document, `VERSION`, `required-fixes.json` |
| `tools/client-gen/` | client regeneration pipeline (fix_spec.py, openapi-generator config) |
| `examples/`, `templates/`, `docs/` | tfplugindocs inputs and output (`docs/` is generated) |
| `docker/` | local NetBox 4.7 stack for acceptance tests |

## Everyday commands

```sh
make build test lint          # compile, unit tests, golangci-lint (v2 config)
make gen && make docs         # regenerate everything after changing spec/, overrides or the generator
git diff --exit-code          # CI enforces generated code and docs are committed
```

## Acceptance tests

```sh
make demo-token               # https://demo.netbox.dev account + v2 token -> .env.demo (resets daily)
set -a; . ./.env.demo; set +a
export NETBOX_REQUESTS_PER_SECOND=5
TF_ACC=1 go test ./internal/provider/gen/ipam/ -run TestAccPrefix_basic -v
TF_ACC=1 go test ./internal/provider/gen/... -run TestAcc -p 2 -parallel 2   # everything
make sweep                    # remove leftover tfacc-* objects
```

Tests can also target a local stack: `make docker-up`, then follow `docker/README.md` to obtain a v2 token
and export `NETBOX_SERVER_URL=http://localhost:8000` / `NETBOX_API_TOKEN`. In sandboxes without access to
`checkpoint-api.hashicorp.com`, point `TF_ACC_TERRAFORM_PATH` at a local Terraform binary or the whole test
binary fails while the harness looks for one. To run a scenario with a locally built provider, use a CLI
configuration with a `dev_overrides { "elliot/netbox" = "/path/to/dir" }` block (Terraform 1.16 rejects the
attribute form). `make sweep` deletes every `tfacc-*` object on the configured server.

## Bumping the NetBox version

1. Fetch the new document: `curl -sS 'https://<netbox>/api/schema/?format=json' > spec/netbox-<v>.openapi.json`
   and update `spec/VERSION`.
2. `make client-gen` (needs Java 17+, downloads openapi-generator 7.11.0 into `.cache/`).
3. `make gen && make docs`, then `go build ./... && go test ./internal/...`.
4. Review `go run ./internal/gen -list` for new collections and `docs/RESOURCES.md` for name changes;
   add overrides for new FK targets or fixtures as needed.
5. Update the version check in `internal/provider/provider.go` if the supported major.minor changes.

## Releasing

Releases are built by GoReleaser from `v*` tags (`.github/workflows/release.yml`). The Terraform Registry
requires the repository to be public, GPG-signed checksums (RSA or DSA key, `GPG_PRIVATE_KEY` and
`PASSPHRASE` secrets) and `terraform-registry-manifest.json` (protocol 6.0, already present).
