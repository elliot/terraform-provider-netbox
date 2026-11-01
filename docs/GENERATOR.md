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
* Attribute names separate the words NetBox runs together in a few
  properties and spell counters `<singular>_count` (`naming.Attribute`:
  `ipaddresses` -> `ip_address_ids`, `virtualmachine_count` ->
  `virtual_machine_count`, `count_ipaddresses` -> `ip_address_count`).
  Integer IDs inside nested objects and in data-source-only collections are
  renamed to `<field>_id` through the overrides (`rear_ports[].rear_port_id`).
* `tags` is a set of tag **slugs**. `custom_fields` is a `DynamicAttribute`
  handled by `internal/customfields` (definition cache per object type,
  validation of names, coercion of NetBox's `{value,label}` and related-object
  representations to the configured scalar types). Data sources expose
  `custom_fields` as a JSON string instead, because the framework does not
  allow dynamic values inside list data source items. Untyped JSON fields are
  JSON strings (`jsonencode({...})`).

## Attribute rules (see `internal/gen/build/build.go`)

| API shape | Terraform |
|---|---|
| required property | `Required` |
| nullable string / FK | `Optional`; `null` is sent as JSON null |
| nullable number, nullable choice, timestamp | `Optional+Computed`, `UseStateForUnknown`; never sent as null (NetBox derives or rejects blanks) |
| ID set that is the reverse side of a required FK (`provider.accounts`) | `Computed` only |
| many-to-many writable from both sides (`site.asn_ids` / `asn.site_ids`) | generation fails until one side has `read_only` (that side becomes `Computed` only) |
| request-only nested list not echoed on read (`circuit.assignments`) | dropped |
| nullable untyped JSON (`local_context_data`, `constraints`) | `Optional`; removing it sends an explicit `null` (the generated client drops nil JSON, so that update goes out as a raw PATCH via `conv.PatchWithNulls`) |
| non-nullable optional string that may be blank | `Optional+Computed`, default `""`, always sent |
| non-nullable optional string with a pattern (colour) | `Optional+Computed`, `UseStateForUnknown`, sent when known |
| choice / boolean / non-nullable number | `Optional+Computed`, `UseStateForUnknown`, sent when known |
| integer list, string list, tags | `Optional+Computed` Set with empty default, always sent |
| array of objects | `Optional` ListNestedAttribute |
| `id`, `url`, `display`, `created`, `last_updated` | `Computed` |
| other read-only fields (`*_count`, `display_url`, ...) | data sources only |

Update always uses PATCH built from the full plan; unknown or null values of
computed attributes are omitted so NetBox keeps its defaults. On read, strings
keep the configured value when NetBox only trimmed whitespace (or changed the
case of an attribute with `format: mac` / `wwn`), and floats keep it when equal
at the field's precision, so normalisation never produces a diff. The `format`
comes from the spec when NetBox declares one, otherwise from the overrides
(4.7 declares none); it also adds a colon-notation validator, because NetBox
rewrites dashed or dotted MAC addresses and the result would not match.

## Overrides reference

Files in `generator/overrides/` are merged in file-name order; the same
resource may appear in several files (naming in one, fixtures in another), but
keep each resource's attribute overrides in a single file: later files win
silently. Overrides that hide or force attributes are listed in
[ROADMAP.md](ROADMAP.md) and should be revisited on every spec bump.
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
    deprecated: "..."          # DeprecationMessage on the resource and both data sources
                               #   (true: default message; a deprecated list operation in the spec sets it too)
    schema_version: 1          # resource schema Version; above 0 the resource implements
                               #   UpgradeState with the upgraders in internal/upgrade
    attributes:
      parent:                  # keyed by API property name
        name: parent_id        # Terraform attribute name
        kind: fk               # force a kind (fk, string, int, ...)
        target: interface      # FK target resource (for docs and sweepers)
        skip: true             # drop the attribute
        requires_replace: true
        sensitive: true
        computed: false        # force Optional-only (no server default tracking)
        read_only: true        # keep as Computed only (reverse side of a relation)
        expose: true           # include a read-only API property as a Computed attribute;
                               #   the prior state is kept when the API returns null (token secret)
        optional: true         # make a required property optional
        default: 1             # static default for an integer/FK attribute (makes it Optional+Computed)
        precision: 6           # decimals NetBox stores; drives float semantic equality (default 6)
                               #   and adds a conv.MaxDecimals validator. latitude/longitude get 6 automatically
        format: mac            # mac or wwn: colon-notation validator, letter case ignored on read
        ordered_list: true     # List instead of Set
        description: "..."
        enum: [a, b]           # replace the allowed values
        deprecated: "Use y."   # DeprecationMessage on the resource and data source attribute
                               #   (also nested and data-source-only attributes); true: default
                               #   message; false: ignore `deprecated: true` in the spec
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
      upgrade:                 # TestAcc<Resource>_upgrade (see Upgrade tests)
        skip: reason           # skip the upgrade test only
        config: |              # complete HCL both the release and this build accept;
          ...                  #   replaces trying update, then basic
