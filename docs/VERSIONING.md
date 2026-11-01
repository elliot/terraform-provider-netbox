# Versioning and compatibility policy

This policy applies from `v1.0.0`. Before 1.0, minor releases may still contain breaking changes; they are
listed under *Breaking changes* in the [changelog](../CHANGELOG.md).

## Semantic versioning

Provider releases are numbered `MAJOR.MINOR.PATCH` following [Semantic Versioning](https://semver.org/). The
major version only changes for breaking changes to the provider's own surface: the resources, data sources,
attributes and provider settings that a configuration or a state file depends on.

| Release | Contains |
|---|---|
| Patch (`1.2.3` -> `1.2.4`) | bug fixes, documentation, dependency updates, a NetBox patch-level spec bump with no schema change visible to users |
| Minor (`1.2.x` -> `1.3.0`) | new resources, data sources, attributes and provider settings; a new pinned NetBox minor; deprecations |
| Major (`1.x` -> `2.0.0`) | breaking changes, including the removal of everything deprecated during the previous major |

A change is **breaking** when an existing configuration, state or workflow that worked on the previous release
can fail or behave differently without being edited. In particular:

* removing or renaming a resource, data source, attribute or provider setting;
* changing an attribute's type (for example string to number, or list to set);
* making an optional attribute required, or adding a new required attribute;
* changing a default value or the behaviour an existing configuration relies on (what is sent to NetBox, what is
  tracked in state, whether a change forces replacement);
* changing an import ID or resource identity format;
* raising the minimum supported Terraform version or the minimum supported NetBox version.

Not breaking: adding resources, data sources, optional or computed attributes and provider settings; new
values accepted by a choice attribute; new validators that reject input NetBox already rejected at apply time;
fixing a behaviour that contradicted the documentation; marking an attribute deprecated. State changes that
the provider migrates automatically (schema versions with a state upgrader) are not breaking either, but the
state written by a newer release cannot be read by an older one: do not downgrade the provider in place.

Attributes holding secrets stay `Sensitive`. Write-only variants of those attributes may be added in a 1.x
minor release, next to the existing ones, without changing them.

## Deprecation policy

Nothing the provider controls is removed without a deprecation period:

1. The resource, data source or attribute is marked deprecated in a minor release: it keeps working, its schema
   carries a `DeprecationMessage` (Terraform prints a warning on every plan that uses it) naming the replacement,
   and the changelog lists it.
2. It is removed no earlier than the next major release. The changelog of that release lists the removal under
   *Breaking changes* with upgrade notes.

Where a renamed attribute has a direct replacement, the replacement is added in the same minor release as the
deprecation, so a configuration can move to it before the old name is removed.

## NetBox releases and provider releases

Each provider release is generated from, and tested against, exactly one pinned NetBox OpenAPI document
([`spec/VERSION`](../spec/VERSION)). The mapping is:

* **NetBox patch release** (`4.7.3` -> `4.7.4`): picked up in a provider patch or minor release, depending on
  whether it adds anything to the schema.
* **NetBox minor release** (`4.7` -> `4.8`): a provider minor release. New NetBox collections and fields become
  new resources and attributes.
* **Something NetBox deprecates or renames** (a field, a choice, a whole endpoint) while the old form still
  exists in its API: handled through the deprecation policy above. The provider keeps the attribute, marks it
  deprecated and adds the replacement next to it; the old attribute is removed in the next provider major
  release, not in the minor release that pins the new NetBox version.
* **Something NetBox deletes outright** (the field is gone from the API, so the provider cannot keep sending or
  reading it): the attribute is removed in the release that pins that NetBox version, and the changelog lists it
  under *Breaking changes* with upgrade notes. Where that would break working configurations, the provider keeps
  the previous NetBox spec pinned until the next major instead. NetBox normally deprecates before it deletes, so
  this should be rare.
* **NetBox 5.0**: supported by a provider minor release when the API changes allow it. A provider 2.0 follows
  if NetBox removes things that cannot be handled as above.

## Supported NetBox versions

The NetBox minor of the pinned spec is fully supported: CI runs the acceptance suite against it nightly. At
start-up the provider warns (without failing) when the server runs a different minor version.

Older NetBox 4.x releases, back to 4.5 (the first with v2 API tokens), are supported on a best-effort basis:
resources and attributes work where the older API has them, with the known gaps listed under
[Older NetBox releases](../README.md#older-netbox-releases). Resources and attributes that need a newer NetBox
than the server runs fail at apply time; that is not a breaking change.

Moving the pinned spec to a newer NetBox minor is not breaking either: the previous minor becomes best effort.
Raising the minimum NetBox version, so that existing resources stop working on a release that was best effort
before, is breaking and only happens in a major release.

## Supported Terraform versions

The minimum Terraform version is 1.12, because every resource implements resource identity (`import` blocks
with `identity = { id = ... }`). CI runs the unit tests on the minimum version and the latest Terraform release.
Raising the minimum is a breaking change and only happens in a major release.

## Verifying releases

Every release is built by GitHub Actions from the tagged commit, its checksums are GPG-signed and every asset has
a build provenance attestation. The Terraform Registry checks the signature on install; how to verify a release
yourself is described in [SECURITY.md](../SECURITY.md#verifying-a-release).
