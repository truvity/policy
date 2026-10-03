#!/usr/bin/env bash
# Render the web chart's values for each environment, check them with Helm
# against the generated values.schema.json, and record what the failing modules
# say. Needs out/helm (pkl run gen.pkl) first.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
spike="$(cd "$here/.." && pwd)"
out="$spike/out/gitops"
rm -rf "$out" && mkdir -p "$out"
status=0

for env in devel prod; do
  mkdir -p "$out/$env"
  pkl eval -f yaml "$here/$env/web.pkl" > "$out/$env/web.values.yaml" || status=1
  # A scratch chart that holds only the generated schema: Helm validates the
  # rendered values with it exactly as it would at install.
  chart="$(mktemp -d)"
  cp "$spike/out/helm/web/values.schema.json" "$chart/"
  printf 'apiVersion: v2\nname: web\nversion: 0.0.0\n' > "$chart/Chart.yaml"
  if helm lint "$chart" -f "$out/$env/web.values.yaml" > "$out/$env/helm-lint.txt" 2>&1; then
    echo "helm lint $env: pass"
  else
    echo "helm lint $env: FAIL"; status=1
  fi
  rm -rf "$chart"
done
diff -u --label devel --label prod "$out/devel/web.values.yaml" "$out/prod/web.values.yaml" > "$out/devel-vs-prod.diff"

: > "$out/failures.txt"
for f in "$here"/failing/*.pkl; do
  name="$(basename "$f" .pkl)"
  if msg="$(pkl eval "$f" 2>&1 >/dev/null)"; then
    echo "$name: EVALUATED (expected a failure)" | tee -a "$out/failures.txt"; status=1
  else
    echo "$name: $(printf '%s\n' "$msg" | sed -n '2p;3p' | tr '\n' ' ')" | tee -a "$out/failures.txt"
  fi
done
exit $status
