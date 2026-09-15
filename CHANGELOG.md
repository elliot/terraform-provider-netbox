# Changelog

All notable changes to this project are documented in this file. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Provider scaffolding for NetBox 4.7 on terraform-plugin-framework (protocol 6).
- Provider configuration (`server_url`, `api_token`, rate limiting, request/write delays,
  request serialisation, retries with backoff, timeouts, extra headers, TLS options,
  `skip_version_check`, `user_agent`), each also settable through `NETBOX_*` environment variables.
- HTTP transport chain with client-side token-bucket rate limiting, fixed delays, retry with
  exponential backoff (honouring `Retry-After`), request logging with token redaction.
- Version check against `GET /api/status/` on configure (warning when the server is not 4.7.x).
- Local NetBox 4.7 docker compose stack and `scripts/demo-token.sh` for the public demo.
