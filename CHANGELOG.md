# Changelog

All notable changes to this project are documented in this file. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - 2026-11-02

### Changed

- First stable release. From 1.0 the provider follows the
  [versioning policy](https://github.com/elliot/terraform-provider-netbox/blob/main/docs/VERSIONING.md):
  breaking changes only in a new major version, deprecated attributes kept until then, and each release pinned
  to one NetBox spec (4.7.4 for this release; NetBox 4.5 and 4.6 supported on a best-effort basis). Pin the
  provider with `version = "~> 1.0"`. There are no schema or behaviour changes from 0.6.0, so upgrading needs
  no configuration or state changes.

## [0.6.0] - 2026-11-01

The last 0.x release before 1.0: it finishes the pre-1.0 attribute renames, so upgrading needs a few
configuration edits (see *Breaking changes*). From 1.0 the provider follows the compatibility policy in
[docs/VERSIONING.md](https://github.com/elliot/terraform-provider-netbox/blob/main/docs/VERSIONING.md). The
provider is installed from the [Terraform Registry](https://registry.terraform.io/providers/elliot/netbox)
with `source = "elliot/netbox"`; the network mirror and `scripts/install.sh` remain for off-registry hosts.

### Breaking changes

- Attribute names are consistent with the provider's conventions (`ip_address`, `<field>_id`,
  `<singular>_count`) ([#58](https://github.com/elliot/terraform-provider-netbox/pull/58)). Resources:
  - `netbox_service.ipaddress_ids` -> `ip_address_ids`
  - `netbox_front_port` and `netbox_front_port_template`: `rear_ports[].rear_port` -> `rear_ports[].rear_port_id`
  - `netbox_rear_port` and `netbox_rear_port_template`: `front_ports[].front_port` -> `front_ports[].front_port_id`
    (computed)

  Data sources (singular and list) follow the resource renames above, and also rename:
  - `virtualmachine_count` -> `virtual_machine_count` (device role, platform, site, tenant, cluster)
  - `ipaddress_count` -> `ip_address_count` (VRF, tenant); `count_ipaddresses` -> `ip_address_count` and
    `count_fhrp_groups` -> `fhrp_group_count` (interface, VM interface)
  - `devicetype_count`, `moduletype_count`, `inventoryitem_count` -> `device_type_count`,
    `module_type_count`, `inventory_item_count` (manufacturer; inventory item role for the last)
  - `coolingfeed_count` -> `cooling_feed_count`, `powerfeed_count` -> `power_feed_count`,
    `wirelesslan_count` -> `wireless_lan_count`, `tunnel.terminations_count` -> `termination_count`,
    `custom_field_choice_set.choices_count` -> `choice_count`
  - `netbox_cable_termination.cable` -> `cable_id`, `netbox_script.module` -> `module_id`

  **Upgrading:** no state commands are needed; the next refresh reads the renamed attributes from NetBox.
  Terraform rejects the old names, so update configurations. Most renames can be applied with:

  ```sh
  sed -i -E \
    -e 's/\bipaddress_ids\b/ip_address_ids/g' \
    -e 's/\bvirtualmachine_count\b/virtual_machine_count/g' \
    -e 's/\bcount_ipaddresses\b|\bipaddress_count\b/ip_address_count/g' \
    -e 's/\bcount_fhrp_groups\b/fhrp_group_count/g' \
    -e 's/\bdevicetype_count\b/device_type_count/g' \
    -e 's/\bmoduletype_count\b/module_type_count/g' \
    -e 's/\binventoryitem_count\b/inventory_item_count/g' \
    -e 's/\bcoolingfeed_count\b/cooling_feed_count/g' \
    -e 's/\bpowerfeed_count\b/power_feed_count/g' \
    -e 's/\bwirelesslan_count\b/wireless_lan_count/g' \
    -e 's/\bterminations_count\b/termination_count/g' \
    -e 's/\bchoices_count\b/choice_count/g' \
    *.tf
  ```

  Rename `rear_port =` to `rear_port_id =` inside `rear_ports = [...]` blocks, references to
  `.front_ports[*].front_port`, and `data.netbox_cable_termination.<name>.cable` / `data.netbox_script.<name>.module`
  by hand: those names are too generic to rewrite safely.
- Data sources return `custom_fields = "{}"` instead of null for objects without custom fields, so
  `jsondecode()` and `lookup(jsondecode(...), "key", null)` work on every object. Configurations that compared
  `custom_fields` with `null` should compare with `"{}"` or check `length(jsondecode(...)) == 0`
  ([#60](https://github.com/elliot/terraform-provider-netbox/pull/60)).

### Added

- Support NetBox 4.7.4: the pinned OpenAPI document, API client, resources and docs are regenerated from it
  ([#53](https://github.com/elliot/terraform-provider-netbox/pull/53)):
  - `netbox_cable`: `profile` accepts `trunk-8c1p`, `trunk-12c1p` and `trunk-24c1p`.
  - `netbox_circuits`, `netbox_circuit_terminations`, `netbox_virtual_circuits` and
    `netbox_virtual_circuit_terminations`: `provider_network` filters.
  - NetBox now declares object IDs as 64-bit integers; the provider already used Int64 attributes, so nothing
    changes in configurations.

### Changed

- Terraform 1.12 is the documented and tested minimum (resource identity needs it); CI runs the unit tests on
  Terraform 1.12, 1.15, 1.16 and 1.17.
- `netbox_service` and `netbox_service_template` need NetBox 4.7: NetBox 4.5 and 4.6 require `ports`, which the
  provider does not expose (the README previously suggested setting it, which was not possible).

### Security

- `golang.org/x/net` 0.62.0 fixes quadratic complexity in its HTML parser (GO-2026-6860). The provider does not
  link that package, so it was not affected; the update keeps vulnerability scanners quiet
  ([#51](https://github.com/elliot/terraform-provider-netbox/pull/51)).

## [0.5.0] - 2026-10-26

### Added

- Support NetBox 4.7.3 ([#43](https://github.com/elliot/terraform-provider-netbox/pull/43))

### Changed

- `deps`: Build with Go 1.26.10 for the eMudhra CA distrust fix ([#47](https://github.com/elliot/terraform-provider-netbox/pull/47))

## [0.4.1] - 2026-10-19

### Security

- Release binaries are built with Go 1.26.9, which fixes standard library vulnerabilities reachable from the
  provider's HTTP client: an HTTP/1 connection desync after a rejected `CONNECT` through a proxy
  (GO-2026-6605), HTTP/2 framing headers accepted when malformed (GO-2026-6610) and MIME header parsing that
  ignored its memory limit (GO-2026-6608), plus fixes in `crypto/tls`. Exploiting them needs a malicious or
  compromised NetBox server or proxy; upgrading from 0.4.0 is recommended. No configuration changes
  ([#40](https://github.com/elliot/terraform-provider-netbox/pull/40)).

## [0.4.0] - 2026-10-11

### Added

- `netbox_provider` (and its data sources) exposes the provider's accounts as a computed `account_ids` set
  ([#31](https://github.com/elliot/terraform-provider-netbox/pull/31)).

### Changed

- Legacy NetBox v1 API tokens (40 characters) are accepted again, with a deprecation warning on every plan;
  NetBox 5.0 removes them, so switch to a v2 token (`nbt_<key>.<secret>`). Malformed tokens now fail with the
  specific reason ([#22](https://github.com/elliot/terraform-provider-netbox/pull/22)).
- `latitude` and `longitude` on `netbox_site` and `netbox_device` reject more than six decimal places at plan
  time. NetBox stores six and silently rounded finer values, which then differed from the configuration;
  configurations with seven or more decimals now fail validation
  ([#23](https://github.com/elliot/terraform-provider-netbox/pull/23)).

### Fixed

- Several `netbox_custom_field_value` resources on the same object no longer lose values when applied in
  parallel: NetBox kept only one of overlapping updates, so a field could silently disappear or show up as an
  inconsistent result after apply ([#33](https://github.com/elliot/terraform-provider-netbox/pull/33)).

## [0.3.0] - 2026-10-02

### Added

- Support for NetBox 4.7.2: the pinned OpenAPI document, API client, resources and docs are regenerated from
  it. All additions are backwards compatible:
  - `netbox_vlan_translation_policy` and `netbox_vlan_translation_rule`: `tags`, `custom_fields` and
    `created` (NetBox 4.7.1).
  - `netbox_data_source`: `tags` (NetBox 4.7.1).
  - `netbox_interface` and `netbox_interface_template`: `type` accepts `infiniband-hdr-2x`,
    `infiniband-ndr-2x` and `infiniband-xdr-2x` (NetBox 4.7.1).
  - `netbox_export_template`: `auto_sync_enabled` (NetBox 4.7.2).
  - `netbox_vlans`: `related_to_site` and `related_to_site__n` filters (NetBox 4.7.2).

## [0.2.0] - 2026-10-01

### Added

- `netbox_user` and `netbox_user_group` expose the permissions assigned to them as a computed `permission_ids`
  set. `netbox_permission.user_ids` / `group_ids` still own the assignment.
- `netbox_owner.group_id` defaults to `1` (NetBox's "no group"), so owners can be created without one.

### Changed

- Data source `filters[*].name` values are checked against the endpoint's query parameters at plan time, so
  `terraform validate` rejects a typo instead of NetBox ignoring the parameter and returning every object. The
  error suggests the closest valid names. Configurations with unknown filter names now fail.
- MAC addresses (`netbox_mac_address`, `netbox_interface`, `netbox_vm_interface`) and WWNs (`netbox_interface`)
  must be written in colon notation. Dashed, dotted and bare-hex spellings previously failed after apply with
  an inconsistent-result error and are now rejected at plan time; letter case is still free.
- Faster CI: Go build and module caches are keyed per commit, restored incrementally and written only from
  `main` (the GoReleaser snapshot build drops from about 12 minutes to one or two when `netbox/` is
  unchanged). Both CodeQL scans run on every pull request; the Go scan skips the generated `netbox/` client.

### Fixed

- Removing a nullable JSON attribute (`local_context_data`, `constraints`, `netbox_custom_field.default`, ...)
  from the configuration now clears it in NetBox instead of failing with "Provider produced inconsistent
  result". Affects `netbox_device`, `netbox_virtual_machine`, `netbox_permission`, `netbox_data_source`,
  `netbox_module_type`, `netbox_module_type_profile`, `netbox_config_context_profile`,
  `netbox_config_template`, `netbox_export_template`, `netbox_custom_field` and `netbox_event_rule`.
- Five data source examples used filter names that do not exist (`cooling_source`, `module_bay`,
  `power_panel`, `rack`, `q` on cable terminations); they now look up the parent and filter by its ID.

### Security

- Secret attributes are marked `Sensitive` (redacted in plans and CLI output): `netbox_wireless_lan.auth_psk`,
  `netbox_wireless_link.auth_psk`, `netbox_ike_policy.preshared_key`, `netbox_fhrp_group.auth_key`,
  `netbox_webhook.secret` and `netbox_data_source.parameters` (backend credentials), on the resources and
  their data sources. Outputs that expose these values now need `sensitive = true`.

## [0.1.1] - 2026-09-24

### Changed

- Building from source requires Go 1.26; terraform-plugin-framework 1.19.0, terraform-plugin-go 0.31.0 and
  all other Go dependencies updated.
- `SHA256SUMS` is always GPG-signed: the release job fails without the key in the `release` environment,
  verifies the signature after upload and can pin the expected key fingerprint (`RELEASE_KEY_FINGERPRINT`).

### Fixed

- The release workflow skipped GPG signing of `SHA256SUMS` even when the signing secrets were configured.

### Security

- Release assets carry Sigstore build provenance attestations (`gh attestation verify`, see `SECURITY.md`).
- Hardened CI and release workflows (no `pull_request_target`, least-privilege tokens, no cache in release
  builds, exact tool pins, zizmor in CI) and a 7-day cooldown on dependency updates.
- CodeQL code scanning for the Go sources and the GitHub Actions workflows.

## [0.1.0] - 2026-09-24

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

[Unreleased]: https://github.com/elliot/terraform-provider-netbox/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/elliot/terraform-provider-netbox/compare/v0.6.0...v1.0.0
[0.6.0]: https://github.com/elliot/terraform-provider-netbox/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/elliot/terraform-provider-netbox/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/elliot/terraform-provider-netbox/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/elliot/terraform-provider-netbox/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/elliot/terraform-provider-netbox/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/elliot/terraform-provider-netbox/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/elliot/terraform-provider-netbox/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/elliot/terraform-provider-netbox/releases/tag/v0.1.0
