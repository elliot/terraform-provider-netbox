# Validation: ipam + core shard

Validated against https://demo.netbox.dev (NetBox 4.7, Django 6.1) on
2026-09-15 with `NETBOX_REQUESTS_PER_SECOND=2`. Fixtures live in
`generator/overrides/ipam_core.yaml` (merged after `ipam.yaml`, `core.yaml`
and `_fixtures_pilot.yaml`, so they take precedence). Examples under
`examples/resources/netbox_<name>/` and `examples/data-sources/netbox_<name>/`
for every resource below are hand-maintained (no `.generated` marker).

Every test runs create -> in-place update -> singular + list data source ->
import with `ImportStateVerify` -> empty plan.

## Results

| Resource | Test | Result | Notes |
|---|---|---|---|
| netbox_aggregate | TestAccAggregate_basic | PASS | RIR + tenant, `date_added`, comments |
| netbox_asn | TestAccAsn_basic | PASS | RIR, tenant; site link via `netbox_site.asn_ids` (see G1) |
| netbox_asn_range | TestAccAsnRange_basic | PASS | RIR, tenant, start/end update |
| netbox_fhrp_group | TestAccFhrpGroup_basic | PASS | protocol change vrrp2 -> vrrp3, auth_type/auth_key |
| netbox_fhrp_group_assignment | TestAccFhrpGroupAssignment_basic | PASS | device + virtual interface, `interface_type = "dcim.interface"`, priority update |
| netbox_ip_address | TestAccIpAddress_basic | PASS | VRF, tenant, `nat_inside_id`, role, dns_name |
| netbox_ip_range | TestAccIpRange_basic | PASS | VRF, tenant, role, `mark_utilized`, status |
| netbox_ipam_role | TestAccIpamRole_basic | PASS | weight |
| netbox_prefix | TestAccPrefix_basic | PASS | VRF, tenant, role, `scope_type/scope_id` (dcim.site), `is_pool`, `mark_utilized` |
| netbox_rir | TestAccRir_basic | PASS | `is_private`, comments |
| netbox_route_target | TestAccRouteTarget_basic | PASS | tenant |
| netbox_service | TestAccService_basic | PASS | device parent, `port_mappings`, `ipaddress_ids` bound to an interface IP |
| netbox_service_template | TestAccServiceTemplate_basic | PASS | `port_mappings` set update |
| netbox_vlan | TestAccVlan_basic | PASS | VLAN group, tenant, role, status, rename |
| netbox_vlan_group | TestAccVlanGroup_basic | PASS | `vid_ranges` update, `scope_type/scope_id`, tenant |
| netbox_vlan_translation_policy | TestAccVlanTranslationPolicy_basic | PASS | |
| netbox_vlan_translation_rule | TestAccVlanTranslationRule_basic | PASS | `remote_vid` update |
| netbox_vrf | TestAccVrf_basic | PASS | rd, tenant, `enforce_unique`, import/export route targets |
| netbox_data_source (core) | TestAccDataSource_basic | PASS | `sync_interval`, `ignore_rules`, JSON `parameters` |

19 PASS, 0 SKIP, 0 generator-blocked.

Full-shard run: `go test ./internal/provider/gen/ipam/ ./internal/provider/gen/core/ -run 'TestAcc(Aggregate|Asn|AsnRange|FhrpGroup|FhrpGroupAssignment|IpAddress|IpRange|IpamRole|Prefix|Rir|RouteTarget|Service|ServiceTemplate|Vlan|VlanGroup|VlanTranslationPolicy|VlanTranslationRule|Vrf|DataSource)_basic' -parallel 4`.

## Scenario

`examples/scenarios/ipam/main.tf` builds a coherent DC1 address plan (RIR,
IPv6 aggregate, VRF with import/export route targets, two roles, site +
router + SVI, VLAN group + 2 VLANs, container prefix + 2 children, DHCP IP
range, VIP / SVI / loopback addresses, ASN range + ASN, VRRP group assigned
to the SVI, SSH service bound to the SVI address, and two list data sources).
With the locally built provider (`dev_overrides`) it was applied on the demo
(30 resources added), re-planned (`No changes`) and destroyed (30 destroyed).

