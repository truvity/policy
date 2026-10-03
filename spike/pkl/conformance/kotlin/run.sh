#!/usr/bin/env bash
# Regenerate the Kotlin classes from the contract and run the Kotlin validators.
# Prints one JSON line per (fixture, validator). Needs network for the generator
# jar and Gradle dependencies on first use.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
spike="$(cd "$here/../.." && pwd)"
repo="$(cd "$spike/../.." && pwd)"
cd "$here"
if [ ! -x .tools/pkl-codegen-kotlin ]; then
  mkdir -p .tools
  curl -fsSL -o .tools/pkl-codegen-kotlin https://github.com/apple/pkl/releases/download/0.31.1/pkl-codegen-kotlin
  chmod +x .tools/pkl-codegen-kotlin
fi
c="$spike/contract"
rm -rf src/main/kotlin/gen
# Platform is left out on purpose: the generator refuses union types
# (`Int|String`), which `platform` uses.
.tools/pkl-codegen-kotlin --output-dir src/main/kotlin/gen \
  "$c"/urlshortener/*.pkl "$c/Showcase.pkl" "$c/Fragments.pkl" "$c/Service.pkl" "$spike/vocab/Vocab.pkl" >&2
gradle -q --console=plain --no-daemon run --args="$spike $repo $spike/out/jsonschema" 2>&1 | grep '^{' || { echo "gradle run failed" >&2; exit 1; }
