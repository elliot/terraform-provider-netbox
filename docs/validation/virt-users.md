# Validation: virtualization + users (NetBox 4.7.0, demo.netbox.dev)

Shard: `cluster`, `cluster_group`, `cluster_type`, `virtual_disk`,
`virtual_machine`, `virtual_machine_type`, `vm_interface` (virtualization);
`owner`, `owner_group`, `permission`, `token`, `user`, `user_group` (users).

Fixtures and attribute tuning live in `generator/overrides/virt_users.yaml`.
Every acceptance test runs create -> in-place update -> data sources (by id
and list filter) -> import with `ImportStateVerify` -> empty plan against
https://demo.netbox.dev with `NETBOX_REQUESTS_PER_SECOND=2`.

## Results

| Resource | Test | Result | Notes |
|---|---|---|---|
| `netbox_cluster` | `TestAccCluster_basic` | PASS | update adds group, tenant, `dcim.site` scope, status, comments |
| `netbox_cluster_group` | `TestAccClusterGroup_basic` | PASS | auto fixture |
| `netbox_cluster_type` | `TestAccClusterType_basic` | PASS | auto fixture |
| `netbox_virtual_disk` | `TestAccVirtualDisk_basic` | PASS | needs `virtual_machine.disk` computed (see below) |
| `netbox_virtual_machine` | `TestAccVirtualMachine_basic` | PASS | cluster + site + type, `vcpus = 2.5`, memory, disk, `start_on_boot`, `local_context_data` |
| `netbox_virtual_machine_type` | `TestAccVirtualMachineType_basic` | PASS | auto fixture |
| `netbox_vm_interface` | `TestAccVmInterface_basic` | PASS | tagged mode, untagged/tagged VLANs, VRF, MTU, enabled=false |
| `netbox_owner` | `TestAccOwner_basic` | PASS | members: user group + user |
| `netbox_owner_group` | `TestAccOwnerGroup_basic` | PASS | auto fixture |
| `netbox_permission` | `TestAccPermission_basic` | PASS | after dropping `permissions` from user/group (generator issue 3) |
| `netbox_token` | `TestAccToken_basic` | PASS | after `pepper_id: computed` (generator issue 2); secret not exposed (issue 1) |
| `netbox_user` | `TestAccUser_basic` | PASS | update: names, email, `is_active=false`, `group_ids`; import ignores `password` |
| `netbox_user_group` | `TestAccUserGroup_basic` | PASS | auto fixture |

13 PASS, 0 SKIP.

Scenario `examples/scenarios/virtualization/main.tf` (cluster type/group,
site-scoped cluster, VM type, VM with two interfaces and two virtual disks, an
IP address assigned to `eth0`, owner group + owner, user group + user +
permission; 17 resources): `terraform apply` created all 16
resources, a second `apply` only recorded the learned `disk` output,
`terraform plan -detailed-exitcode` returned 0 and `terraform destroy`
removed everything (built provider via `dev_overrides`, 2 requests/s).

## NetBox behaviours discovered

* **Virtual machine `disk` is derived from virtual disks.** As soon as a
  `netbox_virtual_disk` exists, NetBox overwrites `virtual_machine.disk` with
  the sum of the disk sizes. With the generator's default mapping (nullable
  int -> plain `Optional`) a VM without `disk` in its configuration showed a
  perpetual `disk = 10240 -> null` update after its first disk was created.
  `virt_users.yaml` forces `disk` to `computed: true` (Optional+Computed with
  `UseStateForUnknown`), so the server value is tracked; a VM that has disks
  should simply not set `disk`.
  Corollary observed in the scenario: NetBox recomputes the total in
  `VirtualDisk.save()`, so two disks created concurrently (Terraform's default
  parallelism) raced and the VM ended up with `disk = 40960` although its two
  disks total 245760 (`virtual_disk_count = 2`). Creating disks one at a time
  (`-parallelism=1` or `depends_on` between disks) avoids it; the provider
  cannot fix it. Read-only `virtual_disk_count` is exposed on the data source.
* **Tokens: `pepper_id` is populated server-side.** v2 tokens (the 4.7
  default) come back with `pepper_id = 1` although the request never sent it.
  Mapped as plain nullable `Optional` this produced `Provider produced
  inconsistent result after apply: .pepper_id: was null, but now
  cty.NumberIntVal(1)`. Forced to `computed: true` in the overrides.
* **Tokens: the secret is returned once.** `POST /api/users/tokens/` returns
  `token` (the full `nbt_<key>.<secret>` string) exactly once; every later
  read returns `"token": null` and only the 12-character `key`. The demo API
  also returns an `allowed_ips` list that is missing from the 4.7.0 OpenAPI
  document.