The site references its ASN through `netbox_site.asn_ids` (see G1).

## NetBox behaviours discovered

* **Aggregates may not overlap each other**, independent of RIR. The demo
  already has `10.0.0.0/8`, `100.64.0.0/10`, `172.16.0.0/12` and
  `192.168.0.0/16`, so the fixture uses `2001:db8:213::/48` and the scenario
  `2001:db8:5ce::/48`.
* **`asn` is globally unique** (`400: asn: ASN with this ASN already exists.`).
  The test uses 4200213007, the scenario 4200213001 (and range
  4200213000-4200213099); keep them distinct.
* `netbox_ip_range` start/end addresses **must carry the prefix length**
  (`10.213.5.10/24`); NetBox stores and returns them unchanged, no diff.
* `netbox_service.ipaddress_ids` must reference IPs **assigned to an
  interface of the parent device/VM**; an IP on `lo0` of the same device works.
* `netbox_fhrp_group_assignment.interface_type` is `dcim.interface` or
  `virtualization.vminterface`; the singular data source only supports `id`
  and `filters` (no natural key).
* `netbox_vlan.vid` must fall inside the group's `vid_ranges` when
  `group_id` is set; `vid_ranges` is a list of `[min, max]` pairs and
  round-trips without diffs.
* `netbox_data_source.sync_interval` is a **choice in minutes**:
  1, 60, 720, 1440, 10080, 43200. `parameters` (JSON string) and multi-line
  `ignore_rules` round-trip cleanly.
* `netbox_prefix.scope_type/scope_id` (`dcim.site`) and
  `netbox_vlan_group.scope_type/scope_id` round-trip cleanly;
  `is_pool`, `mark_utilized`, `enforce_unique` show no perpetual diffs.
* No address normalisation diffs were observed for `/24` host addresses,
  `/32` loopbacks or IPv6 `/48` aggregates.
* Test infrastructure: without `TF_ACC_TERRAFORM_PATH`,
  terraform-plugin-testing calls `checkpoint-api.hashicorp.com` to locate a
  CLI and the whole package fails in a sandboxed network ("failed to find or
  install Terraform CLI ... context deadline exceeded"). Point it at a local
  binary.

## Generator issues

### G1. Symmetric many-to-many written from both sides (site.asn_ids vs asn.site_ids) - resolved

`netbox_asn.site_ids` and `netbox_site.asn_ids` exposed the same relation.
Both were `Optional+Computed` sets with an empty default that is *always
sent*, so after `netbox_asn` attached a site the *site* resource planned to
remove it again on the next refresh:

```
  # netbox_site.dc1 will be updated in-place
  ~ resource "netbox_site" "dc1" {
      ~ asn_ids          = [
          - 11,
        ]
        ...
    }
Plan: 0 to add, 1 to change, 0 to destroy.
```

In the generated ASN test this surfaced as a non-empty plan on
`netbox_site.test` after the update step; applying it would detach the ASN,
after which the ASN's own `site_ids` would flip back, so the two resources
never converged.

Resolution (coordinator, `generator/overrides/ipam.yaml`): `asns.attributes.sites`
is now `read_only`, so the relation is written only through
`netbox_site.asn_ids`; `netbox_asn.site_ids` remains as a computed,
read-only attribute on the resource and data source. The ASN fixture, `examples/resources/netbox_asn` and the scenario attach
the ASN from the site side. A generic fix for any other writable reverse M2M
remains worthwhile: for `fk_list` attributes omit the value from the request
when the config is null instead of always sending `[]`.

### G2. Choice-int validator message quotes integers (cosmetic)

For `netbox_data_source.sync_interval` the plan-time error reads
`value must be one of: ["1" "60" "720" "1440" "10080" "43200"], got: 86400`.
The validation itself is correct and helpful; the values should be rendered
as numbers and the attribute description should mention the unit (minutes).

No other generator bugs were found in this shard; every attribute exercised
above round-trips without perpetual diffs.
