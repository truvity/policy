# Testing

**The rule.** Two tiers. A gate that needs no credentials, no container and
no cluster, and a cluster tier that runs the thing.
[repository.md §2](../contracts/repository.md) and
[0005](../decisions/0005-kind-is-the-gate.md).

**Why two.** A gate that needs a cluster is a gate people run once a week and
then argue with. A gate that never touches a cluster cannot see the failures
that matter — a resource nothing reconciles, a task that runs before what it
needs, an operator that accepted a manifest and did nothing.

## The tiers

| Tier | Needs | Runs |
|---|---|---|
| the gate | the checkout | `just check`: build, test, lint, vulnerabilities, drift, the canary |
| the cluster (required) | a container runtime | `just cluster-all`: stand up, verify, prove each operator acts, then install the example and exercise it |

Both run on ordinary hosted runners. Nothing here needs a machine of ours,
which is what lets a stranger reproduce the whole thing.

Beyond these two there is a third tier that belongs to a **private consumer**:
a real cluster with an identity plane, provisioning and cloud services. It is
deliberately not here, because the public repository must be runnable by
someone with none of that.

**Which cluster a repository uses is not a preference, and the two kinds of
repository should answer it differently.**

A **public** repository stands up a local cluster. That is forced rather than
chosen: a fork's pull request must never reach your own infrastructure, and a
contributor has none of it.

A **private** repository has no such constraint, and taking the local cluster
anyway costs it the thing it actually needs. Its CI can reach a shared
development cluster, which has the identity plane, the provisioning and the
network policy that a local one cannot have — exactly the third tier above.
Standing up a throwaway cluster to avoid one that is already there trades a
better test for a slower one.

The suite does not change either way. That is the point of writing it against
a chart and a set of probes rather than against an environment: the same
`install`, `smoke` and `identity` steps run in both, and only the cluster
they are pointed at differs.

## What the gate proves

Unit tests, and the two that are really contract tests: the schema and the
type describe the same fields, and **what the chart renders is validated with
the schema the binary validates against at start-up**. Without the second,
the chart keeps setting a key the binary stopped reading and the service runs
on a default nobody chose, with no signal but behaviour.

Rendering a chart needs no network and no cluster, so chart tests are part of
the hermetic gate rather than a separate job.

## What only a cluster proves

The gate cannot see any of these, and each one has happened here:

- a task that runs before the thing it needs exists — a database, a
  configuration file, an account — which fails on the *pod*, never created,
  while the install sits at "in progress";
- a custom resource nothing reconciles, which is accepted and stored and
  never becomes anything, and looks like success from the chart's side;
- a broker that reports healthy and refuses every stream, because a limit it
  reads only at start-up was never set;
- an operator whose setting is not hot-reloadable, so the fix is applied and
  has no effect until a restart;
- credentials that are correct and rights that are not.

## What the cluster installs is what a release ships

`just example-snapshot` runs `.goreleaser.yaml` itself — the release
configuration, not a second build path — for one architecture, into the
box's own registry, and packages the chart from what it pushed with the
same tool a release uses, `helmctl`. `example-install` installs that
`.tgz`, never the chart's source directory: the release contract's own
rule (docs/contracts/release.md §7) is that a published artifact is tested
as published, and a chart bug that only shows up once images are pushed
and digests are baked in is exactly what a source-directory install would
never see. See `hack/example-snapshot.sh` for the two things this loop
does differently from a real release — where the images go, and one
architecture instead of every one — and nothing else.

## Two install paths, and why both must render the same

The cluster lane here installs a chart the way `just example-install`
(`examples/url-shortener/hack/install.sh`) does it: `helm upgrade
--install`, which holds a Helm release record. That
record is what makes `lookup` return something, `.Release.IsUpgrade` and
`.Release.IsInstall` tell the truth, and `helm install`/`upgrade` stamp the
`meta.helm.sh/release-name` annotation on every object.

A consumer's own platform commonly does something else entirely: a GitOps
controller renders the chart with `helm template` and applies the output
directly, with **no Helm release record at all**. `lookup` returns empty,
those `.Release` fields are meaningless, and the annotation is never
stamped — the only thing that identifies a release's objects there is the
`app.kubernetes.io/instance` label the chart itself renders. A hook is
affected too: such a controller has no first-install/upgrade distinction of
its own, so it maps `post-install` and `post-upgrade` to one phase and runs
it on **every** sync, not only the first install or a version change.

