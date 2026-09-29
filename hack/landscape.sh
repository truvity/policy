#!/usr/bin/env bash
# Regenerates the repository table in docs/landscape.md from the GitHub
# API, so the table is never hand-maintained: latest tag, what each
# repository ships, and its one-line purpose are all read from
# origin/master and the repository's releases at run time.
#
# Needs `gh`, authenticated, and the network — deliberately NOT part of
# `just check`, which needs neither (docs/landscape.md, "Keeping this page
# true"). The layer each repository is grouped under, and the "How they
# fit" prose, are curation and are not regenerated here.
set -uo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

out="docs/landscape.md"
start='<!-- landscape:table:start -->'
end='<!-- landscape:table:end -->'

# repo|layer, in the order the table renders.
repos=(
  "access-roster|Identity and secrets"
  "openbao|Identity and secrets"
  "audit|Identity and secrets"
  "gateway|Edge"
  "cloudflare|Edge"
  "tailscale|Edge"
  "cnpg-cluster|Data"
  "observability|Observability"
  "ci-workflows|CI"
  "ci-actions|CI"
  "ci-plane|CI"
  "ci-cache|CI"
  "nats-auth-callout|Cluster add-ons"
  "argocd-ecr-updater|Cluster add-ons"
  "amazon-eks-pod-identity-webhook|Cluster add-ons"
  "ocictl|Developer tooling"
  "workstation|Developer tooling"
  "gemaal|Developer tooling"
  "github-structure|Developer tooling"
  "policy|Doctrine"
)

raw() { gh api -H "Accept: application/vnd.github.raw" "repos/truvity/$1/contents/$2" 2>/dev/null || true; }
tree() { gh api "repos/truvity/$1/git/trees/master?recursive=true" --jq '.tree[].path' 2>/dev/null || true; }

latest_tag() {
  local repo="$1" t
  t=$(gh release list -R "truvity/$repo" --limit 1 --json tagName --jq '.[0].tagName // empty' 2>/dev/null || true)
  if [ -z "$t" ]; then
    t=$(git ls-remote --tags "https://github.com/truvity/$repo.git" 2>/dev/null \
        | awk '{print $2}' | sed 's#refs/tags/##' | grep -v '\^{}$' \
        | sort -V | tail -1)
    [ -n "$t" ] && t="$t (unreleased)"
  fi
  echo "${t:-none}"
}

# The first sentence of the first real paragraph after the README's title —
# skipping badge lines — condensed to one line for a table cell; the
# repository's own README is the source of truth for anything longer.
purpose() {
  local repo="$1" readme
  readme=$(raw "$repo" README.md)
  printf '%s\n' "$readme" | awk '
    /^# / { seen=1; next }
    seen && !started && /^\[!\[/ { next }
    seen && !started && NF==0 { next }
    seen && !started && /^#/ { exit }
    seen && started && NF==0 { exit }
    seen && NF>0 { started=1; buf = buf==""? $0 : buf" "$0 }
    END {
      gsub(/\*\*/, "", buf)
      # Markdown links to plain text FIRST — a URL'"'"'s own dots must never
      # be mistaken for a sentence end.
      buf = gensub(/\[([^]]+)\]\([^)]+\)/, "\\1", "g", buf)
      # A version number'"'"'s dot ("Kubernetes 1.35+") is not a sentence end
      # either — protect digit.digit, restored after the split below.
      buf = gensub(/([0-9])\.([0-9])/, "\\1@@DOT@@\\2", "g", buf)
      # First sentence; if it is a short bold tagline (few words), fold in
      # the sentence after it too, so a title like "The policy is the
      # product." is not the whole cell.
      if (match(buf, /^[^.]+\./)) {
        first = substr(buf, 1, RLENGTH)
        nwords = split(first, warr, / /)
        if (nwords <= 6) {
          rest = substr(buf, RLENGTH + 2)
          if (match(rest, /^[^.]+\./)) {
            result = first " " substr(rest, 1, RLENGTH)
          } else {
            result = first
          }
        } else {
          result = first
        }
      } else {
        result = buf
      }
      gsub(/@@DOT@@/, ".", result)
      print result
    }
  '
}

