# Terraform Provider for NetBox

`terraform-provider-netbox` manages objects in a [NetBox](https://netbox.dev) **4.7** instance
through its REST API using the Terraform Plugin Framework (protocol 6). Resources and data
sources are generated from the pinned NetBox 4.7.0 OpenAPI schema; the provider adds client-side
rate limiting, fixed request delays, retries with backoff (`Retry-After` aware) and supports only
NetBox **v2 API tokens** (`nbt_<key>.<secret>`).

> **Work in progress.** Nothing here is released yet; names, attributes and behaviour may change
> without notice until `v0.1.0`.

## Quick start

```hcl
provider "netbox" {
  server_url          = "https://netbox.example.com" # or NETBOX_SERVER_URL
  api_token           = var.netbox_token             # or NETBOX_API_TOKEN, must be a v2 token
  requests_per_second = 5                            # optional client-side throttle
}
```

## Development

```sh
make build            # compile
make test             # unit tests (downloads a Terraform CLI on first run)
make demo-token       # provision a token on https://demo.netbox.dev into .env.demo
set -a; . ./.env.demo; set +a
make testacc          # acceptance tests against the NetBox in the environment
make docker-up        # or run NetBox 4.7 locally (see docker/README.md)
```

Licensed under the [Mozilla Public License 2.0](LICENSE).
