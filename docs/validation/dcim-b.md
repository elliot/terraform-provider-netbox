# Validation: dcim shard B (device components, templates, modules, VC)

Target: https://demo.netbox.dev (NetBox 4.7), provider built from this tree,
`NETBOX_REQUESTS_PER_SECOND=2`. Every resource runs the generated
`TestAcc<Name>_basic` flow: create -> in-place update -> singular and list data
source lookups -> import with `ImportStateVerify` -> empty plan.

Fixtures and per-resource tuning live in `generator/overrides/dcim_b.yaml`
(device, device_type, interface and cable keep their fixtures in
`_fixtures_pilot.yaml`). Hand-maintained examples are under
`examples/resources/netbox_<name>/` and `examples/data-sources/netbox_<name>/`;
the end-to-end scenario is `examples/scenarios/dcim-b/main.tf`.

## Results

RESULTS_TABLE

## Scenario

SCENARIO_SECTION

## NetBox behaviours discovered

BEHAVIOURS

## Generator issues

GENERATOR_ISSUES
