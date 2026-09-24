# Validation: circuits + tenancy

Validated on 2026-09-15 against https://demo.netbox.dev (NetBox 4.7,
`demo-netbox-stable`) with `NETBOX_REQUESTS_PER_SECOND=2`.

Every acceptance test runs create -> in-place update -> singular and list
data sources -> import with `ImportStateVerify` -> empty plan. Fixtures and
attribute tuning live in `generator/overrides/circuits_tenancy.yaml`.

## Results

| Resource | Test | Result | Notes |
|---|---|---|---|
| `netbox_circuit` | `TestAccCircuit_basic` | PASS | `assignments` dropped (see generator issue 1). Update covers provider account, tenant, status, install/termination dates, commit rate, distance + unit, comments. |
| `netbox_circuit_group` | `TestAccCircuitGroup_basic` | PASS | update adds tenant, comments |
| `netbox_circuit_group_assignment` | `TestAccCircuitGroupAssignment_basic` | PASS | `member_type = "circuits.circuit"`; update sets `priority` |
| `netbox_circuit_termination` | `TestAccCircuitTermination_basic` | PASS | `termination_type = "dcim.site"`; update covers speeds, xconnect, pp_info, mark_connected |
| `netbox_circuit_type` | `TestAccCircuitType_basic` | PASS | update adds color + tag |
| `netbox_provider` | `TestAccProvider_basic` | PASS | `account_ids` dropped (see generator issue 2) |
| `netbox_provider_account` | `TestAccProviderAccount_basic` | PASS | passed only after issue 2 was worked around |
| `netbox_provider_network` | `TestAccProviderNetwork_basic` | PASS | update sets service_id |
| `netbox_virtual_circuit` | `TestAccVirtualCircuit_basic` | PASS | update adds provider account, tenant, status |
| `netbox_virtual_circuit_termination` | `TestAccVirtualCircuitTermination_basic` | PASS | full device chain; interface `type = "virtual"`; role peer -> hub |
| `netbox_virtual_circuit_type` | `TestAccVirtualCircuitType_basic` | PASS | auto fixture |
| `netbox_contact` | `TestAccContact_basic` | PASS | update sets group_ids, title, phone, email, address, link |
| `netbox_contact_assignment` | `TestAccContactAssignment_basic` | PASS | `object_type = "dcim.site"`; update sets priority |
| `netbox_contact_group` | `TestAccContactGroup_basic` | PASS | update nests under a parent |
| `netbox_contact_role` | `TestAccContactRole_basic` | PASS | auto fixture |
| `netbox_tenant` | `TestAccTenant_basic` | PASS | update adds group, tag, comments |
| `netbox_tenant_group` | `TestAccTenantGroup_basic` | PASS | update nests under a parent |

17 / 17 pass, 0 skipped.

### Scenario

`examples/scenarios/circuits/main.tf` (26 resources: tenant group/tenant,
contact group/role/contact assigned to a site, provider + account + network,
circuit type, circuit with an A termination at the site and a Z termination
on the provider network, circuit group + assignment, device chain with two
virtual interfaces, virtual circuit type, virtual circuit with hub and spoke
terminations, two list data sources) was applied with the locally built
provider: `apply` created 26 objects, a second `plan` was empty
(`-detailed-exitcode` 0) and `destroy` removed all 26.

## NetBox behaviours discovered

* **`circuits.assignments` is not writable and not returned.** `POST
  /api/circuits/circuits/` with `assignments: [{"group": 5, "priority":
  "primary"}]` fails with `{"assignments": ["Related object not found using
  the provided attributes: {'group': 5, 'priority': 'primary'}"]}` (the
  nested serializer only resolves *existing* assignment objects), and the
  circuit detail response does not contain an `assignments` key at all
  (`GET` returns no such field, `PATCH {"assignments": []}` is silently
  ignored). Group membership must be managed through
  `/api/circuits/circuit-group-assignments/`.
* **`providers.accounts` is a reverse relation.** Creating a provider
  account makes the provider's `accounts` list non-empty on the next read.
  It cannot be meaningfully set from the provider side.
