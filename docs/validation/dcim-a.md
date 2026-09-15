# Validation: dcim shard A (facilities, racks, power, cooling, catalogue)

Target: https://demo.netbox.dev (NetBox 4.7), provider built from this tree,
`NETBOX_REQUESTS_PER_SECOND=2`. Every resource runs the generated
`TestAcc<Name>_basic` flow: create -> in-place update -> singular and list data
source lookups -> import with `ImportStateVerify` -> empty plan.

Fixtures and per-resource tuning live in `generator/overrides/dcim_a.yaml`.
Hand-maintained examples are under `examples/resources/netbox_<name>/` and
`examples/data-sources/netbox_<name>/`; the end-to-end scenario is
`examples/scenarios/dcim-a/main.tf`.

Run the shard:

```sh
set -a; source .env.demo; set +a
export TF_ACC=1 NETBOX_REQUESTS_PER_SECOND=2
go test ./internal/provider/gen/dcim/ -count=1 -v -timeout 60m -parallel 4 \
  -run 'TestAcc(Site|Region|SiteGroup|Location|Rack|RackGroup|RackRole|RackType|RackReservation|PowerPanel|PowerFeed|PowerPort|PowerPortTemplate|PowerOutlet|PowerOutletTemplate|CoolingSource|CoolingFeed|CoolingIntake|CoolingIntakeTemplate|CoolingOutflow|CoolingOutflowTemplate|Manufacturer|Platform|DeviceRole|ModuleTypeProfile|ModuleBayType|CableBundle|MacAddress)_basic$'
```

## Results

28 resources, 28 PASS, 0 skipped (full run 2026-09-15, ~6 min wall clock).

| Resource | Result | Attributes exercised beyond the required ones |
|---|---|---|
| `netbox_region` | PASS | parent_id, description, comments, tags |
| `netbox_site_group` | PASS | parent_id, description, comments, tags |
| `netbox_site` | PASS | status, region_id, group_id, tenant_id, facility, time_zone, physical/shipping address, latitude/longitude (6 decimals), comments, asn_ids, tags |
| `netbox_location` | PASS | parent_id, status, tenant_id, facility, description, comments, tags |
| `netbox_rack` | PASS | location_id, tenant_id, role_id, status, facility_id, serial, asset_tag, form_factor, width, u_height, starting_unit, desc_units, weight/max_weight/weight_unit, outer_*, mounting_depth, airflow, cooling_capability, comments, tags; rack_type_id via the scenario |
| `netbox_rack_group` | PASS | description, comments, tags |
| `netbox_rack_role` | PASS | color, description, comments, tags |
| `netbox_rack_type` | PASS | form_factor, width, u_height, starting_unit, desc_units, outer_*, weight/max_weight/weight_unit, mounting_depth, cooling_capability, cooling_capacity, comments, tags |
| `netbox_rack_reservation` | PASS | units (set, grown in place), status, tenant_id, comments, tags; user created with `netbox_user` |
| `netbox_power_panel` | PASS | location_id, description, comments, tags |
| `netbox_power_feed` | PASS | rack_id, status, type, supply, phase, voltage, amperage, max_utilization, mark_connected, tenant_id, comments, tags |
| `netbox_power_port` | PASS | label, type, maximum_draw, allocated_draw, mark_connected, description, tags |
| `netbox_power_port_template` | PASS | device_type_id, label, type, maximum_draw, allocated_draw, description |
| `netbox_power_outlet` | PASS | label, type, status, color, power_port_id, feed_leg, mark_connected, description, tags |
| `netbox_power_outlet_template` | PASS | device_type_id, label, type, color, power_port_id (-> template), feed_leg, description |
| `netbox_cooling_source` | PASS | location_id, type change, status, fluid_type, cooling_capacity, description, comments, tags |
| `netbox_cooling_feed` | PASS | rack_id, status, cooling_capacity, max_flow, max_flow_unit, tenant_id, description, comments, tags |
| `netbox_cooling_intake` | PASS | label, type, diameter, diameter_unit, max_flow, max_flow_unit, cooling_outflow_id, description, tags |
| `netbox_cooling_intake_template` | PASS | device_type_id, label, type, diameter, diameter_unit, max_flow, max_flow_unit, description |
| `netbox_cooling_outflow` | PASS | label, type, diameter, diameter_unit, cooling_intake_id, description, tags |
| `netbox_cooling_outflow_template` | PASS | device_type_id, label, type, diameter, diameter_unit, cooling_intake_id (-> template), description |
| `netbox_manufacturer` | PASS | description, comments, tags |
| `netbox_platform` | PASS | parent_id, manufacturer_id, config_template_id, description, comments, tags |
| `netbox_device_role` | PASS | color, vm_role, parent_id, config_template_id, description, comments, tags |
| `netbox_module_type_profile` | PASS | schema (JSON string, nested object round-trips), description, comments, tags |
| `netbox_module_bay_type` | PASS | manufacturer_id, color, description, comments, tags |
| `netbox_cable_bundle` | PASS | description, comments, tags |
| `netbox_mac_address` | PASS | mac_address change, assigned_object_type/id (dcim.interface), description, comments, tags |

