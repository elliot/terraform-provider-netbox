# Changelog

All notable changes to this project are documented in this file. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

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
- HTTP transport chain with client-side token-bucket rate limiting, fixed delays, retry with
  exponential backoff (honouring `Retry-After`), request logging with token redaction.
- Version check against `GET /api/status/` on configure (warning when the server is not 4.7.x).
- Local NetBox 4.7 docker compose stack and `scripts/demo-token.sh` for the public demo.
