---
name: acc-test
description: Run acceptance tests for named NetBox resources against demo.netbox.dev (pinned NetBox version) or a local Docker NetBox (older versions). Creates and deletes real objects.
disable-model-invocation: true
argument-hint: "<Resource>[,<Resource>...] [netbox-version]"
---

Run the acceptance tests for: $ARGUMENTS

1. **Resolve tests.** For each resource, find its test function and package:
   `grep -rn 'func TestAcc<Resource>_' internal/provider/`. Build a single `-run` regex
   (`'TestAccSite_basic|TestAccPrefix_basic'`) and the list of packages it lives in. If a test is generated with
   `t.Skip`, say so and suggest adding a fixture (see the generator-overrides skill) instead of running it.

2. **Pick the target.** The pinned version is in `spec/VERSION`.
   - No version argument, or it matches the pinned major.minor: use **demo.netbox.dev**. Reuse `.env.demo` if
     present; otherwise run `make demo-token`. Then `set -a; . ./.env.demo; set +a` and
     `export NETBOX_REQUESTS_PER_SECOND=5`. On 401/403 the demo has reset: re-run `make demo-token`.
   - An older version: use the **local stack**.
     `NETBOX_IMAGE_TAG=v<major.minor> make docker-up`, then
     `NETBOX_URL=http://localhost:8000 SKIP_DEMO_LOGIN=1 DEMO_USERNAME=admin DEMO_PASSWORD=admin OUT_FILE=.env.local ./scripts/demo-token.sh`
     and source `.env.local`. Expect the provider's version-mismatch warning; set `NETBOX_SKIP_VERSION_CHECK=true`
     only if it gets in the way.

3. **Terraform binary.** If `checkpoint-api.hashicorp.com` is unreachable (sandboxes), export
   `TF_ACC_TERRAFORM_PATH=$(command -v terraform)` or the test binary fails on a timeout.

4. **Run** only the resolved tests:
   `TF_ACC=1 go test <packages> -run '<regex>' -count=1 -v -timeout 60m`.
   Never run the whole suite unless explicitly asked.

5. **Report** pass/fail per test with the relevant error lines. For failures, say whether it is a provider bug,
   a fixture problem or server-side behaviour (NetBox 500s, demo reset), and propose the fix.

6. **Clean up.** If any test failed mid-apply, offer `make sweep`. First state which server `NETBOX_SERVER_URL`
   points at, because sweep deletes every `tfacc-*` object there. For the local stack, offer `make docker-down`
   when done. Never print the token or the contents of `.env.demo` / `.env.local`.
