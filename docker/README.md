# Local NetBox 4.7 for acceptance tests

```sh
make docker-up            # docker compose -f docker/docker-compose.yml up -d --wait
open http://localhost:8000  # superuser admin / admin (override with SUPERUSER_NAME / SUPERUSER_PASSWORD)
make docker-down          # stops everything and deletes the volumes
```

The stack is NetBox `v4.7` (netbox-docker image), PostgreSQL 18 and Valkey 8.
The first start runs migrations and can take a few minutes; `--wait` blocks
until the `/login/` health check passes.

## Getting a v2 API token

The provider only accepts **v2 tokens** (`nbt_<key>.<secret>`, sent as
`Authorization: Bearer ...`). The `SUPERUSER_API_TOKEN` created by the
container entrypoint is a legacy v1 token and will be rejected. v2 tokens
require at least one pepper: the compose file sets a test-only
`API_TOKEN_PEPPER_1` (override it with the environment variable of the same
name); without it NetBox answers the provisioning call with a 500.

### Option A: provision through the API (what CI does)

```sh
resp=$(curl -sS -X POST http://localhost:8000/api/users/tokens/provision/ \
  -H 'Content-Type: application/json' -H 'Accept: application/json' \
  -d '{"username":"admin","password":"admin","version":2,"write_enabled":true,"description":"local dev"}')
key=$(echo "$resp" | jq -r .key)       # NOTE: returned WITHOUT the nbt_ prefix
secret=$(echo "$resp" | jq -r .token)  # shown exactly once
export NETBOX_SERVER_URL=http://localhost:8000
export NETBOX_API_TOKEN="nbt_${key#nbt_}.${secret}"
curl -sS -H "Authorization: Bearer $NETBOX_API_TOKEN" "$NETBOX_SERVER_URL/api/status/"
```

The same flow works for the public demo; `make demo-token` wraps it and
creates a throw-away demo account first (see `scripts/demo-token.sh`).

### Option B: from inside the container

```sh
docker compose -f docker/docker-compose.yml exec netbox \
  /opt/netbox/venv/bin/python /opt/netbox/netbox/manage.py nbshell
```

```python
from users.models import Token, User
u = User.objects.get(username="admin")
t = Token(user=u, version=2, write_enabled=True, description="local dev")
t.save()
print(f"nbt_{t.key}.{t.plaintext_secret}")   # visible only right after save()
```

Field names follow NetBox 4.7's `users.models.Token`; the API route in
option A is the stable, documented way and is preferred.

### Option C: the UI

Log in as the superuser, open *Admin > Authentication > API Tokens > Add*,
pick *Version 2*, and copy the full `nbt_...` value from the confirmation
page. It is never shown again.

## Running the acceptance tests

```sh
export NETBOX_SERVER_URL=http://localhost:8000
export NETBOX_API_TOKEN=nbt_...
make testacc                          # everything
make testacc ACC_SHARD=TestAccDcim    # one app
make sweep                            # delete leftover tfacc-* objects
```
