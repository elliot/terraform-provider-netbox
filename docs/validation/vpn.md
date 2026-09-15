# Validation: vpn shard (NetBox 4.7, https://demo.netbox.dev)

Resources: `netbox_ike_proposal`, `netbox_ike_policy`, `netbox_ipsec_proposal`,
`netbox_ipsec_policy`, `netbox_ipsec_profile`, `netbox_tunnel_group`,
`netbox_tunnel`, `netbox_tunnel_termination`, `netbox_l2vpn`,
`netbox_l2vpn_termination`.

Fixtures live in `generator/overrides/vpn.yaml`; every test runs
create -> in-place update -> data sources (by id and list filter) -> import
with `ImportStateVerify` -> empty plan.

## Results

| Resource | Test | Result | Notes |
|---|---|---|---|
| `netbox_ike_proposal` | `TestAccIkeProposal_basic` | PASS | Auto fixture failed on update (`authentication_algorithm: This field may not be blank.`); fixture now always sets the algorithm. Update with an AEAD cipher still fails (G1). |
| `netbox_ike_policy` | `TestAccIkePolicy_basic` | PASS | Auto fixture failed (`version: This field is required.`); IKEv1 + `mode = "main"` fixture, proposals added on update. IKEv2 policies cannot be updated (G1). |
| `netbox_ipsec_proposal` | `TestAccIpsecProposal_basic` | PASS | Auto fixture failed (`Encryption and/or authentication algorithm must be defined`). Update with an AEAD cipher still fails (G1). |
| `netbox_ipsec_policy` | `TestAccIpsecPolicy_basic` | PASS | Fixture sets `pfs_group`; policies without PFS cannot be updated (G1). |
| `netbox_ipsec_profile` | `TestAccIpsecProfile_basic` | PASS | Was skipped (FK deps); full IKE/IPsec chain fixture, `mode` esp -> ah in place. |
| `netbox_tunnel_group` | `TestAccTunnelGroup_basic` | PASS | Auto fixture already passed; explicit fixture adds `comments`. |
| `netbox_tunnel` | `TestAccTunnel_basic` | PASS | Auto fixture failed (`status: This field is required.`, G3); fixture sets `status`, `group_id`, `tunnel_id`, changes `encapsulation` in place. |
| `netbox_tunnel_termination` | `TestAccTunnelTermination_basic` | PASS | Was skipped; device chain + `virtual` tunnel interface, `role` hub -> peer and `outside_ip_id` added in place. `role` must be set (G3). |
| `netbox_l2vpn` | `TestAccL2vpn_basic` | PASS | Explicit fixture adds `identifier`, `status`, `type` change and import/export route targets on update. |
| `netbox_l2vpn_termination` | `TestAccL2vpnTermination_basic` | PASS | Was skipped; VLAN termination, re-pointed to a second VLAN plus a tag on update. |

10 PASS / 0 SKIP / 0 FAIL (`go test ./internal/provider/gen/vpn/ -run 'TestAcc.*_basic$' -parallel 3`, 200 s, 2026-09-15).

Scenario `examples/scenarios/vpn/main.tf` (crypto stack, tunnel group, IPsec
tunnel with hub/spoke terminations on device interfaces with outside IPs, VXLAN
EVPN L2VPN with a VLAN termination, 31 resources): `terraform apply` created 26 resources, a second `terraform plan` was empty, `terraform destroy` removed all 26 (provider built from the working tree, `dev_overrides`, `NETBOX_REQUESTS_PER_SECOND=2`).

## NetBox behaviours discovered

| Resource | Behaviour | Consequence |
|---|---|---|
| `netbox_tunnel` | `status` is required by the API (`status: This field is required.`) although `WritableTunnelRequest` does not list it as required and the model has a default. | Always set `status`; the schema still shows it as Optional+Computed (see generator issue G3). |
| `netbox_tunnel_termination` | `role` is required by the API (`role: This field is required.`) although `WritableTunnelTerminationRequest` does not list it as required. | Always set `role` (see G3). |
| `netbox_ike_policy` | `version` is required (`version: This field is required.`); `mode` is required for IKEv1 (`Mode is required for selected IKE version`) and forbidden for IKEv2 (`Mode cannot be used for selected IKE version`). NetBox returns `mode: null` for IKEv2 policies. | IKEv1 fixture sets `mode`; IKEv2 examples omit it. |
| `netbox_ike_proposal` | `authentication_algorithm` is mandatory for non-AEAD ciphers; NetBox returns `null` when it is unset (AEAD `*-gcm`). | Fixture always sets it (CBC ciphers). |
| `netbox_ipsec_proposal` | `Encryption and/or authentication algorithm must be defined`. | Auto fixture failed; override sets both. |
| `ike_proposal`, `ike_policy`, `ipsec_proposal`, `ipsec_policy` | The nullable choice fields (`authentication_algorithm`, `encryption_algorithm`, `mode`, `version`, `pfs_group`) reject both `null` and `""` on PATCH: `This field may not be blank.` NetBox's `ChoiceField` maps `null` to `""` and these serializers do not set `allow_blank`. Omitting the key works. | Clearing these fields through the API is impossible; sending `null` for an unset field breaks every update (G1). |
| `netbox_l2vpn` | `identifier` (VNI/VPN id) and route targets accepted as expected; `slug` unique. | - |
| `netbox_l2vpn_termination` | `assigned_object_type` accepts `ipam.vlan`, `dcim.interface`, `virtualization.vminterface`; switching the assigned VLAN is an in-place PATCH. | - |
| `netbox_tunnel_termination` | `outside_ip_id` accepts an IP address assigned to another interface of the same device; switching `role` is in place. | - |
| all vpn objects | The demo instance is periodically reset; object IDs restart at 1, so hard-coded IDs in examples are placeholders only. | - |

