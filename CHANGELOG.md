# Changelog

All notable changes to this project are documented in this file. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Building from source requires Go 1.26; terraform-plugin-framework 1.19.0, terraform-plugin-go 0.31.0 and
  all other Go dependencies updated.
- `SHA256SUMS` is always GPG-signed: the release job fails without the key in the `release` environment,
  verifies the signature after upload and can pin the expected key fingerprint (`RELEASE_KEY_FINGERPRINT`).
- Faster CI: Go build and module caches are keyed per commit, restored incrementally and written only from
  `main` (the GoReleaser snapshot build drops from about 12 minutes to one or two when `netbox/` is
  unchanged). Both CodeQL analyses (Go and workflows) run on every pull request, so the CodeQL summary check
  can always compare against `main`; the Go analysis no longer scans the generated client in `netbox/`, which
  cuts it from about 40 minutes to about 10.

### Fixed

- The release workflow skipped GPG signing of `SHA256SUMS` even when the signing secrets were configured.

### Security

- Release assets carry Sigstore build provenance attestations (`gh attestation verify`, see `SECURITY.md`).
- Hardened CI and release workflows (no `pull_request_target`, least-privilege tokens, no cache in release
  builds, exact tool pins, zizmor in CI) and a 7-day cooldown on dependency updates.
- CodeQL code scanning for the Go sources and the GitHub Actions workflows.
- Secret attributes are marked `Sensitive` (redacted in plans and CLI output): `netbox_wireless_lan.auth_psk`,
  `netbox_wireless_link.auth_psk`, `netbox_ike_policy.preshared_key`, `netbox_fhrp_group.auth_key`,
  `netbox_webhook.secret` and `netbox_data_source.parameters` (backend credentials), on the resources and
  their data sources. Outputs that expose these values now need `sensitive = true`.

## [0.1.0] - 2026-09-16

### Added

- `custom_fields` is a native Terraform object with per-key plans, name validation against the NetBox
  definitions and coercion of selection/object values; `netbox_custom_field_value` sets one field on any object.
- Generated resources, data sources and list data sources for every CRUD collection of the
  NetBox 4.7 API (132 resources, 266 data sources), with import and resource identity by ID.
- Allocation resources `netbox_available_ip_address`, `netbox_available_prefix`,
  `netbox_available_vlan`, `netbox_available_asn`, and `netbox_device_primary_ip` /
  `netbox_virtual_machine_primary_ip`.
- go-netbox compatible API client regenerated for NetBox 4.7 (`netbox/`).
- Provider scaffolding for NetBox 4.7 on terraform-plugin-framework (protocol 6).
- Provider configuration (`server_url`, `api_token`, rate limiting, request/write delays,
  request serialisation, retries with backoff, timeouts, extra headers, TLS options,
  `skip_version_check`, `user_agent`), each also settable through `NETBOX_*` environment variables.
- Only NetBox v2 API tokens (`nbt_<key>.<secret>`) are supported; v1 tokens are rejected at configure time.
- HTTP transport chain with client-side token-bucket rate limiting, fixed delays, retry with
  exponential backoff (honouring `Retry-After`), request logging with token redaction.
- Version check against `GET /api/status/` on configure (warning when the server is not 4.7.x).
- Local NetBox 4.7 docker compose stack and `scripts/demo-token.sh` for the public demo.
- Off-registry release: unsigned GitHub Releases (signed only when a GPG key is configured), a provider
  network mirror published to GitHub Pages by `tools/mirror-index`, `scripts/install.sh` for Terraform's
  implied local mirror, a filesystem mirror for air-gapped installs, `make dist mirror mirror-check
  install-local` and `docs/INSTALL.md`. The Terraform Registry listing under the reserved namespace
  `elliot/netbox` is pending, so this is the only way to install the provider today. 32-bit builds were
  dropped from the release matrix.

### Changed

- Update requests carry only the attributes whose planned value differs from state; optional nullable
  fields are omitted on create instead of being sent as null.
- `form_factor` (rack types), `status` (tunnels), `role` (tunnel terminations) and `version` (IKE policies)
  are required, matching NetBox validation.
- Reverse sides of relations are computed-only (`netbox_asn.site_ids`, `netbox_rear_port.front_ports`,
  `netbox_rear_port_template.front_ports`) or not exposed (`netbox_provider.account_ids`,
  `netbox_circuit.assignments`, `netbox_user.permission_ids`).
- Resources no longer expose `display` and `last_updated`: both change on every write and produced a
  `(known after apply)` line in every plan. They remain available on the data sources; `id`, `url` and
  `created` stay on resources and are stable across updates.
- Nullable numbers, choices and timestamps with server defaults are `Optional+Computed` and are never sent as null.

### Fixed

- Float rounding, whitespace trimming and MAC/WWN upper-casing by NetBox no longer produce diffs.
- Re-sending unchanged front/rear port mappings no longer fails with "must make a unique set".
- Nullable choice fields (IKE `mode`, IPsec `pfs_group`, ...) no longer break updates with "may not be blank".
- Token secrets and keys are exposed once on create and kept in state; device and VM primary IPs set through
  the primary-IP resources no longer plan a removal on the parent.

### Known limitations

See [docs/ROADMAP.md](docs/ROADMAP.md).

[Unreleased]: https://github.com/elliot/terraform-provider-netbox/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/elliot/terraform-provider-netbox/releases/tag/v0.1.0
