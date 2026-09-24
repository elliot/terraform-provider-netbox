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

> **Status: pre-release.** Names, attributes and behaviour may change until `v0.1.0`.

### Supported NetBox versions

The provider tracks the NetBox 4.x release line. Each provider release is generated from, and tested
against, one pinned NetBox OpenAPI document; the pinned version lives in [`spec/VERSION`](spec/VERSION)
and moving to a new NetBox release is mostly a matter of pinning the new spec and regenerating (see
[Bumping the NetBox version](docs/DEVELOPMENT.md#bumping-the-netbox-version)).
At start-up the provider reads `/api/status/` and warns (without failing) when the server's minor
version differs from the one it was generated for; set `skip_version_check = true` to silence it.
v2 API tokens require NetBox 4.5 or later.

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
* **v2 API tokens only** (`nbt_<key>.<secret>`, NetBox 4.5+), sent as `Authorization: Bearer`.
* **Resource identity and import** by NetBox ID for every resource.
* **Generic data-source filters**: any query parameter of the NetBox API can be used
  (`filters = [{ name = "tenant_id", value = "3" }]`).
* **Allocation resources**: `netbox_available_ip_address`, `netbox_available_prefix`,
  `netbox_available_vlan`, `netbox_available_asn`, and `netbox_device_primary_ip` /
  `netbox_virtual_machine_primary_ip` to break the device → interface → IP → device cycle.

## Installation

The provider is not on the public Terraform Registry yet. Releases on GitHub ship zips for Linux and
macOS (`amd64`, `arm64`) and install through any of Terraform's off-registry channels while keeping the
normal `source = "elliot/netbox"` address:

```sh
# 1. one machine: verified download into Terraform's implied local mirror, no CLI config needed
curl -fsSL https://raw.githubusercontent.com/elliot/terraform-provider-netbox/main/scripts/install.sh | sh
```

```hcl
# 2. teams: provider network mirror published on GitHub Pages (~/.terraformrc)
provider_installation {
  network_mirror {
    url     = "https://elliot.github.io/terraform-provider-netbox/"
    include = ["elliot/netbox"]
  }
  direct {
    exclude = ["elliot/netbox"]
  }
}
```

A filesystem mirror for air-gapped hosts, lock-file guidance for mixed Linux/macOS teams and the local
build/verify loop (`make dist mirror mirror-check`) are described in [docs/INSTALL.md](docs/INSTALL.md).

## Usage

```hcl
terraform {
  required_providers {
    netbox = {
      source  = "elliot/netbox"
      version = "~> 0.1"
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
| `api_token` | `NETBOX_API_TOKEN` | | v2 token `nbt_<key>.<secret>` |
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
  every field as a JSON string (`jsondecode(data.netbox_site.x.custom_fields)`), because dynamic values cannot
  be nested in list results. `netbox_custom_field_value` sets a single field on objects this configuration does not manage.
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
make client-gen       # regenerate the API client in netbox/ (openapi-generator 7.11.0)
make demo-token       # provision a v2 token on https://demo.netbox.dev into .env.demo
set -a; . ./.env.demo; set +a
NETBOX_REQUESTS_PER_SECOND=5 make testacc ACC_SHARD='TestAccSite_basic|TestAccPrefix_basic'
make registry-smoke   # published registry release end to end against the demo
make docker-up        # or run the pinned NetBox version locally (docker/README.md)
make sweep            # delete leftover tfacc-* objects
```

The generator, the overrides format and the testing workflow are described in
[docs/GENERATOR.md](docs/GENERATOR.md); repository layout and release steps in
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Per-app validation reports against the public demo live in
`docs/validation/`.

### Known limitations

* Bulk endpoints, ETag/`If-Match`, `add_tags`/`remove_tags`, background jobs, custom scripts, plugins, GraphQL
  and v1 tokens are out of scope.
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
