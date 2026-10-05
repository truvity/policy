# Adopting the shape from a private repository

**The rule.** A private repository consumes the same contracts, the same
chart interface and the same libraries as a public one. What differs is
where the gate runs, where a release is published, and how its packages are
fetched — never the shape itself. [0005](../decisions/0005-kind-is-the-gate.md)
and [0004](../decisions/0004-policy-is-public.md).

**Why it is written down separately.** This repository must stay runnable by
a stranger with none of an estate's infrastructure — that is the whole of
[0004](../decisions/0004-policy-is-public.md) — so it can never itself
describe a shared cluster, a private registry or a token. A private
repository has all three, and using a public repository's own defaults
anyway — a local cluster it does not need, an OSS build tool with no
destination configured — is not caution, it is leaving on the table exactly
the depth a real platform gives for free. This guide says which defaults a
private repository keeps and which it replaces, so that adopting these
contracts does not mean re-deriving that split from the contracts' own
public-repository framing.

## What is identical to a public consumer

Nothing about the shape itself changes. A private repository is held to
exactly the same things a public one is:

- **The service contract.** One configuration file plus a committed schema,
  the two probe endpoints, JSON logs on stderr, drain on `SIGTERM` — every
  rule in [contracts/service.md](../contracts/service.md) and
  [contracts/config.md](../contracts/config.md) applies unchanged. A private
  service that configures itself differently because "nobody outside sees
  it" is a service the next reader cannot operate without reading its
  source, which is the exact failure the contract exists to rule out.
