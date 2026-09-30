# Changelog

What changed for someone consuming this repository, newest first, one
heading per hand-cut release (minor or major). The prose bullets are written
for a consumer; the commit subjects under them are the GitHub Release's own
list. Automatic patch releases get no heading: their notes are the GitHub
Release's, and the next hand-cut heading covers them. `## Unreleased

### Contracts

- **Automatic patch releases need no CHANGELOG heading.** C5 and
  [release.md §3](docs/contracts/release.md) now require a heading for every
  hand-cut (minor or major) tag only. A patch cut by the auto-release bot
  may leave `## Unreleased` open; its notes are the GitHub release's. The
  next hand-cut release closes `## Unreleased` into its own heading, covering
  everything since the previous hand-cut heading, patches included. A checker
  allows a missing heading only for `vX.Y.Z` with Z > 0 whose `X.Y` equals
  the latest heading's.

## v1.33.0 — 2026-09-29

### Features

- **The URL shortener's chart can verify the database server.**
  `database.tls.mode` (`require`, the default, or `verify-full`),
  `database.tls.rootCA.{configMapName,key}` and `database.clusterDomain`.
  Under `verify-full` the connection URL of `urls`, `redirect` and the
  migration carries `sslmode=verify-full&sslrootcert=<mounted root>`, the host
  is dialled by its fully-qualified service name (the form a server
  certificate carries) and the root ConfigMap is mounted as a directory into
  those three workloads, so a rotation is picked up. `require` renders
  byte-identically to before. The end-to-end suite still reaches the database
  through a port-forward, which cannot present the service name, so it keeps
  `sslmode=disable` against its own box database.
- **The URL shortener's `stat` no longer receives `DATABASE_PASSWORD`.** It
  has no database; the unused credential is gone from its Deployment.

## v1.32.0 — 2026-09-29

### Features

- **A publisher can authenticate to the broker with its workload identity.**
  The `nats` configuration fragment gains an optional `tls` object
  (`caFile`, `serverName`): the client dials TLS, presents the certificate
  the service's own `tls` block loads, and verifies the broker's certificate
  against a bundle of its own; the broker maps the identity in it to a user
  with its own permissions. New `transport.Identity.ClientTo(roots,
  serverName)` builds the client configuration (the certificate is re-read on
  every connect, a rotation drops no connection already made, the server is
  verified by chain and name). The URL shortener's chart takes
  `events.tls.{enabled,serverName,caConfigMap,caKey}`, for `redirect` only,
  off by default; when on, redirect sends no token, so a certificate the
  broker cannot map is refused instead of succeeding as the token's account.
  Turning it on needs `tls.mode` for redirect not `off` and a trust bundle
  ConfigMap for the broker. A service that does not declare the key ignores
  it: `stat` and `log` keep their tokens.

### Dependencies

- **chore(url-shortener): the `stat` component moves to OpenTelemetry
  instrumentation 2.31.1.** OpenTelemetry's OkHttp library dropped
  `OkHttpTelemetry.newInterceptor`; its only entry point is now
  `createCallFactory`, which returns a `Call.Factory` where the Connect Kotlin
  client wants an `OkHttpClient`. `stat` wraps the finished client (identity
  socket factory and trust manager included) and routes `newCall` through the
  instrumented factory, so the spans, metrics and trace propagation are
  unchanged and a test now proves an outgoing Connect call carries a
  `traceparent` and a client span. `opentelemetry.version` is pinned to 1.65.0
  next to the other Spring BOM overrides: without it the BOM resolves the
  OpenTelemetry API and SDK down to 1.49.0 under an instrumentation built on
  1.65.0.

### Fixes