ships() {
  local repo="$1" files charts gomod=no importable=no cli reusable bits=() toplevel

  files=$(tree "$repo")
  charts=$(printf '%s\n' "$files" | grep -c '^charts/[^/]*/Chart\.yaml$' || true)
  [ "$charts" -gt 0 ] 2>/dev/null && bits+=("charts:$charts")

  if printf '%s\n' "$files" | grep -qx 'go\.mod'; then
    gomod=yes
    toplevel=$(printf '%s\n' "$files" | grep -oE '^[A-Za-z0-9_.-]+/' | sort -u \
      | grep -vE '^(cmd|internal|\.github|docs|charts|examples|hack|test|tests|schemas|ts|python|kotlin|bin|lint|scripts|third_party)/' || true)
    [ -n "$toplevel" ] && importable=yes
    if [ "$importable" = yes ]; then
      bits+=("Go module (importable)")
    else
      bits+=("Go module (not importable)")
    fi
  fi

  if printf '%s\n' "$files" | grep -qE '^\.goreleaser\.ya?ml$|^\.ko\.ya?ml$'; then
    goreleaser_content=""
    printf '%s\n' "$files" | grep -qx '.goreleaser.yaml' && goreleaser_content+=$(raw "$repo" .goreleaser.yaml)
    printf '%s\n' "$files" | grep -qx '.ko.yaml' && goreleaser_content+=$(raw "$repo" .ko.yaml)
    printf '%s' "$goreleaser_content" | grep -qE 'kos:|dockers_v2:' && bits+=("images")
  fi

  cli=$(printf '%s\n' "$files" | grep -oE '^cmd/[^/]+' | sort -u | sed 's#cmd/##' | paste -sd, - | sed 's/,/, /g')
  [ -n "$cli" ] && bits+=("CLI: $cli")

  printf '%s\n' "$files" | grep -qE '(^|/)action\.ya?ml$' && bits+=("action")

  reusable=""
  while read -r f; do
    [ -z "$f" ] && continue
    raw "$repo" "$f" | grep -q 'workflow_call:' && reusable="$reusable,$(basename "$f")"
  done < <(printf '%s\n' "$files" | grep -E '^\.github/workflows/.*\.ya?ml$')
  reusable="${reusable#,}"
  [ -n "$reusable" ] && bits+=("reusable workflows: $(echo "$reusable" | sed 's/,/, /g')")

  local result="" b
  for b in "${bits[@]}"; do
    if [ -z "$result" ]; then result="$b"; else result="$result; $b"; fi
  done
  echo "$result"
}

tmp=$(mktemp)
{
  echo "$start"
  echo
  echo "| Layer | Repository | Purpose | Ships | Latest tag |"
  echo "|---|---|---|---|---|"
  for entry in "${repos[@]}"; do
    repo="${entry%%|*}"
    layer="${entry#*|}"
    echo "landscape.sh: $repo" >&2
    p=$(purpose "$repo")
    s=$(ships "$repo")
    t=$(latest_tag "$repo")
    echo "| $layer | [\`$repo\`](https://github.com/truvity/$repo) | $p | $s | $t |"
  done
  echo
  echo "$end"
} > "$tmp"

awk -v start="$start" -v end="$end" -v tablefile="$tmp" '
  BEGIN { while ((getline line < tablefile) > 0) table = table line "\n" }
  $0 == start { printf "%s", table; skip=1; next }
  $0 == end   { skip=0; next }
  skip != 1 { print }
' "$out" > "$out.new" && mv "$out.new" "$out"
rm -f "$tmp"

echo "landscape table regenerated from the GitHub API — review the diff before committing" >&2
