#!/usr/bin/env bash
# Generate Go structs from the contract with the official pkl-go generator
# (the Pkl package pkl.golang). 0.14.0 needs Pkl 0.32; 0.13.2 runs on 0.31.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
spike="$(cd "$here/../.." && pwd)"
cd "$here"
c="$spike/contract"
pkg=spike.invalid/pklconf/gen
rm -rf gen
pkl run package://pkg.pkl-lang.org/pkl-go/pkl.golang@0.13.2#/gen.pkl -- --output-path . \
  --mapping spike.vocab.Vocab=$pkg/vocab \
  --mapping spike.contract.Fragments=$pkg/fragments \
  --mapping spike.contract.Service=$pkg/service \
  --mapping spike.contract.Platform=$pkg/platform \
  --mapping spike.contract.Showcase=$pkg/showcase \
  --mapping spike.contract.urlshortener.Web=$pkg/web \
  --mapping spike.contract.urlshortener.Urls=$pkg/urls \
  --mapping spike.contract.urlshortener.Redirect=$pkg/redirect \
  --mapping spike.contract.urlshortener.Stat=$pkg/stat \
  --mapping spike.contract.urlshortener.Prober=$pkg/prober \
  --mapping spike.contract.urlshortener.Migrate=$pkg/migrate \
  --mapping spike.contract.urlshortener.LogArchiver=$pkg/logarchiver \
  --mapping spike.contract.urlshortener.Shortener=$pkg/shortener \
  "$c"/urlshortener/*.pkl "$c/Platform.pkl" "$c/Showcase.pkl" "$c/Fragments.pkl" "$c/Service.pkl" "$spike/vocab/Vocab.pkl"
