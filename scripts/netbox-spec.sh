#!/usr/bin/env bash
# Extract the OpenAPI document of a NetBox release from its official image,
# without starting the web server or needing an API token.
#
#   scripts/netbox-spec.sh 4.7.2                      # -> spec/netbox-4.7.2.openapi.json
#   scripts/netbox-spec.sh 4.7.2 /tmp/netbox.json     # explicit output path
#
# Starts PostgreSQL and Valkey from docker/docker-compose.yml under a separate
# project name (so a running `make docker-up` stack is left alone), migrates an
# empty database (the schema generator reads django_content_type) and runs
# `manage.py spectacular` in the netboxcommunity/netbox:v<VERSION> image. The
# result is byte-identical to GET /api/schema/?format=json on that release;
# only info.version differs ("4.7.2-Docker-5.1.1" offline), so it is rewritten
# to the "4.7.2 (4.7)" form the HTTP endpoint reports.
#
# Needs docker with the compose plugin and python3.
set -euo pipefail

version="${1:-}"
version="${version#v}"
if ! [[ "$version" =~ ^([0-9]+)\.([0-9]+)\.[0-9]+$ ]]; then
  echo "usage: $0 <netbox-version, e.g. 4.7.2> [output]" >&2
  exit 2
fi
major_minor="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"

cd "$(git rev-parse --show-toplevel)"
out="${2:-spec/netbox-${version}.openapi.json}"

project="tfnetbox-spec-$$"
compose=(docker compose -p "$project" -f docker/docker-compose.yml)
workdir="$(mktemp -d)"
# The container does not run as the caller; let it write the document.
chmod 0777 "$workdir"
cleanup() {
  "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$workdir"
}
trap cleanup EXIT

export NETBOX_IMAGE_TAG="v${version}"
echo "==> Starting PostgreSQL and Valkey (project $project)" >&2
"${compose[@]}" up -d --wait --wait-timeout 300 postgres valkey >&2

echo "==> Migrating and generating the schema with netboxcommunity/netbox:v${version}" >&2
# The image prints its configuration banner on stdout, so the document is
# written to a file rather than captured.
"${compose[@]}" run --rm --no-deps -T -v "$workdir:/out" --entrypoint "" netbox sh -euc '
  cd /opt/netbox/netbox
  /opt/netbox/venv/bin/python manage.py migrate --no-input >/dev/null
  /opt/netbox/venv/bin/python manage.py spectacular --format openapi-json --file /out/openapi.json
' >&2

reported="$(python3 -I -c 'import json, sys; print(json.load(open(sys.argv[1]))["info"]["version"])' "$workdir/openapi.json")"
case "$reported" in
  "$version" | "$version"-* | "$version "*) ;;
  *)
    echo "error: the image reports NetBox '$reported', expected $version" >&2
    exit 1
    ;;
esac

# Rewrite only the first "version" key, which is info.version (line 5).
sed "0,/\"version\": \"[^\"]*\"/s//\"version\": \"${version} (${major_minor})\"/" "$workdir/openapi.json" > "$workdir/fixed.json"
python3 -I -c '
import json, sys
doc = json.load(open(sys.argv[1]))
want = sys.argv[2]
if doc["info"]["version"] != want:
    sys.exit("error: info.version is %r after the rewrite, expected %r" % (doc["info"]["version"], want))
' "$workdir/fixed.json" "${version} (${major_minor})"

mkdir -p "$(dirname "$out")"
mv "$workdir/fixed.json" "$out"
echo "==> Wrote $out (NetBox $reported, $(wc -c < "$out") bytes)" >&2