* **`permission.user_ids/group_ids` and `user.permission_ids` are the same
  M2M relation.** Assigning a permission to a user or group from the
  permission side immediately shows up in the user's/group's `permissions`
  list; with both sides modelled as `Optional+Computed` sets defaulting to
  `[]`, the user and group resources produced a perpetual
  `permission_ids = [- 2]` diff after `netbox_permission.test` was updated
  with `user_ids`/`group_ids`. Listing the permission on both sides would be
  a dependency cycle, so `virt_users.yaml` drops `permissions` from
  `netbox_user` and `netbox_user_group`; permissions are assigned from
  `netbox_permission` only (same model as the e-breuninger provider).
  `user.group_ids` has no writable reverse side on `netbox_user_group`, so it
  stays.
* **Owner requires an owner group** (`group` is `required` even though
  the schema marks it nullable). Owners accept users and user groups as
  members; owner membership does not appear on the user resource.
* **Cluster scope**: `scope_type` (`dcim.site`, `dcim.location`,
  `dcim.region`, `dcim.sitegroup`) + `scope_id` are optional and can be set
  in a later update. A VM may set `site_id` together with `cluster_id` when
  the cluster has no scope or the same site.
* **VM `vcpus`** is a decimal (`2.5` round-trips exactly); `memory`/`disk`
  are MB. `start_on_boot` (`on`/`off`/`laststate`) is new in 4.7.
* **VM interfaces**: `mode = "tagged"` with `untagged_vlan_id` +
  `tagged_vlan_ids` and `vrf_id` round-trip cleanly; `mac_address` on the
  interface is effectively read-only in 4.2+ (MAC addresses are separate
  `netbox_mac_address` objects linked via `primary_mac_address_id`) and was
  left out of fixtures and examples.
* **Users**: `password` is write-only; the resource keeps the configured
  value in state, re-sends it on every PATCH (harmless) and the import step
  ignores it. `is_active = false` and `group_ids` update in place.
* **Permissions**: `object_types` use `app_label.model`, `constraints` is
  arbitrary JSON (object or list) and round-trips through `jsonencode`.

## Generator issues (not patched here; recorded for the generator owner)

1. **Token secret not exposed on `netbox_token`** (resource `token`,
   attributes `token`, `key`). Both are `readOnly` in the `Token` schema, so
   the classifier makes them data-source-only; the resource therefore never
   captures the secret that only the create response contains, which makes
   the resource useless for bootstrapping API access. Proposed fix: for
   read-only properties that are present in the create response but
   nullable/absent afterwards (`token`), add a `write_once`/`create_only`
   attribute override that emits a `Computed` + `Sensitive` +
   `UseStateForUnknown` attribute populated from the create response and kept
   from prior state on read/import (like `password` today); `key` can be a
   normal `Computed` attribute. Until then the example documents the gap.
2. **Nullable scalars with a server-side default** (`token.pepper_id`,
   `virtual_machine.disk`). The rule "nullable scalar -> Optional" assumes
   NetBox returns `null` when the client sends nothing; when NetBox fills a
   value the result is `inconsistent result after apply` (create) or a
   perpetual diff (refresh). Worked around per attribute with
   `computed: true`. Proposed fix: treat nullable integers/floats as
   Optional+Computed with `UseStateForUnknown` by default, and let PATCH send
   an explicit `null` only when the attribute is present-but-null in the
   configuration (needs config inspection, not just plan) so users can still
   clear them.
3. **Bidirectional M2M lists** (`users.permissions`, `groups.permissions`
   vs `permissions.users/groups`). Both ends of the same relation are emitted
   as sets with an empty default, guaranteeing a diff on one side. Worked
   around with `skip: true` on the reverse side. Proposed fix: detect
   reverse accessors (property whose target's request schema has a list
   pointing back at this resource) and emit them read-only (data source only)
   or Optional+Computed without a default.
4. **Spec gap, not a generator bug**: `Token.allowed_ips` (list of CIDRs) is
   returned by the 4.7.0 API but absent from the OpenAPI document, so it
   cannot be managed.

## Files

* `generator/overrides/virt_users.yaml`: fixtures for all 13 resources,
  `virtual-machines.disk.computed`, `tokens.pepper_id.computed`,
  `users.permissions.skip`, `groups.permissions.skip`.
* `examples/resources/netbox_<name>/{resource.tf,import.sh}` and
  `examples/data-sources/netbox_<name>/data-source.tf` (singular and plural)
  for every resource in the shard, hand-maintained.
* `examples/scenarios/virtualization/main.tf`.
