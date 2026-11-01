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

The generated client (`netbox/`) compiles as a single package and needs about 5.9 GB at Go's default
`GOGC=100` on a cold cache. The `Makefile` therefore exports `GOGC=25` (about 4.2 GB peak, same wall
time) and `build`, `test`, `docs` and `docs-check` first run `make build-client`, which compiles
`netbox/` on its own so it is not built next to other packages. Override with `make test GOGC=100`.
On machines with under 8 GB, or when other builds run at the same time, also pass `GOFLAGS=-p=1`.

`make lint` installs the golangci-lint version pinned by `GOLANGCI_LINT_VERSION` in the `Makefile`
into `.bin/` (gitignored) with `go install`, using the repo's Go toolchain, and runs that copy
instead of whatever is on `PATH`. This matters because a golangci-lint built with an older Go than
`go.mod` targets fails at config load. Run `make lint-install` to install it without linting. CI
reads the same pin through `make lint-version`, and Renovate bumps it via the annotation on that line.

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

## Bumping the NetBox version

1. Extract the new document with `scripts/netbox-spec.sh <v>` (writes `spec/netbox-<v>.openapi.json`; needs
   Docker, about five minutes), `git rm` the old one, update `spec/VERSION` and `packageVersion` in
   `tools/client-gen/config.yaml`. The script migrates an empty database and runs `manage.py spectacular`
   in the `netboxcommunity/netbox:v<v>` image; the result is identical to `GET /api/schema/?format=json` on
   that release, without plugins (the demo runs one) or a token.
2. `go run ./tools/netbox-diff -old spec/netbox-<old>.openapi.json -new spec/netbox-<v>.openapi.json` lists
   what the release changes in the provider (breaking or additive). Both documents must sit next to
   `required-fixes.json`; the tool refuses otherwise, because the generator silently skips missing fixes.
3. `make client-gen` (needs Java 17+, downloads openapi-generator 7.26.0 into `.cache/`).
4. `make gen && make docs`, then `go build ./... && make test`.
5. Review `go run ./internal/gen -list` for new collections and `docs/RESOURCES.md` for name changes;
   add overrides for new FK targets or fixtures as needed.
6. Update the version check in `internal/provider/provider.go` if the supported major.minor changes.

The generator jar is pinned by SHA-256 (`OAG_SHA256` next to `OAG_VERSION` in
`tools/client-gen/generate.sh`), which is verified before every run and must be updated together with
`OAG_VERSION`; `.github/workflows/client-gen.yml` regenerates the client on any change to `spec/`,
`tools/client-gen/` or `netbox/` and fails on drift, so commit the regenerated `netbox/` with the bump.
Acceptance tests (`.github/workflows/acceptance.yml`) run nightly, on demand via *Run workflow* (optionally
against another `netbox_image_tag`), and on pull requests labelled `run-acceptance`; every shard records the
NetBox version it tested in the run summary.

### NetBox release watch

`.github/workflows/netbox-watch.yml` runs daily (logic in `scripts/netbox-watch.sh`). When NetBox publishes a
release newer than `spec/VERSION` and its image exists, it extracts the document, runs `tools/netbox-diff`,
tries the bump (client-gen, gen, docs, build, unit tests), runs the latest provider release's acceptance
tests against the new image, and runs smoke tests on demo.netbox.dev once the demo runs that release. It
then opens one `NetBox vX.Y.Z released` issue (label `netbox-release`, assigned to the repository owner) with
a verdict:

| Verdict | When |
|---|---|
| Release required (fix) | the latest release fails acceptance tests against the new NetBox |
| Release required | a new major.minor (the provider warns outside `SupportedNetBoxMajorMinor`) |
| Release required (breaking) | an attribute, value, filter or resource disappears or changes type, or becomes required or read-only |
| Release recommended | additive changes only: the released provider cannot use them |
| No release needed | the generated surface is unchanged |

