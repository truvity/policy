#!/usr/bin/env bash
#
# Does THIS checkout still satisfy the example that proves it?
#
# The url-shortener example lives in its own repository
# (docs/decisions/0014-the-worked-example-lives-in-its-own-repository.md). That
# repository depends on this module by released version, so a change to a
# contract here is only seen there after a release and a bump. This script
# closes that gap: it builds and tests the example's branch against THIS
# checkout, so a change that breaks the example is seen in the pull request
# that made it.
#
#   hack/example-canary.sh                 # clones ../url-shortener at master if absent
#   URL_SHORTENER_DIR=~/src/url-shortener hack/example-canary.sh
#   URL_SHORTENER_REF=some-branch hack/example-canary.sh
#
# What it does, and nothing else:
#
#   1. takes a copy of the example's tracked files (the clone is never
#      modified, so a working clone of your own is safe to point it at);
#   2. makes a go.work that uses the copy AND this checkout, so the example's
#      `require github.com/truvity/policy vX` is overridden by what is on
#      disk here;
#   3. only with URL_SHORTENER_LOCAL_LIB=1: points any chart that depends on
#      the library chart `service-lib` at charts/service-lib in this checkout
#      instead of the released one, for the same reason. OFF by default,
#      because the example's own tests may pin how that dependency is
#      declared (its repository and version), and rewriting it would turn
#      them red for a reason that is not a broken contract;
#   4. runs the example's hermetic gate: `go build ./... && go test ./...`,
#      under the example's own devbox environment when it has one.
#
# It needs the network (a clone, Go modules, `helm dependency build` of the
# charts' OCI dependencies) and no credentials. It holds none and publishes
# nothing, which is what makes it safe to run on a fork's pull request.
set -euo pipefail

here="$(cd "$(dirname "$0")/.." && pwd)"
repo="${URL_SHORTENER_REPO:-https://github.com/truvity/url-shortener}"
ref="${URL_SHORTENER_REF:-master}"
src="${URL_SHORTENER_DIR:-$here/../url-shortener}"

say() { printf 'example-canary: %s\n' "$*"; }

if [ ! -e "$src/.git" ]; then
    say "no checkout at $src; cloning $repo at $ref"
    if ! git clone --quiet --depth 1 --branch "$ref" "$repo" "$src"; then
        cat >&2 <<EOF
example-canary: could not clone $repo at $ref.
  If that repository or ref does not exist yet, there is nothing to check
  against: this is the expected failure until its first commit on $ref lands.
  Otherwise set URL_SHORTENER_DIR to an existing checkout, or URL_SHORTENER_REF
  to a ref that exists.
EOF
        exit 1
    fi
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

say "copying the tracked files of $src"
git -C "$src" ls-files -z | tar -C "$src" --null --ignore-failed-read -T - -cf - | tar -C "$work" -xf -

# The module and the chart library, from this checkout. A go.work outside the
# copy: nothing is written into the clone, and GOWORK wins over any go.work
# the example might carry.
mkdir "$work.workspace"
trap 'rm -rf "$work" "$work.workspace"' EXIT
(cd "$work.workspace" && GOWORK=off go work init "$work" "$here")
export GOWORK="$work.workspace/go.work"
say "workspace: $(tr '\n' ' ' < "$GOWORK")"

# Charts that depend on service-lib: use this checkout's (opt-in, see above).
# The dependency is rewritten in place and the lock beside it dropped, because
# it records the digest of the archive being replaced. A line-based edit, not
# a YAML library: the layout of a Chart.yaml dependency is fixed by Helm, and
# nothing here needs more than its `version:` and `repository:` lines.
if [ "${URL_SHORTENER_LOCAL_LIB:-0}" = 1 ]; then
rewritten="$(python3 -I - "$work" "$here/charts/service-lib" <<'PY'
import pathlib
import sys

root, lib = pathlib.Path(sys.argv[1]), sys.argv[2]


def indent(line: str) -> int:
    return len(line) - len(line.lstrip())


def block(lines: list[str], at: int) -> range:
    """The lines of the list item that holds line `at`."""
    start = at
    while start > 0 and not lines[start].lstrip().startswith("- "):
        start -= 1
    base = indent(lines[start])
    end = at + 1
    while end < len(lines) and (not lines[end].strip() or indent(lines[end]) > base):
        end += 1
    return range(start, end)


for chart in sorted(root.rglob("Chart.yaml")):
    lines = chart.read_text(encoding="utf-8").splitlines(keepends=True)
    changed = False
    for at, line in enumerate(lines):
        if line.strip().removeprefix("- ").strip() != "name: service-lib":
            continue
        for i in block(lines, at):
            key = lines[i].strip().removeprefix("- ").strip()
            pad = lines[i][: indent(lines[i])]
            if key.startswith("version:"):
                lines[i] = f"{pad}{'- ' if lines[i].lstrip().startswith('- ') else ''}version: 0.0.0\n"
                changed = True
            elif key.startswith("repository:"):
                lines[i] = f"{pad}{'- ' if lines[i].lstrip().startswith('- ') else ''}repository: file://{lib}\n"
                changed = True
    if changed:
        chart.write_text("".join(lines), encoding="utf-8")
        chart.with_name("Chart.lock").unlink(missing_ok=True)
        print(chart.parent)
PY
)"
    if [ -n "$rewritten" ]; then
        say "charts now using this checkout's service-lib:"
        printf '    %s\n' "$rewritten" | sed "s#$work/##"
    else
        say "no chart depends on service-lib; only the Go module is exercised"
    fi
else
    say "charts keep the service-lib they declare (URL_SHORTENER_LOCAL_LIB=1 swaps in this checkout's)"
fi

gate() { go build ./... && go test ./...; }

say "go build ./... && go test ./... against this checkout"
cd "$work"
if [ -f devbox.json ] && command -v devbox >/dev/null 2>&1; then
    # One line, no function body: devbox re-evaluates the command it is given
    # and a multi-line string does not survive that.
    devbox run -c "$work" -- bash -c 'go build ./... && go test ./...'
else
    gate
fi
say "the example builds and passes against this checkout"
