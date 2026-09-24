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

`make lint` only runs golangci-lint over the hand-written packages listed in `LINT_PKGS` in the
`Makefile`; linting the regenerated client (`netbox/`) and generated resources
(`internal/provider/gen/`) alongside them exhausts the GitHub runner's memory. CI runs the unit-test
matrix against Terraform 1.13, 1.14 and 1.16.

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

## Registry smoke test

After a release is published, check that what users will actually download works:

```sh
make registry-smoke                       # latest ~> 0.1 release against demo.netbox.dev
VERSION=0.1.1 make registry-smoke         # a specific release
EXPECTED_KEY_ID=CAA7EF848084A7F1 make registry-smoke   # also pin the signing key
```

`scripts/registry-smoke.sh` copies `examples/scenarios/registry-demo` to a temporary directory and runs it
with a CLI configuration that only allows direct registry installs, so `dev_overrides` and anything under
`~/.terraform.d/plugins` (from `make install-local` or `scripts/install.sh`) cannot stand in for the
published build. It fails unless `terraform init` installs `elliot/netbox` from the registry with a verified
signature, the scenario applies with every `check` block passing, and the plans after apply, after an
import round-trip and after an in-place update are all empty. Everything is destroyed at the end, also on
failure. Names carry a random `tfacc-registry-*` prefix and the prefix is a random /24 of `198.18.0.0/15`, so
parallel runs do not collide. It uses `NETBOX_SERVER_URL` / `NETBOX_API_TOKEN` when set, then `.env.demo`,
and runs `make demo-token` otherwise.

## Bumping the NetBox version

1. Fetch the new document: `curl -sS 'https://<netbox>/api/schema/?format=json' > spec/netbox-<v>.openapi.json`
   and update `spec/VERSION`.
2. `make client-gen` (needs Java 17+, downloads openapi-generator 7.11.0 into `.cache/`).
3. `make gen && make docs`, then `go build ./... && go test ./internal/...`.
4. Review `go run ./internal/gen -list` for new collections and `docs/RESOURCES.md` for name changes;
   add overrides for new FK targets or fixtures as needed.
5. Update the version check in `internal/provider/provider.go` if the supported major.minor changes.

The generator jar is pinned by SHA-256 (`OAG_SHA256` next to `OAG_VERSION` in
`tools/client-gen/generate.sh`), which is verified before every run and must be updated together with
`OAG_VERSION`; `.github/workflows/client-gen.yml` regenerates the client on any change to `spec/`,
`tools/client-gen/` or `netbox/` and fails on drift, so commit the regenerated `netbox/` with the bump.
Acceptance tests (`.github/workflows/acceptance.yml`) run nightly, on demand via *Run workflow*, and on
pull requests labelled `run-acceptance`.

## Releasing

Releases are built by GoReleaser from `v*` tags (`.github/workflows/release.yml`):

1. `git tag v0.2.0 && git push origin v0.2.0`.
2. The build job runs in the `release` deployment environment; approve the deployment when it has a
   required reviewer.
3. The workflow builds `linux`/`darwin`/`freebsd`/`windows` × `amd64`/`arm64` zips, `SHA256SUMS` and the
   registry manifest, publishes the GitHub Release, attests build provenance for every asset, then runs
   `tools/mirror-index` and deploys the provider network mirror to GitHub Pages. Earlier versions are
   carried over from the live mirror so they stay installable.
4. `SHA256SUMS` is signed with the key stored as `GPG_PRIVATE_KEY` / `PASSPHRASE` environment secrets;
   the job fails before building if they are missing. After the upload the workflow re-verifies the
   signature and, when the `RELEASE_KEY_FINGERPRINT` repository variable is set, checks that the imported
   key is the expected one. The public Terraform Registry verifies that signature (and needs a public
   repository plus the `terraform-registry-manifest.json` already present); the same public key must be
   registered under the `elliot` namespace.

The repository settings that protect this pipeline (environment reviewers, tag rulesets, immutable
releases, SHA-pinning policy) are listed in [SECURITY.md](../SECURITY.md).

Before tagging, `make dist mirror mirror-check` reproduces the release build locally and installs it through
both the network mirror and `scripts/install.sh`; the `package` job in `test.yml` does the same on every pull
request. Installation channels for users are documented in [INSTALL.md](INSTALL.md).
