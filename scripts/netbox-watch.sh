#!/usr/bin/env bash
# Logic of .github/workflows/netbox-watch.yml, kept here so it can be run and
# tested outside Actions. Both subcommands talk to GitHub through `gh`
# (GH_TOKEN) and write step outputs to $GITHUB_OUTPUT when it is set.
#
#   scripts/netbox-watch.sh detect
#     Decides what the run does for the newest NetBox release (or VERSION):
#       mode=analyse  a release newer than spec/VERSION has an image and no issue yet
#                     (or FORCE=true)
#       mode=demo     its issue is open and still waits for demo.netbox.dev
#                     (label demo-pending: the demo ran an older release, was
#                     down, or the smoke tests did not run)
#       mode=none     nothing to do
#     Env: REPOSITORY (owner/repo), VERSION (optional), FORCE (true/false).
#     Outputs: mode, version, pinned, release_tag, issue.
#
#   scripts/netbox-watch.sh report
#     Turns the job results into a verdict and creates or updates the
#     "NetBox vX.Y.Z released" issue (analyse) or comments on it (demo).
#     Env: REPOSITORY, MODE, VERSION, PINNED, RELEASE_TAG, ISSUE, RUN_URL,
#     ASSIGNEE, ANALYSE_DIR (verdict.json, surface.md, trial.md),
#     ANALYSE_RESULT, TRIAL_OUTCOME, COMPAT_RESULT, DEMO_STATUS, DEMO_VERSION.
set -euo pipefail

LABEL="netbox-release"
PENDING_LABEL="demo-pending"
UPSTREAM="netbox-community/netbox"
IMAGE="docker.io/netboxcommunity/netbox"

cd "$(git rev-parse --show-toplevel)"

output() { # output <key> <value>
  echo "$1=$2"
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "$1=$2" >>"$GITHUB_OUTPUT"
  fi
}

semver_re='^[0-9]+\.[0-9]+\.[0-9]+$'

# newer <a> <b>: true when version a sorts after b.
newer() {
  [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -n1)" = "$1" ]
}

major_minor() { echo "$1" | cut -d. -f1,2; }

issue_title() { echo "NetBox v$1 released"; }

# find_issue <version>: prints "<number> <state> <labels,...>" of the issue for
# that version, or nothing.
find_issue() {
  local title
  title="$(issue_title "$1")"
  gh issue list --repo "$REPOSITORY" --label "$LABEL" --state all --limit 100 \
    --search "\"$title\" in:title" --json number,state,title,labels |
    jq -r --arg t "$title" '.[] | select(.title == $t)
      | "\(.number) \(.state) \([.labels[].name] | join(","))"' | head -n1
}

detect() {
  local pinned_version version release_tag found number state labels
  pinned_version="$(tr -d '[:space:]' <spec/VERSION)"
  version="${VERSION:-}"
  version="${version#v}"
  if [ -z "$version" ]; then
    version="$(gh api "repos/$UPSTREAM/releases/latest" --jq .tag_name)"
    version="${version#v}"
  fi
  if ! [[ "$version" =~ $semver_re ]]; then
    echo "::error::'$version' is not a NetBox release version"
    exit 1
  fi
  release_tag="$(gh api "repos/$REPOSITORY/releases/latest" --jq .tag_name 2>/dev/null || true)"
  output version "$version"
  output pinned "$pinned_version"
  output release_tag "${release_tag:-}"

  # A manually requested version is analysed even when it is not newer, which
  # also makes it possible to dry-run the workflow on the pinned release.
  if [ -z "${VERSION:-}" ] && ! newer "$version" "$pinned_version"; then
    echo "NetBox $version is the latest release and spec/VERSION pins $pinned_version: nothing to do."
    output mode none
    return
  fi
  if ! docker manifest inspect "$IMAGE:v$version" >/dev/null 2>&1; then
    echo "::notice title=NetBox $version::released, but $IMAGE:v$version is not published yet; checking again on the next run"
    output mode none
    return
  fi

  found="$(find_issue "$version")"
  if [ -z "$found" ] || [ "${FORCE:-false}" = "true" ]; then
    output issue "${found%% *}"
    output mode analyse
    return
  fi
  read -r number state labels <<<"$found"
  output issue "$number"
  if [ "$state" = "OPEN" ] && [[ ",$labels," == *",$PENDING_LABEL,"* ]]; then
    output mode demo
  else
    echo "Issue #$number already covers NetBox $version (set FORCE=true to analyse it again)."
    output mode none
  fi
}

