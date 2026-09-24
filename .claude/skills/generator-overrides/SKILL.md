---
name: generator-overrides
description: How to add, fix or reshape a generated NetBox resource or data source through generator/overrides/*.yaml and internal/gen, including acceptance-test fixtures. Use whenever a change touches internal/provider/gen/, generator/overrides/, a generated attribute's name/kind/computed behaviour, or a skipped/failing generated TestAcc test.
---

Generated resources are never edited by hand. Every change goes through `generator/overrides/*.yaml` (per-resource
data) or `internal/gen/` (rules that apply to every resource), then regeneration.

## Workflow

1. Find the names: `go run ./internal/gen -list` prints Terraform resource names (used by `-only`). Override
   keys are different: API path segments like `sites`, or `app/segment` when a segment exists in several apps
   (`virtualization/interfaces`).
2. Inspect the current model: `go run ./internal/gen -dump -only <name>`.
3. Find where the resource is already overridden: `grep -n '^  <key>:' generator/overrides/*.yaml`. Keep all
   attribute overrides for one resource in that single file. Files merge in name order and later files win
   silently. `_naming.yaml` only resolves name collisions.
4. Edit the override (keys below), or change `internal/gen/build/build.go` if the rule should apply generically.
5. Regenerate only what you touched: `go run ./internal/gen -only <name>[,<name>...]`. A run without `-only`
   deletes and rewrites every generated file, but it is what CI checks, so finish with `make gen` and
   `make docs` and commit all of their output.
6. `go build ./... && go test ./internal/provider/gen/<app>/` (unit), then run the resource's acceptance test
   (`/acc-test <Resource>`).
7. If the override hides or forces an attribute (`skip`, `read_only`, `computed`, `expose`, `enum`), add or
   update its row in the "Things to circle back to" table in `docs/ROADMAP.md`.

## Override keys

See the full reference in docs/GENERATOR.md ("Overrides reference"). The most used ones:

```yaml
resources:
  <key>:
    name: interface              # Terraform name without netbox_
    skip: true / skip_reason     # do not generate
    lookups: [name]              # singular data source lookup attributes
    serial_test: true            # resource.Test instead of ParallelTest
    attributes:
      <api_property>:
        name: parent_id          # FKs are <field>_id, M2M are <singular>_ids
        kind: fk                 # force a kind
        target: interface        # FK target (docs + sweeper ordering)
        read_only: true          # reverse side of a relation -> Computed only
        computed: false          # Optional only, no server-default tracking
        expose: true             # read-only property as Computed (write-once secrets)
        requires_replace / sensitive / optional / ordered_list / precision / enum / description
    test:
      skip: reason
      basic: |                   # full HCL for step 1
      update: |                  # step 2, must be an in-place update
      checks: { status: active }
      import_verify_ignore: [x]
```

## Fixture rules

- Go `text/template`: `{{.Name}}` is a unique `tfacc-xxxxxxxx`, `{{.Snake}}` the same with underscores.
- Every object a fixture creates must carry `tfacc-` in its name, slug or description, or `make sweep` cannot
  clean it up.
- The resource under test must be `<terraform_type>.test`; supporting objects can have any label.
- `update` must change only in-place attributes (the generated test asserts no replacement).
- Without a fixture the generator synthesises one only when every required attribute is a plain scalar;
  otherwise it emits a `t.Skip(...)` naming the override to add. Replacing that skip with a real fixture is
  the usual fix.
