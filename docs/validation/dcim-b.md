# Validation: dcim shard B (device components, templates, modules, VC)

Target: https://demo.netbox.dev (NetBox 4.7.0), provider built from this tree,
`NETBOX_REQUESTS_PER_SECOND=2`. Every resource runs the generated
`TestAcc<Name>_basic` flow: create -> in-place update -> singular and list data
source lookups -> import with `ImportStateVerify` -> empty plan.

Fixtures and per-resource tuning live in `generator/overrides/dcim_b.yaml`
(device, device_type, interface and cable keep their fixtures in
`_fixtures_pilot.yaml`). Hand-maintained examples are under
`examples/resources/netbox_<name>/` and `examples/data-sources/netbox_<name>/`
(plus the list data sources); the end-to-end scenario is
`examples/scenarios/dcim-b/main.tf`.

## Results

| Terraform resource | Test | Result | Notes |
|---|---|---|---|
| `netbox_device` | `TestAccDevice_basic` | PASS | pilot fixture |
| `netbox_device_type` | `TestAccDeviceType_basic` | PASS | pilot fixture |
| `netbox_device_bay` | `TestAccDeviceBay_basic` | PASS | parent-type device; update installs a `u_height = 0` child device |
| `netbox_device_bay_template` | `TestAccDeviceBayTemplate_basic` | PASS | device type needs `subdevice_role = "parent"` |
| `netbox_interface` | `TestAccInterface_basic` | PASS | pilot fixture |
| `netbox_interface_template` | `TestAccInterfaceTemplate_basic` | PASS | PoE, mgmt_only, enabled covered |
| `netbox_console_port` | `TestAccConsolePort_basic` | PASS | `speed`, `mark_connected` covered |
| `netbox_console_port_template` | `TestAccConsolePortTemplate_basic` | PASS | |
| `netbox_console_server_port` | `TestAccConsoleServerPort_basic` | PASS | |
| `netbox_console_server_port_template` | `TestAccConsoleServerPortTemplate_basic` | PASS | |
| `netbox_front_port` | `TestAccFrontPort_basic` | PASS | update must change the mapping set, see G1 |
| `netbox_front_port_template` | `TestAccFrontPortTemplate_basic` | PASS | same as above |
| `netbox_rear_port` | `TestAccRearPort_basic` | PASS | `front_ports` dropped from the schema, see G2 |
| `netbox_rear_port_template` | `TestAccRearPortTemplate_basic` | PASS | `front_ports` dropped from the schema, see G2 |
| `netbox_module` | `TestAccModule_basic` | PASS | status/serial/asset_tag update |
| `netbox_module_bay` | `TestAccModuleBay_basic` | PASS | `installed_module` dropped from the schema, see G3 |
| `netbox_module_bay_template` | `TestAccModuleBayTemplate_basic` | PASS | |
| `netbox_module_type` | `TestAccModuleType_basic` | PASS | no `name`/`slug`: singular data source by `id`/filters only |
| `netbox_inventory_item` | `TestAccInventoryItem_basic` | PASS | update links role, manufacturer and a `dcim.interface` component |
| `netbox_inventory_item_role` | `TestAccInventoryItemRole_basic` | PASS | |
| `netbox_inventory_item_template` | `TestAccInventoryItemTemplate_basic` | PASS | |
| `netbox_cable` | `TestAccCable_basic` | PASS | pilot fixture |
| `netbox_virtual_chassis` | `TestAccVirtualChassis_basic` | PASS | `master_id` not exercised, see B5 |
| `netbox_virtual_device_context` | `TestAccVirtualDeviceContext_basic` | PASS | tenant, identifier and status update |

24 resources: 24 PASS, 0 SKIP. Three schema adjustments were needed
(`generator/overrides/dcim_b.yaml`): `rear-ports`/`rear-port-templates`
`front_ports: { skip: true }` and `module-bays` `installed_module: { skip: true }`.

Run the shard with:

