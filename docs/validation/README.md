# Validation reports

One report per validation shard. The pass/fail counts below come from a full
run of the repo's acceptance workflow (Actions run 92, `workflow_dispatch` on
`main` at `d57b8a0`, 2026-10-11): 12 shards, one docker-compose NetBox
(`v4.7` image tag) each. The narrative in the reports was written against
https://demo.netbox.dev (NetBox 4.7.0) before the generator fixes in commits
`39df732` (validation findings), `38e5743` (changed-only PATCH, no nulls on
create) and `92de7af` (native `custom_fields`). Issues they describe as
"worked around" or "proposed" may therefore be fixed already; see the "Fixed"
list in [ROADMAP.md](../ROADMAP.md) and the CHANGELOG. The generator-issue
sections were not rewritten: the run does not prove those fixed, since the
overrides that skip `circuits.assignments` and `providers.accounts` are still
in place. The NetBox behaviour notes remain accurate.

Run 92 totals: 139 tests, 138 pass, 1 fail
(`TestAccCustomFieldValue_basic` in the `manual` shard); the `internal/provider`
shard (2 tests) has no report here.

| Report | Scope | Result in run 92 |
|---|---|---|
| [dcim-a.md](dcim-a.md) | sites, racks, power, cooling, catalogue (28) | 28 pass (`dcim` shard 52/52) |
| [dcim-b.md](dcim-b.md) | devices, components, templates, modules, cables, virtual chassis (24) | 24 pass (`dcim` shard 52/52) |
| [ipam-core.md](ipam-core.md) | ipam + core data sources (19) | 19 pass (`ipam` 18, `core` 1) |
| [extras-wireless.md](extras-wireless.md) | extras + wireless (15) | 15 pass (`extras` 12, `wireless` 3) |
| [circuits-tenancy.md](circuits-tenancy.md) | circuits + tenancy (17) | 17 pass (`circuits` 11, `tenancy` 6) |
| [virt-users.md](virt-users.md) | virtualization + users (13) | 13 pass (`virtualization` 7, `users` 6) |
| [vpn.md](vpn.md) | vpn (10) | 10 pass |
| [manual.md](manual.md) | hand-written allocation and primary IP resources (6) | 10 of 11 tests pass; `TestAccCustomFieldValue_basic` fails |