- **A peer certificate must be a single-identity leaf.** The transport
  packages (Go, Python, and the Kotlin example's trust manager) now hold a
  peer to two rules of the SPIFFE X509-SVID specification they did not
  check: the leaf is not a certificate authority and may not sign
  certificates or revocation lists, and it carries exactly one URI name, of
  any scheme (before, a second URI of another scheme was skipped and the
  first `spiffe` one read). A certificate that fails either is refused, on
  the server side and the client side alike. In Python, pass
  `getpeercert(binary_form=True)` as `der=` to `verify_peer` so the
  authority check can run; the URI rule needs nothing new.

## v1.31.0 — 2026-09-29

### Behaviour change

- **feat(url-shortener): every component runs as its own ServiceAccount,
  always.** The shared application account is gone and there is no toggle
  back. `redirect`, `urls`, `web`, `stat` and `log` each run as
  `<release>-<component>` (renamable under
  `serviceAccount.components.<component>.name`; annotations under
  `.annotations`), and this chart now creates those five accounts plus the
  migration's, so rendering changes for everyone, goldens included. The
  chart's own grants follow its call graph: `urls` admits exactly `web`'s
  and `stat`'s accounts, `redirect` admits nobody internally, and `web` and
  `stat` accept an answer only from `urls`' own account; external
  `tls.peers` grants are unchanged.
  **`log` keeps the cloud binding:** its account is
  `serviceAccount.app.name` when that is set (with `app.annotations`, on
  that account only), else `<release>-log`, so a platform that bound a cloud
  role to the app account for the archive bucket changes nothing. Nothing
  else runs as that account any more. A platform that bound anything ELSE to
  the old shared name (for example broker permissions for `redirect` or
  `stat`) must bind the new names, or set
  `serviceAccount.components.<component>.name` to the old one for at most one
  component. Two components on one name, or one on the migration's, is
  refused at render. The e2e prober's and Job's `tls.peers` must name
  `<release>-urls` and `<release>-redirect`.
- **docs(contract): C14, each component runs as its own ServiceAccount.**
  A workload identity is namespace plus account, so a shared account makes
  components indistinguishable to an allow-list. Checked by the chart's own
  render test.

## v1.30.2 — 2026-09-29

### Fixes

- **fix(url-shortener): stat's peer check always failed after a successful
  call, so every click was counted up to maxDeliver times.** The Kotlin
  component checked the answering workload's identity in an OkHttp
  application interceptor, which has no access to the connection, so the
  check refused every answer, including from the right peer. The request
  had already been served by then: the click was counted, the message was
  not acknowledged, and the stream redelivered it until `maxDeliver` (five
  times), over-counting each click. The peer is now verified in the
  handshake, by a trust manager that checks the chain and then the
  workload identity against the configured peers, so a peer that is not
  admitted is refused before any request bytes are sent. Tests run a real
  server that requires a client certificate: an admitted peer is called
  exactly once, and a peer with another identity sees no request. Anyone who
  copied the interceptor-based check needs the same change. The Go and
  Python packages already verify in the handshake and are unaffected.
- **The e2e prober and suite now fail on over-counting.** Both waited for the
  click count to read the expected number and passed at the first look, which
  is why a click counted five times went unnoticed. The `stat` journey and
  the suite's stat test now require the count to stay at the expected number
  for a hold afterwards, long enough to outlast a redelivery (the consumer's
  ack wait is 30s). In the e2e chart the prober's hold is
  `prober.statSettle`, default `60s`, `0s` to turn it off; a pass therefore
  takes about that much longer, and `prober.interval` is the pause between
  passes.

## v1.30.1 — 2026-09-29

### Fixes

- **fix(url-shortener): stat crashes with NoSuchMethodError (kotlinx-coroutines
  ABI).** v1.30.0 moved the Connect Kotlin client to 0.9.0, which is built
  against kotlinx-coroutines 1.11.0 and OkHttp 5.4.0, while the Spring Boot
  BOM in the stat build silently kept coroutines at 1.8.1. The first call to
  the urls service died with `Job.cancel$default` missing and never
  completed. Both are now pinned to what the client declares, and a test
  makes a real call through the client so a bump of one without the other
  fails the build. Anyone who copied the stat build's Gradle setup with
  connect-kotlin 0.9.0 needs the same two pins.

## v1.30.0 — 2026-09-29

### Contracts

- **The platform contract names four more values the explicit chart
  interface can hand an application chart.** §10 adds
  `identityProviders.<name>.issuer_url`/`.client_id_list` (the workforce
  issuer and audience for a chart that verifies bearer tokens itself),
  `access.issuer`/`.audience`/`.signOutUrl` (for a chart behind a gateway
  that signs the browser in and forwards a bearer — the platform names
  the audience from its own client registration, never from a naming
  convention the chart could derive), and `route.surfaces[].name`/
  `.hostname`/`.parentRef` (additional named routes, on a separate
  hostname or the primary's). The existing `postgres.serverTLS` row now
  says the platform creates the server certificate and the CA secret
  itself — the chart only ever receives the two Secret names and must
  not create either object. A new non-normative note documents `product`
  as an escape hatch for values no platform is better placed to name than
  the product's own configuration, and says a chart must not rely on it
  for anything this contract should name instead.

### Example

- **The url-shortener chart takes a transport mode per serving component, so
  the URL service can be `strict` while the redirect service stays
  `permissive`.** `tls.components.urls.mode` and
  `tls.components.redirect.mode` (`off | permissive | strict`, unset means
  `tls.mode`) override the release-wide default for the two components that
  serve an authenticated boundary; `web` and `stat` only call out and have no
  mode of their own (they present an identity whenever the URL service
  authenticates, and dial its port for its effective mode). The schema
  refuses `strict` for `redirect` and a key for any other component, and the
  render refuses a `strict` redirect that is only inherited. Default renders
  are byte-identical.
- **Behaviour change: `tls.mode: strict` on the url-shortener chart is now
  refused unless `tls.components.redirect.mode` is set.** The redirect
  service is fronted by a gateway that terminates TLS and forwards
  cleartext, so a strict redirect refuses its only caller; the render says
  so and names the fix. To make only the URL service strict, set
  `tls.mode: permissive` and `tls.components.urls.mode: strict`. The same
  rule applies to the e2e chart's `tls` block, which gains the same
  `tls.components` (so the prober dials `urls` and `redirect` each on its own
  port) and refuses a `strict` redirect target, written or inherited.
  `hack/identity-smoke.sh` now proves the counter over `strict` for the URL
  service alone.
- **The url-shortener-e2e suite Job can present a workload identity.**
  `job.tls.enabled` (off by default; byte-identical when off) mounts the CSI
  identity into the Job, requests it as the Job's own ServiceAccount (with a
  Role to ask for it, unless `tls.grantRequest` is false) and hands the suite
  `E2E_URLS_TLS`, `E2E_REDIRECT_TLS` and `E2E_TLS_*`, which the suite reads
  through the new `e2e/tlsenv` package so every call it makes presents the
  certificate and dials the target's port for its mode. It reads the same
  `tls` block as the prober. As with the prober, the consumer adds the Job's
  ServiceAccount to the application chart's `tls.peers.urls` and
  `tls.peers.redirect` before making `urls` strict; see
  `docs/guides/testing.md`.

### Documentation

- **New: `docs/landscape.md`, one page for the 20 public repositories the
  component contract applies to.** A table grouped by layer (identity and
  secrets, edge, data, observability, CI, cluster add-ons, developer
  tooling, doctrine) names each repository's purpose, what it ships and its
  latest tag; a "How they fit" section draws the boundary between
  neighbouring layers from each repository's own `Neighbours` section. The
  table is generated, never hand-maintained — `just landscape`
  (`hack/landscape.sh`) re-fetches it from the GitHub API — so it is linked
  from both `README.md` and `docs/README.md` rather than restated in
  either.

## v1.29.1 — 2026-09-29

### Contracts

- **The component contract names three exceptions and how a repository
  declares one.** C2 exempts a `type: library` chart (it renders no
  values.yaml surface of its own); C9 exempts a fork of a non-MIT
  upstream (a derivative work cannot relicense itself); C11 clarifies
  that a component named after its own repository is only wrong when it
  is the sole image published under that prefix. A repository declares an
  exception in `.github/policy-conformance.yaml`'s new `exempt:` map — see
  the contract's new [Exemptions](docs/contracts/component.md#exemptions)
  section — which `truvity/ci-actions`' `policy-conformance` action now
  reads.


### Security

- **`examples/url-shortener/web`'s `@opentelemetry/exporter-metrics-otlp-http`
  and `@opentelemetry/exporter-trace-otlp-http` bumped to v0.222.0**
  (from v0.205.0), which pulls `@opentelemetry/core` to v2.11.0. The
  version in use resolved `@opentelemetry/core` v2.1.0, affected by
  GHSA-8988-4f7v-96qf (unbounded memory allocation parsing an inbound W3C
  `baggage` header); fixed upstream in v2.8.0.

### Changes

- **New: `lint/biome.base.jsonc` and `lint/.editorconfig`, beside the
  existing `lint/golangci-depguard.yaml`.** A repository's own
  `biome.jsonc` extends the Biome base — indent, quotes and line width
  chosen to match code nobody wrote against this file, and Biome's
  `recommended` lint preset with nothing added, since the canon
  ([node.md](docs/canon/node.md)) names no TypeScript rule of its own.
  `.editorconfig` is copy-only, like the depguard block: EditorConfig has
  no `extends`. This repository dogfoods both — `ts/biome.jsonc` and
  `examples/url-shortener/web/biome.jsonc` extend the base, and the root
  `.editorconfig` is the copy — and `just lint` runs `biome check` and
  `editorconfig-checker` (the universal checks only; indentation WIDTH
  stays each language's own formatter's job). See
  [lint/README.md](lint/README.md).
- **`charts/url-shortener-e2e` can give its prober a workload identity.**
  A new top-level `tls` block (off by default, byte-identical render
  unless turned on) mounts a CSI identity and presents it when the prober
  calls `urls` and `redirect`, on the same terms
  `charts/url-shortener`'s own client-only components (`stat`, `web`)
  already use. This is what lets the application chart's `urls` component
  move to `tls.mode: strict` without leaving the prober calling cleartext
  against a listener that no longer serves it — see
  [`docs/guides/testing.md`](docs/guides/testing.md#the-prober) for how to
  turn it on and what the application chart's own `tls.peers` needs.

### Documentation

- **New guide: `docs/guides/private-consumer.md`.** What a private
  repository keeps identical to a public consumer (the service contract,
  the chart interface, the telemetry variables, the depguard copy, the test
  chart and prober shape), what it replaces (the PR gate moves to the
  shared development cluster per 0005; the release goes to a private
  registry; Go/npm/pip package access each work differently), and the order
  to migrate an existing service onto the shape.

## v1.29.0 — 2026-09-29

### Contracts

- **New: the component contract,
  [`docs/contracts/component.md`](docs/contracts/component.md).** The
  rules C1–C13 every public repository that ships charts, images, Go
  libraries, Pulumi components, CLIs or actions is held to. It replaces
  the two doctrines that used to live beside the shared CI
  (`ci-workflows/docs/component-contract.md` and
  `ci-plane/docs/normalization.md`); where they disagreed, it decides:
  a committed chart version is `0.0.0` (`0.0.0-dev` is retired), and the
  CHANGELOG has one `## vX.Y.Z` heading per tag (grouped headings are
  retired).
- **New: [`docs/glossary.md`](docs/glossary.md)** — estate, platform,
  ring, tier, lane, component, service, consumer and the environment
  names, as these documents use them.
- **Every contract carries a version header** (`Version: 1.0 ·
  Effective: 2026-09-29`). A change to a contract is an entry under this
  sub-heading; [`release.md`](docs/contracts/release.md) §3 says so.
- **`repository.md` §2 names what `check` actually runs** — `build`,
  `test`, `lint`, `drift` and `leak-canary` — and says it needs no
  credentials and no cluster, rather than no network: a first build
  downloads modules. `vuln` is its own scheduled workflow, never part of
  the gate (component contract C10).
- **`release.md` §7 cites the right rule** for the kind install:
  [decision 0005](docs/decisions/0005-kind-is-the-gate.md) and rule 2 of
  the repository contract, not rule 6 (which is dependency bumps).

### Changes

- **The example's Go services log to stderr, not stdout.** The service
  contract (§4) has always said stderr; `internal/runtime/runtime.go` wrote
  to stdout. A collector that reads both streams sees no difference; one
  that reads only stdout stops seeing the Go components' logs, as it
  already did not see the Kotlin, TypeScript and Python components'.
- **Every image base is pinned by digest.** The four Dockerfiles, the
  release's `kos:` block (which never read `.ko.yaml`, so the Go images
  were built on whatever `latest` was) and `.ko.yaml` name
  `cgr.dev/chainguard/<image>:latest@sha256:…`; renovate moves the pins.
- **`just check` no longer runs `vuln`.** Vulnerability scanning is its own
  workflow, `security.yaml`, on push, pull request and a daily schedule —
  a new advisory no longer turns every pull request red. Every
  `devbox.json` entry names a version, and the toolchain lint refuses
  `latest`.

### Documentation

- **The README answers a newcomer's questions in the component contract's
  order**, with an install section pinned to a tag, `Consumers` and
  `Neighbours` sections, and a `Status` that is true: the repository has
  been tagged since v0.1.0, the Kotlin loader is not published, and two
  guides remain. The Python wheel and the Kotlin loader join the "What
  ships" table.
- **Stale claims removed**: a `just cluster-smoke` recipe that no longer
  exists, an `infra.enabled` flag replaced by the two-chart split, guides
  listed as "still to come" that shipped, a counter described as Go that is
  Kotlin, and `just ts-schemas` (the recipe is `just schemas`).
- **This file has one heading per tag.** It stopped at v0.4.6 behind three
  `Unreleased` sections while v0.4.7 to v1.28.7 shipped. Each of those
  sections' bullets now sits under the first tag that contained it, and
  every tag lists its GitHub Release's commit subjects.

## v1.28.7 — 2026-09-29

- fix(url-shortener-e2e): skip the archive check's S3 lookup in-cluster before it shells out

## v1.28.6 — 2026-09-28

- **The `url-shortener-infra` chart tags every AWS resource it creates
  with `project`, not just `cluster`.** Found on an install whose
  platform's permissions boundary denied `iam:CreateRole` outright
  because the Role's request carried `cluster` but not `project` — a
  boundary that requires both, common enough to be worth the chart
  carrying by default rather than every consumer discovering it the same
  way. The Role, the Policy, the Bucket's own `tagging.tagSet` and the
  PodIdentityAssociation all get both tags now. New, optional
  `cloud.project` value: unset (the default), it is this release's own
  namespace, which is what "the project" already means on the platforms
  this chart targets; set, it wins. Also new: `cloud.tags`, a map of
  extra tags folded into every one of those resources, which can never
  override `cluster` or `project`.

Commits in this release:

- fix(url-shortener-infra): tag every AWS resource with cluster AND project

## v1.28.5 — 2026-09-28

- fix(url-shortener e2e): wait for the PROMOTED rollout, not whatever generation was already live

## v1.28.4 — 2026-09-28

- **The url-shortener example's `urls` service no longer extends its
  span-attribute allow-list with `error`.** `urls` serves only Connect
  RPC through otelconnect and never imports `internal/api`, so nothing in
  it ever set that attribute — `redirect` is the one that does, through
  `internal/api.Tracing`, and keeps the extension.

Commits in this release:

- Wire the stat client's outbound span through the filtered OpenTelemetry SDK
- url-shortener: drop urls' unused error span-attribute extension

## v1.28.3 — 2026-09-28

- Assert no span attribute escapes its service's own allow-list on a live trace

## v1.28.2 — 2026-09-28

- **REMOVAL: the `url-shortener` application chart no longer has an
  `images.e2e` value.** Left over from the in-chart verification hook
  removed earlier in this file, it was unread by every template in the
  chart — the suite's image is stamped into the separate
  `url-shortener-e2e` TEST CHART's own `images.e2e` instead (see
  `docs/guides/testing.md`). `helmctl package` already narrows a
  release's image manifest to what each chart declares
  (`RestrictImagesToDeclared`), so dropping the key here only stops this
  chart from being stamped with a digest it never used; a chart install
  refuses an `images.e2e` value now (`additionalProperties: false`).