```sh
set -a; source .env.demo; set +a
export TF_ACC=1 NETBOX_REQUESTS_PER_SECOND=2
export TF_ACC_TERRAFORM_PATH=$(command -v terraform)   # avoids the checkpoint download
go test ./internal/provider/gen/dcim/ -count=1 -v -timeout 40m -parallel 4 -run \
  'TestAcc(Device|DeviceType|DeviceBay|DeviceBayTemplate|Interface|InterfaceTemplate|ConsolePort|ConsolePortTemplate|ConsoleServerPort|ConsoleServerPortTemplate|FrontPort|FrontPortTemplate|RearPort|RearPortTemplate|Module|ModuleBay|ModuleBayTemplate|ModuleType|InventoryItem|InventoryItemRole|InventoryItemTemplate|Cable|VirtualChassis|VirtualDeviceContext)_basic$'
```

## Scenario (`examples/scenarios/dcim-b/main.tf`)

Applied and destroyed on the demo with the locally built provider
(`dev_overrides`), 49 resources:

* switch device type with interface (PoE), console, power port, module bay and
  inventory item templates; two switches instantiated from it inside a virtual
  chassis (`virtual_chassis_id`, `vc_position`, `vc_priority`);
* NetBox instantiated the templates: each switch was born with
  `GigabitEthernet0/0`, `GigabitEthernet1/0/1-4`, `Console`, `PSU1`/`PSU2`,
  `Network Module 1`, `PSU 1`/`PSU 2`; a `data "netbox_interface"` with
  `device_id` + `name` filters reads one of them back;
* explicit uplink interfaces, a console server port, a power port, a second
  module bay and an installed module per switch; the module type's eight
  `TenGigabitEthernet1/{module}/N` templates were replicated as
  `TenGigabitEthernet1/2/1-8` (bay position `2`);
* a patch panel with a 12-position MPO rear port and four LC front ports mapped
  to positions 1-4;
* a `smf-os2` cable between the two switches and one from a switch interface to
  a front port (`dcim.frontport` termination);
* an inventory item attached to an interface and a virtual device context.

`terraform apply` -> `terraform plan` reported `No changes`; `terraform destroy`
removed all 49 objects and left nothing behind (`?q=tfacc-dcim-b` on devices,
device types, module types, virtual chassis, sites, manufacturers, roles: 0).

## NetBox behaviours discovered

* **B1 - templates instantiate on device creation only.** Components are
  copied from the device type when the device is created (and from the module
  type when a module is installed with `replicate_components`). A template
  added later does not touch existing devices. In Terraform, add
  `depends_on = [netbox_interface_template.x, ...]` to the device so templates
  exist first; components created this way are not managed by Terraform (read
  them with data sources).
* **B2 - component names must be unique per device.** Creating a
  `netbox_interface` whose name collides with a templated or module-replicated
  interface fails with `The fields device, name must make a unique set`.
  Module templates use `{module}` which becomes the module bay `position`, so
  pick explicit names/positions that do not collide.
* **B3 - front/rear port mappings (4.7 multi-position model).** A front port
  carries `positions` (default 1) and `rear_ports = [{ position, rear_port,
  rear_port_position }]`. `(rear_port, rear_port_position)` must be unique
  across all front ports of the device. Re-sending the current mapping set in
  a PATCH/PUT is rejected with
  `rear_ports: {"0":{"__all__":["The fields rear_port, rear_port_position must make a unique set."]}}`
  (verified with curl: identical set -> 400, changed position -> 200, PATCH
  without `rear_ports` -> 200). See G1.
* **B4 - `installed_module` on module bays is read-only in practice.** The
  4.7.0 serializer lists it as writable, but any write containing the key
  (even `null`) answers `500 ... The following fields do not exist in this
  model: installed_module`. The installed module is derived from
  `netbox_module.module_bay_id`.
* **B5 - virtual chassis master.** NetBox does not elect a master
  automatically; `master_id` must reference a device that is already a member.
  Members join via `netbox_device.virtual_chassis_id`, which already depends on
  the chassis, so setting `master_id` on the chassis in the same configuration
  would be a dependency cycle. Set it in a second apply step, or leave it
  empty; the scenario and test leave it empty.
* **B6 - device bays.** Only devices whose type has
  `subdevice_role = "parent"` can have bays; the installed device's type must
  be `subdevice_role = "child"` with `u_height = 0`.
* **B7 - `netbox_module_type` has no name or slug.** The singular data source
  supports `id` and `filters` only (`manufacturer` + `model`).
