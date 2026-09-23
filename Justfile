# Development commands. Tools come from devbox (`devbox shell`, or direnv);
# CI runs each recipe as its own job.
#
# `check` is the gate, and it needs nothing but this checkout: no network, no
# containers, no toolchain beyond devbox. A contributor runs exactly what CI
# runs. Recipes that need more (a cluster, a registry) are their own jobs and
# say so.

# Everything CI runs
[doc("Everything CI runs")]
check: build test lint vuln drift leak-canary

# Compile everything
[doc("Compile everything")]
build:
    go build ./...
    cd examples/url-shortener && go build ./...

# The unit tests. They need no network and no services, which is the whole
# point of the gate.
[doc("The unit tests")]
test:
    go test ./...
    # The example is its own module, so `./...` at the root does not reach it.
    # It is also the only place the contracts are proved rather than stated,
    # which makes it the part of this repository that must not go untested.
    cd examples/url-shortener && go test ./...

# Report known vulnerabilities in what this module depends on
[doc("Report known vulnerabilities")]
vuln:
    govulncheck ./...
    cd examples/url-shortener && govulncheck ./...

# The local cluster the charts and the example are tested against: the same
# operators a deployment carries. Idempotent — running it against an existing
# cluster upgrades in place, which is what makes it a development loop rather
# than only a CI step. NOT part of `check`, which needs nothing but the
# checkout; this needs a container runtime.
[doc("Create or upgrade the local cluster")]
cluster:
    bash hack/kind/up.sh

# Ask whether each thing in the box is usable, which is not the same question
# as whether it installed.
[doc("Ask whether the box is usable")]
cluster-verify:
    bash hack/kind/verify.sh

# Prove an operator ACTS: a database becomes a database, a stream becomes a
# stream, a bucket becomes a bucket. This is the question a renderer cannot
# answer and the reason the box is a cluster.
[doc("Prove every operator acts")]
cluster-smoke:
    bash hack/kind/smoke.sh

# Remove it
[doc("Remove the local cluster")]
cluster-down:
    kind delete cluster --name policy

# Regenerate what the TypeScript package carries from `schemas/`.
[doc("Regenerate the schemas the TypeScript package carries")]
ts-schemas:
    node ts/scripts/generate-schemas.mjs

# Generated code is committed. A schema changed without regenerating would
# leave the two loaders validating different documents, which is the one
# thing this repository exists to prevent. Needs no network: the generator
# reads this checkout and writes into it.
[doc("Fail if generated code was not regenerated")]
drift: ts-schemas
    git diff --exit-code -- ts/src/schemas.generated.ts

# The TypeScript package: install, typecheck, test, build, and check what a
# publish would ship. NOT part of `check`, which needs nothing but the
# checkout: this fetches from a registry. CI runs it as its own job.
[doc("The TypeScript package: install, typecheck, test, build")]
ts:
    #!/usr/bin/env bash
    set -euo pipefail
    cd ts
    yarn install --immutable
    yarn typecheck
    yarn test --run
    rm -rf dist && yarn build

    # What a publish would ship: the compiled package, and no test.
    #
    # The listing is captured ONCE rather than piped into `grep -q`. A `-q`
    # grep exits at the first match, the writer gets EPIPE, and under
    # `pipefail` the whole pipeline then fails — a check that reports a
    # failure precisely when it finds what it was looking for.
    shipped=$(yarn pack --dry-run 2>&1)
    grep -q 'dist/index.js' <<<"$shipped"
    ! grep -qE 'dist/.*\.test\.' <<<"$shipped"

# The rules that hold for every file in this repository.
#
# Deliberately no `set -e`: a grep that finds nothing exits non-zero, and
# under `set -e` with `pipefail` that ends the recipe silently — a gate that
# reports nothing and fails is worse than no gate. Each check sets `fail`
# itself.
[doc("The rules that hold for every file")]
lint:
    #!/usr/bin/env bash
    set -uo pipefail
    fail=0

    # `config verify` FIRST: a settings block in the wrong place is accepted
    # silently by `run` and rejected only here, so without this a lint
    # setting can spend releases doing nothing.
    golangci-lint config verify || fail=1
    golangci-lint run ./... || fail=1
    # The example carries the import ban too. A reference implementation
    # exempt from the rules it demonstrates is a reference to nothing.
    ( cd examples/url-shortener && golangci-lint run ./... ) || fail=1

    # Nothing built is committed. A binary in a public repository's history is
    # in every clone forever, and carries the build machine's paths. The
    # largest file here is prose; 1 MiB is a build output.
    big=$(git ls-files -s | awk '{print $2" "$4}' \
            | git cat-file --batch-check='%(objectsize) %(rest)' \
            | awk '$1 > 1048576 {print "    "$2}')
    if [ -n "$big" ]; then
        echo "LINT: build output does not belong in a public repository's history:"
        echo "$big"
        fail=1
    fi

    # A `;` inside a mermaid sequenceDiagram is a statement separator: it
    # splits the message and GitHub renders nothing at all.
    if hits=$(git ls-files -z '*.md' | xargs -0 -r \
            grep -InE '^[[:space:]]*[A-Za-z][A-Za-z0-9_]*[[:space:]]*-?->>?.*;'); then
        echo "LINT: a ';' in a mermaid message stops the diagram rendering:"
        echo "$hits" | sed 's/^/    /'
        fail=1
    fi

    # Every relative link in a Markdown file resolves. A contract nobody can
    # follow is a contract nobody reads.
    broken=""
    while IFS= read -r -d '' f; do
        while read -r link; do
            case "$link" in http*|mailto:*|'') continue ;; esac
            target="$(dirname "$f")/${link%%#*}"
            [ -e "$target" ] || broken+="    $f -> $link"$'\n'
        done < <(grep -oE '\]\([^)]+\)' "$f" | sed 's/^](//;s/)$//')
    done < <(git ls-files -z '*.md')
    if [ -n "$broken" ]; then
        echo "LINT: a relative link does not resolve:"
        printf '%s' "$broken"
        fail=1
    fi

    [ "$fail" = 0 ] && echo "lint clean"
    exit $fail

# This repository is public and its history cannot be unpublished — a rewrite
# changes the SHAs but not what was already fetched. So the rule (mechanism
# only; particulars are the consuming estate's) is enforced mechanically
# rather than remembered.
[doc("Refuse a particular that must never be published")]
leak-canary:
    hack/leak-canary.sh
