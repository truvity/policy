# Testing

**The rule.** Two tiers. A hermetic gate that needs nothing but the checkout,
and a cluster tier that runs the thing.
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
| the cluster | a container runtime | `just cluster-all`: stand up, verify, prove each operator acts, then install the example and exercise it |

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

A further test asks a Jaeger-API query endpoint (`E2E_TRACES_URL`) for
the redirect's own trace, by a trace id the test injects itself via a
`traceparent` header, and asserts every hop contributed a span. The kind
box carries no trace store (0005's amendment), so this is a `t.Skip`
there — it is meant for the two tiers that do have one.

## Traps

**A test that shells out is cached on a stale pass.** When only the rendered
files change, the test's own inputs have not, so the toolchain reuses the old
result. Embed what the test reads into the test's package.

**Assert the tool ran, not just that the command exited zero.** A check whose
match pattern stops matching passes forever, silently. Prove a failing case
fails.

**A probe of a URL is not a probe of the service.** Whatever is in front may
answer, and cheerfully, while the thing behind it is down.
