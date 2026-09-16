#!/usr/bin/env bash
# Regenerate the NetBox API client in ./netbox from the pinned OpenAPI document.
#
#   make client-gen              # uses spec/netbox-<VERSION>.openapi.json
#   OPENAPI_GENERATOR_JAR=...    # optional: path to openapi-generator-cli jar
#
# Requirements: java 17+, python3, gofmt. The openapi-generator jar (7.11.0, the
# version go-netbox uses) is downloaded from Maven Central into .cache/ if absent
# and is checked against the pinned OAG_SHA256 below before every run.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

OAG_VERSION="7.11.0"
# SHA-256 of openapi-generator-cli-${OAG_VERSION}.jar on Maven Central. Central
# publishes only .sha1/.md5 next to this artifact (there is no .jar.sha256), so
# this digest was taken from the jar whose published checksums match:
#   sha1 9261333ecbfd8738956b09be0beff611b47fbaff
#   md5  4931bca886d30823b3ba2c7f36607925
# Update both OAG_VERSION and OAG_SHA256 together when bumping the generator.
OAG_SHA256="113c25df5a781d5a1fc2b883f12fe8f263db285ab12e15854d5b15306e1bf7fc"
NETBOX_VERSION="$(cat spec/VERSION)"
SPEC="spec/netbox-${NETBOX_VERSION}.openapi.json"
CACHE="${ROOT}/.cache"
JAR="${OPENAPI_GENERATOR_JAR:-${CACHE}/openapi-generator-cli-${OAG_VERSION}.jar}"
OUT="${ROOT}/netbox"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

verify_jar() {
  # Checks "$1" against $OAG_SHA256; prints nothing on success.
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s  %s\n' "$OAG_SHA256" "$1" | sha256sum -c --status -
  elif command -v shasum >/dev/null 2>&1; then
    printf '%s  %s\n' "$OAG_SHA256" "$1" | shasum -a 256 -c --status -
  else
    echo "!! neither sha256sum nor shasum found; cannot verify ${1}" >&2
    return 1
  fi
}

if [[ ! -f "$JAR" ]]; then
  mkdir -p "$(dirname "$JAR")"
  echo ">> downloading openapi-generator-cli ${OAG_VERSION}"
  curl -sSL -o "$JAR" "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/${OAG_VERSION}/openapi-generator-cli-${OAG_VERSION}.jar"
fi

echo ">> verifying ${JAR}"
if ! verify_jar "$JAR"; then
  echo "!! checksum mismatch for ${JAR}" >&2
  echo "!!   expected sha256: ${OAG_SHA256}" >&2
  echo "!!   actual   sha256: $( (sha256sum "$JAR" 2>/dev/null || shasum -a 256 "$JAR" 2>/dev/null) | awk '{print $1}')" >&2
  if [[ -n "${OPENAPI_GENERATOR_JAR:-}" ]]; then
    echo "!! OPENAPI_GENERATOR_JAR points at a jar that is not openapi-generator-cli ${OAG_VERSION};" >&2
    echo "!! unset it to let this script download the pinned version, or update OAG_SHA256." >&2
  else
    rm -f "$JAR"
    echo "!! deleted the cached jar; re-run 'make client-gen' to download it again." >&2
  fi
  exit 1
fi

echo ">> normalising spec ${SPEC}"
python3 tools/client-gen/fix_spec.py "$SPEC" "$WORK/openapi.json"

echo ">> generating into ${WORK}/out"
JAVA_OPTS="${JAVA_OPTS:--Xmx4g} -DmaxYamlCodePoints=99999999" \
java -jar "$JAR" generate \
  --config tools/client-gen/config.yaml \
  --input-spec "$WORK/openapi.json" \
  --output "$WORK/out" \
  --inline-schema-options RESOLVE_INLINE_ENUMS=true \
  --http-user-agent "terraform-provider-netbox/${NETBOX_VERSION}" \
  --global-property apis,models,supportingFiles=client.go:configuration.go:response.go:utils.go \
  >"$WORK/generate.log" 2>&1 || { tail -50 "$WORK/generate.log"; exit 1; }

echo ">> installing generated files into ${OUT}"
mkdir -p "$OUT"
# Remove previously generated files (keep hand-written *_extra.go and tests).
find "$OUT" -maxdepth 1 -name '*.go' ! -name '*_extra.go' ! -name '*_extra_test.go' -delete
cp "$WORK"/out/*.go "$OUT"/

echo ">> post-processing"
python3 tools/client-gen/post_process.py "$OUT" "$NETBOX_VERSION"
GOIMPORTS_VERSION="v0.37.0"
GOIMPORTS="$(go env GOPATH)/bin/goimports"
if [[ ! -x "$GOIMPORTS" ]]; then
  echo ">> installing goimports ${GOIMPORTS_VERSION}"
  (cd /tmp && GOFLAGS=-mod=mod go install "golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}")
fi
"$GOIMPORTS" -w "$OUT"/*.go
gofmt -w "$OUT"/*.go

echo ">> building"
go build ./netbox/
echo ">> done: $(ls "$OUT"/*.go | wc -l) files"
