# Installing the provider without the Terraform Registry

The provider is not (yet) published on `registry.terraform.io`. Terraform still resolves it under its
normal address, `elliot/netbox`, through one of the three supported off-registry channels below. All of
them keep `terraform init`, the dependency lock file and `required_providers` version constraints working
exactly as they do for registry providers.

| Channel | Best for | Needs |
|---|---|---|
| [Install script](#1-install-script-implied-local-mirror) | one machine, CI runners, quick start | `curl`, nothing else |
| [Network mirror](#2-network-mirror-github-pages) | teams, many machines, version pinning | one CLI config file |
| [Filesystem mirror / air-gapped bundle](#3-filesystem-mirror-and-air-gapped-bundles) | no internet, corporate artifact stores | a shared directory |

Every GitHub Release ships `terraform-provider-netbox_<version>_<os>_<arch>.zip` for **linux** and
**darwin** (`amd64`, `arm64`), plus `freebsd`/`windows`, a `_SHA256SUMS` file and the registry
manifest. Releases are unsigned unless a GPG key is configured (only the public registry requires the
signature; Terraform itself verifies mirror downloads with the `h1:` hashes in the lock file).

Reference the provider the same way in every case:

```hcl
terraform {
  required_providers {
    netbox = {
      source  = "elliot/netbox"
      version = "~> 0.1"
    }
  }
}
```

## 1. Install script (implied local mirror)

```sh
curl -fsSL https://raw.githubusercontent.com/elliot/terraform-provider-netbox/main/scripts/install.sh | sh
# or a specific release:
VERSION=0.1.0 sh scripts/install.sh
```

The script detects the OS and CPU, downloads the zip and `SHA256SUMS` from the GitHub Release, verifies
the checksum and unpacks the binary to

```
~/.terraform.d/plugins/registry.terraform.io/elliot/netbox/<version>/<os>_<arch>/terraform-provider-netbox_v<version>
```

This is one of Terraform's [implied local mirror directories](https://developer.hashicorp.com/terraform/cli/config/config-file#implied-local-mirror-directories),
so **no CLI configuration is required**: `terraform init` finds the provider there and never contacts the
registry for it. Set `TF_PLUGIN_DIR` to install somewhere else (for example a directory named in a
`filesystem_mirror` block), `REPO`/`RELEASE_BASE_URL` to install from a fork or an internal artifact store,
and `OS`/`ARCH` to override detection. The unpacked layout is used because Terraform only recognises the
packed (zip) layout for plain `x.y.z` versions, not pre-releases or snapshots.

On macOS, Terraform plugins need no notarization. The script removes the `com.apple.quarantine` attribute
in case the archive was fetched by a browser earlier; downloads made by `curl` never carry it.

## 2. Network mirror (GitHub Pages)

Every release also refreshes a static [provider network mirror](https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol)
at `https://elliot.github.io/terraform-provider-netbox/` (`index.json` and `<version>.json` under
`registry.terraform.io/elliot/netbox/`; the JSON points at the zips attached to the GitHub Releases).
Add this to `~/.terraformrc` (Linux/macOS) or to the file named by `TF_CLI_CONFIG_FILE`:

```hcl
provider_installation {
  network_mirror {
    url     = "https://elliot.github.io/terraform-provider-netbox/"
    include = ["elliot/netbox"]
  }
  direct {
    exclude = ["elliot/netbox"]
  }
}
```

`include`/`exclude` keep every other provider on the normal registry path. Terraform requires an `https`
URL for network mirrors and a trailing slash. Version constraints, `terraform init -upgrade` and
`terraform providers lock` all work against the mirror.

To host the mirror elsewhere (an internal web server, S3, Artifactory generic repo), copy the
`dist/mirror` tree produced by `make dist mirror` or the `mirror` artifact of the release workflow. With
`make mirror MIRROR_ARGS='-base-url https://artifacts.example.com/netbox/'` the JSON references zips
hosted at that prefix; without `-base-url` the zips are copied into the tree and referenced relatively,
which gives a self-contained directory.

## 3. Filesystem mirror and air-gapped bundles

A filesystem mirror is a directory in either layout Terraform understands:

```
registry.terraform.io/elliot/netbox/terraform-provider-netbox_0.1.0_linux_amd64.zip   # packed
registry.terraform.io/elliot/netbox/0.1.0/linux_amd64/terraform-provider-netbox_v0.1.0 # unpacked
```

Point Terraform at it explicitly:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "/opt/terraform/providers"
    include = ["elliot/netbox"]
  }
  direct {
    exclude = ["elliot/netbox"]
  }
}
```

To build a bundle for machines without internet access, run this once on a connected machine that can
reach the network mirror (or has the provider installed through the script):

```sh
terraform providers mirror -platform=linux_amd64 -platform=darwin_arm64 -platform=darwin_amd64 /opt/terraform/providers
```

`terraform providers mirror` writes the packed layout for every platform requested, ready to copy to the
offline hosts.

## Lock files on mixed platforms

Mirrors provide only the `h1:` hash of the platform that was installed (the packed layout also yields
`zh:` hashes). A `.terraform.lock.hcl` created on macOS therefore fails `terraform init` on a Linux CI
runner with "does not match any of the checksums recorded in the dependency lock file". Record every
platform your team uses once and commit the lock file:

```sh
# network mirror
terraform providers lock -net-mirror=https://elliot.github.io/terraform-provider-netbox/ \
  -platform=linux_amd64 -platform=darwin_arm64 -platform=darwin_amd64
# filesystem mirror / install script (only the platforms present in the directory)
terraform providers lock -fs-mirror=$HOME/.terraform.d/plugins -platform=linux_amd64 -platform=darwin_arm64
```

`terraform providers lock` does not read `provider_installation` from the CLI configuration: without
`-net-mirror` or `-fs-mirror` it asks the public registry, which does not know this provider.

## Development builds

`make install-local` compiles the provider for the current machine into the implied local mirror in the
unpacked layout as version `0.0.0-dev`; pin `version = "0.0.0-dev"` to use it (pre-release versions are
only selected when named exactly). `dev_overrides` remains the fastest loop while editing code, see
[DEVELOPMENT.md](DEVELOPMENT.md).

## Verifying a release locally

```sh
make dist            # goreleaser snapshot: dist/*.zip, *_SHA256SUMS, *_manifest.json (unsigned)
make mirror          # dist/mirror/registry.terraform.io/elliot/netbox/{index.json,<v>.json,*.zip}
make mirror-check    # serves dist/mirror over local HTTPS, runs terraform init + providers lock,
                     # then installs through scripts/install.sh from file://dist/ (Linux)
```

The same three steps run in CI for every pull request (`package` job in `.github/workflows/test.yml`), so a
tag only publishes what has already been installed at least once.

## Publishing on the Terraform Registry later

Nothing above changes when the provider is eventually published: users switch `provider_installation`
off (or keep it; a mirror takes precedence only for the providers it `include`s). The registry additionally
requires a public repository, a GPG-signed `SHA256SUMS` (`GPG_PRIVATE_KEY` and `PASSPHRASE` secrets make
the release workflow sign automatically) and the `terraform-registry-manifest.json` that is already part
of every release.
