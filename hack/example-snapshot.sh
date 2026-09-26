#!/usr/bin/env bash
#
# Build the example's images and package its chart EXACTLY the way a release
# does: GoReleaser builds and pushes every image and records what it pushed,
# then helmctl reads that and bakes the digests into the chart — see
# .goreleaser.yaml's own header and docs/contracts/release.md §5 and §7.
#
# What is different here, and only here:
#
#   - the destination is the kind box's own registry, not ghcr.io, and no
#     GitHub release is cut — the same two environment variables the
#     release configuration's header already documents for a local loop;
#   - every image is built for ONE architecture, the box's own, rather than
#     every architecture a release always carries. The box's node is one
#     architecture, so a second one would only be built and discarded — the
#     same reasoning the old `example-images` recipe gave for `ko build
#     --platform`. That narrowing is produced by
#     hack/goreleaser-snapshot-config from THIS file, never a second
#     release configuration kept beside it.
#
# NEVER --snapshot: GoReleaser's snapshot mode also stops every image being
# pushed (see .goreleaser.yaml), which is exactly the one thing this needs.
# `--skip=validate` is what makes a commit that is not a tag releasable
# instead.
set -euo pipefail
cd "$(dirname "$0")/.."

SNAPSHOT_REGISTRY=${SNAPSHOT_REGISTRY:-localhost:$(grep '^REGISTRY_PORT=' hack/kind/versions.env | cut -d= -f2)}
IMAGE_TAG=${IMAGE_TAG:-snapshot-$(git rev-parse --short HEAD)}

export IMAGE_REPO="${SNAPSHOT_REGISTRY}/url-shortener"
export KO_DOCKER_REPO="${SNAPSHOT_REGISTRY}/url-shortener"
export IMAGE_TAG
export GITHUB_RELEASE_DISABLED=true

# A buildx builder of THIS script's own, on the HOST's network — never
# whatever builder happens to be current. Two things go wrong otherwise:
#
#   - a "docker-container" builder runs BuildKit in a separate container
#     with its OWN network namespace, where "localhost" is itself and
#     nothing is listening on it. A push there does not fail, it retries a
#     connection forever until something times out.
#   - the "default" driver shares the host's network (so "localhost" is
#     right), but it is the plain docker driver, and GoReleaser's
#     `dockers_v2` pipe always asks for an SBOM attestation — which that
#     driver refuses outright ("Attestation is not supported for the
#     docker driver").
#
# `--driver-opt network=host` is both fixes at once: a real BuildKit
# container (so attestations work), sharing the host's network namespace
# (so "localhost" reaches the box's registry). Idempotent, and never torn
# down by this script — `just cluster-down` removes it, the same as the
# registry container `hack/kind/up.sh` starts.
BUILDER=policy-example-snapshot
if ! docker buildx inspect "$BUILDER" >/dev/null 2>&1; then
    docker buildx create --name "$BUILDER" --driver docker-container --driver-opt network=host >/dev/null
fi
export BUILDX_BUILDER="$BUILDER"

step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }

scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT

step "the release configuration, for linux/${SNAPSHOT_ARCH:-$(go env GOARCH)}"
go run ./hack/goreleaser-snapshot-config \
    -platform "linux/${SNAPSHOT_ARCH:-$(go env GOARCH)}" \
    -in .goreleaser.yaml \
    -out "$scratch/goreleaser.yaml"

step "the images, built and pushed to ${SNAPSHOT_REGISTRY}"
goreleaser release --config "$scratch/goreleaser.yaml" --clean --skip=validate,announce

step "the chart, packaged from what was just pushed"
rm -rf dist/charts
helmctl goreleaser-manifest --goreleaser-dist dist -o dist/goreleaser-manifest.json
helmctl package \
    --chart examples/url-shortener/charts/url-shortener \
    --manifest dist/goreleaser-manifest.json \
    --require-image-digests \
    --output dist/charts

ls dist/charts