Data sources: every singular data source (`data.netbox_<name>` by `id`) and
list data source (`data.netbox_<plural>` with an `id` filter) passed as part of
the flow. `netbox_mac_address` and `netbox_rack_reservation` have no natural
key, so their singular data sources only support `id` and `filters`.

## Scenario

`examples/scenarios/dcim-a/main.tf` (39 resources) was applied, re-planned
(empty plan) and destroyed against the demo with the locally built provider:

* region (Europe -> Germany) -> site FRA1 (tenant, site group, coordinates,
  time zone) -> locations (Floor 2 -> Cage A)
* manufacturer -> rack type; rack role, rack group -> rack A01 referencing the
  rack type; user -> rack reservation (units 1-4)
* power panel -> power feeds A/B into the rack; cooling source (chiller) ->
  cooling feed into the rack
* manufacturer -> platform, device role hierarchy, module type profile, module
  bay type, device type with power port/outlet and cooling intake/outflow
  templates -> device in the rack with extra power port/outlet, cooling
  intake/outflow, interface and MAC address; a cable bundle
* read-back via `data.netbox_site` and `data.netbox_power_feeds`

```sh
go build -o /tmp/tfp-dcim-a/terraform-provider-netbox .
cat > ~/.terraformrc-dcim-a <<'RC'
provider_installation {
  dev_overrides {
    "elliot/netbox" = "/tmp/tfp-dcim-a"
  }
  direct {}
}
RC
export TF_CLI_CONFIG_FILE=~/.terraformrc-dcim-a NETBOX_REQUESTS_PER_SECOND=2
cd examples/scenarios/dcim-a && terraform apply && terraform plan && terraform destroy
```

Note: with Terraform 1.16 the CLI config must spell `dev_overrides { ... }` as
a block; the documented `dev_overrides = { ... }` attribute form is rejected
("items inside the provider_installation block must all be blocks").

## NetBox behaviours discovered

* **Rack types copy their attributes onto racks.** When `rack_type_id` is
  set, NetBox overwrites the rack's `form_factor`, `width`, `u_height`,
  `starting_unit`, `desc_units`, `outer_width/height/depth`, `outer_unit`,
  `weight`, `max_weight`, `weight_unit` and `mounting_depth` with the type's
  values on every save. The nullable ones were Optional-only and produced
  "Provider produced inconsistent result after apply: .outer_width was null,
  but now 600" on create; `dcim_a.yaml` now marks them `computed: true` so a
  rack that references a type leaves them unset and reads them back. Do not
  set a different value than the type in configuration; NetBox silently
  replaces it and Terraform reports an inconsistent result. Attaching a type
  to an existing rack whose stored dimensions differ triggers the same error
  once (the values in state are stale); a second apply converges.
* **MAC addresses are normalised to upper case** (`02:00:5e:...` is stored as
  `02:00:5E:...`). Write MACs upper-case in configuration; lower-case input
  fails with an inconsistent-result error (see generator issue 1).
