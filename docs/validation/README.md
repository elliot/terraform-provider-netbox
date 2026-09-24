# Validation reports

One report per validation shard, written by the agents that ran the acceptance
suite and scenarios against https://demo.netbox.dev (NetBox 4.7.0).

> These reports were written **before** the generator fixes in commits
> `39df732` (validation findings), `38e5743` (changed-only PATCH, no nulls on
> create) and `92de7af` (native `custom_fields`). Several issues they describe
> as "worked around" or "proposed" are fixed; see the "Fixed" list in
> [ROADMAP.md](../ROADMAP.md) and the CHANGELOG. The NetBox behaviour notes
> remain accurate.

| Report | Scope | Result at the time |
|---|---|---|
| [dcim-a.md](dcim-a.md) | sites, racks, power, cooling, catalogue (28) | 28 pass |
| [dcim-b.md](dcim-b.md) | devices, components, templates, modules, cables, virtual chassis (24) | 24 pass |
| [ipam-core.md](ipam-core.md) | ipam + core data sources (19) | 19 pass |
| [extras-wireless.md](extras-wireless.md) | extras + wireless (15) | 15 pass |
| [circuits-tenancy.md](circuits-tenancy.md) | circuits + tenancy (17) | 17 pass |
| [virt-users.md](virt-users.md) | virtualization + users (13) | 13 pass |
| [vpn.md](vpn.md) | vpn (10) | 10 pass |
| [manual.md](manual.md) | hand-written allocation and primary IP resources (6) | 7 tests pass |
