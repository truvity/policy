#!/usr/bin/env bash
# Line counts: the generators (the budget is about 3k), the contract in Pkl
# against the hand-written JSON Schemas it replaces, and what is generated.
# "code" excludes blank lines and comment-only lines.
cd "$(dirname "$0")/.."
repo="$(cd ../.. && pwd)"
count() { # file... -> "total code"
  awk 'BEGIN{t=0;c=0} {t++} !/^[[:space:]]*(\/\/|#|\/\/\/)/ && !/^[[:space:]]*$/ {c++} END{printf "%d %d", t, c}' "$@"
}
row() { printf '%-46s %6s %6s\n' "$1" "$(count "${@:2}" | cut -d' ' -f1)" "$(count "${@:2}" | cut -d' ' -f2)"; }
echo "== our generators (Pkl)                         lines   code"
row "gen/JsonSchema.pkl (jsonschema)" gen/JsonSchema.pkl
row "gen/Helm.pkl (helm: schema + values.yaml)" gen/Helm.pkl
row "gen/Zod.pkl (TypeScript + zod)" gen/Zod.pkl
row "gen/Pydantic.pkl (Python + pydantic)" gen/Pydantic.pkl
row "gen/Probes.pkl (boundary fixtures)" gen/Probes.pkl
row "gen/Model.pkl (reflection -> shared IR)" gen/Model.pkl
row "gen/Index.pkl (module graph + chart list)" gen/Index.pkl
row "gen.pkl (pkl:Command entry point)" gen.pkl
row "gen/Load.pkl (YAML -> typed instance: oracle)" gen/Load.pkl
row "vocab/Annotations.pkl (annotation vocabulary)" vocab/Annotations.pkl
echo "-- total generator code (all of the above)"
row "TOTAL" gen/*.pkl gen.pkl vocab/Annotations.pkl
echo
echo "== official generators: our glue only"
row "conformance/go/generate.sh (pkl-go)" conformance/go/generate.sh
row "conformance/kotlin/run.sh generate step" conformance/kotlin/run.sh
echo
echo "== the contract: Pkl vs the hand-written JSON Schema it reproduces"
row "Pkl: vocab/Vocab.pkl" vocab/Vocab.pkl
row "Pkl: contract/Fragments.pkl" contract/Fragments.pkl
row "Pkl: contract/Service.pkl" contract/Service.pkl
row "Pkl: contract/Platform.pkl" contract/Platform.pkl
row "Pkl: contract/urlshortener/*.pkl (8 modules)" contract/urlshortener/*.pkl
row "Pkl: contract/Charts.pkl + charts/*/values.pkl" contract/Charts.pkl charts/*/values.pkl
row "Pkl TOTAL (contract + vocabulary + charts)" vocab/Vocab.pkl contract/*.pkl contract/urlshortener/*.pkl charts/*/values.pkl
row "JSON: schemas/service.json + fragments/*.json" "$repo"/schemas/service.json "$repo"/schemas/fragments/*.json
row "JSON: url-shortener schemas (6 + log + shortener)" "$repo"/examples/url-shortener/schemas/*.json "$repo"/examples/url-shortener/log/src/url_shortener_log/log.schema.json "$repo"/config/testdata/shortener.schema.json
row "JSON TOTAL" "$repo"/schemas/service.json "$repo"/schemas/fragments/*.json "$repo"/examples/url-shortener/schemas/*.json "$repo"/examples/url-shortener/log/src/url_shortener_log/log.schema.json "$repo"/config/testdata/shortener.schema.json
echo
echo "== generated"
row "out/jsonschema/** (20 documents)" $(find out/jsonschema -name '*.json')
row "out/helm/**" $(find out/helm -type f)
row "out/ts/contract.zod.ts" out/ts/contract.zod.ts
row "out/py/contract_pyd.py" out/py/contract_pyd.py
row "Go: pkl-gen-go output" $(find conformance/go/gen -name '*.go')
row "Kotlin: pkl-codegen-kotlin output (no platform)" $(find conformance/kotlin/src/main/kotlin/gen/kotlin -name '*.kt')
echo
echo "== hand-written loader tests, for scale (the runners of this spike)"
row "conformance runners (go, ts, py, kt, driver)" conformance/go/main.go conformance/ts/run.mjs conformance/py/run.py conformance/kotlin/src/main/kotlin/Main.kt conformance/run.py conformance/semdiff.py conformance/compose/main.go
