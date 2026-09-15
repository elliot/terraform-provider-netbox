#!/usr/bin/env bash
# Provision a NetBox v2 API token on https://demo.netbox.dev (or any NetBox
# with NETBOX_URL) and write it to .env.demo for the acceptance tests.
#
#   ./scripts/demo-token.sh            # random throw-away demo account
#   DEMO_USERNAME=me DEMO_PASSWORD=pw ./scripts/demo-token.sh
#   NETBOX_URL=http://localhost:8000 SKIP_DEMO_LOGIN=1 DEMO_USERNAME=admin DEMO_PASSWORD=admin ./scripts/demo-token.sh
#
# Steps:
#   1. GET  /plugins/demo/login/            -> csrftoken cookie
#   2. POST /plugins/demo/login/            -> creates (or signs in to) the demo account
#   3. POST /api/users/tokens/provision/    -> {"key": "...", "token": "..."} (v2, write enabled)
#   4. GET  /api/status/ with the new token -> sanity check
#
# The token is written only to the output file, never printed.
# The demo database is reset daily, so re-run this when tests start failing with 403.
set -euo pipefail

NETBOX_URL="${NETBOX_URL:-https://demo.netbox.dev}"
NETBOX_URL="${NETBOX_URL%/}"
OUT_FILE="${OUT_FILE:-.env.demo}"
DESCRIPTION="${TOKEN_DESCRIPTION:-terraform-provider-netbox}"
SKIP_DEMO_LOGIN="${SKIP_DEMO_LOGIN:-0}"
CURL_OPTS=(--silent --show-error --fail-with-body --max-time 60)

for bin in curl python3; do
  command -v "$bin" >/dev/null 2>&1 || { echo "error: $bin is required" >&2; exit 1; }
done

json_get() { # json_get <key>  (reads JSON on stdin, prints string value or empty)
  python3 -c 'import json,sys; d=json.load(sys.stdin); v=d.get(sys.argv[1], ""); print(v if isinstance(v,str) else json.dumps(v))' "$1"
}

random_string() { # random_string <len>
  python3 -c 'import secrets,string,sys; a=string.ascii_lowercase+string.digits; print("".join(secrets.choice(a) for _ in range(int(sys.argv[1]))))' "$1"
}

# Reuse credentials from an existing .env.demo unless overridden, so re-runs
# keep the same account (idempotent-ish; the demo DB is wiped daily anyway).
if [[ -z "${DEMO_USERNAME:-}" && -z "${DEMO_PASSWORD:-}" && -f "$OUT_FILE" ]]; then
  # shellcheck disable=SC1090
  DEMO_USERNAME="$(grep -E '^DEMO_USERNAME=' "$OUT_FILE" | head -n1 | cut -d= -f2- | tr -d '"')" || true
  DEMO_PASSWORD="$(grep -E '^DEMO_PASSWORD=' "$OUT_FILE" | head -n1 | cut -d= -f2- | tr -d '"')" || true
fi
DEMO_USERNAME="${DEMO_USERNAME:-tfacc-$(random_string 10)}"
DEMO_PASSWORD="${DEMO_PASSWORD:-$(random_string 24)}"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT
cookies="$workdir/cookies.txt"
provision_url="$NETBOX_URL/api/users/tokens/provision/"
payload="$(python3 -c 'import json,sys; print(json.dumps({"username":sys.argv[1],"password":sys.argv[2],"version":2,"write_enabled":True,"description":sys.argv[3]}))' \
  "$DEMO_USERNAME" "$DEMO_PASSWORD" "$DESCRIPTION")"

provision() { # -> 0 on success, provision.json filled
  local code
  code="$(curl --silent --show-error --max-time 60 -o "$workdir/provision.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' -H 'Accept: application/json' \
    --data "$payload" "$provision_url")" || return 1
  [[ "$code" == "201" || "$code" == "200" ]]
}

