# Validation: extras + wireless

Validated 2026-09-15 against https://demo.netbox.dev (NetBox 4.7) with the
generated provider, `NETBOX_REQUESTS_PER_SECOND=2`, Terraform v1.16.2 for the
scenario. Fixtures and tuning live in `generator/overrides/extras_wireless.yaml`.

Each acceptance test runs create -> in-place update -> data sources (by id and
list with an `id` filter) -> import with `ImportStateVerify` -> empty plan.

## Results

| Resource | Test | Result | Notes |
|---|---|---|---|
| `netbox_config_context` | `TestAccConfigContext_basic` | PASS | `data` JSON round-trips (nested objects, arrays); assignment via `site_ids`; `serial_test` |
| `netbox_config_context_profile` | `TestAccConfigContextProfile_basic` | PASS | `schema` JSON round-trips |
| `netbox_config_template` | `TestAccConfigTemplate_basic` | PASS | `environment_params` JSON; see whitespace note below |
| `netbox_custom_field` | `TestAccCustomField_basic` | PASS | fixture from `_fixtures_pilot.yaml`, unchanged; `serial_test` |
| `netbox_custom_field_choice_set` | `TestAccCustomFieldChoiceSet_basic` | PASS | `extra_choices` as `[[value,label],...]`; `choice_colors` needs colour names; `serial_test` |
| `netbox_custom_link` | `TestAccCustomLink_basic` | PASS | Jinja in `link_text`/`link_url` stored verbatim; `serial_test` |
| `netbox_event_rule` | `TestAccEventRule_basic` | PASS | `conditions` JSON round-trips; webhook action via `action_object_type`/`action_object_id`; `serial_test` |
| `netbox_export_template` | `TestAccExportTemplate_basic` | PASS | |
| `netbox_journal_entry` | `TestAccJournalEntry_basic` | PASS | needed `created_by: { computed: true }` override (see below) |
| `netbox_notification_group` | `TestAccNotificationGroup_basic` | PASS | `group_ids` + `user_ids` (netbox_user_group / netbox_user) |
| `netbox_tag` | `TestAccTag_basic` | PASS | synthesised fixture, unchanged |
| `netbox_webhook` | `TestAccWebhook_basic` | PASS | `secret` is returned by the API and verified on import; `serial_test` |
| `netbox_wireless_lan` | `TestAccWirelessLan_basic` | PASS | group, VLAN, tenant, WPA auth incl. `auth_psk` (returned in clear) |
| `netbox_wireless_lan_group` | `TestAccWirelessLanGroup_basic` | PASS | nested via `parent_id` |
| `netbox_wireless_link` | `TestAccWirelessLink_basic` | PASS | two `ieee802.11ax` interfaces with `rf_role` on two devices; `distance` float |

15 PASS, 0 SKIP.

Scenario `examples/scenarios/extras/main.tf` (18 resources): `terraform apply`
-> `terraform plan` (no changes) -> `terraform destroy` succeeded on the demo.
The device's rendered configuration (`POST /api/dcim/devices/<id>/render-config/`)
contained the hostname plus the NTP and syslog lines from the config context:

```
hostname tfacc-extras-ams-sw1
ntp server 10.10.0.1
ntp server 10.10.0.2
logging host 10.10.0.3 port 514
```

## NetBox behaviours discovered

* **Custom field selection values round-trip as `{value,label}`.** With a
  `select` field the site is written as
  `custom_fields = jsonencode({ tfacc_extras_support_tier = "gold" })`; the API
  returns `{"tfacc_extras_support_tier": {"value": "gold", "label": "Gold"}}`.
  `conv.CustomFieldsFromAPI` / `AllCustomFieldsFromAPI` unwrap this, so the
  resource shows no diff and the `netbox_site` data source yields
  `jsondecode(custom_fields)["tfacc_extras_support_tier"] == "gold"`. The
  resource only tracks keys present in the configuration (other global custom
  fields on the shared demo never cause drift); the data source exposes all.
* **`choice_colors` takes colour names, not hex.** Valid values are `blue`,
  `indigo`, `purple`, `pink`, `red`, `orange`, `yellow`, `green`, `teal`,
  `cyan`, `gray`, `black`, `white`. Hex codes fail with
  `400 choice_colors: {"gold":["\"ffc107\" is not a valid choice."]}`. The
  attribute description now documents this (override).
* **`order_alphabetically = true` reorders `extra_choices` server-side.** The
  API returns the sorted list, so an unsorted configuration produces
  `Provider produced inconsistent result after apply: .extra_choices[2][0]: was
  "gold", but now "silver"`. Configure the list already sorted (fixture and
  examples do). `extra_choices` is an ordered list of `[value, label]` pairs.
