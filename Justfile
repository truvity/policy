# Development commands. Tools come from devbox (`devbox shell`, or direnv);
# CI runs each recipe as its own job.
#
# `check` is the gate, and it needs nothing but this checkout: no network, no
# containers, no toolchain beyond devbox. A contributor runs exactly what CI
# runs. Recipes that need more (a cluster, a registry) are their own jobs and
# say so.

# Everything CI runs
check: lint leak-canary

# The rules that hold for every file in this repository.
#
# Deliberately no `set -e`: a grep that finds nothing exits non-zero, and
# under `set -e` with `pipefail` that ends the recipe silently — a gate that
# reports nothing and fails is worse than no gate. Each check sets `fail`
# itself.
lint:
    #!/usr/bin/env bash
    set -uo pipefail
    fail=0

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
leak-canary:
    hack/leak-canary.sh
