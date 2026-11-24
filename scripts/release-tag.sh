#!/bin/sh
# Tag the current commit with the version from package.json (v1.2.3) and push it.
# The release workflow builds the muxon binaries and publishes the GitHub release from that tag. Run
# by the "version" workflow after a "Release" PR merges, or by hand after `pnpm release:version` on main.
set -eu
cd "$(dirname "$0")/.."
version=$(node -p "require('./package.json').version")
tag="v$version"
if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
  echo "tag $tag already exists; nothing to do"
  exit 0
fi
[ -z "$(git status --porcelain)" ] || { echo "working tree not clean"; exit 1; }
git tag -a "$tag" -m "muxon $version"
git push origin "$tag"
echo "pushed $tag"
