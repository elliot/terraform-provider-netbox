# Security policy

## Supported versions

Security fixes land on `main` and ship in the next release. Only the latest minor release line receives
fixes while the provider is pre-1.0.

## Reporting a vulnerability

Please do **not** open a public issue. Report it privately through GitHub's
[private vulnerability reporting](https://github.com/elliot/terraform-provider-netbox/security/advisories/new)
(Security tab → *Report a vulnerability*) with a description, affected versions and, if possible, a
reproduction. You should get an acknowledgement within a week; fixes are coordinated through a GitHub
security advisory and credited unless you prefer otherwise.

Vulnerabilities in NetBox itself belong upstream at
[netbox-community/netbox](https://github.com/netbox-community/netbox/security).

## Verifying a release

Every release zip, `SHA256SUMS` file and registry manifest carries a
[build provenance attestation](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations)
signed through Sigstore. It proves the file was built by this repository's release workflow from a
specific tag and commit:

```sh
gh attestation verify terraform-provider-netbox_<version>_linux_amd64.zip \
  --repo elliot/terraform-provider-netbox
```

`scripts/install.sh` checks the zip against `SHA256SUMS`, and Terraform checks mirror downloads against the
`h1:` hashes in `.terraform.lock.hcl`. When the GPG secrets are configured, `SHA256SUMS` is also signed
(`SHA256SUMS.sig`), which is what the public Terraform Registry verifies.

## How the build and release pipeline is protected

The measures follow [Open source security at Astral](https://astral.sh/blog/open-source-security-at-astral)
and are checked in CI by [zizmor](https://docs.zizmor.sh) (`.github/workflows/zizmor.yml`). The Go sources
and the workflows are also scanned by [CodeQL](https://codeql.github.com/) (`.github/workflows/codeql.yml`) on
every pull request, on `main` and weekly; results appear under Security → Code scanning.

In the repository:

* No `pull_request_target` or `workflow_run` triggers; pull requests from forks never receive a write token
  or secrets.
* Every action is pinned to a full commit SHA, and zizmor's impostor-commit audit checks each SHA belongs to
  its upstream repository. Tools installed by actions (GoReleaser, golangci-lint, zizmor) are pinned to exact
  versions, and the openapi-generator jar is pinned by SHA-256.
* Workflows start from `permissions: {}` and each job requests only the scopes it uses; checkouts do not
  persist the token.
* Release builds run without the Actions cache, inside the `release` deployment environment that holds
  the signing secrets, and publish provenance attestations.
* Renovate proposes dependency updates only after a 7-day cooldown; security fixes bypass it.

In the GitHub settings (maintainer checklist):

* `release` environment: required reviewer, deployment limited to `v*` tags, `GPG_PRIVATE_KEY` /
  `PASSPHRASE` stored as environment secrets rather than repository secrets.
* Actions: *Require actions to be pinned to a full-length commit SHA*; default workflow permissions
  *read repository contents*; *Allow GitHub Actions to create and approve pull requests* off.
* Rulesets: `main` requires pull requests and passing checks and blocks force-pushes and deletion; `v*`
  tags cannot be updated or deleted; no bypass actors.
* Releases: [immutable releases](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/immutable-releases)
  enabled, so published assets cannot be replaced.
* Account: a phishing-resistant second factor (passkey or security key).
* Private vulnerability reporting enabled.
