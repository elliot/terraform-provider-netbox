## Provider schema diff: v1.0.0 -> HEAD

Breaking: **7**, additive: **6**, notices: **4**.

Descriptions changed on 2 attributes. They are not classified, but allowed values (validators) are not part of the schema and only show up there; check the docs diff for removed values.

Breaking changes found: existing configurations may fail or plan unexpected changes.

### Breaking changes

| Kind | Name | Attribute | Change |
|---|---|---|---|
| resource | `demo_site` | `group_id` | attribute removed |
| resource | `demo_site` | `region_id` | optional -> computed-only: configurations that set it fail |
| resource | `demo_site` | `secret` | now write-only: the value is no longer stored in state or readable |
| resource | `demo_site` | `slug` | optional -> required: configurations that omit it fail |
| resource | `demo_site` | `status` | optional+computed -> optional: an omitted value is now null instead of the server value, so existing resources plan a change |
| resource | `demo_site` | `tag_ids` | type changed from list(number) to set(number) |
| resource | `demo_vlan` |  | removed |

### Additive changes

| Kind | Name | Attribute | Change |
|---|---|---|---|
| provider |  | `insecure` | new optional attribute |
| resource | `demo_site` | `legacy` | computed-only -> optional+computed |
| resource | `demo_site` | `ports.mtu` | new optional attribute |
| resource | `demo_site` | `tenant_id` | new optional attribute |
| resource | `demo_site` | `timeouts.delete` | new optional attribute |
| resource | `demo_tenant` |  | new resource |

### Notices

| Kind | Name | Attribute | Change |
|---|---|---|---|
| provider |  | `api_token` | now sensitive (root module outputs that expose it need `sensitive = true`) |
| resource | `demo_site` |  | schema version increased from 0 to 1 (state is upgraded on the next refresh) |
| resource | `demo_site` | `facility` | optional -> optional+computed: an omitted value is now filled in by the provider instead of null |
| resource | `demo_site` | `facility` | deprecated |

