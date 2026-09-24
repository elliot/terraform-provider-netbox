---
name: bump-netbox
description: Move the provider to a new NetBox release (e.g. 4.8.0) by pinning its OpenAPI document and regenerating the client, resources and docs.
disable-model-invocation: true
argument-hint: "<netbox-version, e.g. 4.8.0>"
---

Bump the pinned NetBox version to $ARGUMENTS. Follow docs/DEVELOPMENT.md ("Bumping the NetBox version"). Work
on a `feat/netbox-<version>` branch.

1. **Spec.** Fetch the schema from a NetBox running exactly that version (the demo if it matches, otherwise
   `NETBOX_IMAGE_TAG=v<major.minor> make docker-up`):
   `curl -sS 'http://<netbox>/api/schema/?format=json' > spec/netbox-<version>.openapi.json`.
   Update `spec/VERSION`, `git rm` the old spec file, and check `spec/required-fixes.json` still applies (its
   entries may now be fixed upstream).
2. **Client.** `make client-gen` (Java 17+, Python 3). Commit all of `netbox/`; CI fails on drift.
3. **Resources and docs.** `make gen && make docs`, then `go build ./... && make test && make lint`.
4. **Review what changed.**
   - `go run ./internal/gen -list`: new collections need a decision (generate, add overrides, or `skip` with a
     reason).
   - `git diff docs/RESOURCES.md`: renamed or removed resources are breaking changes; call them out.
   - New generated tests that `t.Skip` for lack of a fixture: add fixtures via the generator-overrides skill.
   - Revisit every row of the "Things to circle back to" table in `docs/ROADMAP.md`; drop overrides the new
     spec makes unnecessary.
5. **Version wiring.** If major.minor changed, update `SupportedNetBoxMajorMinor` in
   `internal/provider/provider.go`, the default `NETBOX_IMAGE_TAG` in `docker/docker-compose.yml`, the
   "Start NetBox" step name in `.github/workflows/acceptance.yml`, the netbox image rule in
   `.github/renovate.json5`, and any `4.x`-specific text in README.md, docker/README.md and docs/DEVELOPMENT.md.
6. **Verify** with `/acc-test` on a representative set (at least one resource per changed app, plus anything
   whose schema changed), against demo.netbox.dev if it already runs the new version.
7. **Roadmap.** Remove the bump item from `docs/ROADMAP.md` and update its status table. Do not edit
   `CHANGELOG.md`.
8. Open the PR titled `feat: support NetBox <version>`, listing new resources, breaking renames and any new
   overrides. Suggest adding the `run-acceptance` label so CI runs the full matrix.
