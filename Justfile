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
    # Resolving and installing is this package's build: it is what proves the
    # lock still describes something that exists.
    cd python && uv sync --frozen --quiet
    # The example's Python component, for the same reason.
    cd examples/url-shortener/log && uv sync --frozen --quiet
    # The Kotlin loader. `assemble` rather than `build`, so this stays a
    # build: the tests are the test recipe's job.
    cd kotlin && gradle assemble --console=plain --quiet
    # The example's Kotlin component, which builds the RPC client from the
    # same schema the Go server is generated from.
    cd examples/url-shortener/stat && gradle assemble --console=plain --quiet

# The unit tests. They need no network and no services, which is the whole
# point of the gate.
[doc("The unit tests")]
test:
    go test ./...
    # The example is its own module, so `./...` at the root does not reach it.
    # It is also the only place the contracts are proved rather than stated,
    # which makes it the part of this repository that must not go untested.
    #
    # Its chart tests shell out to helm, which devbox supplies — so they are
    # part of the hermetic gate rather than a separate job: rendering a chart
    # needs no network and no cluster.
    cd examples/url-shortener && go test ./...
    # The Python loader, against the SAME fixtures. It is in the hermetic
    # gate because it needs nothing but this checkout: the interpreter and
    # the package manager are declared, and the lock is committed.
    cd python && uv run --frozen pytest -q
    # The archiver: what it decides about batching, keys and ordering, and
    # what its probes answer. No broker and no store — the one call it makes
    # against a store is a five-line double, which is what the one-method
    # protocol in archive.py is for.
    cd examples/url-shortener/log && uv run --frozen pytest -q
    # The Kotlin loader, against the SAME fixtures as the other three.
    cd kotlin && gradle test --console=plain --quiet
    cd examples/url-shortener/stat && gradle test --console=plain --quiet

# Report known vulnerabilities in what this module depends on
[doc("Report known vulnerabilities")]
vuln:
    govulncheck ./...
    cd examples/url-shortener && govulncheck ./...

# The local cluster the charts and every example are tested against: servers
# only (Postgres, NATS with JetStream, an S3 stand-in, a local registry), no
# operator and nothing that names any example — see hack/kind/README.md.
# Idempotent — running it against an existing cluster upgrades in place,
# which is what makes it a development loop rather than only a CI step. NOT
# part of `check`, which needs nothing but the checkout; this needs a
# container runtime.
[doc("Create or upgrade the local cluster")]
cluster:
    bash hack/kind/up.sh

# Ask each server in the box a REAL question — a query, a publish and a
# consume, a put and a get, a push and a pull — not whether it installed.
# There is no operator on this box any more (see hack/kind/README.md), so
# this is also the question `cluster-smoke` used to ask: there is nothing
# left for a separate step to prove.
[doc("Ask whether the box is usable")]
cluster-verify:
    bash hack/kind/verify.sh

# Build the example's images and package its chart exactly the way a release
# does — `.goreleaser.yaml` itself, then `helmctl` — into the box's own
# registry, one architecture instead of every one. See
# hack/example-snapshot.sh and docs/guides/testing.md for why: what the
# cluster installs must be what a release ships.
[doc("Build the example's images and chart exactly as a release does, into the local registry")]
example-snapshot:
    bash hack/example-snapshot.sh

# Stand in for the url-shortener-infra chart, which kind never installs (see
# docs/decisions/0005-kind-is-the-gate.md): the database, the two roles, the
# stream and the bucket the application chart's values point at, under the
# EXACT names examples/url-shortener/e2e/fixture reads off the charts. Must
# run before `example-install`, which the fixture provisions for.
[doc("Provision what the infra chart would, by name")]
example-fixture:
    bash examples/url-shortener/e2e/fixture/apply.sh

# Install the example's PACKAGED application chart — the .tgz `example-snapshot`
# produced, never the source directory (docs/contracts/release.md §7: a
# published artifact is tested as published) — on top of what example-fixture
# provisioned.
[doc("Install the example into the local cluster")]
example-install:
    bash examples/url-shortener/hack/install.sh

# Prove the example WORKS. This is the question the whole box exists to
# answer: a redirect is served, an event crosses the broker, and a counter a
# different service owns moves. Rendering a chart cannot ask it.
#
# A Go suite (examples/url-shortener/e2e/suite), not a shell script: it
# reaches every Service through github.com/truvity/gemaal/pkg/harness, the
# same toolbox a private repository's shared-cluster suite and a
# post-promotion suite use — see docs/guides/testing.md. E2E_NAMESPACE is
# what turns it on; `go test ./...` on its own (`just test`) stays hermetic.
[doc("Prove the example works end to end")]
example-smoke:
    cd examples/url-shortener && E2E_NAMESPACE="${NS:-shortener}" E2E_APP_RELEASE="${APP:-example}" \
        E2E_BUCKET="${BUCKET:-url-shortener-archive}" go test ./e2e/suite/... -count=1 -v