# --- report ------------------------------------------------------------------

result_line() { # result_line <job result or outcome>
  case "$1" in
    success) echo "passed" ;;
    failure) echo "**failed**" ;;
    cancelled) echo "cancelled" ;;
    skipped | "") echo "not run" ;;
    *) echo "$1" ;;
  esac
}

demo_line() {
  case "${DEMO_STATUS:-}" in
    passed) echo "smoke tests passed on demo.netbox.dev (NetBox ${DEMO_VERSION:-unknown})" ;;
    failed) echo "**smoke tests failed** on demo.netbox.dev (NetBox ${DEMO_VERSION:-unknown})" ;;
    pending) echo "pending: demo.netbox.dev still runs NetBox ${DEMO_VERSION:-unknown}; retried daily" ;;
    *) echo "not run (${DEMO_STATUS:-no result}); retried daily" ;;
  esac
}

# demo_done: true when the demo produced a result (passed or failed). Pending,
# errors (demo down, token provisioning failed) and skipped runs are retried.
demo_done() { [ "${DEMO_STATUS:-}" = "passed" ] || [ "${DEMO_STATUS:-}" = "failed" ]; }

# unreleased_commits: feat/fix subjects on HEAD since the latest release tag.
unreleased_commits() {
  [ -n "${RELEASE_TAG:-}" ] || return 0
  git rev-parse -q --verify "refs/tags/$RELEASE_TAG" >/dev/null || return 0
  git log --format='%s' "$RELEASE_TAG..HEAD" | grep -E '^(feat|fix)(\([^)]*\))?!?: ' || true
}

verdict() { # prints "<headline>|<explanation>"
  local breaking=0 additive=0 dir="${ANALYSE_DIR:-}"
  if [ "${ANALYSE_RESULT:-}" != "success" ] || [ ! -f "$dir/verdict.json" ]; then
    echo "Needs a look|The analysis job did not finish, so the API difference is unknown; see the run."
    return
  fi
  breaking="$(jq .breaking "$dir/verdict.json")"
  additive="$(jq .additive "$dir/verdict.json")"
  if [ "${COMPAT_RESULT:-}" = "failure" ]; then
    echo "Release required (fix)|The latest provider release ($RELEASE_TAG) fails acceptance tests against NetBox $VERSION."
  elif [ "$(major_minor "$VERSION")" != "$(major_minor "$PINNED")" ]; then
    echo "Release required|NetBox $(major_minor "$VERSION") is a new minor version: the provider warns on servers outside $(major_minor "$PINNED").x until SupportedNetBoxMajorMinor moves."
  elif [ "$breaking" -gt 0 ]; then
    echo "Release required (breaking)|$breaking breaking change(s) in the generated provider surface."
  elif [ "$additive" -gt 0 ]; then
    echo "Release recommended|$additive additive change(s): the current release cannot use the new attributes, values or filters."
  else
    echo "No release needed|The generated provider surface is unchanged. Bumping the pinned spec is optional housekeeping."
  fi
}

