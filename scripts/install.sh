#!/bin/sh
# Install a released terraform-provider-netbox into Terraform's implied local
# mirror directory so `source = "elliot/netbox"` works without the Terraform
# Registry and without any CLI configuration.
#
#   curl -fsSL https://raw.githubusercontent.com/elliot/terraform-provider-netbox/main/scripts/install.sh | sh
#   VERSION=0.1.0 sh scripts/install.sh
#
# Environment overrides:
#   VERSION            release to install (default: latest GitHub release)
#   REPO               GitHub repository (default: elliot/terraform-provider-netbox)
#   RELEASE_BASE_URL   where <name>_<version>_<os>_<arch>.zip and *_SHA256SUMS live
#                      (default: https://github.com/$REPO/releases/download/v$VERSION/;
#                       file:///path/to/dist/ works for local GoReleaser output)
#   TF_PLUGIN_DIR      plugin directory (default: $HOME/.terraform.d/plugins)
#   OS / ARCH          override detection (linux|darwin|freebsd, amd64|arm64)
#
# The archive is verified against the release's SHA256SUMS and unpacked into
# Terraform's filesystem mirror layout:
#   $TF_PLUGIN_DIR/registry.terraform.io/elliot/netbox/<version>/<os>_<arch>/terraform-provider-netbox_v<version>
# (The "packed" layout, a zip in the provider directory, is not used because
# Terraform only recognises it for plain x.y.z versions, not pre-releases.)
set -eu

REPO=${REPO:-elliot/terraform-provider-netbox}
NAMESPACE=${NAMESPACE:-elliot}
NAME=${NAME:-netbox}
HOSTNAME_=${REGISTRY_HOSTNAME:-registry.terraform.io}
TF_PLUGIN_DIR=${TF_PLUGIN_DIR:-$HOME/.terraform.d/plugins}
PROJECT="terraform-provider-$NAME"

log() { printf '%s\n' "$*" >&2; }
die() { log "error: $*"; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "$1 is required"; }

need curl

# --- platform -------------------------------------------------------------
if [ -z "${OS:-}" ]; then
  case "$(uname -s)" in
    Linux)   OS=linux ;;
    Darwin)  OS=darwin ;;
    FreeBSD) OS=freebsd ;;
    *) die "unsupported operating system: $(uname -s) (set OS=linux|darwin|freebsd)" ;;
  esac
fi
if [ -z "${ARCH:-}" ]; then
  case "$(uname -m)" in
    x86_64|amd64)  ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
    *) die "unsupported architecture: $(uname -m) (set ARCH=amd64|arm64)" ;;
  esac
fi

# --- version --------------------------------------------------------------
if [ -z "${VERSION:-}" ]; then
  log "Resolving latest release of $REPO ..."
  VERSION=$(curl -fsSL -H 'Accept: application/vnd.github+json' \
    "https://api.github.com/repos/$REPO/releases/latest" |
    sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)
  [ -n "$VERSION" ] || die "could not determine the latest release; set VERSION=x.y.z"
fi
VERSION=${VERSION#v}
BASE=${RELEASE_BASE_URL:-https://github.com/$REPO/releases/download/v$VERSION/}
case "$BASE" in */) ;; *) BASE="$BASE/" ;; esac

ZIP="${PROJECT}_${VERSION}_${OS}_${ARCH}.zip"
SUMS="${PROJECT}_${VERSION}_SHA256SUMS"

# --- download + verify ----------------------------------------------------
TMP=$(mktemp -d 2>/dev/null || mktemp -d -t tfnetbox)
trap 'rm -rf "$TMP"' EXIT INT TERM

log "Downloading $BASE$ZIP ..."
curl -fsSL -o "$TMP/$ZIP" "$BASE$ZIP" || die "download failed (is $VERSION released for ${OS}_${ARCH}?)"
curl -fsSL -o "$TMP/$SUMS" "$BASE$SUMS" || die "download of $SUMS failed"

expected=$(awk -v f="$ZIP" '{ sub(/^\*/, "", $2) } $2 == f { print $1 }' "$TMP/$SUMS")
[ -n "$expected" ] || die "$ZIP is not listed in $SUMS"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$TMP/$ZIP" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$TMP/$ZIP" | awk '{print $1}')
elif command -v openssl >/dev/null 2>&1; then
  actual=$(openssl dgst -sha256 "$TMP/$ZIP" | awk '{print $NF}')
else
  die "need sha256sum, shasum or openssl to verify the download"
fi
[ "$expected" = "$actual" ] || die "checksum mismatch for $ZIP: expected $expected, got $actual"
log "Checksum verified."

# --- install --------------------------------------------------------------
DEST="$TF_PLUGIN_DIR/$HOSTNAME_/$NAMESPACE/$NAME/$VERSION/${OS}_${ARCH}"
mkdir -p "$TMP/unpack"
if command -v unzip >/dev/null 2>&1; then
  unzip -q -o "$TMP/$ZIP" -d "$TMP/unpack"
elif command -v python3 >/dev/null 2>&1; then
  python3 -m zipfile -e "$TMP/$ZIP" "$TMP/unpack"
elif command -v bsdtar >/dev/null 2>&1; then
  bsdtar -xf "$TMP/$ZIP" -C "$TMP/unpack"
else
  die "need unzip, python3 or bsdtar to extract $ZIP"
fi
BIN=$(find "$TMP/unpack" -maxdepth 1 -type f -name "${PROJECT}_v*" | head -n 1)
[ -n "$BIN" ] || die "$ZIP does not contain a ${PROJECT}_v* binary"
rm -rf "$DEST"
mkdir -p "$DEST"
cp "$BIN" "$DEST/"
chmod 0755 "$DEST/$(basename "$BIN")"
for extra in LICENSE README.md CHANGELOG.md; do
  [ -f "$TMP/unpack/$extra" ] && cp "$TMP/unpack/$extra" "$DEST/" || true
done
if [ "$OS" = darwin ] && command -v xattr >/dev/null 2>&1; then
  xattr -dr com.apple.quarantine "$DEST" 2>/dev/null || true
fi

log ""
log "Installed $PROJECT $VERSION (${OS}_${ARCH}) to"
log "  $DEST/$(basename "$BIN")"
log ""
log "Terraform picks it up automatically (implied local mirror). Reference it with:"
log ""
log "  terraform {"
log "    required_providers {"
log "      $NAME = {"
log "        source  = \"$NAMESPACE/$NAME\""
log "        version = \"$VERSION\""
log "      }"
log "    }"
log "  }"
log ""
log "Teams on mixed platforms should record every platform's hash in the lock file:"
log "  terraform providers lock -fs-mirror=\"$TF_PLUGIN_DIR\" -platform=linux_amd64 -platform=darwin_arm64 -platform=darwin_amd64"
