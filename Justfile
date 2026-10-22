# Development commands. Tools come from devbox (`devbox shell`, or direnv);
# CI runs each recipe as its own job.
#
# `check` is the gate, and it needs no credentials, no containers and no
# toolchain beyond devbox — only what the lock files name, which a first
# build downloads. A contributor runs exactly what CI runs. Recipes that need
# more (a cluster, a schema from the network) are their own jobs and say so.
#
# `vuln` is NOT in it (docs/contracts/component.md, C10): a new advisory is
# news about the world, not about the change under review, and one with no
# released fix would turn every pull request red on a finding nobody can act
# on. It runs in .github/workflows/security.yaml, on a schedule.

# Everything CI requires
[doc("Everything CI requires")]
check: build test lint leak-canary

# Compile everything
[doc("Compile everything")]
build:
    go build ./...

# The unit tests. They need no network and no services, which is the whole
# point of the gate.
[doc("The unit tests")]
test:
    go test ./...

# Report known vulnerabilities in what this module depends on. Its own
# workflow (security.yaml), never the gate — see `check`.
[doc("Report known vulnerabilities")]
vuln:
    govulncheck ./...

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

# Remove it, and the registry container beside it — disk is a shared resource
# on the machine this usually runs on, and a container `up.sh` started is this
# recipe's to remove.
[doc("Remove the local cluster")]
cluster-down:
    kind delete cluster --name policy
    docker rm -f kind-registry >/dev/null 2>&1 || true

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
    GOTOOLCHAIN=local golangci-lint run ./... || fail=1

    # The whitespace rules every file holds to, whatever language it is
    # written in (lint/.editorconfig, copied to the root as this
    # repository's own copy — lint/README.md): a final newline, LF, UTF-8,
    # no stray trailing whitespace but a Markdown hard break. Indentation
    # WIDTH is deliberately NOT checked here — that is gofmt's job, already
    # run above. A line-count check cannot tell a real violation from a raw
    # string or a template that legitimately indents differently; chasing
    # that false-positive class file by file is not what a gate is for.
    editorconfig-checker -disable-indentation -disable-indent-size || fail=1

    # The toolchain is declared once: every manifest entry names a version.
    python3 hack/toolchain-check.py || fail=1

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

# Regenerates docs/landscape.md's repository table from the GitHub API —
# latest tag, what each repository ships, its one-line purpose. NOT part of
# `check`: it needs `gh`, authenticated, and the network, and the gate needs
# neither. Read the diff before committing it.
[doc("Regenerate the landscape page's table from the GitHub API")]
landscape:
    hack/landscape.sh

# Does THIS tree still honour the contracts it publishes? Builds and tests the
# downstream example repository (truvity/url-shortener, which moved out of
# here) against this checkout, through a throw-away go.work, so a contract
# change that would break the example breaks it in the pull request that made
# it. Advisory in CI (see .github/workflows/ci.yaml), runnable by anyone:
#
#   just example-canary                     # clones ../url-shortener at master if absent
#   URL_SHORTENER_DIR=~/src/url-shortener just example-canary
#   URL_SHORTENER_REF=some-branch just example-canary
#   URL_SHORTENER_LOCAL_LIB=1 just example-canary   # charts use this checkout's service-lib
#
# NOT part of `check`: it needs the network and another repository.
[doc("Build and test the downstream example against this checkout (advisory)")]
example-canary:
    bash hack/example-canary.sh
