#!/usr/bin/env bash
# End-to-end check of the packaging output produced by `make dist mirror`:
#
#   1. serves dist/mirror over HTTPS with a throw-away self-signed certificate
#      and runs `terraform init` against it as a provider network mirror,
#   2. runs `terraform providers lock` for the Linux/macOS platforms,
#   3. installs the same release through scripts/install.sh (file:// base URL)
#      into a temporary plugin directory and runs `terraform init` with no CLI
#      configuration at all (implied local mirror).
#
# Requirements: terraform, python3, openssl, curl. Linux only for step 1: Go
# programs on macOS ignore SSL_CERT_FILE, so the self-signed certificate is not
# trusted there (steps 2-3 still work).
#
# Environment: MIRROR_DIR (dist/mirror), DIST_DIR (dist), TERRAFORM (terraform),
# PORT (8443), NAMESPACE (elliot), NAME (netbox).
set -euo pipefail

MIRROR_DIR=${MIRROR_DIR:-dist/mirror}
DIST_DIR=${DIST_DIR:-dist}
TERRAFORM=${TERRAFORM:-terraform}
PORT=${PORT:-8443}
NAMESPACE=${NAMESPACE:-elliot}
NAME=${NAME:-netbox}
HOST=127.0.0.1

MIRROR_DIR=$(cd "$MIRROR_DIR" && pwd)
DIST_DIR=$(cd "$DIST_DIR" && pwd)
ROOT=$(cd "$(dirname "$0")/.." && pwd)

index="$MIRROR_DIR/registry.terraform.io/$NAMESPACE/$NAME/index.json"
[ -f "$index" ] || { echo "missing $index; run 'make dist mirror' first" >&2; exit 1; }
version=$(python3 -c 'import json,sys; print(sorted(json.load(open(sys.argv[1]))["versions"])[-1])' "$index")
platform="$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')"
echo "==> version $version, host platform $platform"

work=$(mktemp -d)
cleanup() {
  [ -n "${server_pid:-}" ] && kill "$server_pid" 2>/dev/null || true
  rm -rf "$work"
}
trap cleanup EXIT

write_config() { # $1 = dir
  mkdir -p "$1"
  cat >"$1/main.tf" <<EOF
terraform {
  required_providers {
    $NAME = {
      source  = "$NAMESPACE/$NAME"
      version = "$version"
    }
  }
}
EOF
}

# --- 1. network mirror ------------------------------------------------------
if [ "$(uname -s)" = Linux ]; then
  echo "==> serving $MIRROR_DIR on https://$HOST:$PORT/"
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -days 2 \
    -keyout "$work/key.pem" -out "$work/cert.pem" -subj "/CN=$HOST" \
    -addext "subjectAltName=IP:$HOST,DNS:localhost" >/dev/null 2>&1
  python3 - "$MIRROR_DIR" "$HOST" "$PORT" "$work/cert.pem" "$work/key.pem" <<'EOF' &
import http.server, os, ssl, sys
root, host, port, cert, key = sys.argv[1], sys.argv[2], int(sys.argv[3]), sys.argv[4], sys.argv[5]
os.chdir(root)
class H(http.server.SimpleHTTPRequestHandler):
    def log_message(self, *a): pass
srv = http.server.ThreadingHTTPServer((host, port), H)
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
ctx.load_cert_chain(cert, key)
srv.socket = ctx.wrap_socket(srv.socket, server_side=True)
srv.serve_forever()
EOF
  server_pid=$!
  for _ in $(seq 1 50); do
    curl -fsS --cacert "$work/cert.pem" "https://$HOST:$PORT/" >/dev/null 2>&1 && break
    sleep 0.2
  done
  curl -fsS --cacert "$work/cert.pem" "https://$HOST:$PORT/registry.terraform.io/$NAMESPACE/$NAME/index.json" >/dev/null

  cat >"$work/netmirror.tfrc" <<EOF
provider_installation {
  network_mirror {
    url = "https://$HOST:$PORT/"
  }
}
EOF
  write_config "$work/net"
  (
    cd "$work/net"
    export TF_CLI_CONFIG_FILE="$work/netmirror.tfrc" SSL_CERT_FILE="$work/cert.pem" TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1
    echo "==> terraform init (network mirror)"
    "$TERRAFORM" init -input=false -no-color | grep -E "Installing|Installed|Terraform has been" || true
    grep -q "h1:" .terraform.lock.hcl || { echo "lock file has no h1: hash" >&2; cat .terraform.lock.hcl; exit 1; }
    echo "==> terraform providers lock -platform=linux_amd64 -platform=darwin_arm64 -platform=darwin_amd64"
    "$TERRAFORM" providers lock -no-color -platform=linux_amd64 -platform=darwin_arm64 -platform=darwin_amd64 | tail -n 3
    n=$(grep -c "h1:" .terraform.lock.hcl)
    [ "$n" -ge 3 ] || { echo "expected at least 3 h1: hashes, got $n" >&2; cat .terraform.lock.hcl; exit 1; }
    "$TERRAFORM" providers -no-color | grep -q "$NAMESPACE/$NAME" || { echo "provider not listed" >&2; exit 1; }
  )
  kill "$server_pid" 2>/dev/null || true
  server_pid=
else
  echo "==> skipping network mirror step on $(uname -s)"
fi

# --- 2. install.sh into an implied local mirror -----------------------------
echo "==> scripts/install.sh from file://$DIST_DIR/"
export TF_PLUGIN_DIR="$work/plugins"
VERSION="$version" RELEASE_BASE_URL="file://$DIST_DIR/" sh "$ROOT/scripts/install.sh" 2>&1 | grep -E "Checksum|Installed|error" || true
zip="$TF_PLUGIN_DIR/registry.terraform.io/$NAMESPACE/$NAME/terraform-provider-${NAME}_${version}_${platform}.zip"
[ -f "$zip" ] || { echo "install.sh did not create $zip" >&2; exit 1; }

# Terraform's implied local mirror is ~/.terraform.d/plugins; point HOME at a
# scratch dir so the test does not touch the real one.
mkdir -p "$work/home/.terraform.d"
ln -s "$TF_PLUGIN_DIR" "$work/home/.terraform.d/plugins"
write_config "$work/fs"
(
  cd "$work/fs"
  export HOME="$work/home" TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1
  unset TF_CLI_CONFIG_FILE
  echo "==> terraform init (implied local mirror, no CLI config)"
  "$TERRAFORM" init -input=false -no-color | grep -E "Installing|Installed|Terraform has been" || true
  grep -q "h1:" .terraform.lock.hcl || { echo "lock file has no h1: hash" >&2; cat .terraform.lock.hcl; exit 1; }
  echo "==> terraform providers lock -fs-mirror (all platforms present in the plugin dir)"
  "$TERRAFORM" providers lock -no-color -fs-mirror="$TF_PLUGIN_DIR" -platform="$platform" | tail -n 2
)
echo "==> packaging smoke test passed"
