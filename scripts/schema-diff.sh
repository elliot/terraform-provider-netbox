#!/usr/bin/env bash
# Compares the provider schema of the working tree with a released version
# (`make schema-diff`, and the Schema diff workflow on pull requests):
#
#   1. builds the provider and dumps its schema through a dev_overrides CLI
#      configuration,
#   2. installs the base release from the registry with `terraform init` and
#      dumps its schema,
#   3. runs tools/schema-diff on the two documents.
#
# Usage: scripts/schema-diff.sh [BASE_VERSION]   (default: latest release)
#
# Environment: TERRAFORM (terraform), PROVIDER (elliot/netbox),
# ALLOW_BREAKING (non-empty passes -allow-breaking), SCHEMA_DIFF_OUT (also write
# the report there), GITHUB_STEP_SUMMARY (the report is appended when set),
# KEEP_WORK (non-empty keeps the temporary directory). Exits with the status of
# tools/schema-diff: 1 on breaking changes not allowed, 2 on errors.
set -euo pipefail

TERRAFORM=${TERRAFORM:-terraform}
PROVIDER=${PROVIDER:-elliot/netbox}
NAME=${PROVIDER#*/}
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

export CHECKPOINT_DISABLE=1 TF_IN_AUTOMATION=1

base=${1:-}
if [ -z "$base" ]; then
  base=$(go run ./tools/schema-diff latest "$PROVIDER")
fi
base=${base#v}

work=$(mktemp -d)
cleanup() { [ -n "${KEEP_WORK:-}" ] && echo "work dir: $work" >&2 || rm -rf "$work"; }
trap cleanup EXIT

echo "==> building the working tree provider" >&2
mkdir -p "$work/bin" "$work/current" "$work/base"
go build -o "$work/bin/terraform-provider-$NAME" .

cat >"$work/dev.tfrc" <<EOF
provider_installation {
  dev_overrides {
    "$PROVIDER" = "$work/bin"
  }
  direct {}
}
EOF
# The base must come from the registry even if the user's CLI configuration
# overrides the provider.
: >"$work/empty.tfrc"

write_config() {
  local version=""
  [ -z "$2" ] || version="version = \"= $2\""
  cat >"$1/main.tf" <<EOF
terraform {
  required_providers {
    $NAME = {
      source = "$PROVIDER"
      $version
    }
  }
}
EOF
}

echo "==> dumping the working tree schema" >&2
write_config "$work/current" ""
# With dev_overrides no `terraform init` is needed (nor allowed to matter).
TF_CLI_CONFIG_FILE="$work/dev.tfrc" "$TERRAFORM" -chdir="$work/current" providers schema -json >"$work/current.json"

echo "==> installing $PROVIDER $base from the registry and dumping its schema" >&2
write_config "$work/base" "$base"
TF_CLI_CONFIG_FILE="$work/empty.tfrc" "$TERRAFORM" -chdir="$work/base" init -input=false -no-color >"$work/init.log" 2>&1 ||
  { cat "$work/init.log" >&2; exit 2; }
TF_CLI_CONFIG_FILE="$work/empty.tfrc" "$TERRAFORM" -chdir="$work/base" providers schema -json >"$work/base.json"

args=(-old "$work/base.json" -new "$work/current.json" -old-label "v$base" -new-label "${SCHEMA_DIFF_NEW_LABEL:-working tree}"
  -provider "registry.terraform.io/$PROVIDER" -out "$work/report.md")
[ -n "${ALLOW_BREAKING:-}" ] && args+=(-allow-breaking)

status=0
go run ./tools/schema-diff "${args[@]}" || status=$?
[ -f "$work/report.md" ] || exit "$status"
cat "$work/report.md"
[ -n "${SCHEMA_DIFF_OUT:-}" ] && cp "$work/report.md" "$SCHEMA_DIFF_OUT"
[ -n "${GITHUB_STEP_SUMMARY:-}" ] && cat "$work/report.md" >>"$GITHUB_STEP_SUMMARY"
exit "$status"