report_body() {
  local dir="${ANALYSE_DIR:-}" headline explanation commits
  IFS='|' read -r headline explanation <<<"$(verdict)"
  commits="$(unreleased_commits)"
  cat <<EOF
## Verdict: $headline

$explanation

| Check | Result |
|---|---|
| Spec extraction and provider diff | $(result_line "${ANALYSE_RESULT:-}") |
| Trial bump (\`make client-gen gen docs\`, build, unit tests) | $(result_line "${TRIAL_OUTCOME:-}") |
| ${RELEASE_TAG:-latest release} acceptance tests on \`netboxcommunity/netbox:v$VERSION\` | $(result_line "${COMPAT_RESULT:-}") |
| demo.netbox.dev smoke tests | $(demo_line) |

Pinned: NetBox $PINNED. Release notes: https://github.com/$UPSTREAM/releases/tag/v$VERSION. Run: ${RUN_URL:-n/a}.

EOF
  if [ -f "$dir/surface.md" ]; then
    cat "$dir/surface.md"
    echo
  fi
  if [ -f "$dir/trial.md" ]; then
    cat "$dir/trial.md"
    echo
  fi
  echo "### Next steps"
  echo
  echo "1. Bump with \`/bump-netbox $VERSION\` or docs/DEVELOPMENT.md. The run's \`netbox-spec-$VERSION\` artifact holds the extracted spec (\`scripts/netbox-spec.sh $VERSION\` recreates it)."
  echo "2. Label the bump PR \`run-acceptance\`, then release with **Actions → Prepare release**."
  if [ -n "$commits" ]; then
    echo
    echo "Already on \`main\` since $RELEASE_TAG ($(echo "$commits" | wc -l | tr -d ' ') feat/fix commit(s) a release would ship):"
    echo
    while IFS= read -r line; do echo "- $line"; done <<<"$commits"
  fi
  echo
  echo "<sub>Opened by .github/workflows/netbox-watch.yml. Close it once the bump is released.</sub>"
}

ensure_labels() {
  gh label create "$LABEL" --repo "$REPOSITORY" --color 0e8a16 \
    --description "A NetBox release the provider has to be checked against" --force >/dev/null
  gh label create "$PENDING_LABEL" --repo "$REPOSITORY" --color fbca04 \
    --description "Waiting for demo.netbox.dev to run the release" --force >/dev/null
}

report() {
  : "${MODE:?}" "${VERSION:?}" "${PINNED:?}" "${REPOSITORY:?}"
  local body pending_flag issue="${ISSUE:-}"
  body="$(mktemp)"
  # shellcheck disable=SC2064 # expand now: $body is local
  trap "rm -f '$body'" EXIT
  ensure_labels

  if [ "$MODE" = "demo" ]; then
    # Stay quiet until the demo runs the release and the smoke tests finish.
    if ! demo_done; then
      echo "demo.netbox.dev: $(demo_line); #$issue stays $PENDING_LABEL"
      return
    fi
    {
      echo "demo.netbox.dev: $(demo_line)."
      echo
      echo "Run: ${RUN_URL:-n/a}"
    } >"$body"
    gh issue comment "$issue" --repo "$REPOSITORY" --body-file "$body" >/dev/null
    gh issue edit "$issue" --repo "$REPOSITORY" --remove-label "$PENDING_LABEL" >/dev/null
    echo "Commented on #$issue"
    return
  fi

  report_body >"$body"
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    cat "$body" >>"$GITHUB_STEP_SUMMARY"
  fi
  pending_flag=(--remove-label "$PENDING_LABEL")
  if ! demo_done; then
    pending_flag=(--add-label "$PENDING_LABEL")
  fi
  if [ -z "$issue" ]; then
    local labels="$LABEL"
    if ! demo_done; then
      labels="$LABEL,$PENDING_LABEL"
    fi
    local assign=()
    if [ -n "${ASSIGNEE:-}" ]; then
      assign=(--assignee "$ASSIGNEE")
    fi
    # An owner that cannot be assigned must not cost the notification.
    gh issue create --repo "$REPOSITORY" --title "$(issue_title "$VERSION")" \
      --label "$labels" "${assign[@]}" --body-file "$body" ||
      gh issue create --repo "$REPOSITORY" --title "$(issue_title "$VERSION")" \
        --label "$labels" --body-file "$body"
  else
    gh issue edit "$issue" --repo "$REPOSITORY" --body-file "$body" "${pending_flag[@]}" >/dev/null
    gh issue comment "$issue" --repo "$REPOSITORY" --body "Analysis re-run: ${RUN_URL:-n/a}" >/dev/null
    echo "Updated #$issue"
  fi
}

case "${1:-}" in
  detect) detect ;;
  report) report ;;
  *)
    echo "usage: $0 detect|report" >&2
    exit 2
    ;;
esac