Commits in this release:

- Drop the unused images.e2e value from the url-shortener app chart
- Fix TestRedirectTraceCrossesEveryComponent to seed a real redirect

## v1.28.1 — 2026-09-28

- **The `url-shortener-e2e` chart has a new, optional `job.annotations`
  value** (empty by default), rendered on the Job's own metadata only,
  never the pod template's. A Job's spec is immutable, so a GitOps
  controller applying this chart with a values change at an already
  deployed version — turning on `traces.tokenExchange`, say — fails to
  patch it; `job.annotations` lets a platform put its own
  force/replace-on-change annotation there for a controller that reads
  one to decide it may delete and recreate the Job instead.

Commits in this release:

- Let a platform set annotations on the url-shortener-e2e Job; fix a flaky transport test

## v1.28.0 — 2026-09-28

- **The `url-shortener-e2e` chart's `traces.url` can now be an
  authenticated trace store.** New, optional values:
  `traces.tokenExchange.{tokenURL,client,audience,expirationSeconds}` and
  `traces.caConfigMap`/`traces.caConfigMapKey`. Set, the Job projects a
  ServiceAccount token and trades it at `tokenURL` for a bearer token
  scoped to `client` (RFC 8693 token exchange), which
  `examples/url-shortener/e2e/suite`'s `TestRedirectTraceCrossesEveryComponent`
  now sends as `Authorization: Bearer` when it queries the trace store —
  optionally verified against a CA bundle from `caConfigMap`, independent
  of the trust store the token exchange itself is verified against (the
  new `examples/url-shortener/e2e/traceauth` package doc comment explains
  why the two are never the same). Unset (the default, and the only shape
  the kind tier ever renders), nothing changes: the chart wires no auth
  and the suite sends no Authorization header, exactly as before.

Commits in this release:

- Let the e2e trace test authenticate to a private trace store

## v1.27.0 — 2026-09-28

- **REMOVAL: the `url-shortener` application chart no longer has a
  `verification` block, and no longer renders a post-install/post-upgrade
  hook Job (nor the ServiceAccount/Role/RoleBinding it ran under).** The
  hook duplicated what the separate `url-shortener-e2e` TEST CHART now
  does as a plain Job — see `docs/guides/testing.md`'s "The suite, as a
  released test chart" — with none of a hook's own sharp edges: a GitOps
  controller maps every `post-install`/`post-upgrade` hook to one phase
  and runs it on **every** sync, not only the first install or a version
  change, which a plain Job does not. This is a **breaking values
  change**: `verification.*` is refused by the chart's schema now
  (`additionalProperties: false`), not silently ignored. Run the e2e suite
  with the `url-shortener-e2e` test chart instead. The e2e suite itself
  keeps exactly two ways to run — outside-in
  (`e2e/fixture.Resolve`, the kind/local loop) and as the test chart's Job
  (`E2E_NAMES_FROM_ENV=1`) — the hook was a third, and is gone from
  `examples/url-shortener/e2e/suite` along with it.

Commits in this release:

- Remove the url-shortener app chart's verification hook

## v1.26.3 — 2026-09-28

- **The `url-shortener-e2e` chart's prober Deployment now sets
  `strategy: {type: Recreate}`.** With one replica, RollingUpdate's default
  `maxUnavailable` of 25% rounds down to 0, so a new prober version that
  cannot start (bad image, crash) left the OLD pod running indefinitely —
  still producing synthetic traffic, which hides a broken prober from any
  monitoring gate reading its metrics. Recreate stops the old pod first, so
  a broken rollout shows up as missing traffic instead. Not configurable —
  see `templates/prober.yaml`'s own comment.

Commits in this release:

- Recreate the url-shortener-e2e prober Deployment instead of rolling

## v1.26.2 — 2026-09-28

- **The `url-shortener-infra` chart's database and its two role names can
  now be derived from the tenant's namespace and installName, opt-in via a
  new `postgres.tenantScopedNames` (default `false`), instead of always
  being a fixed `url_shortener` / `_owner` / `_app`.** The chart's local
  Postgres server is shared by every install the same way its NATS broker
  already is, and only the stream was scoped to the tenant before this — a
  `primary` install's own dedicated CNPG Cluster never noticed, but two
  installs on a box with one shared Postgres (the kind lane's own fixture,
  standing in for that Cluster) collided on all three names, and the second
  install's fixture run reset the credentials the first one was already
  connected with. The default stays exactly what it always was — every
  existing consumer keeps its fixed names unchanged — and an explicit
  `postgres.database`/`ownerRole`/`runtimeRole` always wins over the derived
  name regardless of the flag. Kind/fixture installs opt in
  (`e2e/fixture/names.go`); nothing else needs to. See
  `templates/_helpers.tpl`'s `url-shortener-infra.postgresBase` for the
  exact derivation (a valid Postgres identifier, truncated with a hash
  suffix past 57 bytes so two tenants never collide even there) and
  `docs/guides/testing.md`.

Commits in this release:

- Scope the kind fixture's Postgres database and roles to the tenant

## v1.26.1 — 2026-09-27

- **The `url-shortener-e2e` chart's Job no longer sets
  `ttlSecondsAfterFinished` by default.** It is now optional, and UNSET
  unless a deployment sets it (still floored at 120s by the schema when
  it does). A GitOps controller applying this chart with self-heal on
  treats a Job that deleted itself as missing from the live state and
  recreates it — running the whole suite, including its DDL case, again
  on a loop, forever, at a fixed version. The chart already names the Job
  after its own version so a plain `apply` converges at every version;
  the previous version's Job is left for the next version's apply to
  prune, or for `helm uninstall`, never a TTL. See
  `docs/guides/conformance.md`, and a new conformance check
  (`examples/url-shortener/charts/conformance_test.go`) that fails on any
  plain (non-hook) Job a chart renders that still sets
  `ttlSecondsAfterFinished`.

Commits in this release:

- Stop the url-shortener-e2e Job from setting ttlSecondsAfterFinished by default

## v1.26.0 — 2026-09-27

- **The `url-shortener-e2e` chart can now run an always-on prober beside
  its e2e Job.** `prober.enabled` (default `false`) turns on a Deployment
  that walks the same three journeys the e2e suite proves once — create a
  short link, resolve it, watch its counter move — in a loop, against a
  release nobody is otherwise calling: a fresh install with no traffic
  always looks healthy, and a bake window needs steady signal to read
  before, during and after a rollout. It is a SEPARATE workload from the
  suite's Job on purpose (one proves a release IS healthy, once; the other
  proves it STAYS healthy) and shares its request-making code with the
  suite through a new package, `examples/url-shortener/e2e/journey`, so the
  two can never drift on what a journey even is. Metrics are OpenTelemetry
  (`probe_journey_total{journey,result}`, `probe_journey_duration_seconds{journey}`),
  wired the same way as every other component here (decision 0006): no
  endpoint configured, no export. A new image, `url-shortener/prober`,
  ships alongside the others from the same `.goreleaser.yaml`. See
  `docs/guides/testing.md`.

- **A third chart, `url-shortener-e2e`, ships alongside `url-shortener` and
  `url-shortener-infra`.** It renders a plain Job that runs the example's
  own e2e suite — the same `e2e` image the outside-in loop already runs —
  against an application release already installed in the same namespace,
  reading every name the suite needs (the database, its two roles, the
  runtime role's password Secret, the stream and its subjects and durable
  consumers, the archive bucket) as values rather than by rendering another
  chart. `mode: full | tenant` picks whether the one case that issues DDL
  directly at the database runs. Released, versioned and packaged the same
  way as the other two, by the same `.goreleaser.yaml` + `helmctl` flow.
  See `docs/guides/testing.md`, "The suite, as a released test chart".

Commits in this release:

- Add an always-on synthetic-traffic prober to url-shortener-e2e
- Add url-shortener-e2e test chart: run the suite as a Job after deploy

## v1.25.0 — 2026-09-27

- **This repository's own tag line now starts at v1.25.0, not v1.0.0.** The
  url-shortener chart continues a line that used to be published from
  another repository, whose last release was in the 1.22.x series; see
  `docs/contracts/release.md`.

