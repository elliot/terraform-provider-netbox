# Roadmap, todos and sharp edges

Status as of `v0.5.0` (2026-11-01; NetBox 4.7.4 support is merged on `main` and unreleased). Items that have
been actioned are removed from this file; `CHANGELOG.md` is the record of what shipped.

## Where things stand

| Area | State |
|---|---|
| Resources / data sources | 133 resources (126 generated, 7 hand-written) and 266 data sources; see [RESOURCES.md](RESOURCES.md) |
| Spec | pinned at NetBox 4.7.4 (`spec/VERSION`); 4.7.4 support merged after `v0.5.0` |
| Acceptance | nightly workflow on `main` (docker-compose, 12 shards, one pinned NetBox each) green since `v0.5.0`; the 4.7.4 bump passed on its PR. The workflow also runs on `workflow_dispatch` and on PRs labelled `run-acceptance` |
| Per-app validation reports | [validation/](validation/README.md) (historical; superseded by the nightly acceptance run) |
| CI | build, lint (hand-written packages), generated code and docs up to date, `tfplugindocs validate`, unit tests on Terraform 1.12 (minimum) / 1.15 / 1.16 / 1.17, GoReleaser snapshot with network-mirror and install-script smoke test, client regeneration drift check (pinned openapi-generator jar checksum), daily NetBox release watch (`netbox-watch.yml`), zizmor workflow audit, CodeQL, PR auto-labelling; actions pinned to SHAs, least-privilege tokens, Renovate weekly with a 7-day cooldown (Dependency Dashboard: issue #46) |
| Release | `v0.5.0` is the latest release. GitHub Releases with eight platform zips, GPG-signed `SHA256SUMS` (verified for `v0.4.0`, `v0.4.1`, `v0.5.0`) and SLSA build provenance attestations for every asset; network mirror at `https://elliot.github.io/terraform-provider-netbox/` and `scripts/install.sh` for off-registry installs; see [INSTALL.md](INSTALL.md) |
| Registry | live at [registry.terraform.io/providers/elliot/netbox](https://registry.terraform.io/providers/elliot/netbox) (community tier) with every signed release from `v0.1.1`; `terraform init` installs from it without CLI configuration |

## v1.0

Remaining work before `v1.0.0`; the compatibility promise is described in [VERSIONING.md](VERSIONING.md).

- Decide how data sources expose `custom_fields` (JSON string today) before it is frozen.
- Remaining repository settings from the maintainer checklist in [SECURITY.md](../SECURITY.md): the "require
  full-length commit SHA" Actions policy, read-only default workflow permissions and private vulnerability
  reporting (maintainer only).

Release PR checklist for `v1.0.0`:

- [ ] Drop the "Status: pre-1.0" banner from `README.md`.
- [ ] Bump the version pins to `~> 1.0` in `README.md`, `docs/INSTALL.md`, `templates/index.md.tmpl` (if it
      carries one) and `examples/provider/provider.tf`, then `make docs`.
- [ ] Run **Prepare release** with the explicit version `1.0.0` (git-cliff would otherwise propose a 0.x
      version, see [DEVELOPMENT.md](DEVELOPMENT.md#releasing)).

## 1.x

Additive work that does not need a major release:

- Expose `circuit.assignment_ids` and `module_bay.installed_module_id` as read-only computed attributes on the
  resources (NetBox 4.7.3 returns both on read; the data sources already have them).
- `cf_<name>` custom-field filters on list data sources (not in the OpenAPI document; derive from the definition cache).
- Natural-key imports (`terraform import netbox_site.x slug:dc1`).
- ETag / `If-Match` concurrency control (absent from the schema).
- An ephemeral resource for API tokens (the secret is only returned once).
- Write-only variants of secret attributes, next to the existing `Sensitive` ones.

## Backlog

Behaviour that is deliberately conservative since v0.1: reverse sides of relations are read-only; nullable numbers and choices track the server value; `custom_fields` tracks only configured keys.

### Generator polish
- Reverse-side computed lists (`asn.site_ids`, `rear_port.front_ports`) are unknown during updates of the owning resource; a state-copying plan modifier would remove that line from plans but risks an inconsistent-result error when the other side changes in the same apply.
- Description polish: rack dimensions copied from the rack type, `extra_choices` as `[value, label]` pairs, `sync_interval` in minutes, list data source item shapes.
- Per-field float precision in overrides for every decimal field (only `latitude`/`longitude` carry a precision today, applied by the generator; other floats fall back to 6 on read and have no decimals validator).

### Beyond CRUD
- Action endpoints as data sources: cable traces and paths, `render-config` for devices and VMs, available IP/prefix/VLAN/ASN listings.
- Bulk operations (absent from the schema).
- `Token.allowed_ips` once the OpenAPI document declares it.

### Later
- NetBox 4.8 bump following [DEVELOPMENT.md](DEVELOPMENT.md); expect new collections and field additions only.
- Plugin API support behind an opt-in (plugins do not appear in the core schema).

## Sharp edges for users

General
- Use v2 tokens (`nbt_<key>.<secret>`). Legacy 40-character v1 tokens work but warn on every run; NetBox removes them in 5.0.
- Attributes with a server default are `Optional+Computed`: removing them from the configuration keeps the current value rather than resetting it. Set them explicitly to change them.
- NetBox trims whitespace from text fields and upper-cases MAC addresses and WWNs; the provider keeps your configured value when the difference is only that. MAC addresses and WWNs must be written in colon notation (`aa:bb:cc:dd:ee:ff`): NetBox rewrites other notations, so the provider rejects them at plan time.
- Reverse sides of relations are read-only: attach ASNs from `netbox_site.asn_ids`, permissions from `netbox_permission`, provider accounts by creating `netbox_provider_account`.
- `custom_fields` tracks only the keys you configure; the data sources return every field as a JSON string. Unknown field names fail at plan time.
- Data-source `filters` names are checked against the endpoint's query parameters at plan time (with suggestions for typos); custom-field filters (`cf_<name>`) are not accepted yet.
- Rate limiting is off unless `requests_per_second` (or `NETBOX_REQUESTS_PER_SECOND`) is set; shared instances such as the demo need it.
- `terraform providers lock` ignores the CLI configuration's mirrors; pass `-net-mirror` or `-fs-mirror` explicitly on hosts that cannot reach the registry (see [INSTALL.md](INSTALL.md)).

DCIM
- Component templates instantiate only when a device is created: add `depends_on` from the device to its templates, and read the generated components with data sources.
- Component names are unique per device; explicit component resources must not collide with templated ones.
- Front/rear port mappings are unique per `(rear_port, rear_port_position)`; the provider only sends them when they change because NetBox rejects re-sent identical mappings.
- Attaching a `rack_type_id` copies the type's dimensions onto the rack on every save; do not set conflicting values.
- Virtual chassis: set `master_id` in a second apply after members joined via `netbox_device.virtual_chassis_id`.
- Device bays need `subdevice_role = "parent"` on the parent type and a `u_height = 0` child type.
- Power feed defaults (voltage, amperage) are deployment settings, not constants.

IPAM
- Aggregates may not overlap regardless of RIR; `asn` is globally unique; IP range boundaries need a prefix length.
- `service.ip_address_ids` must reference IPs assigned to the parent device or VM.
- VLAN `vid` must fall inside the group's `vid_ranges`.

Extras and wireless
- `choice_colors` takes colour names (`blue`, `gray`, ...), not hex.
- `order_alphabetically = true` reorders `extra_choices` server-side; configure them sorted.
- `journal_entry.created_by` is the token's user.
- Secrets NetBox returns in clear text (`auth_psk`, `preshared_key`, `auth_key`, webhook `secret`, data-source `parameters`) are `Sensitive`: redacted in plans, but stored in state like every attribute. Webhook `additional_headers` is not marked; keep credentials out of it or protect the state.

Virtualization, users, VPN, circuits
- A VM's `disk` is derived from its virtual disks; concurrent disk creation can race, use `depends_on` or `-parallelism=1`.
- Token secrets are returned once; `netbox_token.token` is kept from state afterwards.
- `owner` requires an owner group although the schema says nullable.
- IKE: `mode` is required for IKEv1 and forbidden for IKEv2; nullable choice fields cannot be cleared through the API.
- `netbox_circuit` does not manage group membership; use `netbox_circuit_group_assignment` (the data sources expose `assignment_ids`).
- Allocated IPs, prefixes and ASNs do not remember their parent after import.

## Sharp edges for contributors

- Never run `go run ./internal/gen` without `-only` while someone else is regenerating; a full run deletes and rewrites every generated file.
- Generated examples are only rewritten when a `.generated` marker exists in the directory; every current example is hand-maintained.
- `spec/required-fixes.json` feeds both the client generator and the provider generator; after changing it run `make client-gen` and `make gen` together or constructors will not match.
- `make lint` and CI lint only the packages in `LINT_PKGS` (Makefile): linting the generated client and resources exhausts the GitHub runner's memory. Append new hand-written packages to that list.
- Building release targets needs about 4 GB per target while the generated client compiles; `make dist DIST_PARALLELISM=1` on small machines. CI runs GoReleaser with `--parallelism 1` and `GOGC=50` for the same reason; the snapshot job restores a cross-compiled build cache from `main`, so only a change to `netbox/` pays the full compile (about 100 s per target).
- In sandboxes set `TF_ACC_TERRAFORM_PATH` to a local Terraform binary; otherwise the test harness contacts `checkpoint-api.hashicorp.com` and the whole test binary fails on timeout.
- Terraform 1.16 CLI configuration wants `dev_overrides { ... }` as a block; the documented `dev_overrides = { ... }` form is rejected.
- Terraform 1.16.2 can crash and write an empty state when a provider returns an inconsistent result on create; every such bug therefore orphans objects.
- Terraform only recognises the packed (zip) filesystem-mirror layout for plain `x.y.z` versions; snapshots and pre-releases must use the unpacked layout, which is what `scripts/install.sh` and `make install-local` do.
- `make sweep` deletes every `tfacc-*` object on whatever `NETBOX_SERVER_URL` points at.
- The demo resets daily: accounts, tokens and IDs disappear; `make demo-token` provisions a new account.
- Overrides for one resource should live in one file; files merge in name order and later files win, which is easy to miss.

## Things to circle back to

Attributes hidden or forced by overrides (revisit on every spec bump):

| Resource | Attribute | Override | Why |
|---|---|---|---|
| circuit | assignments | skip | owned by netbox_circuit_group_assignment; 4.7.3 returns it on read, so it can become a computed `assignment_ids` |
| rear_port, rear_port_template | front_ports | read_only | reverse side of front_port.rear_ports; NetBox accepts writes from both sides, the front port owns the mapping |
| module_bay | installed_module | skip | owned by netbox_module.module_bay_id; the 4.7.2 write error is fixed in 4.7.3, so it can become a computed `installed_module_id` |
| asn | sites | read_only, target | reverse side of site.asn_ids; the read items (`ASNSite`) name no target, so it is set explicitly for the symmetric-pair check |
| user, user_group | permissions | read_only | reverse side of permission.user_ids / group_ids; exposed as computed `permission_ids` |
| service, service_template | protocol, ports | skip | deprecated in 4.7 for port_mappings (the spec says so only in the description, not with `deprecated: true`). Not exposed as deprecated attributes: NetBox derives them from `port_mappings` (null for multi-protocol services) and derives `port_mappings` from them, so writing both sides of one value through `Optional+Computed` attributes produces inconsistent-result errors (stale `ports` re-sent after a `port_mappings` change; the empty `port_mappings` default overriding `ports`). Exposing them needs mutually exclusive attributes without state carry-over; never part of a release, so nothing to keep for 1.x |
| device_type | front_image, rear_image | skip | multipart upload |
| data_source | type | choice enum | free string in the schema, three shipped backends |
| devices, virtual_machines | primary_ip4, primary_ip6 | computed | managed by the primary-IP resources |
| tokens | token, key | expose | write-once secret |
| journal_entry | created_by | computed | set by the server |

Whole collections skipped: image attachments (multipart), notifications, subscriptions, bookmarks, saved filters, table configs (per-user UI state).

Spec gaps NetBox should fix or we should patch in `spec/`: `Token.allowed_ips` missing; allocation request schemas under-describe accepted fields (4.7.3 documents the available-prefixes request; available IPs, VLANs and ASNs still do not); `add_tags`/`remove_tags`, ETag headers and `?background=true` are absent.