* **Text fields are whitespace-trimmed by the API** (DRF `CharField`
  `trim_whitespace`). A heredoc `template_code` ending in a newline is stored
  without it and Create fails with an inconsistent-result error. Wrap heredocs
  in `trimspace(...)` (examples and scenario do). See generator gap 1.
* **`journal_entry.created_by` is filled by the server** with the token's user.
  The generator classified it as a nullable FK (`Optional`), so the provider
  sent `null`, and the read-back user id was an inconsistent result. Override
  `created_by: { computed: true }` makes it `Optional+Computed` and unknown
  values are then omitted from the request.
* **`notification_group` accepts an empty membership** (a POST with only
  `name` succeeds). Members are `group_ids` (users.group) and `user_ids`.
* **`event_rule`** requires `action_object_type` (`extras.webhook` or
  `extras.notificationgroup`) and `action_object_id`; `event_types` is a list of
  `object_created|object_updated|object_deleted|job_*`. `conditions` JSON is
  returned exactly as sent.
* **`webhook.payload_url`** must be an absolute URL; the synthesised fixture
  (`payload_url = "{{.Name}}"`) is rejected. `secret` and `additional_headers`
  are returned by the API (import verify passes).
* **`wireless_lan.auth_psk` / `wireless_link.auth_psk`** are returned in clear
  text by the API (not write-only); the provider marks them `Sensitive` (import
  verify still compares them).
* **`wireless_link`** requires both interfaces to be wireless types
  (`ieee802.11*`); `rf_role` (`ap`/`station`) is optional but realistic.
  `distance` is a float and round-trips at `1.5`.
* **List filters are silently ignored when unknown.** `tags?for_object_type=`
  and `webhooks?payload_url__ic=` return the unfiltered list; the spec-listed
  names are `for_object_type_id` (numeric content type) and `q`. `tags?object_types=`
  wants content-type IDs, not `app.model` labels. The notification-groups
  endpoint has no filters in the spec but `?name=` works live.
* **Terraform 1.16.2 crash.** When the provider returned an inconsistent
  result on Create (the `template_code` trimming above) Terraform panicked in
  `statefile.StatesMarshalEqual` with `Instance netbox_config_template.base has
  status ObjectStatus(0), which cannot be saved in state` and left an empty
  state file, orphaning already-created objects. That is a Terraform core bug
  (the provider error is legitimate), but it makes the whitespace issue below
  more damaging than a plain error.

## Generator gaps (not patched; overrides cannot express them)

1. **Whitespace-trimmed strings** (`internal/gen/build` / `internal/gen/render`).
   NetBox trims leading and trailing whitespace of every text field
   (`template_code`, `body_template`, `additional_headers`, `comments`,
   `link_text`, `description`, ...). A multi-line HCL heredoc therefore never
   matches what the API returns, and the failure surfaces as
   `Provider produced inconsistent result after apply` on Create (no plan-time
   hint). Proposed fix: emit a custom string type for `KindString` attributes
   whose `StringSemanticEquals` treats values equal when
   `strings.TrimSpace(a) == strings.TrimSpace(b)` (as done for JSON via
   `jsontypes.Normalized`), or a plan modifier that trims the planned value.
   Until then the examples use `trimspace(<<-EOT ... EOT)`.
2. **`extra_choices` ordering with `order_alphabetically`** is a NetBox
   behaviour rather than a bug, but the generator could document it: the
   `any_list` kind is rendered as an ordered `List` of `List(String)` and the
   description ("Extra Choices.") gives no hint that value/label pairs are
   expected. Proposed fix: derive the description from the schema
   (`minItems: 2, maxItems: 2` -> "list of `[value, label]` pairs").
3. **Server-populated nullable FKs** (`journal_entry.created_by`) need a manual
   `computed: true`. Proposed fix: when a request schema property has
   `default` semantics that the spec cannot express, nothing can be inferred;
   but the generator could treat every nullable FK named `created_by` /
   `owner` as `Optional+Computed` by default (both are filled by the server).
4. **Unfiltered list data sources.** Unknown filter names are ignored by NetBox,
   so a typo returns every object. Proposed fix: validate `filters[*].name`
   against the spec's parameter list for the endpoint (the generator already
   counts them: `[filters] 97`) and emit a `stringvalidator.OneOf` or at least a
   warning diagnostic.

None of these blocks a resource; no test is skipped.

## Files

* `generator/overrides/extras_wireless.yaml` - fixtures for the 13 resources
  that had none, `serial_test` for the global extras objects, the
  `journal_entry.created_by` override and the `choice_colors` description.
* `examples/resources/netbox_<name>/{resource.tf,import.sh}` and
  `examples/data-sources/netbox_<name>[s]/data-source.tf` for all 15 resources
  (hand-maintained, `.generated` markers removed).
* `examples/scenarios/extras/main.tf` - the scenario above.