# The whole cluster tier, from nothing.
#
# No identity step: transport identity testing moved off the box entirely
# (0005) and examples/url-shortener/hack/identity-smoke.sh is not yet
# ported to wherever it lands — a later task, not this one.
[doc("The whole cluster tier, from nothing")]
cluster-all: cluster cluster-verify example-snapshot example-fixture example-install example-smoke

# Remove it, and the registry container and the snapshot builder beside it —
# disk is a shared resource on the machine this usually runs on, and a
# container `up.sh` or `example-snapshot.sh` started is this recipe's to
# remove.
[doc("Remove the local cluster")]
cluster-down:
    kind delete cluster --name policy
    docker rm -f kind-registry >/dev/null 2>&1 || true
    docker buildx rm policy-example-snapshot >/dev/null 2>&1 || true

# Render and validate every chart.
#
# The name is the repository contract's, not a description of the tool: a
# person moving between repositories should not have to read this file to
# find out what the chart check is called here, and a repository that has
# the job under another name has made every caller special. This one was
# called `kubeconform` until it was noticed that policy was failing its own
# contract.
#
# NOT part of `check`: it fetches the schema for a custom resource, and the
# gate needs nothing but the checkout. It sits beside `ts` for the same
# reason — both are real checks that happen to need the network.
[doc("Render and validate every chart")]
charts:
    bash hack/kubeconform.sh

# Regenerate the chart goldens. Read the diff BEFORE running this: a golden
# updated without being read is a golden that records whatever happened.
#
# The pattern matches BOTH chart render tests. `TestWhatTheChartRenders`
# alone does not: the infrastructure chart's is
# `TestWhatTheInfraChartRenders`, which that pattern does not contain, so
# this recipe regenerated one chart's goldens and silently left the other's
# stale. The only symptom was a CI failure on a change the author had
# already run `just golden` for.
[doc("Regenerate the chart goldens")]
golden:
    cd examples/url-shortener && UPDATE_GOLDEN=1 go test ./charts/... -run 'TestWhatThe(Infra)?ChartRenders' -count=1

# Regenerate the RPC code from the schema.
[doc("Regenerate the RPC code from the schema")]
protos:
    cd examples/url-shortener && buf lint && buf generate

# Regenerate what every loader carries from `schemas/`.
[doc("Regenerate the schemas the loaders carry")]
schemas:
    node hack/generate-schemas.mjs

# Generated code is committed. A schema changed without regenerating would
# leave the two loaders validating different documents, which is the one
# thing this repository exists to prevent. Needs no network: the generator
# reads this checkout and writes into it.
[doc("Fail if generated code was not regenerated")]
drift: schemas protos
    git diff --exit-code -- ts/src/schemas.generated.ts python/src/truvity_policy/schemas.py
    # The RPC code too. It is committed rather than generated at build time,
    # because a build step between a checkout and a compiler is a step that
    # has to work on every machine forever — and the first thing it breaks
    # is the editor, which cannot resolve a symbol that does not exist yet.
    git diff --exit-code -- examples/url-shortener/internal/gen

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

    # The front end, which depends on the package just built. LAST, because
    # everything above is about the package itself and runs in its
    # directory — a `cd` before the publish check above would run it against
    # a different package, which is exactly what the first version of this
    # did: `yarn pack --dry-run` in a private package with no dist, and a
    # recipe that failed after every step had succeeded.
    cd ../examples/url-shortener/web
    yarn install --immutable
    yarn typecheck
    yarn test --run
    yarn build

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
    # The Python loader: one tool for lint and formatting, so formatting is
    # never a second opinion, and a type checker, because an annotation
    # nothing checks is a comment that rots.
    # The Kotlin compiler with warnings as errors, which is where a JVM
    # project's lint lives: there is no separate linter to run, and a
    # warning nobody fails on is a warning nobody reads.
    for jvm in kotlin examples/url-shortener/stat; do
        ( cd "$jvm" && gradle compileKotlin compileTestKotlin --console=plain --quiet ) || fail=1
    done

    for py in python examples/url-shortener/log; do
        ( cd "$py" \
            && uv run --frozen ruff check . \
            && uv run --frozen ruff format --check . \
            && uv run --frozen mypy ) || fail=1
    done
    # The example carries the import ban too. A reference implementation
    # exempt from the rules it demonstrates is a reference to nothing.
    ( cd examples/url-shortener && golangci-lint run ./... ) || fail=1

    # The toolchain is declared once and resolves to what it declares.
    # Node is the case that needs saying: corepack's shims take precedence
    # over anything devbox installed and fetch their own yarn, so a pin can
    # be correct, unused, and silently overridden all at once.
    python3 hack/toolchain-check.py || fail=1

    # The two package manifests still hold the placeholder the release
    # stamps over. A version edited by hand here ships as itself — and a
    # package published at 0.0.0 cannot be taken back.
    python3 hack/stamp-version.py --check || fail=1

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