The issue also lists the feat/fix commits already on `main` since the latest tag. While the demo still runs
an older release the issue carries `demo-pending` and the smoke tests are retried daily; the result is added
as a comment. The run's `netbox-spec-<v>` artifact holds the extracted document for the bump. The workflow
never opens pull requests (SECURITY.md). Run it by hand with *version* set to the pinned release to dry-run
it; *force* analyses a version again even when its issue exists.

## Releasing

Pull request titles are Conventional Commit subjects (checked by `.github/workflows/pr-title.yml`), and
each squash-merged title becomes a draft changelog entry: `feat` under *Added*, `fix` under *Fixed*,
`fix(deps)`, `perf` and `revert` under *Changed*, any `(security)` scope under *Security*, and a `!` (or a
`BREAKING CHANGE:` footer) under *Breaking changes*. `docs`, `ci`, `test`, `build`, `chore`, `refactor` and
`style` are left out. The mapping lives in `cliff.toml`.

1. Run **Actions → Prepare release → Run workflow** on `main`. Leave *version* empty to let
   [git-cliff](https://git-cliff.org) work it out, or enter one. On 0.x a feature or a breaking change
   bumps the minor version, otherwise the patch. The `[bump]` settings in `cliff.toml`
   (`features_always_bump_minor`, `breaking_always_bump_major = false`) only make a difference while the
   major version is 0; from 1.0 on, git-cliff bumps the major for a breaking change (`!` or a
   `BREAKING CHANGE:` footer), the minor for a feature and the patch otherwise, with no configuration change.
   It never proposes 1.0.0 by itself, so enter `1.0.0` explicitly for that release. The workflow runs `scripts/prepare-release.sh`, which turns
   `[Unreleased]` into a `## [X.Y.Z] - <date>` section that merges any hand-written entries with the drafted
   ones, then pushes it to a `release/vX.Y.Z` branch.
2. Open the pull request from the link in the job summary, titled `chore(release): vX.Y.Z`. Edit
   `CHANGELOG.md` there: reword the entries for users, drop duplicates and internal changes, and add upgrade
   notes for anything breaking.
3. Squash-merge it. The Release workflow sees the `chore(release): vX.Y.Z` commit on `main`, creates the
   annotated `vX.Y.Z` tag on it and dispatches itself on that tag.
4. The build job runs in the `release` deployment environment; approve the deployment when it has a
   required reviewer. The version's `CHANGELOG.md` section becomes the GitHub Release notes, and the
   build fails before anything is published if that section is missing.
5. The workflow builds `linux`/`darwin`/`freebsd`/`windows` × `amd64`/`arm64` zips, `SHA256SUMS` and the
   registry manifest, publishes the GitHub Release, attests build provenance for every asset, then runs
   `tools/mirror-index` and deploys the provider network mirror to GitHub Pages. Earlier versions are
   carried over from the live mirror so they stay installable.
6. `SHA256SUMS` is signed with the key stored as `GPG_PRIVATE_KEY` / `PASSPHRASE` environment secrets;
   the job fails before building if they are missing. After the upload the workflow re-verifies the
   signature and, when the `RELEASE_KEY_FINGERPRINT` repository variable is set, checks that the imported
   key is the expected one. The public Terraform Registry verifies that signature (and needs a public
   repository plus the `terraform-registry-manifest.json` already present); the same public key must be
   registered under the `elliot` namespace.

`make changelog` (optionally `VERSION=0.2.0`) does step 1 locally without pushing anything, and
`make release-notes VERSION=0.2.0` prints the notes a release would get. Pushing a `v*` tag by hand still
releases, provided `CHANGELOG.md` on that tag has its section; to re-run a release, run the Release
workflow on the tag.

The repository settings that protect this pipeline (environment reviewers, tag rulesets, immutable
releases, SHA-pinning policy) are listed in [SECURITY.md](../SECURITY.md).

Before tagging, `make dist mirror mirror-check` reproduces the release build locally and installs it through
both the network mirror and `scripts/install.sh`; the `package` job in `test.yml` does the same on every pull
request. Installation channels for users are documented in [INSTALL.md](INSTALL.md).