- **A repository-wide chart test now enforces the rules a chart needs to
  behave the same whether it is installed with `helm install`/`upgrade` or
  rendered with `helm template` and applied directly, with no Helm release
  record at all — the shape a GitOps controller uses.** It statically scans
  every chart's templates for `lookup`, `.Release.IsUpgrade`/`.IsInstall`/
  `.Revision`, a value generated once in a template (`randAlphaNum` and
  friends, `genPrivateKey`, `genCA`, `genSelfSignedCert`,
  `derivePassword`), and a template selecting an object by the
  `meta.helm.sh/release-name` annotation — and asserts every rendered
  workload and Service carries the `app.kubernetes.io/instance` label
  instead. See `docs/guides/conformance.md` and
  `docs/guides/testing.md#two-install-paths-and-why-both-must-render-the-same`.

Commits in this release:

- Add render-and-apply chart conformance rules; set the next tag to v1.25.0

## v0.9.3 — 2026-09-27

- **The url-shortener example's e2e readiness check now finds a release's
  Deployments when the chart was rendered by a GitOps controller rather
  than installed by helm.** `TestDeploymentsAreReady` reaches
  `github.com/truvity/gemaal/pkg/harness`'s `ReleaseDeployments` and
  `WaitForDeployments`, which used to decide a Deployment's release only
  by the `meta.helm.sh/release-name` annotation `helm install/upgrade`
  stamps. A release deployed with `helm template` and applied directly
  never gets that annotation, only the standard `app.kubernetes.io/instance`
  label every chart carries — bumped to gemaal v0.24.1, which falls back
  to that label when the annotation is absent.

Commits in this release:

- fix(url-shortener): bump gemaal to v0.24.1 for the instance-label fallback

## v0.9.2 — 2026-09-26

- fix(url-shortener): verification hook no longer needs helm

## v0.9.1 — 2026-09-26

- **The url-shortener example's `log` archiver no longer crash-loops when its
  broker credential is rotated.** The credential is a short-lived file the
  platform replaces in place, and the archiver already re-read it on every
  reconnect — but `nats-py` treats a server closing the connection over an
  expired credential as fatal rather than retryable: it closes the client
  outright instead of handing the error to its own reconnect logic (unlike
  the Go and Kotlin clients against the same broker), so every fetch after
  that raised the same error forever. The archiver now dials again itself
  the one time this happens, which reads the credential file fresh, and
  only lets a second failure in a row surface. See
  `examples/url-shortener/log/src/url_shortener_log/pull.py`.

Commits in this release:

- Fix log archiver crash loop on NATS credential rotation

## v0.9.0 — 2026-09-26

- **Every exported span now passes through an attribute allow-list.** Each
  language's telemetry starter (`telemetry.Start` in Go, `telemetry.start()`
  in Python, `start()` in TypeScript) drops any span attribute that is not
  on a short default list of OpenTelemetry semantic-convention keys before
  the span leaves the process — an attribute nobody thought about is now
  ABSENT, not exported because some instrumentation library happened to add
  it. A service extends the list in code, as an argument to the starter;
  there is no configuration key or environment variable for it. The example
  service wires its own additions the same way (see the diff for exactly
  which keys). Kotlin's example wires the same rule through the
  OpenTelemetry Spring Boot starter's own extension point, needing no
  additions of its own. See `docs/guides/logging-and-telemetry.md`.

- **The cluster lane is now a required check for merges.** All tests, including
  the end-to-end suite, must be green before a pull request can merge. Run
  `just cluster-all` locally to verify before pushing.

Commits in this release:

- Drop span attributes not on an allow-list before export
- feat(url-shortener): optional post-sync verification hook
- feat(url-shortener): publish the e2e suite as a verification image

## v0.8.1 — 2026-09-26

- **The url-shortener example's cluster suite is now one Go program, not a
  shell script.** `just example-smoke` runs
  `examples/url-shortener/e2e/suite` in place of the retired
  `hack/smoke.sh`. It reaches every Service through
  `github.com/truvity/gemaal/pkg/harness` (>=0.24.0, which added a kind
  tier: a port-forward to the exact Pod behind a Service stands in for the
  direct ClusterIP a shared cluster reaches), asserts through Service
  endpoints only — never `kubectl exec` — and is meant to run unchanged
  wherever this example is next installed: a private repository's shared
  cluster, or a cluster after a promotion. See `docs/guides/testing.md`.
  Readiness is asserted directly — every Deployment of the release is
  Available with every replica ready, read through the harness's kubectl
  runner — rather than by probing `/health/live`/`/health/ready` through a
  Service: a Pod is only Ready once the kubelet has already run that exact
  probe, so a Service would only widen who can reach the probe listener
  without proving anything new. This also covers `stat` and `log`, which
  carry no Service of their own.