- **The chart interface and its conformance rules.** Names, not values; a
  tier a test install can afford; the two-chart split where a service owns a
  database or a stream. [contracts/platform.md](../contracts/platform.md)
  and the checks in [conformance.md](conformance.md#what-ci-already-checks)
  hold on a private chart exactly as they hold on the example's.
- **The telemetry environment variables.** The chart renders the
  OpenTelemetry SDK's own variables and nothing else — see
  ["What a chart sets, and what it must not"](logging-and-telemetry.md#what-a-chart-sets-and-what-it-must-not).
  A private platform's collector is a real endpoint instead of an absent
  one; the variable names and the "no endpoint means `none`" rule do not
  change because the endpoint now points somewhere.
- **The depguard copy.** [lint/golangci-depguard.yaml](../../lint/golangci-depguard.yaml)
  is pasted into a private repository's `.golangci.yaml` exactly as
  [lint/README.md](../../lint/README.md) describes for any consumer — there
  is no private variant of the import ban. An estate adopting these
  contracts across many repositories does not have to take that on faith
  per repository: a central parity job asserts the pasted block still
  matches, across the whole estate, and reports **n/a** rather than a
  failure for a repository with no Go code. A repository that genuinely
  cannot take the block as written — a fork carrying a dependency the
  canon does not — is a **documented exemption**, not a silent skip; the
  parity job's own configuration is where that exemption is named.
- **The test chart and prober shape.** A service that owns a database or a
  stream ships a third, released chart carrying the suite as a plain Job
  (["the suite, as a released test chart"](testing.md#the-suite-as-a-released-test-chart)),
  and the same chart's prober Deployment if the service wants a continuous
  signal
  (["The prober"](testing.md#the-prober)). Both run against a private
  install exactly as they run against the example's — the suite reads every
  name it needs from an environment variable with a fixture fallback, never
  from a convention only the box that built it has.

## What differs

- **The PR gate.** [0005](../decisions/0005-kind-is-the-gate.md) draws this
  line on purpose: a public repository's fork-originated pull requests must
  never reach real infrastructure, so a local kind cluster is the only
  honest gate it can run. A private repository carries no such constraint,
  and its CI can already reach the shared development cluster — the real
  identity plane, the real provisioning, the real network policy a local
  cluster cannot have. So a private repository's required check runs the
  same suite (["The suite"](testing.md#the-suite)) against that shared
  cluster, installed through the consumer's own tenant harness, rather than
  against kind. Kind stays available for the fast local loop — standing up
  a cluster from nothing, rendering and installing before anything is
  pushed — the same recipes (`just cluster`, `just cluster-all`) still work
  on a laptop; it simply stops being what the *required* check runs.
- **Where a release goes.** The release is still built by the same OSS
  goreleaser configuration shape this repository's own
  [`.goreleaser.yaml`](../../.goreleaser.yaml) uses — one job, images first,
  charts packaged from the digests that job wrote
  ([release.md §5](../contracts/release.md#5-the-build-is-the-release)) —
  pointed at a private registry instead of a public one. That is the "ONE
  destination per repository" rule in the same section: what varies is
  *where*, one environment variable, never what is built. One trap is worth
  knowing before it costs a release: **a field that exists only in
  goreleaser Pro is refused at parse time, not silently ignored** — a
  private repository's configuration copied from an example that assumed
  Pro fails the release before it builds anything, which is the right
  failure, but only if it is expected.
- **Where a chart goes.** Charts are packaged and pushed to a private OCI
  registry by the same registry tool the public path uses, reading the same
  digest-stamped artifact list
  ([release.md §5](../contracts/release.md#5-the-build-is-the-release)).
  Nothing about packaging changes; only the destination the tool is told to
  push to does.
- **How policy's own libraries are consumed.** Pinned by version, in the
  consumer's own manifest and lock file — never vendored, never copied into
  the private repository's tree. [release.md §1](../contracts/release.md#1-one-tag-stamps-everything)
  is the reason: one tag stamps every artifact this repository publishes
  together, and a copy of the source cannot be pinned or bumped the same
  way a dependency can.
- **Private module or package access.** The three ecosystems this
  repository publishes to are not symmetric, and a private consumer meets
  each one differently:
  - **Go.** `github.com/truvity/policy` is a public module on a public
    repository, so `GOPRIVATE` is not needed for it and never will be — see
    ["No token, no installation, no rotation for any consumer"](../decisions/0004-policy-is-public.md#consequences).
    `go get`/`go mod tidy` resolve it exactly like any other public
    dependency.
  - **npm.** `@truvity/policy` is published to GitHub Packages, which —
    unlike npm's own public registry — requires an authenticated request
    for **every** package it serves, including a public one. A private
    consumer scopes `@truvity` to the GitHub Packages registry in its own
    `.npmrc` or `.yarnrc.yml` and supplies a token with `read:packages`, the
    same way it would for any other package the estate hosts there. That
    token is the private repository's concern, not this one's: nothing here
    is gated by it, only reaching it is.
  - **pip.** The wheel is **not** published to any package index; it is
    attached as an asset on the GitHub Release for its tag
    ([`hack/publish.sh`](../../hack/publish.sh)). `pip install
    truvity-policy` resolves nothing. A private consumer either points its
    resolver at the release asset's own URL for the pinned tag, or mirrors
    that asset into whatever private index the estate already runs — but it
    is a fetch-by-URL from the start, not a registry lookup that happens to
    need a token.

## Migrating an existing service onto the shape

The order that finds problems fastest, and the order a survey-first
migration ([AGENTS.md, "bringing another repository to this shape"](../../AGENTS.md))
already argues for on general grounds. Each step is its own pull request.

1. **Charts.** Split into an infrastructure chart and an application chart
   if the service owns a database or a stream
   ([platform.md §11](../contracts/platform.md#11-two-charts-and-who-installs-each)),
   and take every value the two tables in
   [platform.md §10](../contracts/platform.md#10-what-a-platform-passes-by-name)
   list — by name, never derived from a convention only the current
   platform happens to follow.
2. **The composition root.** Replace a dependency-injection container or a
   service locator with a hand-written `main` that constructs dependencies
   in order and passes them as arguments
   ([service.md §2](../contracts/service.md#2-the-composition-root-is-hand-written),
   [0001](../decisions/0001-no-di-containers.md)). Doing this before step 5
   matters: removing the container is the bulk of the change, and depguard
   only holds the result — it does not do the rewrite.
3. **Config.** Collapse whatever configuration surface exists today into one
   file validated against one committed schema, with secrets moved to
   declared environment variables and nowhere else
   ([config.md](../contracts/config.md)).
4. **Telemetry.** Delete any home-grown telemetry configuration — a block in
   the config schema, a flag gated on an environment name — and start the
   SDK from its own environment variables instead
   (["Telemetry"](logging-and-telemetry.md#telemetry)). A chart that used to
   render a custom block now renders the OpenTelemetry variables listed
   above and nothing else.
5. **Depguard.** Paste the shared block into `.golangci.yaml`, run
   `golangci-lint config verify` before `run` — a misplaced block is
   accepted silently otherwise — and resolve every finding as a migration or
   a named exception ([lint/README.md](../../lint/README.md)).
6. **The test chart.** Add the third, released chart carrying the suite as a
   plain Job, and wire the prober if the service should be watched
   continuously rather than proved once
   (["The suite, as a released test chart"](testing.md#the-suite-as-a-released-test-chart),
   ["The prober"](testing.md#the-prober)).
7. **The promotion gate.** Point that same suite at the shared development
   cluster through the consumer's own tenant harness, per
   [0005](../decisions/0005-kind-is-the-gate.md), so the check that gates a
   merge is the depth a local cluster cannot reach — identity, provisioning,
   network policy — rather than a second copy of the kind lane.

**A note on an existing database.** A database is a thing the infrastructure
chart provisions per install, not a thing the application finds lying
around under its old name
([platform.md §6](../contracts/platform.md#6-streams-and-databases-are-things-the-service-finds-not-things-it-makes),
[§11](../contracts/platform.md#11-two-charts-and-who-installs-each)). Moving
an existing service onto the explicit chart interface therefore does not
rename its database in place: the infrastructure chart provisions a new
instance under **the platform's own naming**, and `database.host`
([platform.md §10](../contracts/platform.md#10-what-a-platform-passes-by-name))
names that instance, not whatever the service answered to before. There is
no value to pass that makes the chart keep the old name, because the chart
was never the thing choosing it — the platform was, and it names its own
cluster explicitly for exactly the reason
[platform.md §10](../contracts/platform.md#10-what-a-platform-passes-by-name)
gives: a chart that derived one instead would be a chart that only installs
on the one platform whose convention it guessed.

So a service carrying real data is not a same-database rename, and the
migration step above is not complete until that data has actually moved: a
dump and restore, or whatever bootstrap path the database operator supports
for adopting an existing database into a cluster it will manage from then
on, planned into a maintenance window like any other cutover with a single
writer.

## Checklist

- [ ] Service reads one configuration file, validated against a committed
      schema; secrets are names resolved through one `secrets` source.
- [ ] `main` is hand-written; no DI container remains.
- [ ] The chart takes every value in
      [platform.md §10](../contracts/platform.md#10-what-a-platform-passes-by-name)
      by name; nothing is derived from a platform convention.
- [ ] `lint/golangci-depguard.yaml` is pasted in, verified, and every finding
      is resolved or excepted with a comment; a central parity job can find
      it (or the repository is a documented exemption).
- [ ] The chart renders only the OpenTelemetry SDK's own variables for
      telemetry, and nothing else.
- [ ] A third, released chart carries the suite as a Job; the prober is
      wired if the service wants a continuous signal.
- [ ] The required check installs the published artifacts on the shared
      development cluster through the consumer's own tenant harness, not
      only on kind.
- [ ] The release publishes to the private registry and the private OCI
      registry, from the same OSS goreleaser shape, one job, images before
      charts.
- [ ] A service with existing data has a real migration plan (dump and
      restore, or an operator-supported import) into the platform-named
      database, in a maintenance window — not a value that tries to keep
      the old name.
- [ ] Go, npm and pip consumption are each set up the way their own
      ecosystem needs — no `GOPRIVATE`, an authenticated `@truvity` scope,
      and a fetch of the release asset — not copied from one another.