## Generator issues

### G1 - nullable choice attributes are sent as JSON `null`, NetBox rejects it (blocks updates)

Affected here: `netbox_ike_proposal.authentication_algorithm`,
`netbox_ipsec_proposal.encryption_algorithm` / `authentication_algorithm`,
`netbox_ike_policy.mode`, `netbox_ipsec_policy.pfs_group`. The same rule
applies to every `nullable` choice/choice_int attribute of every app whose
serializer field lacks `allow_blank=True` (e.g. it is worth checking
`netbox_ip_address.role`, `netbox_vlan.qinq_role`).

Reproduction (built provider, demo instance): create an IKEv2 policy, an
AEAD IKE/IPsec proposal and an IPsec policy without `pfs_group`, then change
only `description` and apply:

```
Error: Error updating netbox_ike_proposal
NetBox API error: 400 Bad Request: authentication_algorithm: This field may not be blank.
Error: Error updating netbox_ipsec_proposal
NetBox API error: 400 Bad Request: authentication_algorithm: This field may not be blank.
Error: Error updating netbox_ike_policy
NetBox API error: 400 Bad Request: mode: This field may not be blank.
Error: Error updating netbox_ipsec_policy
NetBox API error: 400 Bad Request: pfs_group: This field may not be blank.
```

Cause: `internal/gen/render/emit.go`, `setterCode`, case
`a.Nullable && (!a.Computed || a.Kind == model.KindChoice || ...)` emits
`if plan.X.IsNull() { body.SetXNil() } ...` for create and PATCH. After
create the state holds `null` (NetBox returns `null`), `UseStateForUnknown`
copies it into the plan, and every PATCH carries `"mode": null`. NetBox's
`ChoiceField.validate_empty_values` turns `null` into `""` and the vpn
serializers do not set `allow_blank`, so the request is rejected. Create only
works because the attribute is unknown (not null) in the plan and is omitted.

Proposed fix: for `KindChoice`/`KindChoiceInt` never send `null` when the
prior state is also null (nothing to clear): in `toPatch` pass the prior
state and emit
`if plan.X.IsNull() { if !state.X.IsNull() { body.SetXNil() } } else if !plan.X.IsUnknown() { body.SetX(...) }`;
on create emit the plain `conv.Known` form. Sending `null` only when the
user removed a previously set value keeps the "clear" semantics for the
serializers that accept it (`allow_blank=True`, where NetBox stores `""` and
the provider already maps `""` back to null in `conv.Choice`). Cheaper
alternative: treat nullable choices like non-nullable ones (`conv.Known`
only) and document that NetBox cannot clear them through the API.

No override can work around this: `computed: false` and `kind: string` both
still hit the explicit-null branch. The fixtures avoid the bug by always
setting these attributes; the acceptance suite therefore stays green but the
realistic IKEv2 / AEAD / no-PFS configurations are not updatable until the
generator is fixed.

### G2 - `netbox_ike_policy.version` is Optional+Computed but required by NetBox

`version: This field is required.` on create without `version`. The spec
lists `WritableIKEPolicyRequest.version` as optional (`spec/required-fixes.json`
even removes it from the non-writable `IKEPolicyRequest`). Proposed fix: add
`"WritableIKEPolicyRequest": {"add": ["version"]}` to
`spec/required-fixes.json` so the attribute is generated as `Required`
(the constructor then takes it). Same pattern as G3.

### G3 - `netbox_tunnel.status` and `netbox_tunnel_termination.role` are Optional+Computed but required by NetBox

`status: This field is required.` / `role: This field is required.` on
create when omitted. `spec/required-fixes.json` contains
`"TunnelRequest": {"remove": ["status"]}` and
`"TunnelTerminationRequest": {"remove": ["role"]}`, which target the
non-writable schemas and go the wrong way for NetBox 4.7. Proposed fix:
`"WritableTunnelRequest": {"add": ["status"]}` and
`"WritableTunnelTerminationRequest": {"add": ["role"]}` (and drop the two
`remove` entries) so both attributes become `Required` and a missing value is
a plan-time error instead of a 400.

### G4 - auto-synthesised fixtures do not know cross-field rules (informational)

`ike_proposal` (non-AEAD cipher without `authentication_algorithm`),
`ipsec_proposal` (no algorithm at all) and `ike_policy` (no `version`) cannot
be exercised by the synthesised fixture; explicit fixtures in
`generator/overrides/vpn.yaml` cover them. No generator change proposed
beyond G1-G3.

## Files

* `generator/overrides/vpn.yaml` - fixtures for all ten resources.
* `internal/provider/gen/vpn/*_resource_test.go` - regenerated with
  `go run ./internal/gen -only ike_policy,...,tunnel_termination`.
* `examples/resources/netbox_<name>/{resource.tf,import.sh}` and
  `examples/data-sources/netbox_<name>/data-source.tf`,
  `examples/data-sources/netbox_<plural>/data-source.tf` - hand-maintained
  for all ten resources.
* `examples/scenarios/vpn/main.tf` - applied and destroyed on the demo.