- **The kind lane now tests the release's own artifacts.** `just
  example-snapshot` runs `.goreleaser.yaml` itself — one architecture,
  pushed into the box's own registry — and packages the chart from what it
  pushed with `helmctl`, the same tool a release uses. `example-install`
  installs that packaged `.tgz`, never the chart's source directory. The
  old `example-images` recipe (`ko build` + `docker build` + `kind load`, a
  second, independent build path) is gone, and with it every `kind load`
  and every override that existed only for it — a packaging bug can no
  longer ship while this lane stays green, which happened once. See
  `docs/guides/testing.md` and `hack/kind/README.md`.

Commits in this release:

- feat(kind): install the release's own artifacts, not a second build
- feat(url-shortener): replace hack/smoke.sh with a Go e2e suite
- fix(ci): fetch full history for the cluster job's release build
- fix(ci): one devbox run for cluster+snapshot, and wait for the registry
- fix(kind): exec into nats-box for the NATS check, not run --rm -i
- fix(kind): make the NATS round trip race-free, not merely pod-ready
- fix(url-shortener): readiness via Deployments, not a widened Service

## v0.8.0 — 2026-09-26

- **The local cluster (`hack/kind`) installs no infra-shaped chart.** It now
  carries servers only — a plain Postgres, NATS with JetStream, an S3
  stand-in, a local registry — and no operator: no CloudNativePG, no NATS
  controller, no cert-manager. `url-shortener-infra` still renders, still
  has goldens, and is still validated against the Kubernetes API's own
  schemas; it is simply never installed here, because doing so proved one
  example's platform choices on infrastructure meant to outlive any one
  example. What it would have provisioned is now the worked example's own
  fixture (`examples/url-shortener/e2e/fixture`), which reads the names it
  needs off the charts rather than repeating them. Workload-identity testing
  (`example-identity`) is removed from this box for the same reason and
  moves to another tier; `identity-smoke.sh` is unchanged and not yet run
  anywhere.

- **The url-shortener example's JetStream stream, its subjects and its
  durable consumer names are now tenant-scoped, not global.** They used to
  be a fixed name (`URL_SHORTENER`, `url-shortener.redirect`), which is
  harmless on a disposable cluster and a silent collision on a shared one:
  two installs that agreed on that name — two CI runs in one namespace, or
  two engineers who each called their copy by the project's own name —
  found each other's stream, and neither install failed. Both charts now
  compute every one of those names from one formula, `installName`
  (defaulting to the release name) plus the release's namespace, so that
  sharing either alone no longer collides. A chart test renders two
  installs, varied one way and then the other, and asserts neither shares
  a name with the other. `docs/guides/events.md` has the rule and the
  trap; `docs/contracts/platform.md` rule 6 has the naming half of "found,
  not made".

- **A log line can be followed to its trace, and a trace to its log lines.**
  Every JSON log record written while a span is current now carries
  `trace_id` and `span_id` — lower-case hex, the W3C forms, the names the
  OpenTelemetry specification recommends outside OTLP — as top-level
  fields, in all four languages. The fields are absent, not empty, when no
  span is current. Go reads the span from the record's own context, so
  `log.InfoContext(ctx, ...)` carries it and a call with no context carries
  nothing; Python and TypeScript read the active OpenTelemetry context, and
  stay optional extras — the fields are simply absent where the SDK was
  never installed; Kotlin wires the OpenTelemetry Logback MDC
  instrumentation, which the Spring Boot starter does not bring in on its
  own. No configuration key and no enable flag: the correlation follows
  whether a span happens to be current, the same way the trace itself does.
  `logging-and-telemetry.md` gains a section naming the fields and the
  degradation.

Commits in this release:

- Fix kubeadmConfigPatches to use v1beta3 extraArgs format
- Slim the kind box to servers only; give url-shortener its own e2e fixture
- Turn off leader election on the single-node box
- feat(logs): carry trace context (trace_id/span_id) onto every log line
- fix(url-shortener): pair the two releases' installName on the local box
- fix(url-shortener): scope JetStream names to namespace and install

## v0.7.6 — 2026-09-25

- **The archiver no longer crashes when the stream is quiet.** An empty
  fetch is reported by the client as the standard `TimeoutError`, and the
  archiver caught only the client's own subclass of it, so the first quiet
  second killed the process and the pod restarted in a loop whenever there
  was no traffic (5-8 restarts in an hour on a quiet cluster, shown as a
  "flaky" pod). It now catches the base class, in one small function with
  a test that fails on the old handler.

- **The redirect service's request spans are named for the route that served
  them.** Every span read `GET /`: the tracing middleware named it at the
  start, when the only route the router has matched is the middleware's own.
  It is now named after the handler has run, so a trace says which handler a
  request reached. Still the route, never the path.

Commits in this release:

- fix(log): a quiet stream is an empty batch, not a crash
- fix(redirect): name request spans for the route that served them

## v0.7.5 — 2026-09-25

- **The archiver's write now shows inside each request's trace.** The write
  and its S3 call are one span in one trace (a batch cannot be any single
  request's child), so a trace opened on a redirect ended at "received" and
  gave no sign the event had been archived. Each message now also gets a
  short `archive.write` child, timed to the flush and linked to it. The
  guide says why the S3 call itself stays under the flush.

- **`logging-and-telemetry.md` gains "A trace that stays whole"**, with
  pointers from the events, RPC and object-storage guides and a new item in
  the conformance review. Spans that exist per service but do not connect
  pass every exporter's health check, so the guide names the five places
  context is dropped (an RPC, a broker, a database, an object store, a
  thread hand-off), the two decisions that are not configuration (trust the
  caller or link to it; parent a single message but *link* a batch), the
  lower-case `traceparent` header NATS needs, and the check that proves it:
  fetch one trace by id and read its tree.

Commits in this release:

- fix(log): show the archive write inside each request's trace

## v0.7.4 — 2026-09-25

- **The example's traces are one graph, not one fragment per service.**
  Every service was exporting spans and no request could be followed
  across them, because each hop began a trace of its own. Five joins,
  each a place context was being dropped:

  - **web to urls**: the web server's client now makes a client span per
    call and puts the trace context on the request. urls **trusts** the
    incoming parent (`otelconnect.WithTrustRemote()`); the default is to
    only *link* to an untrusted caller, which is right for a service
    facing the internet and wrong for one whose callers are its own
    platform.
  - **redirect to the broker to stat and log**: the publisher writes
    `traceparent` into the message in lower case (NATS header names are
    case-sensitive, and an HTTP-style carrier would have written
    `Traceparent` where no other language looks). The Kotlin consumer
    continues the trace; the Python archiver records a span per message
    and one flush span that *links* to them, because a write of hundreds
    of records cannot be the child of any one.
  - **stat to urls**: the outbound call runs on another thread, and the
    current span lives in a thread-local, so the client span had no
    parent. The HTTP client's executor now captures the context where the
    call is enqueued.
  - **database and object store**: every query is a span under its
    request (values are not recorded: they are the URLs people
    shorten), and every S3 call is a span under the flush.

  The header case and the thread hand-off are each covered by a test that
  fails without the change.

Commits in this release:

- Connect the example's traces across services, the broker, the database and the store

## v0.7.3 — 2026-09-25

- fix(migrate): the migration Job spans its run, the fourth service to need it

## v0.7.2 — 2026-09-25

- fix(stat): the service publishes the SDK itself; the chart stays generic
- fix(ts): import the module under test the way this library does
- fix(ts): the Node SDK ignored OTEL_SERVICE_NAME, so the service was filed as a stranger
- fix(urls): shortening a URL twice is the same link, not a panic

## v0.7.1 — 2026-09-25

- fix(urls): Create generates a key when none is given, as advertised

## v0.7.0 — 2026-09-25

- feat(charts): the chart can mint the runtime role's credential
- fix(charts): the generated secret uses the given name, not a derived one
- fix(log): the archiver spans its flush, the same gap as the other two
- fix(stat): the counter was completely dark, not just missing a span
- fix(telemetry): installing exporters is not instrumenting

## v0.6.0 — 2026-09-25

- feat(web): the front end is a front end again
- fix(events): read the broker token on every connect, and resubscribe

## v0.5.0 — 2026-09-24

- **The infrastructure chart provisions the objects an install owns**, and
  takes a `tier` that decides whether it provisions them at all. A `test`
  install mints nothing and runs as the namespace's standing identity; a
  `primary` install makes its own store and the identity that reaches it,
  from names it is given rather than names it derives.

  Those resources had been moved into a stack that runs per cluster. The
  reason that was wrong is scope: there is no "the install" at cluster
  scope, so it served the deployment and left every engineer's copy and
  every CI run with nothing.

- **`platform.md`'s credential advice is corrected.** It recommended "a
  credential the database operator issues for the role it already
  manages"; the operator issues no such thing, and its managed roles take
  a password or no password. The recommendation stands and now says what
  it costs: who owns the CA, why one arrangement makes you take over
  replication's identity, and why splitting the two CAs breaks verifying
  the server. The example still uses a password, because the issuer is a
  platform's to provide.

- **The default is unchanged and the default render is byte-identical.**
  `tier` defaults to `test`, so nothing here is reachable until a
  platform asks for it by name.

- **`platform.md` §11 no longer says a chart may not name a vendor.** The
  bar is that it RENDERS without that cloud, not that it installs on one,
  and the tier is what keeps that honest.

Commits in this release:

- feat(charts): ring2 owns the objects the install owns, credential included
- fix(charts): route the front end, and keep the resolver a separate rule
- fix(charts): withdraw the certificate mode, which nobody can turn on

## v0.4.8 — 2026-09-24

- fix(charts): the managed role declares what the API server defaults

## v0.4.7 — 2026-09-24

- **v0.4.6's `url-shortener-infra` chart cannot be installed. Use v0.4.7.**
  It reached the registry carrying a top-level `images:` map it never
  declares, and its schema sets `additionalProperties: false` — which
  Helm checks before any template runs. So the chart refuses *every*
  install, including one that passes no values at all:

  ```
  - at '': additional properties 'images' not allowed
  ```

  The chart source was never wrong. The packaging tool gave every chart
  in a release every image the build produced, which a repository
  publishing one chart never notices and this one, publishing two, did.
  Fixed upstream in the tool, so nothing here changed but the version of
  it that CI runs.

  Worth keeping as an example of the shape: the release packaged, pushed
  and went green, and the only signal was somebody trying to install the
  result. Rule 7 — *a published artifact is tested as published* — is in
  `contracts/release.md` because of this class, and the test that now
  guards it renders the **packaged** artifact, since rendering the
  source tree cannot see a defect that packaging introduces.

Commits in this release:

- chore(ci): ci-workflows v3.12.2, so the charts package correctly

## v0.4.6 — 2026-09-24

- **A client can authenticate to the broker.** The configuration contract
  has had `nats.tokenFile` all along, described as something the platform
  mounts — and nothing mounted it. Against a broker that authenticates
  its clients, every component that touches the stream failed at connect
  with `Authorization Violation`, which reads like a wrong password
  rather than a missing mount.

  `events.auth.audience` is an **audience**, not a credential: the chart
  projects a ServiceAccount token for it, the pod cannot forge one, and
  nothing here or in a values file is a secret. Empty renders no volume
  at all, which is a broker that admits anonymous clients — what a local
  one does.

- **The route declares the fields the API server defaults.** `group`,
  `kind` and `weight` on a backend reference are filled in if omitted, so
  a chart that leaves them out renders a route that never matches what is
  stored: a permanent difference, in every renderer that compares the
  two, for a route nobody changed.

  Declared rather than ignored. Telling a comparer to skip those fields
  silences the real changes underneath them, and declaring a default is
  not duplication — it is saying which value this chart wants, where a
  reader can see it.

## v0.4.5 — 2026-09-24

- **Every component exports telemetry**, in all four languages, read from
  OpenTelemetry's own environment (decision 0006). The chart carries an
  `otel` block in its values and renders `OTEL_*`; nothing reads
  telemetry from a configuration file and no schema changed.

  **No endpoint means export nothing** — not "export to localhost and
  retry forever", which is what an SDK left to its defaults does. The
  chart sets the exporters to `none`, so a laptop, a test and a cluster
  with no collector all do the same thing, and no component carries an
  enable flag. That flag is the failure 0006 records: a service that
  exported to a console in production because nothing set the
  environment name its code tested.

  **OTLP logs are off.** A node agent already collects stdout into the
  same store under the same namespace, so an exporter buys a second copy
  of what is there — and logs that exist only over OTLP vanish exactly
  when the exporter is what broke.

  In Python and TypeScript the starter is an **optional extra**: the
  loader is what every consumer takes, and a service reading a
  configuration file should not be made to carry an SDK it never starts.

  Go also exports **runtime metrics**, which cost nothing and make an
  empty store unambiguous — without a series that is always present,
  "nothing is arriving" and "this service is quiet" look identical.

## v0.4.4 — 2026-09-24

v0.4.4 was never tagged; what is listed here first shipped in v0.4.5.

- **A route can name its parent's KIND.** It could only ever attach to a
  Gateway, which is the API's default and silently wrong on a cluster
  that serves routes from something else. `route.parentRef.kind` and
  `.group` are the platform's to set, like the name and namespace beside
  them.

  Silently, because there is no good signal: a route whose parent does
  not exist is **Accepted** — the status says so and goes on saying so —
  and the service answers 404 with every pod healthy. The only other tell
  is the listener reporting zero attached routes, which nobody watches.
  Found in a cluster, by the 404.

## v0.4.3 — 2026-09-24

- **v0.4.2 did not publish.** Its Go images went to `ghcr.io/truvity`
  rather than to this example's repository, because `KO_DOCKER_REPO`
  silently overrides a `repositories:` named in the release
  configuration — and it failed only afterwards, on an SBOM written to a
  repository nobody meant to use. The images are built the way the old
  script built them now: the destination in `KO_DOCKER_REPO`, the
  command's own name appended.

  Neither `goreleaser check` nor a snapshot build can see this: a
  snapshot publishes those images to `ko.local` whatever repository is
  named. It is the rule in `release.md` §7 catching its own author.

## v0.4.2 — 2026-09-24

- **The release is two tools and no scripts of ours.** GoReleaser builds
  and pushes every image and records what it pushed; helmctl reads that
  and bakes the digests into the charts. `publish-images.sh` and
  `chart-manifest.py` are both gone — the second of them reproduced a
  schema ocictl owns, inside the repository other repositories copy.

  **One job**, so the ordering cannot be got wrong: the charts are
  packaged from a file that does not exist until the images are pushed.
  v0.4.1 fixed that ordering; this removes the possibility of it.

- **One configuration, three loops.** The image destination, the tag and
  the GitHub-release switch are taken from the environment, so a local
  loop, a CI loop and a release differ in three values and never in what
  is built. One destination per repository — public to a public registry,
  private to a private one, never both.

- **The multi-architecture rule is a test, not a shell assertion.** It was
  a check that inspected images after pushing them, which could only fail
  once a release had happened and could not see the likeliest mistake — a
  platform quietly dropped from the list. It now fails in the gate, and it
  is joined by two more: no Dockerfile may execute anything while the
  image is assembled (which is what makes cross-building a file copy), and
  no registry may be written into the release configuration.

  The build context is the repository root for every image, because the
  release stages files into the build tool's context keeping their paths
  and a Dockerfile can only be written for one context. One `COPY` line
  both builds agree on beats two that can drift.

## v0.4.1 — 2026-09-24

- **The release pins every image by digest, and refuses to publish a chart
  that is not.** The build writes down what it pushed, the chart is
  packaged from that file, and `--require-image-digests` rejects a chart
  with an entry left blank. The image values changed shape to
  `images.<component>.{registry,repository,tag,digest}` — the shape that
  check reads. A chart spelling them any other way passes the check with
  nothing to check, which is worse than not running it.

  The jobs are in the other order now: images first, then the charts
  packaged from their digests. A digest exists only once the image is
  built, so a release that published charts first was always going to
  publish empty ones.


- **The published chart could not render a single Deployment.** Its own
  values said *the release stamps a digest per component here*, and the
  release does not: it publishes the charts and the images in the same run,
  and a digest only exists once the image is built. So the chart reached the
  registry with `digests: {}` and `tag: ""`, and every install of it failed
  at the first `image:` with `no image for web`.

  Nothing in this repository could see it. Every chart test supplied a tag
  or a set of digests, so all of them passed against a chart no consumer
  could use. It was found by installing the published artifact, which is the
  only place the difference exists.

  A published chart now falls back to its own **appVersion** — the version
  its release stamped, and the one thing such a chart always knows about the
  images built beside it. `image.digests` remains, and a deployment that
  needs a rollback to reach an exact image still sets it; what changed is
  that a chart with neither installs instead of refusing.

  There is a test for the published case now, and it fails against the old
  helper with the same message the cluster produced.

- **`platform.md` §10 corrected on the same point.** It claimed a release
  stamps digests into the published chart. It does not, and saying so made
  a chart that cannot be installed look like the intended shape. A platform
  still fills neither field; a *deployment* may pin digests, and that is a
  different actor making a stronger promise about one install.

## v0.4.0 — 2026-09-24

- **`platform.md` §10: what a platform passes a chart, by name.** Rules 1 to
  9 say what a chart may ask for; nothing said what a platform hands it, and
  the gap between those two is where a repository ends up satisfying the
  whole service contract and still being undeployable.

  Found by taking these charts to a real delivery layer and discovering that
  the values it renders and the values these charts read have zero keys in
  common — not a spelling difference, two unrelated interfaces. One side
  passes addresses; the other derives them from a naming convention. The
  second renders perfectly and installs on exactly one platform, and nothing
  says so until the second platform tries.

  Two tables, one per chart, each row naming the decision it answers. The
  test is rule 1's: hand the table to a platform that shares no naming
  convention with the first, and a row it cannot fill is a convention
  wearing a value's clothes.

- **The infrastructure chart takes the platform's decisions.**
  `postgres.labels`, `postgres.scheduling`, `postgres.backup`,
  `postgres.serverTLS` and `events.account`, named for the decision rather
  than for the operator's field — so a platform running a different database
  operator can still say *these instances belong on that pool*. Every one
  defaults to nothing: installing this on an empty cluster renders exactly
  what it rendered before.

  Archiving is a **name**, not a description. The archive is a resource the
  platform made, with its own retention and credential model; a chart that
  described one would be describing the wrong one on every platform but the
  one it was written against, and the difference is only visible when
  somebody tries a restore.

  `events.account` and `events.url` are alternatives rather than a pair. An
  account carries both the broker and the identity, so a server list beside
  one is the chart arguing with the broker about an answer the broker
  already has — and that argument is resolved silently.

- **Rule 6 is checked by a test now, not by reading.** Both charts render in
  the suite and neither may produce the other's kinds: no workload from the
  chart with a separate lifetime, nothing the application chart should be
  finding rather than making. That ordering was discovered by building it
  the other way first, and nothing but a test would notice it being undone.

  Embedding the second chart paid immediately: the platform's labels were
  being written *above* the chart's own rather than merged, producing a
  duplicate `app.kubernetes.io/instance` line. Helm renders it, the API
  server keeps the last, and which one that is depends on the order a
  template happens to write them in.

## v0.3.0 — 2026-09-24

- **The charts are published.** The image repository has pointed at ghcr
  since the first commit and the images have been published since v0.2.0;
  the charts that install them were published nowhere, so nothing could
  install this. Both of them, because they are a pair: Helm runs every hook
  before anything else in the same release, so the chart that migrates a
  database cannot be the chart that creates it.

- **The chart check is called `charts`, which is what the repository
  contract says it is called.** It shipped as `kubeconform` — an accurate
  name for the tool and the wrong name for the recipe. The contract fixes
  these names precisely so that a person moving between repositories, and
  anything automating across them, does not have to read a recipe file to
  find out what the chart check is called here; a repository that has the
  job under another name has made every caller special.

  Found by reading the contract back against the repository that publishes
  it. The failure is invisible from the inside — everything runs, the gate
  is green, and only a caller from outside notices — so the conformance
  guide now says to list the fixed names against `just --list` rather than
  assume them.

## v0.2.0 — 2026-09-24

- **Which cluster a repository tests against is not a preference.** A public
  repository stands up a local one because it is forced to: a fork's pull
  request must never reach your infrastructure, and a contributor has none of
  it. A private repository has no such constraint, and taking the local
  cluster anyway costs it the thing it actually needs — its CI can reach a
  shared development cluster, which has the identity plane, the provisioning
  and the network policy a local one cannot. Standing up a throwaway cluster
  to avoid one that is already there trades a better test for a slower one.
  The suite does not change either way, which is the point of writing it
  against a chart and a set of probes rather than against an environment.

- **Chart goldens, and the second one is the one that earns its keep.** A
  `minimal` render records what the defaults produce, so a changed default is
  a diff in a review rather than a surprise in a cluster. An `everything`
  render sets every value to something other than its default, so a template
  that stopped READING one shows up — which nothing else in the package would
  catch, because a test that asserts a particular key says nothing about the
  other four hundred lines. The failure names the first line that moved,
  rather than printing a thousand.

- **The renders are validated against the Kubernetes API's own schemas.**
  That is a question the chart tests do not ask: they check that a rendered
  file is one the BINARY accepts, and would pass just as happily for a
  Deployment with a misspelled field — because the API server ignores one
  rather than refusing it. Proved by misspelling one. It reads the committed
  goldens, so what is validated is the render a reviewer actually read.

- **The example has a front end, and it is the fourth language.** It serves a
  page and asks the service that owns the tables; it writes nothing and holds
  no database credential, which is the ownership rule seen from the consuming
  side. Its deployment has no password block at all while the two Go services
  do — the difference is visible in the chart, which is where it should be.

- **One code generator for TypeScript, not two.** connect-es v2 builds a
  client from the service descriptor `protoc-gen-es` already emits, so the
  separate Connect plugin the earlier line needed is gone.

- **The runtime image carries no `node_modules`.** The server is bundled into
  one file, which is not only tidiness: the dependency on this repository's
  own loader is a symlink in a checkout, and a symlink is not a thing that
  can be copied into a container. Two things had to be got right for the
  bundle to run — a dependency reached through its CommonJS build calls
  `require` for a Node builtin and an ES module has none, so the bundle
  starts with one; and generated imports must end in `.ts`, because the
  server runs TypeScript directly and Node resolves the path it is given.

- **A fourth loader, in Kotlin**, read against the same fixtures as the other
  three. An empty file parses to a MISSING node on the JVM rather than a null
  one, and checking only the second let it reach the validator — which
  reported "unknown found, object expected". Accurate, and no help at all to
  somebody looking at a blank file.

- **The counter is a Spring Boot service now, and the Go one is deleted.**
  The example has four languages in it and the chart still does not know
  which is which: the same probes on the same paths, the same drain, the same
  account, the same configuration file validated against the same schema.
  Three things had to be got right for that to be true, and each was wrong
  first:

  A framework that serves no traffic still has to stay running. With the web
  application turned off entirely the process started, consumed once and
  exited — and the probe listener never started either, so the symptom was a
  pod reporting "drained" a second after it reported "consuming". The main
  listener is disabled and the management one is not.

  The configuration file is read ONCE, by the composition root. A bean that
  re-read it would have to rediscover the path, and the first version did
  exactly that and found nothing, because the path arrives as an argument
  and a bean has none.

  And the refusal happens before the framework starts. A configuration this
  service will not accept should be one line on stderr, not a framework
  stack trace about a bean that could not be created — which is the same
  refusal with the answer buried in it.

- **The JVM presents an identity as a client**, with TLS 1.3, a certificate
  reloaded when the platform replaces it, and the peer admitted by the
  ACCOUNT in its certificate rather than by the address that answered. Proved
  under strict mutual TLS on a cluster, in the same gate as the Go and
  Python components.

- **The Connect generator for Kotlin is a JAR, not a binary**, so the build
  writes a launcher around it. Every other generator here is a static binary
  the environment manifest pins; this one is a JVM artifact, and pinning it
  in the manifest would mean pinning a jar as if it were a binary while
  pretending the JVM is not already on the machine.

- **The example has an ownership boundary, and all three shapes are now in
  it on purpose.** A new service owns the URL tables; the counter asks it
  instead of writing them. The counter's configuration has no `database`
  block at all, and a test asserts that it does not — the ownership rule
  usually shows up as an absence rather than as a line of code. What used to
  be a database password with write rights on a table it did not own is now
  an address.

  The contrast is the interesting part and it is documented rather than
  tidied away: an RPC for a boundary of ownership, an event for fan-out, and
  a direct read on the redirect path because that is the hot path and a
  second network hop on it is not worth what it buys.

- **One handler serves Connect, gRPC and gRPC-Web on one port**, so the
  protocol is the caller's choice. The smoke test proves both ends of that:
  the counter's call arrives as gRPC, and the same boundary answers an
  ordinary `curl` POST with a JSON body. The second is worth a test because
  it is the difference between a boundary anyone can ask a question of and
  one that needs a generated client.

- **The schema is a file with a linter on it, and the generated code is
  committed.** A field renumbered by hand is a wire incompatibility that no
  compiler catches, because both sides are regenerated from the same file in
  the same commit and agree with each other perfectly. Generation at build
  time was the alternative, and its first casualty is the editor, which
  cannot resolve a symbol that does not exist yet.

- **`listen` leaves the shared envelope**, for the same reason the transport
  block never joined it: a job exits and a consumer answers nothing, so a
  listener is not something every component has. This was found rather than
  reasoned about — the counter had been carrying a `listen` it never read,
  purely so that a test comparing its type to its schema would pass. A field
  every component carries and only some can use is a field a deployment sets
  and watches do nothing, which is the third time this repository has met
  that shape.

- **A client presents an identity too, and the gate now proves it.** The
  counter is the example's first in-cluster RPC client, so it is the first
  component that needs a certificate without serving one. Three things came
  out of making that work on a cluster rather than on paper:

  A chart's own internal callers are the chart's to grant. The allow-list
  value is for callers from OUTSIDE the release — leaving the internal one
  to an operator means a chart whose default configuration cannot talk to
  itself, and the error names a certificate rather than a list nobody
  filled in.

  `permissive` is meaningless for a component that serves nothing. There is
  no second listener to put anywhere, and rendering that mode produced a
  crash loop complaining about a listener address on a component with none.
  A client either presents an identity or it does not.

  And the two allow-lists are different questions. "Who may call this
  service" and "whose answer will this client accept" look alike enough to
  share a value, and must not: a client that checked only the certificate
  chain would accept any workload in the trust domain that happened to
  answer on that address.

- **The example's cluster scripts name the cluster they talk to.** Not
  whatever context is current — `kind create cluster` points the current
  context at whatever it just made, so a second box created in another
  terminal silently moves every command in the script. The symptom is
  "namespace not found" for a namespace that is right there, in the cluster
  you thought you were talking to. It also means the scripts cannot be
  aimed at a real cluster by accident.

- **The deprecated `h2c` wrapper is gone.** Cleartext HTTP/2, which a gRPC
  client needs when there is no TLS to negotiate over, is `Protocols` on the
  standard library's server and transport since Go 1.24. One fewer dependency
  on each side.

## v0.1.0 — 2026-09-24

The first version. It carries:

- **The repository skeleton.** Devbox toolchain, the `just check` gate, the
  leak canary on every commit and in CI, and hosted-runner-only CI.
- **The service contract and the configuration contract**, with the three
  decisions they rest on: hand-wired composition roots, a configuration file
  with secrets in the environment, and a schema with a hand-written type
  rather than a code generator.
- **The canon**: the Go library list, the Node and TypeScript list, a Kotlin
  stub written from the first JVM service rather than before it, the pinned
  toolchain versions with the reason each is a pin, and the build tools.
- **The repository and release contracts**: one product per repository, a
  gate that needs nothing but the checkout, fixed documentation paths, and
  what a public repository is held to on top; one tag stamping every
  artifact, what a version means read from the consumer's side, and the rule
  that adoption is proved by a byte-identical render rather than asserted.
- **The configuration schemas and the Go loader.** Seven shared fragments and
  the service envelope, embedded in the module so that validation needs no
  network; `config.Load`, which validates before it decodes and names the key
  that failed rather than the file; `config.Secret`, which reads the variable
  a configuration names and never the value it carries; and `conformance`,
  which holds a configuration type and a rendered chart to the same schema.
- **The TypeScript loader**, `@truvity/policy`: the same three calls as the Go
  one, carrying the same schemas, tested against the same fixtures and
  wording its refusals the same way, so that a misconfiguration reads
  identically whichever runtime refused it.
- **The import ban**, as a block a repository copies into its own lint
  configuration: no dependency-injection container, no configuration-mapping
  library, and the libraries the canon retired. Each entry names what to use
  instead, because a lint error that only says "no" gets suppressed rather
  than fixed.
- **A conformance guide**: what CI checks for you, what a reviewer checks,
  and the two rules that are checked by eye.
- **The local cluster**: a recipe that stands up Kubernetes with the same
  operators a deployment carries, a check that asks whether each thing is
  usable rather than merely installed, and a smoke test that proves an
  operator ACTS — a database becomes a database, a stream becomes a stream,
  a bucket becomes a bucket. About two minutes from nothing.
- **The worked example**, first three components: a migration job, the
  redirect service and the click counter, hand-wired against the contracts
  with no framework behind them. Each binary has one configuration file, one
  schema, and a test that the two describe the same fields.
- **The example's chart**, and the test that makes the configuration contract
  real: what the chart renders is validated with the schema the BINARY
  validates against at start-up, so the two cannot drift in the direction
  that matters. Five negative fixtures, one per refusal, each failing for its
  own reason.
- **`AGENTS.md`**, the entry point for a reader with no other context:
  what to read first, the four gate checks that surprise people, the writing
  rules, and a second set of instructions for bringing another repository to
  this shape — the survey-before-you-change order, what a survey of real
  services usually finds, and what never to do. Mirrored under whatever
  filename a particular tool looks for.
- **A Python canon**, starting from nothing because there was nothing to
  inherit. Deliberately short: no web framework, no RPC row and no ORM until
  a component needs one, and one weakness written down in advance — a Python
  server cannot swap a certificate underneath a running listener, so it
  recycles its workers or takes the proxy.
- **The Kotlin canon is no longer a stub.** It inherits the JVM stack the
  adopting estate already runs rather than choosing a lighter one, because
  the alternative is two JVM stacks and the newer one wins every later
  argument. The Kubernetes shape is not inherited; it comes from the service
  contract.
- **TypeScript is two lines, split by job.** The 7.0 compiler type-checks
  several times faster and ships no programmatic API, so type checking is
  7.0 and anything that drives the compiler stays on 6.0. Emit was never the
  compiler's job anyway. The framework's build command and its
  code-generating plugins are out of the canon, with the workaround for the
  first and the reason there is none for the second.
- **Fixed recipe names** in the repository contract, so that moving between
  repositories does not mean reading a recipe file to find the linter, and
  **every toolchain entry names a version** — "latest" makes a reproducible
  build a coincidence.
- **The transport is provable on the cluster, both ways.** The chart takes a
  `tls` block: an ephemeral volume the platform mounts an identity into, the
  in-cluster permission a workload needs to ASK for its own certificate, and
  a second port under `permissive` because one listener cannot be both. With
  the default `off`, the render carries no trace of any of it, which a test
  asserts by name rather than by golden.

  A new cluster step proves what only a cluster can: a caller on the list is
  served, a caller holding a REAL identity that is not on the list is closed
  at the handshake, and the service says which account it refused. All three
  halves are asserted — the second alone would pass for a service that
  refuses everyone, and without the third a refusal is indistinguishable from
  a service that is simply broken.
- **Verification is by identity, not by name.** A platform's workload
  certificate carries an identity and usually no host name, so a client
  builds the chain against the trust bundle and reads the identity out of the
  leaf itself. That means turning the standard library's own verification
  off, which looks alarming and is not: the comment sits next to the flag,
  because the next reader's first instinct will be to delete it. Two tests
  hold the property the flag would otherwise destroy — a server outside the
  trust bundle is refused, and so is one that chains correctly but runs as an
  account the client was not told to trust.
- **A pod security context, and the group is the point.** A driver writes
  what it mounts owned by root, so a process running as anyone else cannot
  read its own certificate. It surfaces as a permission error on a
  certificate authority file, or a complaint that a certificate is malformed
  — neither of which mentions identity, and both only once the transport is
  on.
- **The leak canary no longer fires on Kubernetes' own secret path.** Its
  parameter-store pattern matched `/var/run/secrets/`, which is where a pod's
  own credentials are mounted and is therefore in any manifest that reads
  one. A pattern that fires on the most common path convention in the
  ecosystem makes nobody safer: it teaches the next person to rename their
  mount to get past it, and the one after that to stop reading the output. It
  still catches a real parameter path, which is proved rather than assumed.
- **The local cluster can issue workload identities**, and asserts the one
  thing that makes them mean anything. cert-manager, its identity driver and
  that driver's approver, over a self-signed authority.

  **cert-manager's own approver is turned off, deliberately**, and this is
  the finding the box was built to produce. It approves every request for an
  authority it knows, so with it on the driver's approver never gets a say
  and any account that may create a request receives ANY identity it asks
  for — including its neighbour's. Nothing fails: certificates mount,
  services connect, every log line says success, and the attestation is
  decoration.

  Measured, not reasoned about: an account called `alice` submitted a request
  naming another account by hand and was issued a certificate for it. With
  the approver off the same request sits inert and nothing is issued. The
  verification step now asserts the flag, because the two states are
  indistinguishable from every other angle, and the platform contract now
  asks a platform to demonstrate the REFUSAL rather than the issuance.

  The consequence, found by CI rather than by thinking: turning that approver
  off turns it off for **everything**, including the authority's own
  bootstrap. A self-signed root expressed as a certificate needs its request
  approved like any other, and nothing was left to approve it, so the box
  never finished standing up. The box's root is a generated fixture now. A
  real deployment answers this with a policy engine; a throwaway root does
  not need one.
- **Mutual TLS, with the identity the platform gives.** A new `tls` fragment
  and a `transport` package that does three things and no more: load the
  mounted certificate and reload it when it changes, present it as a server
  and as a client, and admit a peer by the ACCOUNT it runs as rather than by
  the address it calls from. It never fetches or mints a certificate, because
  a workload that did would be asserting an identity rather than presenting
  one it was given.

  Three modes. `off` is the default and always will be, so a chart stays
  installable by someone whose platform provides none of this. `permissive`
  serves both on two ports, so an edge migrates one side at a time. `strict`
  serves only the authenticated port.

  Eleven tests over real handshakes, including the two that matter: an
  unlisted peer is closed at the handshake with the reason on the SERVER and
  a bare refusal to the caller, and a rotated certificate is picked up
  without a restart. Both mutation-checked.
- **A guide per aspect**, each with the rule, why it is that way, a table of
  where to look per language, and the traps — the failures that look like
  something else. Configuration, identity and secrets, events, probes and
  rollout, logging and telemetry, exposure, releases and images, and testing,
  indexed by [`docs/guides/README.md`](docs/guides/README.md). The remaining
  five arrive with the components that prove them, and the index says which,
  because a guide written before the code is a description of nothing.
- **The example proves the rollout rule instead of violating it.** Two
  instances of everything routed, a disruption budget, `maxUnavailable: 0`,
  spread across machines, and ONE drain number used three times — the
  service's own timeout in its configuration file, the grace period
  Kubernetes grants, and a pre-stop delay. A new `drain` fragment carries
  the first of those, so the number the chart renders is the number the
  process uses.
- **The example names the account it runs as**, and names two: the migration
  creates tables and grants rights, the services read and write rows, and
  one account for both puts the migration's rights on the request path. The
  chart still grants nothing — it names accounts and leaves their
  annotations open, so a platform binds them by whichever mechanism that
  cluster uses.
- **The route's rule is named**, because a policy attaches to a rule by name
  and a policy whose target names no rule is not refused — it is simply not
  attached, and the route keeps serving without it.
- **Six new chart tests**, each mutation-checked: a rollout with no gap, a
  grace period that outlasts the drain, every route rule named, every
  workload naming an account, the migration and the services on different
  accounts, and — the third instance of one trap — everything a pre-install
  hook references being a hook itself. The first version of that last test
  passed while the install hung, because the account and the job share a
  name and it keyed on the name alone.
- **A platform contract**, `docs/contracts/platform.md`: the other side of
  the seam. What a service asks of whatever runs it — names never values, an
  account and its annotations rather than a grant mechanism, secrets as
  Kubernetes Secrets, a store as an endpoint, a key operation with more than
  one provider, streams it finds rather than makes, a route whose parent it
  is given and whose rules are named — and what a platform owes back. Written
  so that a second platform, run by someone else, can satisfy it without a
  patch to any chart.
- **Logs move to stderr.** stdout is the program's product and stderr its
  commentary, which is the same split for a service, a job and a
  command-line tool, so a binary that grows a subcommand does not have to
  move its logs. The rule that follows is that a service writes nothing to
  stdout, and that every library which logs is wired to the service's logger.
- **The RPC rule says which protocol, not just which library.** The server is
  one Connect handler serving all three protocols; a client in the cluster
  speaks gRPC over cleartext HTTP/2, and speaks Connect over HTTP/1.1 or
  gRPC-Web only where HTTP/2 trailers cannot survive the path. A schema with
  nothing configured to generate from it is a client somebody hand-wrote.
- **Two new rules in the service contract.** A rollout replaces instances
  without a gap — two instances, a disruption budget, no unavailable
  replicas, and one drain constant used by the code, the grace period and a
  pre-stop delay alike. And transport identity belongs to the platform: a
  service presents an identity it is given, reloads it, and checks its peers,
  while the probes listener is exempt and the chart's default is off.
- **Three decisions.** Telemetry is configured by OpenTelemetry's own
  environment, which is the one exception to "configuration is a file" and
  removes the `otel` fragment nothing ever read. There is no service mesh:
  identity is the account a workload runs as, attested by the runtime rather
  than asserted by the workload, and terminated in process. And the twelve
  factors are a map to read these contracts by rather than a label to claim,
  with the two deviations argued instead of footnoted.
- **`nats` takes a `tokenFile`, not a credentials file.** It is the
  workload's own account token, re-read on every reconnect so a rotation
  needs no restart, and the broker asks an authorisation service who the
  bearer is rather than trusting what the client claims.
- **`bucket` takes a `ca`**, because a store inside somebody's own network is
  the ordinary case and is not signed by a public root.
- **The example runs.** Two releases, because a migration hook cannot wait
  for a database its own release creates: one for the database and the
  stream, one for the application. Two database roles with two credentials,
  because the migration creates tables and the services must not be able
  to. And a smoke test that asks the only question rendering cannot: a
  redirect is served, an event crosses the broker, and a counter another
  service owns moves by exactly the number of requests made.
- **The cluster tier is now a pull-request gate**, which is what decision
  0005 said it would become once something consumed it. Five defects it
  caught while the example was built are listed there.
- **The `nats` fragment now describes a CONNECTION only**, and a new
  `nats-consumer` fragment describes what a consumer binds to. The first real
  consumer is what showed that a publisher carrying a `consumer` field it
  never reads is a field somebody will eventually set.
- **A third loader, in Python**, read against the SAME fixtures as the other
  two. That is the point rather than a detail: a contract with one
  implementation is a library, and a contract whose implementations are
  tested against different inputs is two contracts wearing one name — a key
  one loader refuses and another accepts is a configuration that passes a
  chart's test and crashes the service.
- **A Python transport helper**, and the difference it cannot hide. Python's
  `ssl` module has no verification callback, so a peer cannot be admitted or
  refused during the handshake the way the Go package does it: the chain is
  verified by the library and the ACCOUNT is checked immediately afterwards,
  by the caller. The failure that shape invites is invisible — a service
  that builds the context correctly and never makes that call verifies a
  certificate chain and admits anybody holding one, with every other test
  still passing. The test for it is a stranger holding a genuine certificate
  from the same authority, in the same trust domain, for an account nobody
  granted.
- **The example's fourth component is not written in Go**, and almost
  nothing changes. It consumes the request records the redirect service
  publishes and archives them as NDJSON in an object store, and its
  deployment is twenty lines that never mention the language: the same
  probes on the same port, the same drain, the same account, and its
  configuration file validated by the same chart test as the other three. A
  platform that had to know which language a workload was written in would
  be a platform every new language has to be added to.
- **The `bucket` fragment has a consumer**, so "a store is an endpoint, not
  a vendor" is exercised rather than stated: name, region, endpoint, path
  style, certificate authority, and credentials by NAME. The local cluster
  points it at its own object store and the component cannot tell.
- **A batch is named by its first stream sequence and nothing else.** A
  batch is acknowledged only after its object is written, so a failed write
  means the same records are redelivered — and a redelivery begins at the
  same sequence, so it overwrites its own partial attempt rather than
  leaving a second copy beside it. A failed write KEEPS the batch, because a
  consumer that acknowledged what it had not stored would lose it for good.
- **A runtime image with no build step, in a language that has no `ko`.**
  The dependency tree is resolved from the committed lock and installed
  OUTSIDE the image; the Dockerfile is a copy and an entry point. A
  multi-stage build is not the same thing — it still runs a package manager
  while the image is assembled, which is exactly what makes a
  cross-architecture build need emulation.
