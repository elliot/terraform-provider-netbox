# Code generator

Almost everything under `internal/provider/gen/`, the `examples/` tree and
`docs/RESOURCES.md` is produced by `go run ./internal/gen` (`make gen`) from
two inputs:

1. `spec/netbox-<VERSION>.openapi.json`: the pristine NetBox OpenAPI document
   (`spec/VERSION` selects it). `spec/required-fixes.json` corrects the
   `required` lists of a few request schemas and is shared with the client
   generator (`tools/client-gen`).
2. `generator/overrides/*.yaml`: hand-maintained tuning, merged across files.

The API client in `netbox/` is generated separately by `make client-gen`
(openapi-generator, see `tools/client-gen/`). Regenerate it only when the
spec changes.

## Pipeline

```
spec ──► internal/gen/openapi (parse) ──► internal/gen/build (classify) ──► model.Resource[]
                                              ▲
                          generator/overrides ┘
model.Resource[] ──► internal/gen/render ──► internal/provider/gen/<app>/<name>_resource.go
                                          ├─► internal/provider/gen/<app>/<name>_data_source.go
                                          ├─► internal/provider/gen/<app>/<name>_resource_test.go
                                          ├─► internal/provider/gen/all/all.go   (registration)
                                          ├─► examples/resources/netbox_<name>/{resource.tf,import.sh}
                                          ├─► examples/data-sources/netbox_<name>/data-source.tf
                                          ├─► examples/data-sources/netbox_<plural>/data-source.tf
                                          └─► docs/RESOURCES.md
```

Useful commands:

```sh
go run ./internal/gen -list                 # every collection and its Terraform names
go run ./internal/gen -dump -only site,vrf  # classified attributes of some resources
go run ./internal/gen -only site,vrf        # regenerate a subset (fast iteration)
make gen && git diff --exit-code            # what CI enforces
```

## Naming

* Resource: `netbox_<singular of the API path segment>` (`ip-addresses` ->
  `netbox_ip_address`). Data source: same name. List data source:
  `netbox_<path segment>` (`netbox_ip_addresses`).
* Foreign keys: `<field>_id` (Int64). Many-to-many integer lists:
  `<singular>_ids` (Set of Int64). Generic FKs keep `*_object_type` /
  `*_object_id`.
* `tags` is a set of tag **slugs**. `custom_fields` and untyped JSON fields
  are JSON strings (`jsonencode({...})`).

## Attribute rules (see `internal/gen/build/build.go`)

| API shape | Terraform |
|---|---|
| required property | `Required` |
| nullable scalar / FK | `Optional`; `null` is sent as JSON null |
| non-nullable optional string that may be blank | `Optional+Computed`, default `""`, always sent |
| non-nullable optional string with a pattern (colour) | `Optional+Computed`, `UseStateForUnknown`, sent when known |
| choice / boolean / non-nullable number | `Optional+Computed`, `UseStateForUnknown`, sent when known |
| integer list, string list, tags | `Optional+Computed` Set with empty default, always sent |
| array of objects | `Optional` ListNestedAttribute |
| `id`, `url`, `display`, `created`, `last_updated` | `Computed` |
| other read-only fields (`*_count`, `display_url`, ...) | data sources only |

Update always uses PATCH built from the full plan; unknown or null values of
computed attributes are omitted so NetBox keeps its defaults.

## Overrides reference

Files in `generator/overrides/` are merged in file-name order; the same
resource may appear in several files (naming in one, fixtures in another).
Keys are API path segments (`sites`) or `app/segment` when a segment exists
in several apps (`virtualization/interfaces`).

```yaml
resources:
  interfaces:
    name: interface            # Terraform name without the netbox_ prefix
    plural: interfaces         # list data source name
    skip: true                 # do not generate anything
    skip_reason: why
    description: "..."         # resource description
    lookups: [name]            # singular data source lookup attributes (default: name, slug)
    import_verify_ignore: [x]  # attributes ignored by ImportStateVerify
    serial_test: true          # resource.Test instead of resource.ParallelTest
    attributes:
      parent:                  # keyed by API property name
        name: parent_id        # Terraform attribute name
        kind: fk               # force a kind (fk, string, int, ...)
        target: interface      # FK target resource (for docs and sweepers)
        skip: true             # drop the attribute
        requires_replace: true
        sensitive: true
        computed: false        # force Optional-only (no server default tracking)
        optional: true         # make a required property optional
        precision: 6           # float decimals (documentation for now)
        ordered_list: true     # List instead of Set
        description: "..."
        enum: [a, b]           # replace the allowed values
        nested:                # same keys for nested list item properties
          rear_port: { target: rear_port }
    test:
      skip: reason             # skip the acceptance test
      basic: |                 # complete HCL; the resource under test must be
        resource "netbox_interface" "test" { ... }   #   <type>.test
      update: |                # second step; must be an in-place update
        ...
      checks:                  # extra TestCheckResourceAttr on step 1
        status: active
      import_verify_ignore: [x]
```

Templates use Go `text/template`: `{{.Name}}` is a unique `tfacc-xxxxxxxx`
string, `{{.Snake}}` the same with underscores. Every object a fixture
creates must contain the `tfacc-` prefix in its name, slug or description so
sweepers can find it (`make sweep`).

When no fixture is given, the generator synthesises one if every required
attribute is a plain string, choice, number or boolean; otherwise the test is
generated with `t.Skip(...)` naming the override to add.

## Acceptance tests

Generated tests run create -> update (asserting an in-place update) -> data
source lookups (`data.netbox_<name>` by id and `data.netbox_<plural>` with an
`id` filter) -> import with `ImportStateVerify` -> empty plan.

```sh
make demo-token                      # writes .env.demo for https://demo.netbox.dev
set -a; source .env.demo; set +a
export NETBOX_REQUESTS_PER_SECOND=5  # be polite to the shared demo
TF_ACC=1 go test ./internal/provider/gen/dcim/ -run TestAccSite_basic -v
make sweep                           # delete leftover tfacc-* objects
```

## Adding hand-written behaviour

* Hand-written resources live in `internal/provider/manual/` and register
  themselves with `provider.RegisterResource` from `init()`.
* Shared conversions belong in `internal/conv`.
* Never edit generated files; change the generator or the overrides and run
  `make gen`.