```

Templates use Go `text/template`: `{{.Name}}` is a unique `tfacc-xxxxxxxx`
string, `{{.Snake}}` the same with underscores. Every object a fixture
creates must contain the `tfacc-` prefix in its name, slug or description so
sweepers can find it (`make sweep`).

When no fixture is given, the generator synthesises one if every required
attribute is a plain string, choice, number or boolean; otherwise the test is
generated with `t.Skip(...)` naming the override to add.

## Changing the schema after 1.0

From 1.0 on, a provider upgrade must not break a working configuration or
state within the major version. Anything NetBox removes or renames stays as a
deprecated attribute through the 1.x line and is removed in the next major.

**Deprecation.** `deprecated: "<message>"` on an attribute or a resource sets
`DeprecationMessage`, so Terraform warns wherever the configuration uses it,
and appends `**Deprecated:** <message>` to the description (tfplugindocs only
shows a "Deprecated" flag, not the message). Properties and list operations
the spec marks `deprecated: true` get a default message automatically;
override it with your own message, or with `deprecated: false` when the
provider keeps supporting the field. `skip` still wins. A required attribute
cannot be deprecated (users could not stop setting it): make it `optional`
first. `docs/RESOURCES.md` lists deprecated resources and attributes in its
Notes column.

**Rename.** Add the new attribute and keep the old one with `deprecated:
"Use <new> instead."`. When NetBox renames a field it usually keeps the old
property for a release; the new property is generated automatically and the
old one only needs the override. A provider-side rename of one API property
(two Terraform names for the same field) is not supported by the generator
yet, so rename attributes before 1.0 or in a major release.

**Type or shape change.** Bump `schema_version` on the resource and write a
state upgrader by hand:

1. Set `schema_version: N` in the resource's overrides and run `make gen`.
   The generated resource sets `Version: N` and implements
   `resource.ResourceWithUpgradeState` by returning
   `upgrade.For("netbox_<name>", N)`.
2. Add `internal/upgrade/<name>.go` registering one upgrader per older
   version from `init()`:
   `upgrade.Register("netbox_<name>", 0, resource.StateUpgrader{PriorSchema: ..., StateUpgrader: ...})`.
   The framework calls only the upgrader keyed by the version in the state,
   and it must return state of the current version: on the next bump, update
   every existing upgrader to produce the new shape and add one from the
   previous version.
3. `make test` fails (`TestEveryVersionedResourceHasUpgraders`) while a
   version has no upgrader or an upgrader is registered for a version that
   is not older than the resource's. `internal/upgrade/upgrade_test.go` shows
   an upgrader exercised end to end through the protocol server.

All resources are at schema version 0 today.

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

### Upgrade tests

Every generated test file also has `TestAcc<Resource>_upgrade`, which checks
that state written by a published release still plans cleanly with the
current build: step 1 applies the fixture with the release from the Terraform
Registry (`ExternalProviders`), step 2 switches to the in-process provider and
expects an empty plan. The hand-written resources have curated equivalents in
`internal/provider/manual/upgrade_test.go`. Both go through
`acctest.UpgradeTest`, which:

* skips unless `TF_ACC` and `NETBOX_UPGRADE_FROM` (the release, e.g. `0.5.0`)
  are set; `NETBOX_UPGRADE_SOURCE` overrides the registry address
  (default `elliot/netbox`);
* reads the release's schema once per test binary (`terraform providers schema
  -json`) and uses the first configuration that only contains resource types,
  data sources and attributes the release knows: the `update` fixture, then
  the `basic` one. A fixture that uses something newer than the release falls
  back or is skipped with the unknown names in the skip message, so new
  resources and attributes need no override;
* defaults `TF_PLUGIN_CACHE_DIR` to a directory under the user cache, so the
  release is downloaded once rather than by every test.

```sh
NETBOX_UPGRADE_FROM=0.5.0 TF_ACC=1 go test ./internal/provider/gen/dcim/ -run 'TestAccSite_upgrade$' -v
```

A failure means the current build reads the state of an existing resource
differently from the release: a renamed or removed attribute, a changed type
without a state upgrader, or a normalisation that now produces a diff. Fix it
in the generator (or add a state upgrader) unless the change is a deliberate
breaking change. For deliberate ones:

* a renamed attribute is unknown to the release, so fixtures using the new
  name fall back or skip on their own. `test.upgrade.config` supplies a
  configuration that both the release and the current build accept (for
  example one that leaves the renamed attribute out) to keep the resource
  covered, and `test.upgrade.skip` opts a resource out with a reason;
* after a breaking release, the upgrade from the release before it fails by
  design. Run the tests with `NETBOX_UPGRADE_FROM` set to the last
  schema-compatible release (the nightly workflow takes it from the
  `upgrade_from` input) until the breaking release is the latest one, then
  drop overrides that only existed for the old schema.

If the release refuses the NetBox under test (it checks the server version),
set `NETBOX_SKIP_VERSION_CHECK=true` for the run; both providers read it.

## Adding hand-written behaviour

* Hand-written resources live in `internal/provider/manual/` and register
  themselves with `provider.RegisterResource` from `init()`.
* Shared conversions belong in `internal/conv`.
* Never edit generated files; change the generator or the overrides and run
  `make gen`.
