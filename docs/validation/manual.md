# Validation: hand-written resources (`internal/provider/manual`)

Phase 4 adds six hand-written resources that the generator cannot express:
four allocation resources built on NetBox's `available-*` action endpoints and
two primary-IP assignment resources. All were validated against
https://demo.netbox.dev (NetBox 4.7) with `NETBOX_REQUESTS_PER_SECOND=2`.

| Resource | Endpoint used on create | Managed afterwards as | Test |
|---|---|---|---|
| `netbox_available_ip_address` | `POST /api/ipam/prefixes/{id}/available-ips/` or `POST /api/ipam/ip-ranges/{id}/available-ips/` | `/api/ipam/ip-addresses/{id}/` | `TestAccAvailableIpAddress_basic`, `TestAccAvailableIpAddress_ipRange` |
| `netbox_available_prefix` | `POST /api/ipam/prefixes/{id}/available-prefixes/` | `/api/ipam/prefixes/{id}/` | `TestAccAvailablePrefix_basic` |
| `netbox_available_vlan` | `POST /api/ipam/vlan-groups/{id}/available-vlans/` | `/api/ipam/vlans/{id}/` | `TestAccAvailableVlan_basic` |
| `netbox_available_asn` | `POST /api/ipam/asn-ranges/{id}/available-asns/` | `/api/ipam/asns/{id}/` | `TestAccAvailableAsn_basic` |
| `netbox_device_primary_ip` | `PATCH /api/dcim/devices/{id}/` (`primary_ip4` / `primary_ip6`) | same | `TestAccDevicePrimaryIp_basic` |
| `netbox_virtual_machine_primary_ip` | `PATCH /api/virtualization/virtual-machines/{id}/` | same | `TestAccVirtualMachinePrimaryIp_basic` |

Each `_basic` test runs create -> in-place update (`plancheck.ExpectResourceAction(..., Update)`)
-> import with `ImportStateVerify` -> empty plan (`plancheck.ExpectEmptyPlan`).

## Results

```
set -a; source .env.demo; set +a
export NETBOX_REQUESTS_PER_SECOND=2 TF_ACC=1
go test ./internal/provider/manual/ -run TestAcc -count=1 -v -timeout 40m -parallel 2
```

First run (before the fixture fix described below):

```
--- PASS: TestAccAvailableVlan_basic (21.10s)
--- PASS: TestAccAvailableAsn_basic (22.21s)
--- FAIL: TestAccVirtualMachinePrimaryIp_basic (20.40s)
--- FAIL: TestAccDevicePrimaryIp_basic (19.49s)
--- PASS: TestAccAvailableIpAddress_ipRange (16.99s)
--- PASS: TestAccAvailablePrefix_basic (21.17s)
--- PASS: TestAccAvailableIpAddress_basic (21.75s)
FAIL	github.com/elliot/terraform-provider-netbox/internal/provider/manual	80.264s
```

Re-run of the two primary-IP tests (originally with `lifecycle { ignore_changes = [primary_ip4_id, primary_ip6_id] }`
on the parent fixtures; the block has since been removed and the tests pass without it on NetBox 4.7.2):

```
--- PASS: TestAccVirtualMachinePrimaryIp_basic (62.58s)
--- PASS: TestAccDevicePrimaryIp_basic (118.10s)
ok  	github.com/elliot/terraform-provider-netbox/internal/provider/manual	118.124s
```

Latest full run (Actions run 92 (`acceptance.yml`, `workflow_dispatch` on `main` at `d57b8a0`, 2026-10-11, NetBox `v4.7` image from docker-compose, 12 shards)): the `manual` package now holds 11 `TestAcc*` tests (the custom-field tests were added after this report); 10 PASS, 1 FAIL. Both primary-IP tests and all four allocation tests pass. The failure is `TestAccCustomFieldValue_basic` (step 1: the refresh plan after apply was not empty, it plans to create `netbox_custom_field_value.owner` again); it is a provider bug outside this report's resources and is not addressed here.

Earlier run (all seven original tests, demo.netbox.dev 4.7.2):

```
--- PASS: TestAccAvailableAsn_basic (18.40s)
--- PASS: TestAccVirtualMachinePrimaryIp_basic (25.51s)
--- PASS: TestAccAvailableVlan_basic (18.57s)
--- PASS: TestAccDevicePrimaryIp_basic (31.45s)
--- PASS: TestAccAvailableIpAddress_ipRange (14.54s)
--- PASS: TestAccAvailablePrefix_basic (21.82s)
--- PASS: TestAccAvailableIpAddress_basic (19.12s)
PASS
ok  	github.com/elliot/terraform-provider-netbox/internal/provider/manual	83.540s
```

`go vet ./internal/provider/manual/` and `golangci-lint run ./internal/provider/manual/` report 0 issues.

## NetBox behaviours discovered