* **`netbox_asn.site_ids` mirrors `netbox_site.asn_ids`.** Assigning an ASN to
  a site through the site makes the ASN resource in the same configuration
  show a perpetual `site_ids` change. Manage the link from one side only; the
  site fixture uses `lifecycle { ignore_changes = [site_ids] }` on the ASN.
* **Power feed defaults are deployment settings.** `voltage`, `amperage` and
  `max_utilization` default to `POWERFEED_DEFAULT_VOLTAGE/AMPERAGE/MAX_UTILIZATION`
  (the demo returns 220 V / 16 A, not the upstream 120 V / 20 A). Set them
  explicitly if the values matter.
* **`rack_type.form_factor` is required by NetBox** ("This field cannot be
  blank") but the generated schema makes it Optional+Computed; omitting it
  yields a NetBox 400 at apply time instead of a plan-time error (generator
  issue 2).
* Rack reservations need a NetBox **user**; `netbox_user` (username +
  password) works on the demo. `units` is a set and can be grown in place.
* Component templates (`*_template`) take either `device_type_id` or
  `module_type_id`; `power_outlet_template.power_port_id` and
  `cooling_outflow_template.cooling_intake_id` reference the *template*
  resources of the same device type, not device components.
* `cooling_intake.cooling_outflow_id` / `cooling_outflow.cooling_intake_id`
  pair an intake with an outflow of the same device; setting one side is
  enough, the other side is not reflected back into the resource (no drift).
* `module_type_profile.schema` accepts a JSON-schema object written with
  `jsonencode`; nested objects and arrays round-trip without a diff.
* Slugs and names must contain the `tfacc-` prefix for the sweeper; `time_zone`
  takes IANA names (`Europe/Berlin`); `color` is 6 lowercase hex digits.

## Generator issues

1. **`mac_address` case normalisation** (`netbox_mac_address.mac_address`,
   also `netbox_interface.mac_address` and other MAC fields). NetBox
   upper-cases MAC addresses; a lower-case value in configuration produces
   `Provider produced inconsistent result after apply: .mac_address: was
   "02:00:5e:10:20:30", but now "02:00:5E:10:20:30"`. Proposed fix: classify
   properties named `mac_address` (or with `format: mac`) as a `mac` kind that
   uses a case-insensitive semantic-equality string type (or a plan modifier
   that upper-cases the planned value) and validates the `aa:bb:cc:dd:ee:ff`
   shape. Worked around in fixtures and examples by writing MACs upper-case.
2. **`rack_type.form_factor` is wrongly demoted to optional.**
   `spec/required-fixes.json` contains `"WritableRackTypeRequest": {"remove":
   ["form_factor"]}`, so the build step never sees it as required and emits it
   Optional+Computed. NetBox does require it: `POST /api/dcim/rack-types/`
   without `form_factor` answers `400 {"form_factor": ["This field cannot be
   blank."]}` (verified on the demo). Proposed fix: delete that entry from
   `spec/required-fixes.json` and regenerate (client and provider); the
   attribute then becomes `Required` and the error moves to plan time. Worked
   around by always setting `form_factor` in fixtures and examples.
3. **Bidirectional M2M attributes cause drift on the far side**
   (`netbox_asn.site_ids` vs `netbox_site.asn_ids`; likely the same for other
   symmetric relations). Proposed fix: mark the reverse side (`site_ids` on
   ASN, which NetBox exposes writable but which is the same relation) as
   computed-only or data-source-only via a `reverse_of` override, or at least
   default it to Optional+Computed without "always send". Worked around in
   the site fixture with `ignore_changes`.
4. **Nullable numbers that the server fills in need `computed: true`**
   (rack dimensions copied from the rack type, see behaviours). Not a code
   bug -- the `computed: true` override handles it -- but the generated
   description still says "Defaults to the NetBox server default when
   omitted" for values that actually come from the related rack type; a
   per-attribute `description` override could be added if that wording
   matters.

No tests are skipped in this shard.
