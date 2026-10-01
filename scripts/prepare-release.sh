#!/usr/bin/env bash
# Writes the next release section of CHANGELOG.md: git-cliff drafts the entries
# from the Conventional Commit subjects since the last tag (cliff.toml) and
# tools/changelog merges them with any hand-written [Unreleased] entries.
#
#   scripts/prepare-release.sh            # version worked out from the commits
#   scripts/prepare-release.sh 0.2.0      # explicit version
#
# Used by `make changelog` and .github/workflows/prepare-release.yml. Prints the
# version on the last line (and writes version=... to $GITHUB_OUTPUT in CI).
# Set GIT_CLIFF to override how git-cliff is run.
set -euo pipefail

# renovate: datasource=pypi depName=git-cliff
GIT_CLIFF_VERSION="${GIT_CLIFF_VERSION:-2.14.2}"

cd "$(git rev-parse --show-toplevel)"

if [ -z "${GIT_CLIFF:-}" ]; then
  if command -v uvx >/dev/null 2>&1; then
    GIT_CLIFF="uvx --from git-cliff==${GIT_CLIFF_VERSION} git-cliff"
  elif command -v pipx >/dev/null 2>&1; then
    GIT_CLIFF="pipx run --spec git-cliff==${GIT_CLIFF_VERSION} git-cliff"
  elif command -v git-cliff >/dev/null 2>&1; then
    GIT_CLIFF="git-cliff"
  else
    echo "error: need uvx, pipx or git-cliff ${GIT_CLIFF_VERSION} on PATH" >&2
    exit 1
  fi
fi
read -r -a cliff <<<"$GIT_CLIFF"

if [ -z "$(git tag --list 'v*')" ]; then
  echo "error: no v* tags; fetch them (git fetch --tags, or fetch-depth: 0 in CI)" >&2
  exit 1
fi

version="${1:-}"
version="${version#v}"
if [ -z "$version" ]; then
  version="$("${cliff[@]}" --bumped-version 2>/dev/null)"
  version="${version#v}"
fi
if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "error: '$version' is not a semantic version" >&2
  exit 1
fi
if git rev-parse -q --verify "refs/tags/v${version}" >/dev/null; then
  echo "error: tag v${version} already exists" >&2
  exit 1
fi

entries="$(mktemp)"
trap 'rm -f "$entries"' EXIT
"${cliff[@]}" --unreleased --tag "v${version}" --strip all --output "$entries"

go run ./tools/changelog prepare -version "$version" -entries "$entries"

echo "CHANGELOG.md: added the [${version}] section; review and edit the wording before releasing" >&2
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "version=${version}" >>"$GITHUB_OUTPUT"
fi
echo "$version"