* **Allocation endpoints take and return lists.** A single-object JSON body is
  rejected for `available-prefixes`, so every resource POSTs `[{...}]` and
  expects exactly one object back (`allocate[T]` in `common.go`). The
  response objects are the full serializers (`IPAddress`, `Prefix`, `VLAN`,
  `ASN`), so they decode into the generated client models directly.
* **The OpenAPI request schemas under-describe the accepted fields.**
  `AvailableIPRequestRequest` lists only `prefix_length` and
  `PrefixLengthRequest` only `prefix_length`, but NetBox validates the items
  with the regular writable serializers (`IPAddressSerializer`,
  `PrefixSerializer`, `CreateAvailableVLANSerializer`, `ASNSerializer`) after
  injecting the allocated value. All writable fields (`status`, `role`,
  `tenant`, `description`, `tags`, `custom_fields`, `assigned_object_*`,
  `scope_*`, `is_pool`, ...) are honoured on create; the tests exercise
  `description` on create and the rest on update.
* **VRF / RIR / group are set by the server.** `available-ips` and
  `available-prefixes` copy the parent's VRF, `available-asns` copies the
  range's RIR, `available-vlans` sets `group` and `vid`. These are modelled as
  Computed (`vrf_id`, `rir_id`, `vid`) with `UseStateForUnknown`.
* **First free values on a fresh parent.** A new `/24` yields `x.x.x.1/24`
  (network address skipped, `is_pool = false`); an IP range yields its
  `start_address`; a `/24` parent yields `x.x.x.0/28` for `prefix_length = 28`;
  a VLAN group with `vid_ranges = [[100, 199]]` yields VID 100; an ASN range
  yields its `start`. The tests assert these exact values.
* **The parent of an allocated object is not recoverable.** An IP address does
  not reference the prefix or range it came from, a prefix does not reference
  its parent and an ASN does not reference its range, so `prefix_id`,
  `ip_range_id`, `parent_prefix_id`, `prefix_length` and `asn_range_id` are
  null after import (`ImportStateVerifyIgnore` in the tests). Configuring them
  on an imported resource forces a replacement (`RequiresReplace`); this is
  documented in the schema and `import.sh`. A VLAN does reference its group,
  so `vlan_group_id` is refreshed from the API and imports verify cleanly.
* **`family` on an IP address** is a `{value, label}` object (`ChoiceInteger`),
  which `conv.ChoiceInt` unwraps; it is used to detect `ip_address_version`.
* **Primary IP assignment requires the address to be assigned to an interface
  of the same device / VM** (`dcim.interface` / `virtualization.vminterface`),
  which is why the fixtures build the full chain.
* **PATCHing `primary_ip4` on a device returns the whole device object**, from
  which `primary_ip4.id` / `primary_ip6.id` are read back; a minimal struct is
  decoded instead of the strict generated `Device` model.

## Interaction with the generated `netbox_device` / `netbox_virtual_machine`

Originally the generated parent resources exposed `primary_ip4_id` / `primary_ip6_id`
as plain `Optional` FKs: after `netbox_device_primary_ip` set the primary address,
the next refresh of the parent planned `- primary_ip4_id = 226 -> null` (the
first-run failure above) and applying it would have cleared the assignment.

Resolved: `primary_ip4_id` / `primary_ip6_id` are `Optional+Computed` with
`UseStateForUnknown` on the generated parents, so the parent's plan carries the
current value forward and `netbox_*_primary_ip` works without a `lifecycle`
block (confirmed by `TestAccDevicePrimaryIp_basic` and
`TestAccVirtualMachinePrimaryIp_basic` without `ignore_changes`).

## Fixture isolation on the shared demo

Fixtures derive a third octet (`10.201.X.0/24`, `10.203.X.0/24`,
`10.204.X.10-20`, `10.205.X.0/24`, `10.206.X.0/24`) and an ASN block
(`4200000000 + (hash % 900000) * 100`) from the random `tfacc-` name, so
concurrent runs and leftovers rarely collide. All objects carry the `tfacc-`
prefix in `name`, `slug` or `description`, so the existing generated sweepers
(`netbox_ip_address`, `netbox_prefix`, `netbox_vlan`, `netbox_asn`,
`netbox_device`, `netbox_virtual_machine`, ...) clean them up.

## Files

* `internal/provider/manual/common.go`: shared schema attributes, raw-body
  builder (`body`), `allocate[T]`.
* `internal/provider/manual/available_ip_address_resource.go`
* `internal/provider/manual/available_prefix_resource.go`
* `internal/provider/manual/available_vlan_resource.go`
* `internal/provider/manual/available_asn_resource.go`
* `internal/provider/manual/primary_ip_resource.go` (both primary-IP resources)
* `internal/provider/manual/*_test.go`, `helpers_test.go`
* `examples/resources/netbox_{available_ip_address,available_prefix,available_vlan,available_asn,device_primary_ip,virtual_machine_primary_ip}/{resource.tf,import.sh}`
* `main.go`: blank import of `internal/provider/manual`.