Nothing here exercises that second path directly — see
[docs/guides/conformance.md](conformance.md#what-ci-already-checks) for the
rows that enforce it by static scan and by chart test instead, so that a
chart which happens to pass under `helm upgrade --install` cannot rely on
what only that path provides.

## The box installs no infra chart

The local cluster carries SERVERS ONLY — a plain Postgres, NATS with
JetStream, an S3 stand-in, a local registry — and no operator: no
CloudNativePG, no NATS controller. An example's own infra-shaped chart
(one that renders a `Cluster` or a `Stream` custom resource) is never
installed here; installing it would prove one example's platform choices
on infrastructure meant to outlive any one example.

What that chart would have provisioned is instead an example's own
FIXTURE, under the example's own directory, naming what it creates by the
exact names the application chart takes as values — rendered from the
chart, not repeated by hand, so the fixture cannot drift from the
interface it stands in for. See `examples/url-shortener/e2e/fixture` and
`docs/decisions/0005-kind-is-the-gate.md`'s amendment.

The box's Postgres is shared the same way its NATS broker is — ONE server
for every install on the box, not one per tenant — so on this box, and only
on this box, the database and its two role names need to be cluster-global
on the same terms [docs/guides/events.md](events.md#naming) already
describes for the stream. `url-shortener-infra`'s `postgres.tenantScopedNames`
is what turns that on: OFF by default, so every real platform keeps the
chart's fixed names (`url_shortener`, `_owner`, `_app`) exactly as before —
a `primary` install's own CNPG Cluster is never shared with anything, so a
fixed name there costs it nothing, and flipping the default would have
handed an already-bootstrapped Cluster a database and owner that no longer
match what it was created with. The fixture turns it ON (see
`e2e/fixture/names.go`'s call to `Resolve`), which derives whichever of the
three a caller left at its own default from this install's namespace and
installName together — folded into a valid Postgres identifier (lower-cased,
`-` to `_`, started with a letter, and — past 57 of Postgres' 63-byte
identifier limit, leaving room for the longer of the two role suffixes —
truncated with an 8-character hash suffix from its own scope, so two tenants
that truncate to the same prefix still do not collide). An explicit value
always wins over the derived one, flag or no flag. See
`examples/url-shortener/charts/url-shortener-infra/templates/_helpers.tpl`'s
`"url-shortener-infra.postgresBase"` and `"url-shortener-infra.resolvedDatabase"`
for the exact rules. Before this, the fixture created the database and both
roles under the chart's fixed names regardless; two installs in different
namespaces shared all three, and the second one's fixture run reset the
passwords the first one's pods were already connected with.

## The suite

The cluster suite does not assert that things installed. It asserts that the
product **worked**: a row is written, a request is served, an event crosses
the broker, and a counter another service owns moves by exactly the number of
requests made. Exactly, not at least — an upsert that overwrote instead of
incrementing passes "the row exists".

The url-shortener example's suite
(`examples/url-shortener/e2e/suite`) is ONE Go program, meant to run
unchanged in three places: the kind lane above, a private repository's
own shared cluster, and a cluster a release has just been promoted to.
Only the environment differs — it reaches every Service through
`github.com/truvity/gemaal/pkg/harness`, whose `(*Cluster).ServiceURL`
opens a port-forward to the exact Pod behind a Service on a kind-tier
context and dials the ClusterIP directly everywhere else. Nothing in the
suite branches on which tier it is running against.

It is inert unless `E2E_NAMESPACE` is set — `just example-smoke` sets it,
`go test ./...` on its own does not — so `just test` stays hermetic. Set
it (and, on a namespace or release that is not this box's own defaults,
`E2E_APP_RELEASE`/`E2E_BUCKET`) and run it directly:

```sh
E2E_NAMESPACE=shortener go test ./examples/url-shortener/e2e/suite/... -v
```

Most of it asserts through Service endpoints only:

- the migration Job completed and the owner and runtime roles are really
  separate — reached by connecting to the box's own Postgres through the
  harness as each role in turn, since a table's rights are not something
  an HTTP Service can be asked about;
- `urls` creates a short link over Connect, generates one when none is
  given, and shortening the same URL twice returns the same link;
- `redirect` answers 302 with the long URL;
- `stat`'s consumer moves the click counter, read back through `urls` —
  `stat` carries no Service of its own; this is its effect, not its
  endpoint;
- `log` archives the record to the fixture's bucket within its batch
  window.

One test does not go through a Service at all: every Deployment of the
release (`urls`, `redirect`, `web`, `stat` and `log`) is Available with
every replica ready, read through the harness's own kubectl runner. That
is deliberately NOT a Service probe of `/health/live`/`/health/ready` — a
Pod is only marked Ready after the kubelet has already run that exact
probe, so asking the same question again through a Service widens who can
reach the probe listener and proves nothing new. It is also the only test
that covers `stat` and `log`, since neither carries a Service at all.

Before any of that: `TestMain` itself will not run a single test until
`e2e/rollout` (`WaitForPromoted`) says the release has finished rolling
out — closing a race found on a real cluster, where the test chart's own
Job (below) is a separate Application a GitOps controller can sync while
the application release's own Deployments are still catching up. A plain
`kubectl rollout status` wait, called at that moment, sees the OLD
generation already fully rolled out and returns immediately — the suite
would go on to exercise, and `trace_test.go` to inspect a trace produced
by, the PREVIOUS version's pods. When the Job sets `E2E_APP_VERSION` (see
below), `WaitForPromoted` instead waits until every pod carries THAT
version's `app.kubernetes.io/version` label (stamped by every chart's own
`_helpers.tpl`) as well as being fully rolled out; outside-in, with no
separately promoted version to know, it falls back to the plain wait —
still run once, up front, rather than left to whichever test happens to
run first. `E2E_ROLLOUT_TIMEOUT` (a Go duration, e.g. `3m`) overrides how
long each Deployment is given; unset is `harness.DefaultRolloutTimeout`
(2m).

A further test asks a Jaeger-API query endpoint (`E2E_TRACES_URL`) for
the redirect's own trace, by a trace id the test injects itself via a
`traceparent` header, and asserts every hop contributed a span. The kind
box carries no trace store (0005's amendment), so this is a `t.Skip`
there — it is meant for the two tiers that do have one.

That trace store may authenticate its readers. `E2E_TRACES_TOKEN_URL`,
`E2E_TRACES_CLIENT` and `E2E_TRACES_TOKEN_FILE`, set together, tell the
test to trade a projected ServiceAccount token at `E2E_TRACES_TOKEN_URL`
for a bearer token scoped to `E2E_TRACES_CLIENT` (RFC 8693 token
exchange, `examples/url-shortener/e2e/traceauth`) before it queries
`E2E_TRACES_URL`; `E2E_TRACES_CA_FILE`, independently, names a CA bundle
for `E2E_TRACES_URL` alone — never for the token exchange, which is
verified against the suite image's own default trust store, appropriate
for a public issuer. All four unset (the kind tier's only case) sends the
same unauthenticated request this test always sent.

## The suite, as an image

`.goreleaser.yaml` also publishes the compiled suite itself
(`examples/url-shortener/e2e/Dockerfile`, `e2e/hack/build.sh`) — a
`go test -c` binary with a pinned `kubectl` beside it, so a third caller
can run it as a Job rather than as `go test`: a post-promotion
verification, on a cluster whose names are not this box's own defaults.

Every name it needs is an environment variable with a fixture fallback —
`E2E_NAMESPACE`, `E2E_APP_RELEASE`, `E2E_BUCKET`, `E2E_TRACES_URL` as
above, plus:

- `E2E_KCTX` — set it to the EMPTY STRING (present in the Job's env, not
  merely unset) to run with no `--context` at all, so `kubectl` and the
  harness fall back to the Pod's own ServiceAccount instead of this
  package's `kind-policy` default, which does not exist off the box.
- `E2E_S3_ENDPOINT` / `E2E_S3_REGION` — a real S3(-compatible) endpoint
  for the archive check, read with the process's own default AWS
  credential chain rather than the kind box's static test credentials.

Two checks need more than a caller running THIS suite should be handed by
default, and skip cleanly rather than fail when it is absent, so a Job
scoped narrowly still passes on everything else:

- the role-separation check (`db_test.go`) reads a Secret; a `kubectl get
  secret` that comes back `Forbidden` skips that one test rather than
  failing `TestMain` for the whole binary (see `appPasswordOrSkip`);
- the archive check (`log_test.go`) reads a bucket; with no
  `E2E_S3_ENDPOINT` set and no kind fixture to fall back to, it skips
  rather than reaching for credentials nobody asked to grant it (see
  `s3ClientOrSkip`).

A caller that wants either to run sets the matching env and grants the
matching permission — nothing here decides that for every caller by
running unconditionally.

## The suite, as a released test chart

The suite runs two ways, both against a real install, and both the exact
same binary described above — only how it reaches its names differs:

- **outside-in**, from this box's own devbox environment or a laptop
  (`just example-smoke`, or `go test` directly): the suite resolves its
  names by rendering `charts/url-shortener-infra` with `helm`
  (`e2e/fixture.Resolve`), since `helm` is on PATH there and this box's own
  fixture stands in for a real infra chart (see "The box installs no infra
  chart" above);
- **as a Job**, rendered by `charts/url-shortener-e2e` — a THIRD chart,
  released and packaged alongside the other two by the exact same
  `.goreleaser.yaml` + `helmctl` flow, carrying the SAME `e2e` image the
  outside-in loop runs directly. Installed into the SAME namespace as an
  application release already there, it runs the suite against that
  release from inside the cluster.

The Job cannot render the infra chart either — no `helm` in the `e2e`
image, no copy of the charts' embedded source there — so it does not try
to. `charts/url-shortener-e2e/values.yaml` takes every name the suite
needs directly: the database's host, name, and its owner and runtime
roles; the runtime role's password Secret (read into the suite as
`E2E_APP_PASSWORD`, via `secretKeyRef` — no RBAC on Secrets, because the
kubelet resolves it, never this Job's own ServiceAccount token); the
stream, its two subjects and the two durable consumer names; the archive
bucket, its region and endpoint; and, optionally, a traces URL and how to
authenticate to it (`traces.tokenExchange.*`, `traces.caConfigMap` — see
"The suite, as an image" above for what each becomes). Setting
the suite to build its names from THOSE environment variables instead of
rendering a chart. Every case the suite carries runs; nothing here is
missing on purpose.

It also sets `E2E_APP_VERSION` to its OWN `.Chart.AppVersion` — which is
the application chart's too, since the two (plus `url-shortener-infra`)
release under one version together (`.github/workflows/release.yaml`).
That is what lets `WaitForPromoted` (above) prove a run is judging the
version THIS release actually promoted rather than whatever generation
happened to be live when the Job started; `rolloutTimeout` (a Go
duration, e.g. `3m`, empty by default) overrides how long it waits per
Deployment.

`mode` picks which cases run:

| `mode` | Runs |
|---|---|
| `full` (the default) | every case the suite carries |
| `tenant` | every case EXCEPT `TestMigrationRanAndRolesAreSeparate`, which issues DDL directly at the database (`CREATE TABLE`, then `DROP TABLE`, to prove the runtime role cannot) — a probing write that assumes this install owns its database outright, which a tenant sharing one under a prefix (`url-shortener-infra`'s `tier: test`) should not be handed rights to make true |

The Job carries no `helm.sh/hook` annotations at all — it is a PLAIN Job,
applied the same way by `helm upgrade --install` and by a GitOps
controller's `helm template` + apply (see "Two install paths" above). Its
own name folds in THIS CHART'S version
(`charts/url-shortener-e2e/templates/_helpers.tpl`'s
`"url-shortener-e2e.jobName"`), because a Job's spec is immutable: without
that, re-applying an upgraded chart under the same name would be refused
rather than converge. Each release of this chart is therefore a Job
nothing before it ever created; the one before it is left for the NEXT
version's apply to prune, or for `helm uninstall` — not for
`job.ttlSecondsAfterFinished`, which is UNSET by default (see
[conformance.md](conformance.md)) precisely so a GitOps controller's
self-heal cannot recreate a Job that deleted itself. The name alone does
not cover a platform changing only the chart's VALUES at a version
already deployed (Kubernetes still refuses that patch as immutable), so
`job.annotations` — empty by default, rendered on the Job's own metadata
only — lets a platform put its own force/replace annotation there for a
controller that reads one to decide it may delete and recreate the Job
rather than apply in place.

`just example-e2e-chart` runs it on the kind box, after
`just example-install`: installing the packaged `.tgz` under the release
under test's own names (read the same way `hack/install.sh` reads them —
`e2e/fixture/cmd/resolve`, never repeated by hand), waiting for the Job it
renders to reach `Complete`, and printing the Job's own log either way. It
is part of `just cluster-all` and the CI `cluster` job's test step,
alongside `example-smoke` — proving the suite runs as the chart's own Job,
through that Job's scoped RBAC, and not only outside-in.

## The prober

The e2e Job proves a release **IS** healthy, once, and exits. Nothing here
proves it **STAYS** healthy — and a fresh install with no traffic always
looks green, which is exactly the gap a monitoring gate (a bake window a
promotion tool reads before calling a rollout safe) cannot tolerate: it
needs signal before, during and after the rollout, not a single Job's exit
code from before it began.

`charts/url-shortener-e2e/templates/prober.yaml` is that signal: a
Deployment, off by default (`prober.enabled`), that walks the SAME three
journeys the suite proves once — create a short link (`urls`), resolve it
(`redirect`, expecting a 302 back to the long URL), read its click count
back and see it move (`stat`, which carries no Service of its own — this is
its effect, exactly as the suite's own `TestStatMovesTheCounter` reads it)
— in a loop, forever, against the release named by `.Values.appRelease`.

**A separate workload from the suite's Job, deliberately.** They answer two
different questions and belong to two different lifetimes: the Job runs
once and its result is a Job condition; the prober runs for as long as the
release does and its result is a stream of outcomes over time. Folding the
loop into the Job would make "prove it once" and "watch it forever" one
component with two settings fighting over what `mode` even means.

**The request-making code is not duplicated.** Before this existed, the
suite held its own copy of "how to ask `urls` to create a link, how to read
a redirect's status and Location, how to read a click count back" — and the
prober would have needed a second copy, which is exactly the drift a
shared library exists to rule out. `examples/url-shortener/e2e/journey` is
that library: three functions, no `*testing.T`, imported by
`examples/url-shortener/e2e/suite` (which wraps each call with `t.Fatalf`
and the harness's pod-aware error wrapping) and by
`examples/url-shortener/e2e/cmd/prober` (which wraps each call with an
OpenTelemetry metric and a structured log line instead). A protocol change
either would need to follow now has exactly one place to make it.

**Metrics, on the same terms as every other component here (decision
0006).** The prober starts the OpenTelemetry SDK
(`telemetry.Start`, decision 0006 again — see "Telemetry" in
[logging-and-telemetry.md](logging-and-telemetry.md)) and reports a counter
and a histogram: a Prometheus reader sees them as
`probe_journey_total{journey,result}` (`journey` is `urls`, `redirect` or
`stat`; `result` is `success` or `failure`) and
`probe_journey_duration_seconds{journey}`. No endpoint configured — the
chart's default — means no export, exactly like every other exporter here;
nothing about the prober itself decides whether metrics leave the process.

**On kind, read the log, not the metrics.** The local cluster carries no
OpenTelemetry collector (0005's amendment), so a metric this prober
computes is never exported anywhere off the box. What IS always there is
the structured log line `record` in `examples/url-shortener/e2e/cmd/prober`
writes for every pass — `"probe journey succeeded"` or `"probe journey
failed"`, with `journey`, `result` and `duration_seconds` fields — which
`just example-prober` (`examples/url-shortener/hack/install-prober.sh`)
reads back with `kubectl logs` to prove the loop is actually running,
within a bounded time, rather than asking a collector this box does not
have for a series it would never receive.

`just example-prober` runs after `just example-e2e-chart`, on the SAME
release that installed: it enables the prober with `helm upgrade
--reuse-values`, waits for its Deployment to become ready, and fails unless
a successful pass shows up in its log within 60 seconds. It is part of
`just cluster-all` and the CI kind test step, alongside `example-e2e-chart`.

## Traps

**A test that shells out is cached on a stale pass.** When only the rendered
files change, the test's own inputs have not, so the toolchain reuses the old
result. Embed what the test reads into the test's package.

**Assert the tool ran, not just that the command exited zero.** A check whose
match pattern stops matching passes forever, silently. Prove a failing case
fails.

**A probe of a URL is not a probe of the service.** Whatever is in front may
answer, and cheerfully, while the thing behind it is down.
