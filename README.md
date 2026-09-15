# Terraform Provider for NetBox 4.7

`terraform-provider-netbox` manages objects in a [NetBox](https://netboxlabs.com/oss/netbox/) **4.7**
instance through its REST API. It is built on the Terraform Plugin Framework (protocol 6) and is
**generated from the NetBox 4.7.0 OpenAPI document**: every CRUD collection of the API is available
as a resource, a single-object data source and a list data source (132 resources, 266 data sources,
see [docs/RESOURCES.md](docs/RESOURCES.md)).

> **Status: pre-release.** Names, attributes and behaviour may change until `v0.1.0`.

## Highlights

* **Complete 4.7 coverage** including 4.5–4.7 additions (owners, cooling, cable bundles, rack groups,
  module bay types, virtual machine types, VLAN translation, virtual circuits, `port_mappings`).
* **Client-side rate limiting, artificial delays and retries** shared by all resources of a provider
  instance: token bucket (`requests_per_second`/`request_burst`), fixed delay per request and per
  write, optional serialization, exponential backoff with jitter honouring `Retry-After`.
* **v2 API tokens only** (`nbt_<key>.<secret>`, NetBox ≥ 4.5), sent as `Authorization: Bearer`.
* **Resource identity and import** by NetBox ID for every resource.
* **Generic data-source filters**: any query parameter of the NetBox API can be used
  (`filters = [{ name = "tenant_id", value = "3" }]`).
* **Allocation resources**: `netbox_available_ip_address`, `netbox_available_prefix`,
  `netbox_available_vlan`, `netbox_available_asn`, and `netbox_device_primary_ip` /
  `netbox_virtual_machine_primary_ip` to break the device → interface → IP → device cycle.

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
  custom_fields = jsonencode({
    cost_center = "CC-42"
  })
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
| `skip_version_check` | `NETBOX_SKIP_VERSION_CHECK` | `false` | Skip the `/api/status/` version check (a non-4.7 server only produces a warning) |
| `user_agent` | `NETBOX_USER_AGENT` | `terraform-provider-netbox/<version>` | User-Agent header |

### Conventions

* Foreign keys are numeric IDs named `<field>_id`; many-to-many relations are sets named `<field>_ids`.
  Generic relations keep `*_object_type` (`"dcim.interface"`) and `*_object_id`.
* `tags` is a set of tag **slugs**.
* `custom_fields` and other free-form JSON fields (`local_context_data`, `data`, ...) are JSON strings;
  only the custom-field keys present in your configuration are tracked, so fields managed elsewhere never
  cause drift. Selection fields, which 4.7 returns as `{value,label}` objects, are unwrapped to their value.
* Choice fields take the machine value (`active`, `1000base-t`, ...); attributes with a server-side default
  are `Optional` + `Computed`.
* Every resource can be imported by ID (`terraform import netbox_site.x 12`) or with an `import` block using
  `identity = { id = 12 }`.
* Data sources look up by `id`, by a few unique attributes (`name`, `slug`, `address`, ...) or by arbitrary
  API `filters`; list data sources return `items` (optionally capped with `limit`).

## Development

Prerequisites: Go 1.25+, Terraform ≥ 1.12 (the tests download one if missing), Java 17+ and Python 3 only
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
make docker-up        # or run NetBox 4.7 locally (docker/README.md)
make sweep            # delete leftover tfacc-* objects
```

The generator, the overrides format and the testing workflow are described in
[docs/GENERATOR.md](docs/GENERATOR.md); repository layout and release steps in
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Per-app validation reports against the public demo live in
`docs/validation/`.

### Relationship to go-netbox

`netbox/` is a drop-in-shaped regeneration of [go-netbox](https://github.com/netbox-community/go-netbox)
for NetBox 4.7 (upstream stopped at 4.3). The generation pipeline in `tools/client-gen/` reuses go-netbox's
openapi-generator configuration and adds spec fix-ups that make the client resilient to new NetBox choices.

## License

[Mozilla Public License 2.0](LICENSE).
