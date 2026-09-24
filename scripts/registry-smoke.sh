#!/usr/bin/env bash
# Smoke-test the provider as published on the Terraform Registry against a live
# NetBox (https://demo.netbox.dev by default), using
# examples/scenarios/registry-demo.
#
#   ./scripts/registry-smoke.sh                  # latest release matching ~> 0.1
#   VERSION=0.1.1 ./scripts/registry-smoke.sh    # pin one release
#   KEEP=1 ./scripts/registry-smoke.sh           # skip destroy (inspect in the UI)
#   EXPECTED_KEY_ID=CAA7EF848084A7F1 ./scripts/registry-smoke.sh  # pin the signing key
#
# Credentials come from NETBOX_SERVER_URL / NETBOX_API_TOKEN, then from
# .env.demo; when neither is set, scripts/demo-token.sh provisions a demo
# account first.
#
# Steps (each must succeed):
#   1. init from registry.terraform.io only (no dev_overrides, no local mirrors)
#      and check that the checksum file signature was verified
#   2. apply                        -> create everything, check blocks pass
#   3. plan -detailed-exitcode      -> no changes (read-back is stable)
#   4. state rm + import (tag)      -> import by ID, then no changes
#   5. apply with a new description -> in-place update, then no changes
#   6. destroy                      -> everything removed (also on failure)
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scenario_dir="$repo_root/examples/scenarios/registry-demo"
TERRAFORM="${TERRAFORM:-terraform}"
VERSION="${VERSION:-}"
KEEP="${KEEP:-0}"

command -v "$TERRAFORM" >/dev/null 2>&1 || { echo "error: terraform not found (set TERRAFORM=/path/to/terraform)" >&2; exit 1; }

if [[ -z "${NETBOX_API_TOKEN:-}" ]]; then
  if [[ ! -f "$repo_root/.env.demo" ]]; then
    (cd "$repo_root" && ./scripts/demo-token.sh)
  fi
  set -a
  # shellcheck disable=SC1091
  . "$repo_root/.env.demo"
  set +a
fi
: "${NETBOX_SERVER_URL:?NETBOX_SERVER_URL is not set}"
export NETBOX_SERVER_URL NETBOX_API_TOKEN

rand() { python3 -c 'import secrets,sys; print(secrets.randbelow(int(sys.argv[1])))' "$1"; }
suffix="$(python3 -c 'import secrets,string; print("".join(secrets.choice(string.ascii_lowercase+string.digits) for _ in range(6)))')"
prefix="${PREFIX:-tfacc-registry-$suffix}"
# A random /24 of 198.18.0.0/15 so concurrent runs do not collide on a
# server that enforces globally unique prefixes.
network="${NETWORK:-198.$((18 + $(rand 2))).$(rand 256).0/24}"

workdir="$(mktemp -d)"
applied=0
cleanup() {
  local rc=$?
  if [[ "$applied" == "1" && "$KEEP" != "1" && $rc -ne 0 ]]; then
    echo "==> Failure: destroying what was created"
    tf destroy -auto-approve >/dev/null 2>&1 || echo "warning: destroy failed, run 'make sweep' to remove $prefix-* objects" >&2
  fi
  if [[ "$KEEP" == "1" ]]; then
    echo "==> KEEP=1: working directory left at $workdir"
  else
    rm -rf "$workdir"
  fi
  exit $rc
}
trap cleanup EXIT

cp "$scenario_dir/main.tf" "$workdir/"
if [[ -n "$VERSION" ]]; then
  sed -i.bak -E "s/(version[[:space:]]*=[[:space:]]*)\"~> 0\.1\"/\1\"= $VERSION\"/" "$workdir/main.tf"
  rm -f "$workdir/main.tf.bak"
  grep -q "\"= $VERSION\"" "$workdir/main.tf" || { echo "error: could not pin version $VERSION in the scenario" >&2; exit 1; }
fi

