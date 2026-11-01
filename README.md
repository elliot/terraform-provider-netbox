# Terraform Provider for NetBox

[![Tests](https://github.com/elliot/terraform-provider-netbox/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/elliot/terraform-provider-netbox/actions/workflows/test.yml)
[![Acceptance tests](https://github.com/elliot/terraform-provider-netbox/actions/workflows/acceptance.yml/badge.svg?branch=main)](https://github.com/elliot/terraform-provider-netbox/actions/workflows/acceptance.yml)
[![Client is reproducible](https://github.com/elliot/terraform-provider-netbox/actions/workflows/client-gen.yml/badge.svg?branch=main)](https://github.com/elliot/terraform-provider-netbox/actions/workflows/client-gen.yml)
[![CodeQL](https://github.com/elliot/terraform-provider-netbox/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/elliot/terraform-provider-netbox/actions/workflows/codeql.yml)
[![Latest release](https://img.shields.io/github/v/release/elliot/terraform-provider-netbox?include_prereleases&sort=semver)](https://github.com/elliot/terraform-provider-netbox/releases)
[![NetBox](https://img.shields.io/badge/NetBox-4.x-blue)](https://github.com/netbox-community/netbox)
[![Terraform](https://img.shields.io/badge/Terraform-%E2%89%A5%201.12-844FBA?logo=terraform)](https://developer.hashicorp.com/terraform)
[![Go version](https://img.shields.io/github/go-mod/go-version/elliot/terraform-provider-netbox)](go.mod)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](LICENSE)

`terraform-provider-netbox` manages objects in a [NetBox](https://netboxlabs.com/oss/netbox/) **4.x**
instance through its REST API. It is built on the Terraform Plugin Framework (protocol 6) and is
**generated from the NetBox OpenAPI document**: every CRUD collection of the API is available as a
resource, a single-object data source and a list data source (see [docs/RESOURCES.md](docs/RESOURCES.md)
for the current list).

> **Status: stable (1.x).** Releases follow semantic versioning: breaking changes only in a new major version, after a deprecation period; see the [versioning policy](docs/VERSIONING.md) and the [changelog](CHANGELOG.md).

### Supported NetBox versions

The provider tracks the NetBox 4.x release line. Each provider release is generated from, and tested
against, one pinned NetBox OpenAPI document; the pinned version lives in [`spec/VERSION`](spec/VERSION)
and moving to a new NetBox release is mostly a matter of pinning the new spec and regenerating (see
[Bumping the NetBox version](docs/DEVELOPMENT.md#bumping-the-netbox-version)); how NetBox releases map to
provider releases is described in [docs/VERSIONING.md](docs/VERSIONING.md).
At start-up the provider reads `/api/status/` and warns (without failing) when the server's minor
version differs from the one it was generated for; set `skip_version_check = true` to silence it.
v2 API tokens require NetBox 4.5 or later; legacy v1 tokens still work, with a deprecation warning.

#### Older NetBox releases

Only the pinned version (4.7.x) is fully supported. Older 4.x releases were checked once against the
provider's acceptance suite (2026-10-11, official `netboxcommunity/netbox` images, provider generated from the
4.7.2 spec). This is a snapshot, not something CI repeats: the nightly acceptance run covers the pinned version only.

| NetBox | Acceptance tests passed | What fails |
|---|---|---|
| 4.7.x (pinned) | 138 of 139 | One custom-field-value test; a fix (#33) landed on `main` after the run |
| 4.6.10 | 128 of 140 | Everything that is new in 4.7 (see below) |
| 4.5.10 | 122 of 140 | The 4.7 additions, plus the 4.6 additions |
| 4.4.10 | 108 of 140 | Not recommended: v2 tokens do not exist, and many resources return 404 or fail after apply |

What an older server lacks shows up in two ways, so check this list before using a newer resource or attribute:

* **Missing endpoint (`404 Not Found` on create).** Resources that need 4.7: the `netbox_cooling_*` family
  (source, feed, intake, outflow and their templates) and `netbox_module_bay_type`. Resources that need 4.6:
  `netbox_cable_bundle`, `netbox_virtual_machine_type` and `netbox_rack_group`.
* **Attribute NetBox silently drops (`Provider produced inconsistent result after apply`).** An older server
  ignores fields it does not know, so the value you set comes back null. Attributes that need 4.7:
  `cooling_capability` on `netbox_rack` and `netbox_rack_type`, `cooling_capacity` on `netbox_rack_type`, and
  `timeout` on `netbox_webhook`. `choice_colors` on `netbox_custom_field_choice_set` needs 4.6.
* **`netbox_service` and `netbox_service_template` need 4.7.** NetBox 4.5 and 4.6 require `ports`
  (`400 ports: This field is required`), which the provider does not expose: NetBox 4.7 derives it from
  `port_mappings`, so managing both would produce inconsistent results. Manage services on 4.5 and 4.6 outside
  Terraform, or upgrade NetBox.

Apart from the cases above, the suite passed on 4.5 and 4.6 (sites, devices, interfaces, IPAM, circuits,
tenancy, VPN, wireless, users, virtualization). Releases before 4.5 cannot use v2 tokens, and 4.4 was only run
with a legacy v1 token. No provider release targets an older NetBox, so on an older server expect the gaps listed
here and treat the version-check warning as meaningful.

### Acknowledgements

This project builds on the ideas and groundwork of
[e-breuninger/terraform-provider-netbox](https://github.com/e-breuninger/terraform-provider-netbox).
Many thanks to its maintainers and contributors for years of work that made NetBox usable from
Terraform.

That provider is hand-written on top of [fbreckle/go-netbox](https://github.com/fbreckle/go-netbox),
and we kept running into newer NetBox features that the client did not (yet) support, which meant
waiting on or patching two projects before a resource could be added. This provider takes a different
approach: both the API client and the Terraform resources are **generated directly from the NetBox
OpenAPI document**, so new NetBox releases can be picked up by regenerating rather than by hand-porting
each endpoint. It is a separate provider with its own schema conventions, not a drop-in replacement.

## Highlights

* **Full API coverage** generated from the spec, so recent additions (owners, cooling, cable bundles,
  rack groups, module bay types, virtual machine types, VLAN translation, virtual circuits,
  `port_mappings`, ...) are available as soon as the spec that introduces them is pinned.
* **Client-side rate limiting, artificial delays and retries** shared by all resources of a provider
  instance: token bucket (`requests_per_second`/`request_burst`), fixed delay per request and per
  write, optional serialization, exponential backoff with jitter honouring `Retry-After`.
* **v2 API tokens** (`nbt_<key>.<secret>`, NetBox 4.5+), sent as `Authorization: Bearer`; legacy v1 tokens are
  accepted with a deprecation warning (NetBox removes them in 5.0).
* **Resource identity and import** by NetBox ID for every resource.
* **Generic data-source filters**: any query parameter of the NetBox API can be used
  (`filters = [{ name = "tenant_id", value = "3" }]`).
* **Allocation resources**: `netbox_available_ip_address`, `netbox_available_prefix`,
  `netbox_available_vlan`, `netbox_available_asn`, and `netbox_device_primary_ip` /
  `netbox_virtual_machine_primary_ip` to break the device → interface → IP → device cycle.

## Installation

The provider is published on the [Terraform Registry](https://registry.terraform.io/providers/elliot/netbox)
as `elliot/netbox`. Declare it and run `terraform init`; no CLI configuration is needed:

```hcl
terraform {
  required_providers {
    netbox = {
      source  = "elliot/netbox"
      version = "~> 1.0"
    }
  }
}
```

The registry hosts the documentation of every resource and data source and verifies each release's signed
checksums. For hosts that cannot reach the registry, every GitHub Release (Linux, macOS, FreeBSD and Windows,
`amd64` and `arm64`) also installs through Terraform's off-registry channels under the same address: a
verified install script, a provider network mirror on GitHub Pages, and a filesystem mirror for air-gapped
machines. Those channels, lock-file guidance for mixed Linux/macOS teams and the local build/verify loop
(`make dist mirror mirror-check`) are described in [docs/INSTALL.md](docs/INSTALL.md).

Version numbers follow semantic versioning; what counts as a breaking change, the deprecation policy and the
supported NetBox and Terraform versions are described in [docs/VERSIONING.md](docs/VERSIONING.md).

## Usage

```hcl
terraform {
  required_providers {
    netbox = {
      source  = "elliot/netbox"
      version = "~> 1.0"
    }
  }
}

provider "netbox" {
  server_url          = "https://netbox.example.com" # or NETBOX_SERVER_URL
  api_token           = var.netbox_token             # or NETBOX_API_TOKEN (nbt_<key>.<secret>)
  requests_per_second = 10                           # optional client-side throttle
}

resource "netbox_tenant" "acme" {
  name = "ACME"
  slug = "acme"
}

resource "netbox_site" "dc1" {
  name      = "DC1"
  slug      = "dc1"
  status    = "active"
  tenant_id = netbox_tenant.acme.id
  tags      = ["managed-by-terraform"]
  custom_fields = {
    cost_center = "CC-42"
    owner_site  = 12 # object-typed field: the related object's ID
  }
}

data "netbox_prefixes" "dc1" {
  filters = [{ name = "site_id", value = netbox_site.dc1.id }]
}
```

### Provider configuration

| Attribute | Env var | Default | Description |
|---|---|---|---|
| `server_url` | `NETBOX_SERVER_URL` | | Base URL (a trailing `/api` is ignored) |
| `api_token` | `NETBOX_API_TOKEN` | | v2 token `nbt_<key>.<secret>` (a v1 token works, deprecated) |
| `requests_per_second` | `NETBOX_REQUESTS_PER_SECOND` | `0` (off) | Shared token-bucket rate limit |
| `request_burst` | `NETBOX_REQUEST_BURST` | `10` | Token-bucket burst |
| `request_delay_ms` | `NETBOX_REQUEST_DELAY_MS` | `0` | Fixed artificial delay before every request |
| `write_delay_ms` | `NETBOX_WRITE_DELAY_MS` | `0` | Additional delay before POST/PUT/PATCH/DELETE |
| `serialize_requests` | `NETBOX_SERIALIZE_REQUESTS` | `false` | One request at a time regardless of `-parallelism` |
| `max_retries` | `NETBOX_MAX_RETRIES` | `4` | Retries for 429 and 5xx (except 501) and network errors |
| `retry_wait_min_ms` / `retry_wait_max_ms` | `NETBOX_RETRY_WAIT_MIN_MS` / `_MAX_MS` | `1000` / `30000` | Backoff bounds; `Retry-After` is honoured |
| `request_timeout_ms` | `NETBOX_REQUEST_TIMEOUT_MS` | `60000` | Per-attempt timeout |
| `allow_insecure_https` | `NETBOX_ALLOW_INSECURE_HTTPS` | `false` | Skip TLS verification |
| `headers` | | | Extra request headers |
| `skip_version_check` | `NETBOX_SKIP_VERSION_CHECK` | `false` | Skip the `/api/status/` version check (a server on a different minor version only produces a warning) |
| `user_agent` | `NETBOX_USER_AGENT` | `terraform-provider-netbox/<version>` | User-Agent header |

### Conventions

* Foreign keys are numeric IDs named `<field>_id`; many-to-many relations are sets named `<field>_ids`.
  Generic relations keep `*_object_type` (`"dcim.interface"`) and `*_object_id`.
* `tags` is a set of tag **slugs**.
* `custom_fields` is a native object (`{ cost_center = "CC-42", vlan_id = 5, peers = [3, 7] }`): selection
  fields take the choice value, object fields the related object ID, multi-value fields a list and JSON fields
  any value. Field names are validated against the NetBox definitions before anything is sent, and only the keys
  present in your configuration are tracked, so fields managed elsewhere never cause drift. Data sources expose
  every field as a JSON string (`jsondecode(data.netbox_site.x.custom_fields)`, `"{}"` when the object has none),
  because dynamic values cannot be nested in list results. `netbox_custom_field_value` sets a single field on objects this configuration does not manage.
* Other free-form JSON fields (`local_context_data`, `data`, `parameters`, ...) are JSON strings (`jsonencode({...})`).
* Choice fields take the machine value (`active`, `1000base-t`, ...); attributes with a server-side default
  are `Optional` + `Computed`.
* Every resource can be imported by ID (`terraform import netbox_site.x 12`) or with an `import` block using
  `identity = { id = 12 }`.
* Data sources look up by `id`, by a few unique attributes (`name`, `slug`, `address`, ...) or by arbitrary
  API `filters`; list data sources return `items` (optionally capped with `limit`).

## Development

Prerequisites: Go 1.26+, Terraform ≥ 1.12 (the tests download one if missing), Java 17+ and Python 3 only
for regenerating the API client.

```sh
make build            # compile the provider
make test             # unit tests
make gen              # regenerate resources/data sources/tests/examples/doc templates from the spec
make docs             # render docs/ with tfplugindocs
make client-gen       # regenerate the API client in netbox/ (openapi-generator 7.26.0)
make demo-token       # provision a v2 token on https://demo.netbox.dev into .env.demo
set -a; . ./.env.demo; set +a
NETBOX_REQUESTS_PER_SECOND=5 make testacc ACC_SHARD='TestAccSite_basic|TestAccPrefix_basic'
make docker-up        # or run the pinned NetBox version locally (docker/README.md)
make sweep            # delete leftover tfacc-* objects
```

The generator, the overrides format and the testing workflow are described in
[docs/GENERATOR.md](docs/GENERATOR.md); repository layout and release steps in
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Historical per-app validation reports live in `docs/validation/`;
the nightly acceptance run supersedes them, and they are kept for their NetBox behaviour notes.

### Known limitations

* Bulk endpoints, ETag/`If-Match`, `add_tags`/`remove_tags`, background jobs, custom scripts, plugins, GraphQL
  are out of scope.
* Reverse sides of relations are read-only (`netbox_asn.site_ids`, provider accounts, user permissions); manage
  them from the owning side. `netbox_circuit.assignments` is not exposed (use `netbox_circuit_group_assignment`).
* Attributes with a server default are `Optional+Computed`; removing them keeps the current value.
* Image attachments and per-user UI models (notifications, subscriptions, bookmarks, saved filters, table
  configs) are not managed.

The full list, the roadmap and the sharp edges found during validation live in [docs/ROADMAP.md](docs/ROADMAP.md).

### Relationship to go-netbox

`netbox/` is a drop-in-shaped regeneration of [netbox-community/go-netbox](https://github.com/netbox-community/go-netbox)
for the pinned NetBox 4.x spec (upstream releases stopped at 4.3). The generation pipeline in
`tools/client-gen/` reuses go-netbox's openapi-generator configuration and adds spec fix-ups that make the
client resilient to new NetBox choices.

## Security

Report vulnerabilities privately; see [SECURITY.md](SECURITY.md), which also explains how to verify a
release's build provenance.

## License

[Mozilla Public License 2.0](LICENSE).
