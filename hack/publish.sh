#!/usr/bin/env bash
#
# Publish the two packages a tag produces that goreleaser does not build:
# the TypeScript package to GitHub Packages, and the Python wheel as an
# asset on the Release that goreleaser just created.
#
# A script rather than inline workflow steps, for two reasons. A multi-line
# `devbox run -- bash -c '...'` is parsed by devbox before bash ever sees it
# and fails with an error about the first word of the script, which is a
# confusing thing to debug from a release log. And release machinery that
# cannot be run outside CI is release machinery that can only be tested by
# tagging, which is the one way of testing it that costs a version.
#
# So: `hack/publish.sh --dry-run <version>` does everything up to the two
# irreversible actions and stops.
set -euo pipefail

cd "$(dirname "$0")/.."

dry_run=false
if [ "${1:-}" = "--dry-run" ]; then
    dry_run=true
    shift
fi

version="${1:-}"
tag="${2:-v$version}"
if [ -z "$version" ]; then
    echo "usage: publish.sh [--dry-run] <version> [tag]" >&2
    exit 2
fi

echo "==> the TypeScript package"
(
    cd ts
    yarn install --immutable
    rm -rf dist
    yarn build
    # Published version comes from package.json, which hack/stamp-version.py
    # has already written. Asserted here because the alternative — shipping
    # the placeholder — cannot be taken back.
    packed=$(node -p 'require("./package.json").version')
    if [ "$packed" != "$version" ]; then
        echo "publish: ts/package.json says $packed, releasing $version" >&2
        exit 1
    fi
    if [ "$dry_run" = true ]; then
        yarn pack --out "/tmp/policy-$version.tgz"
        echo "    would publish @truvity/policy@$packed"
    else
        # The token reaches yarn through ts/.yarnrc.yml, which interpolates
        # it from GITHUB_PACKAGES_TOKEN and never stores it.
        yarn npm publish --access public
    fi
)

echo "==> the Python wheel"
(
    cd python
    rm -rf dist
    # A wheel and not an sdist: an sdist makes every consumer build a package
    # that has nothing to build.
    uv build --wheel --out-dir dist
    wheel=$(echo dist/*.whl)
    case "$wheel" in
        *"-$version-"*) ;;
        *) echo "publish: built $wheel, releasing $version" >&2; exit 1 ;;
    esac
    if [ "$dry_run" = true ]; then
        echo "    would attach $wheel to $tag"
    else
        gh release upload "$tag" "$wheel" --clobber
    fi
)

echo "published $version"