demo_login() {
  local login_url="$NETBOX_URL/plugins/demo/login/" csrftoken http_code
  echo "==> Fetching CSRF token from $login_url"
  curl "${CURL_OPTS[@]}" -c "$cookies" -o "$workdir/login.html" "$login_url"
  csrftoken="$(awk '$6 == "csrftoken" {print $7}' "$cookies" | tail -n1)"
  if [[ -z "$csrftoken" ]]; then
    echo "error: no csrftoken cookie returned by $login_url" >&2
    return 1
  fi
  echo "==> Creating demo account '$DEMO_USERNAME'"
  http_code="$(curl --silent --show-error --max-time 60 -b "$cookies" -c "$cookies" \
    -o "$workdir/login_resp.html" -w '%{http_code}' \
    -H "Referer: $login_url" \
    -H "X-CSRFToken: $csrftoken" \
    --data-urlencode "csrfmiddlewaretoken=$csrftoken" \
    --data-urlencode "username=$DEMO_USERNAME" \
    --data-urlencode "password=$DEMO_PASSWORD" \
    "$login_url")"
  case "$http_code" in
    200|302|303) return 0 ;;
    500) echo "error: demo login returned HTTP 500 (the username probably already exists with a different password)" >&2
         return 1 ;;
    *) echo "error: demo login returned HTTP $http_code" >&2
       head -c 600 "$workdir/login_resp.html" >&2; echo >&2
       return 1 ;;
  esac
}

# Existing account (credentials from env or a previous .env.demo)? Try to
# provision directly; the demo login endpoint fails for existing usernames.
echo "==> Provisioning a v2 API token for '$DEMO_USERNAME' at $NETBOX_URL"
if ! provision; then
  if [[ "$SKIP_DEMO_LOGIN" == "1" ]]; then
    echo "error: token provisioning failed and SKIP_DEMO_LOGIN=1:" >&2
    head -c 600 "$workdir/provision.json" >&2 || true; echo >&2
    exit 1
  fi
  demo_login || exit 1
  echo "==> Provisioning a v2 API token"
  if ! provision; then
    echo "error: token provisioning failed after creating the demo account:" >&2
    head -c 600 "$workdir/provision.json" >&2 || true; echo >&2
    exit 1
  fi
fi

key="$(json_get key < "$workdir/provision.json")"
secret="$(json_get token < "$workdir/provision.json")"
if [[ -z "$key" || -z "$secret" ]]; then
  echo "error: unexpected provision response (missing key/token)" >&2
  exit 1
fi
# The API returns the key WITHOUT the nbt_ prefix; the wire format is nbt_<key>.<secret>.
key="${key#nbt_}"
token="nbt_${key}.${secret}"

echo "==> Verifying the token against $NETBOX_URL/api/status/"
if ! curl "${CURL_OPTS[@]}" -o "$workdir/status.json" \
    -H "Authorization: Bearer $token" -H 'Accept: application/json' \
    "$NETBOX_URL/api/status/"; then
  echo "error: /api/status/ rejected the new token" >&2
  exit 1
fi
version="$(json_get netbox-version < "$workdir/status.json")"

umask 077
{
  echo "# generated by scripts/demo-token.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ) - do not commit"
  echo "NETBOX_SERVER_URL=\"$NETBOX_URL\""
  echo "NETBOX_API_TOKEN=\"$token\""
  echo "DEMO_USERNAME=\"$DEMO_USERNAME\""
  echo "DEMO_PASSWORD=\"$DEMO_PASSWORD\""
} > "$OUT_FILE"

echo "==> OK: NetBox ${version:-unknown} at $NETBOX_URL, token for '$DEMO_USERNAME' written to $OUT_FILE"
case "$OUT_FILE" in
  /*) echo "    Load it with:  set -a; . \"$OUT_FILE\"; set +a" ;;
  *)  echo "    Load it with:  set -a; . ./$OUT_FILE; set +a" ;;
esac
