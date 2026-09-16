# Roadmap, todos and sharp edges

Status as of the `v0.1.0` candidate on branch `claude/compassionate-hamilton-drdiis` (PR #1).

## Where things stand

| Area | State |
|---|---|
| Resources / data sources | 126 generated resources, 266 data sources, 7 hand-written resources; see [RESOURCES.md](RESOURCES.md) |
| Acceptance on demo.netbox.dev (4.7.0) | 135/135 pass (create, update, data sources, import, empty plan); 7 scenarios applied and destroyed |
| Per-app validation reports | [validation/](validation/README.md) (written before the last generator fixes; see the banner there) |
| Docs | rendered by tfplugindocs; CI enforces `make gen`/`make docs` are committed and runs `make docs-check` |
| Release | GoReleaser (unsigned by default) + network mirror on GitHub Pages + `scripts/install.sh`; snapshot build and install smoke test run on every PR (`package` job); no tag pushed yet, see [INSTALL.md](INSTALL.md) |

## Next steps to `v0.1.0`

1. Enable GitHub Pages (Settings → Pages → Source "GitHub Actions") so the release workflow can publish the network mirror. Optionally add an RSA (or DSA) GPG key as `GPG_PRIVATE_KEY`/`PASSPHRASE` secrets; only the public Terraform Registry needs the signature.
2. Run the docker-compose acceptance workflow (`.github/workflows/acceptance.yml`) once by `workflow_dispatch`; it has only ever been executed against the public demo. Expect to tune shard timeouts.
3. Done: `.github/workflows/test.yml` runs `make docs-check` (`tfplugindocs validate`) and the unit-test matrix now includes Terraform `1.16.*` alongside `1.13.*`/`1.14.*`.
4. Add a client reproducibility job (`make client-gen && git diff --exit-code -- netbox/`) with Java 17 and a pinned SHA-256 for the openapi-generator jar in `tools/client-gen/generate.sh`.
5. The Terraform Registry namespace `elliot` is reserved and `elliot/netbox` is final. Remaining registry
   steps: add the RSA (or DSA) GPG public key to the registry account, add the `GPG_PRIVATE_KEY`/`PASSPHRASE`
   secrets, and publish the repository once through the registry UI after the first tag.
6. Cut `CHANGELOG.md`, tag `v0.1.0`, verify the docs in the Registry preview tool.

## Milestones

### v0.1 (this PR)
Everything above. Behaviour that is deliberately conservative: reverse sides of relations are read-only; nullable numbers and choices track the server value; `custom_fields` tracks only configured keys.

### v0.2: generator polish
- `display` / `last_updated` are plain `Computed` and show `(known after apply)` on every change; give them `UseStateForUnknown` or a plan modifier that copies state when nothing else changed. This is what forced `rear_port.front_ports` to be skipped.
- Reverse-side *nested* lists (`rear_port.front_ports`, `rear_port_template.front_ports`) need a generic read-only rule like the one for ID sets; today they are `skip: true` and missing from the data sources.
- Symmetric many-to-many pairs (`user.permissions` / `permission.user_ids`) are skip-only; decide the owning side per pair or emit the reverse side as computed.
- Re-verify `module_bay.installed_module` (skipped after a NetBox 500 that the "no null on create" fix may have removed) and retire redundant overrides listed below.
- Validate `filters[*].name` in data sources against the endpoint parameter list at plan time (NetBox ignores unknown filters and returns everything).
- A MAC/WWN format validator (`aa:bb:cc:dd:ee:ff`) instead of case folding by property name.
- Allow clearing JSON attributes (`local_context_data`, `data`, `parameters`) by sending an explicit null through `PatchRaw`.
- Description polish: rack dimensions copied from the rack type, `extra_choices` as `[value, label]` pairs, `sync_interval` in minutes, list data source item shapes.
- Per-field float precision in overrides for every decimal field (only sites carry `precision` today; the default is 6).

### v0.3: beyond CRUD
- Action endpoints as data sources: cable traces and paths, `render-config` for devices and VMs, available IP/prefix/VLAN/ASN listings.
- An ephemeral resource for API tokens (the secret is only returned once).
- `cf_<name>` custom-field filters on list data sources (not in the OpenAPI document; derive from the definition cache).
- Bulk operations and ETag / `If-Match` concurrency control (both absent from the schema).
- `Token.allowed_ips` once the OpenAPI document declares it.

### Later
- NetBox 4.8 bump following [DEVELOPMENT.md](DEVELOPMENT.md); expect new collections and field additions only.
- Plugin API support behind an opt-in (plugins do not appear in the core schema).
- Natural-key imports (`terraform import netbox_site.x slug:dc1`).

## Todos (file-anchored)

- [ ] `internal/gen/render/emit.go`: `UseStateForUnknown` for `display` / `last_updated` (see v0.2).
- [ ] `internal/gen/build/build.go`: extend pass 3 (reverse one-to-many) to nested lists; add a `format: mac` check; drop the property-name special case for `mac_address` / `wwn`.
- [ ] `internal/gen/render/datasource.go`: emit a `OneOf` validator (or a warning) for `filters[*].name`.
- [ ] `generator/overrides`: remove `circuits.assignments` and `providers.accounts` skips (both now handled generically; `accounts` should become read-only, not skipped), remove `racks.*`, `virtual-machines.disk` and `tokens.pepper_id` `computed: true` (default now), keep `journal-entries.created_by` but define it in one file only.
- [ ] `generator/overrides/dcim_a.yaml`: drop the `ignore_changes = [site_ids]` in the ASN fixture (`asn.site_ids` is read-only now) and re-run `TestAccAsn_basic`.
- [ ] `internal/provider/manual/primary_ip_resource.go` and the two primary-IP examples: drop the `ignore_changes = [primary_ip4_id, primary_ip6_id]` guidance after confirming the device / VM attributes (now `Optional+Computed`) no longer plan a removal.
- [ ] `tools/client-gen/generate.sh`: pin the jar checksum.
- [ ] `docs/validation/*.md`: refresh after the next full run so they stop describing fixed issues as workarounds.
- [ ] Renovate (`.github/renovate.json5`, monthly) now manages Go modules, GitHub Actions digests, the
      docker compose images and the openapi-generator jar; review its first dependency dashboard.

## Sharp edges for users

General
- Only v2 tokens (`nbt_<key>.<secret>`) work; v1 tokens are rejected at configure time.
- Attributes with a server default are `Optional+Computed`: removing them from the configuration keeps the current value rather than resetting it. Set them explicitly to change them.
- NetBox trims whitespace from text fields and upper-cases MAC addresses; the provider keeps your configured value when the difference is only that.
- Reverse sides of relations are read-only: attach ASNs from `netbox_site.asn_ids`, permissions from `netbox_permission`, provider accounts by creating `netbox_provider_account`.
- `custom_fields` tracks only the keys you configure; the data sources return every field as a JSON string. Unknown field names fail at plan time.
- Data-source `filters` names are not validated yet; a typo returns every object.
- Rate limiting is off unless `requests_per_second` (or `NETBOX_REQUESTS_PER_SECOND`) is set; shared instances such as the demo need it.

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
- `service.ipaddress_ids` must reference IPs assigned to the parent device or VM.
- VLAN `vid` must fall inside the group's `vid_ranges`.

Extras and wireless
- `choice_colors` takes colour names (`blue`, `gray`, ...), not hex.
- `order_alphabetically = true` reorders `extra_choices` server-side; configure them sorted.
- `journal_entry.created_by` is the token's user.
- `auth_psk` on wireless LANs and links is returned in clear text by the API.

Virtualization, users, VPN, circuits
- A VM's `disk` is derived from its virtual disks; concurrent disk creation can race, use `depends_on` or `-parallelism=1`.
- Token secrets are returned once; `netbox_token.token` is kept from state afterwards.
- `owner` requires an owner group although the schema says nullable.
- IKE: `mode` is required for IKEv1 and forbidden for IKEv2; nullable choice fields cannot be cleared through the API.
- `circuit.assignments` is not writable in 4.7; use `netbox_circuit_group_assignment`.
- Allocated IPs, prefixes and ASNs do not remember their parent after import.

## Sharp edges for contributors

- Never run `go run ./internal/gen` without `-only` while someone else is regenerating; a full run deletes and rewrites every generated file.
- Generated examples are only rewritten when a `.generated` marker exists in the directory; every current example is hand-maintained.
- `spec/required-fixes.json` feeds both the client generator and the provider generator; after changing it run `make client-gen` and `make gen` together or constructors will not match.
- In sandboxes set `TF_ACC_TERRAFORM_PATH` to a local Terraform binary; otherwise the test harness contacts `checkpoint-api.hashicorp.com` and the whole test binary fails on timeout.
- Terraform 1.16 CLI configuration wants `dev_overrides { ... }` as a block; the documented `dev_overrides = { ... }` form is rejected.
- Terraform 1.16.2 can crash and write an empty state when a provider returns an inconsistent result on create; every such bug therefore orphans objects.
- `make sweep` deletes every `tfacc-*` object on whatever `NETBOX_SERVER_URL` points at.
- The demo resets daily: accounts, tokens and IDs disappear; `make demo-token` provisions a new account.
- Overrides for one resource should live in one file; files merge in name order and later files win, which is easy to miss.

### CI lint time

`golangci-lint` over the whole module, including the regenerated client in `netbox/` (1,100+ files) and
the generated resources under `internal/provider/gen/`, used more memory than the GitHub runner has, and
the job was killed after about 8 minutes on every push ("The runner has received a shutdown signal").
Linting is now restricted to the hand-written packages through `LINT_PKGS` in the `GNUmakefile`, with the
same package list repeated in the workflow; that completes in about 4 seconds in CI and under a minute
locally. The `golangci-lint-action` already caches its analysis cache, and `setup-go` caches modules and
the build cache, so no extra caching was added. Generated files are also excluded by path in
`.golangci.yml`. If a new hand-written package is added, append it to `LINT_PKGS`.

## Things to circle back to

Attributes hidden or forced by overrides (revisit on every spec bump):

| Resource | Attribute | Override | Why |
|---|---|---|---|
| circuit | assignments | skip | not echoed by the read serializer (now dropped generically) |
| provider | accounts | skip | reverse side of provider_account.provider (generic rule now makes it read-only) |
| rear_port, rear_port_template | front_ports | skip | reverse nested list, perpetual empty updates |
| module_bay | installed_module | skip | NetBox 500 on any write, re-verify |
| asn | sites | read_only | reverse side of site.asn_ids |
| user, user_group | permissions | skip | reverse side of permission.user_ids / group_ids |
| service, service_template | protocol, ports | skip | deprecated in 4.7 for port_mappings |
| device_type | front_image, rear_image | skip | multipart upload |
| data_source | type | choice enum | free string in the schema, three shipped backends |
| devices, virtual_machines | primary_ip4, primary_ip6 | computed | managed by the primary-IP resources |
| racks | outer_*, mounting_depth, weight, max_weight | computed | copied from the rack type (default now) |
| tokens | token, key | expose | write-once secret |
| journal_entry | created_by | computed | set by the server |

Whole collections skipped: image attachments (multipart), notifications, subscriptions, bookmarks, saved filters, table configs (per-user UI state).

Spec gaps NetBox should fix or we should patch in `spec/`: `Token.allowed_ips` missing; allocation request schemas under-describe accepted fields; `add_tags`/`remove_tags`, ETag headers and `?background=true` are absent.