* **B8 - console port `speed` is an integer choice** (1200 ... 115200) and
  `type` may be empty; component templates take exactly one of `device_type_id`
  / `module_type_id`.
* **B9 - demo instance quirk.** The plugin-test harness tries to contact
  checkpoint-api.hashicorp.com when `TF_ACC_TERRAFORM_PATH` is unset and the
  whole test binary aborts if that times out; always set it.

## Generator issues

### G1 - PATCH re-sends unchanged nested lists (`front_port.rear_ports`)

*Resources*: `netbox_front_port`, `netbox_front_port_template`
(attribute `rear_ports`).

*Symptom*: any in-place update that does not change the mapping set fails:

```
Error updating netbox_front_port
NetBox API error: 400 Bad Request: rear_ports: {"0":{"__all__":["The fields
rear_port, rear_port_position must make a unique set."]}}
```

Cause: `Update` builds the PATCH body from the full plan, so the current
`rear_ports` list is sent again and NetBox 4.7.0 tries to insert the mapping
rows before removing the old ones (B3). The acceptance fixtures therefore move
the mapping to `rear_port_position = 2` in the update step; a real user
changing only `label` hits the error.

*Proposed fix*: in the generated `Update`, compare each nested-list attribute
with the prior state and omit it from the PATCH when unchanged (the same
change would be sensible for every attribute: send only what differs from
state). Alternatively add a `patch_when_changed: true` attribute override.

### G2 - computed nested lists maintained by the server cause perpetual diffs

*Resources*: `netbox_rear_port`, `netbox_rear_port_template` (attribute
`front_ports`); the same pattern applies to any Optional nested list that
NetBox fills in from the other side of a relation.

*Symptom* (attribute left as generated, i.e. Optional): after a front port
maps to the rear port, the refresh plan is not empty:

```
# netbox_rear_port_template.test will be updated in-place
~ resource "netbox_rear_port_template" "test" {
    ~ display        = "Rear 1" -> (known after apply)
    - front_ports    = [
        - { front_port = 122, front_port_position = 1, position = 1 },
      ] -> null
    ~ last_updated   = "2026-09-15T13:00:15Z" -> (known after apply)
```

With `computed: true` (Optional+Computed + `listplanmodifier.UseStateForUnknown`)
the list itself no longer diffs, but the framework still marks
`display`/`last_updated` unknown, so an empty update is planned forever:

```
~ resource "netbox_rear_port" "test" {
    ~ display        = "Rear 1" -> (known after apply)
    ~ last_updated   = "2026-09-15T13:08:07Z" -> (known after apply)
      # (10 unchanged attributes hidden)
```

(`terraform show -json`: `front_ports` before == after, only `display` and
`last_updated` in `after_unknown`.)

*Workaround applied*: `front_ports: { skip: true }` on both resources;
mappings are defined on the front port, as in the NetBox UI. The list is also
gone from the data sources.

*Proposed fix*: classify server-maintained reverse relations (`front_ports`
on rear ports/templates) as read-only (Computed only, data-source style) so
they are read but never planned; more generally, when a nested list is
Optional+Computed and absent from config, plan it from state without
triggering the "computed nils to unknown" pass (e.g. a custom plan modifier
that copies state when the config is null, applied before the framework's
comparison, or give `display`/`last_updated` a UseStateForUnknown modifier
only when no other attribute changed).

### G3 - nullable FKs that NetBox cannot write are sent as `null`

*Resource*: `netbox_module_bay` (attribute `installed_module_id`).

*Symptom*: every create fails with
`500 Internal Server Error: error: The following fields do not exist in this
model: installed_module, exception: ValueError` because the nullable FK is
always serialised as `"installed_module": null`.

*Workaround applied*: `installed_module: { skip: true }`.

*Proposed fix*: treat `installed_module` (and reverse one-to-one accessors in
general: the spec marks them writable but they are not model fields) as
read-only in `spec/required-fixes.json` or a generator deny-list, and stop
sending nullable FKs that are absent from the plan on create (only send
explicit `null` on PATCH when the prior state had a value).

### G4 - list data source item shape differs per resource (minor)

`data.netbox_module_types.all.items[*]` has `model` but no `name`, and
`data.netbox_modules` items only expose `display`. Documentation for list data
sources could state the item attributes; no code change required.