* Circuit terminations accept `termination_type` values `dcim.site`,
  `dcim.location`, `dcim.region`, `dcim.sitegroup` and
  `circuits.providernetwork`; both `dcim.site` and `circuits.providernetwork`
  were exercised (test and scenario).
* Virtual circuit terminations were created on interfaces of `type =
  "virtual"`; `peer`, `hub` and `spoke` roles all work, and the role can be
  changed in place.
* `install_date` / `termination_date` (format `date`) round-trip as plain
  `YYYY-MM-DD` strings with no perpetual diff.
* `distance = 12.5` + `distance_unit = "km"` round-trips without a diff
  (after the `Float64Keep` emitter fix made by the coordinator).
* The demo instance occasionally times out (`context deadline exceeded`
  while awaiting headers) under concurrent load; one `TestAccCircuit_basic`
  run left nothing behind after such a timeout during post-test destroy (the
  objects were verified gone via the API).

## Generator issues

### 1. `netbox_circuit.assignments`: nested list generated from a non-functional request field

* Resource / attribute: `netbox_circuit` / `assignments` (nested list of
  `group_id`, `priority`).
* Symptom: setting it fails with the NetBox 400 quoted above; leaving it
  unset works only because the read serializer never returns the key, so
  the attribute is dead weight in the schema and docs.
* Workaround applied: `circuits.attributes.assignments: { skip: true }` in
  `generator/overrides/circuits_tenancy.yaml`.
* Proposed fix: in `internal/gen/build`, drop request properties whose
  schema is an array of `Brief*Serializer_Request` items **and** that are
  absent from the read schema (or mark them read-only when present). More
  generally, never expose a nested write-only list that the read
  serializer does not echo back, since it can never be refreshed.

### 2. Reverse-FK lists (`netbox_provider.account_ids`) produce a perpetual diff

* Resource / attribute: `netbox_provider` / `account_ids` (API `accounts`).
* Error: after creating a `netbox_provider_account`, the refresh plan of the
  parent provider is not empty:

  ```
  # netbox_provider.test will be updated in-place
  ~ resource "netbox_provider" "test" {
      ~ account_ids  = [
          - 2,
        ]
      ~ display      = "tfacc-irkuugcb" -> (known after apply)
      ~ last_updated = "2026-09-15T13:00:18Z" -> (known after apply)
    }
  ```

  This broke `TestAccProviderAccount_basic`, `TestAccCircuit_basic` (step 2
  adds an account) and `TestAccVirtualCircuit_basic`, and the scenario plan.
* Workaround applied: `providers.attributes.accounts: { skip: true }` (also
  removes it from the `netbox_provider` data source, where it would have
  been useful).
* Proposed fix: classify an `fk_list` as **Computed-only** (resource) /
  read-only (data source) when its target resource has a *required* FK
  pointing back at this resource (here `provider_account.provider` is
  required), i.e. when the list is the reverse side of a one-to-many
  relation. Alternatively add an override key such as `read_only: true`
  that keeps the attribute Computed in the resource and present in the data
  source instead of `skip`, which is too coarse.

### 3. (informational) `display` / `last_updated` shown as `(known after apply)` on unrelated diffs

Whenever any attribute changes, `display` and `last_updated` are planned as
unknown. That is correct (they do change), but it makes every spurious diff
noisier; consider `UseStateForUnknown` only for `display` when no
name-like attribute changes. Not a blocker.

## Files touched by this shard

* `generator/overrides/circuits_tenancy.yaml` (new): fixtures for all 17
  resources, descriptions for the ones that had none, two attribute skips.
* `internal/provider/gen/circuits/*`, `internal/provider/gen/tenancy/*`:
  regenerated with `go run ./internal/gen -only ...`.
* `examples/resources/netbox_<name>/{resource.tf,import.sh}` and
  `examples/data-sources/netbox_<name>/data-source.tf` for every resource
  and list data source in the shard (hand-maintained).
* `examples/scenarios/circuits/main.tf` (new).
