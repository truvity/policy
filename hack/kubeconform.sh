#!/usr/bin/env bash
#
# Validate what the charts render against the Kubernetes API's own schemas.
#
# This asks a question none of the chart tests do. They check what the
# configuration contract cares about — that a rendered file is one the
# binary accepts — and they would pass just as happily for a Deployment with
# a misspelled field, because a misspelled field is silently ignored by the
# API server rather than refused.
#
# The goldens are the input, deliberately: they are the renders this
# repository has already committed to, so this validates the thing a reviewer
# actually read.
set -euo pipefail

cd "$(dirname "$0")/.."

# The custom resources are not in the upstream schema set — that is what
# makes them custom — so their kinds are skipped by name rather than by
# turning missing-schema errors off, which would skip a typo in a built-in
# kind too.
SKIP="Cluster,Stream"

# The Kubernetes line the local cluster runs, so this and the box agree
# about what exists. A newer line would accept a field the cluster does not
# have, which is the failure this is here to prevent.
VERSION="$(grep '^NODE_IMAGE=' hack/kind/versions.env | sed 's/.*://')"

echo "==> validating the committed renders against Kubernetes ${VERSION}"
count=0
for golden in examples/url-shortener/charts/testdata/golden/*.yaml; do
    kubeconform \
        -kubernetes-version "${VERSION#v}" \
        -strict \
        -skip "$SKIP" \
        -schema-location default \
        -schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
        -summary \
        "$golden"
    count=$((count + 1))
done

echo "    ${count} render(s) are shapes the API server would accept"