# An explicit provider_installation block disables dev_overrides from
# ~/.terraformrc and the implied local mirrors (~/.terraform.d/plugins,
# where `make install-local` and scripts/install.sh put builds), so the
# provider can only come from the registry.
cat > "$workdir/cli.tfrc" <<'EOF'
provider_installation {
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$workdir/cli.tfrc"
export TF_IN_AUTOMATION=1 TF_INPUT=0
export TF_VAR_prefix="$prefix" TF_VAR_network="$network"
unset TF_PLUGIN_CACHE_DIR

tf() { local cmd="$1"; shift; "$TERRAFORM" -chdir="$workdir" "$cmd" -no-color "$@"; }

expect_no_changes() { # step=<name> expect_no_changes [plan args...]
  local rc=0
  tf plan -detailed-exitcode "$@" > "$workdir/plan.log" 2>&1 || rc=$?
  if [[ $rc -ne 0 ]]; then
    echo "error: plan after '$step' is not empty (exit $rc):" >&2
    cat "$workdir/plan.log" >&2
    exit 1
  fi
  echo "    plan after '$step': no changes"
}

echo "==> NetBox: $NETBOX_SERVER_URL, prefix: $prefix, network: $network"

echo "==> [1/6] terraform init (registry only)"
tf init > "$workdir/init.log" 2>&1 || { cat "$workdir/init.log" >&2; exit 1; }
installed="$(grep -E 'Installed elliot/netbox v' "$workdir/init.log" || true)"
if [[ -z "$installed" ]]; then
  echo "error: elliot/netbox was not installed from the registry:" >&2
  cat "$workdir/init.log" >&2
  exit 1
fi
echo "    ${installed#- }"
# Terraform prints "(self-signed, key ID ...)" or "(signed by ...)" only after
# verifying the GPG signature of the release's SHA256SUMS against the key
# registered for the namespace. EXPECTED_KEY_ID pins that key.
key_id="$(sed -nE 's/.*Installed elliot\/netbox v.*key ID ([0-9A-F]+)\).*/\1/p' "$workdir/init.log")"
if [[ -z "$key_id" ]]; then
  echo "error: the registry download was not reported as signed" >&2
  exit 1
fi
if [[ -n "${EXPECTED_KEY_ID:-}" && "${EXPECTED_KEY_ID^^}" != *"$key_id" ]]; then
  echo "error: signed with key $key_id, expected $EXPECTED_KEY_ID" >&2
  exit 1
fi

echo "==> [2/6] terraform apply"
applied=1
tf apply -auto-approve > "$workdir/apply.log" 2>&1 || { cat "$workdir/apply.log" >&2; exit 1; }
grep -E '^Apply complete' "$workdir/apply.log" | sed 's/^/    /'
if grep -q 'Check block assertion failed' "$workdir/apply.log"; then
  echo "error: a check block failed:" >&2
  grep -A6 'Check block assertion failed' "$workdir/apply.log" >&2
  exit 1
fi

echo "==> [3/6] terraform plan (expect no changes)"
step=apply expect_no_changes

echo "==> [4/6] import round-trip"
tag_id="$(tf output -raw tag_id)"
"$TERRAFORM" -chdir="$workdir" state rm netbox_tag.smoke >/dev/null
tf import netbox_tag.smoke "$tag_id" > "$workdir/import.log" 2>&1 || { cat "$workdir/import.log" >&2; exit 1; }
echo "    imported netbox_tag.smoke from ID $tag_id"
step=import expect_no_changes

echo "==> [5/6] in-place update"
tf apply -auto-approve -var "description=Updated by the registry smoke test" > "$workdir/update.log" 2>&1 \
  || { cat "$workdir/update.log" >&2; exit 1; }
grep -E '^Apply complete' "$workdir/update.log" | sed 's/^/    /'
if ! grep -q '1 changed' "$workdir/update.log"; then
  echo "error: expected exactly one in-place change:" >&2
  cat "$workdir/update.log" >&2
  exit 1
fi
step=update expect_no_changes -var "description=Updated by the registry smoke test"

tf output | sed 's/^/    /'

if [[ "$KEEP" == "1" ]]; then
  echo "==> [6/6] skipped destroy (KEEP=1); run 'terraform -chdir=$workdir destroy' when done"
  applied=0
else
  echo "==> [6/6] terraform destroy"
  tf destroy -auto-approve -var "description=Updated by the registry smoke test" > "$workdir/destroy.log" 2>&1 \
    || { cat "$workdir/destroy.log" >&2; exit 1; }
  applied=0
  grep -E '^Destroy complete' "$workdir/destroy.log" | sed 's/^/    /'
fi

echo "==> OK: registry release works against $NETBOX_SERVER_URL"
