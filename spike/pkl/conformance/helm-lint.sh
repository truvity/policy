#!/usr/bin/env bash
# helm lint each generated chart (values.yaml checked against values.schema.json
# by Helm itself), then one deliberately broken values file, which must fail.
set -uo pipefail
cd "$(dirname "$0")/.."
for c in web urls redirect stat log prober migrate; do
  d="$(mktemp -d)"
  cp out/helm/$c/values.* "$d/"
  printf 'apiVersion: v2\nname: %s\nversion: 0.0.0\n' "$c" > "$d/Chart.yaml"
  if helm lint "$d" 2>&1 | grep -q '0 chart(s) failed'; then echo "helm lint $c: pass"; else echo "helm lint $c: FAIL"; fi
  rm -rf "$d"
done
d="$(mktemp -d)"
cp out/helm/web/values.* "$d/"
printf 'apiVersion: v2\nname: web\nversion: 0.0.0\n' > "$d/Chart.yaml"
sed -i 's/level: info/level: verbose/' "$d/values.yaml"
if helm lint "$d" 2>&1 | grep -q 'level: value must be one of'; then echo "helm lint web with log.level=verbose: rejected, as it must be"; else echo "helm lint web with log.level=verbose: NOT REJECTED"; fi
rm -rf "$d"
