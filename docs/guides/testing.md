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

## The suite

The cluster suite does not assert that things installed. It asserts that the
product **worked**: a row is written, a request is served, an event crosses
the broker, and a counter another service owns moves by exactly the number of
requests made. Exactly, not at least — an upsert that overwrote instead of
incrementing passes "the row exists".

## Traps

**A test that shells out is cached on a stale pass.** When only the rendered
files change, the test's own inputs have not, so the toolchain reuses the old
result. Embed what the test reads into the test's package.

**Assert the tool ran, not just that the command exited zero.** A check whose
match pattern stops matching passes forever, silently. Prove a failing case
fails.

**A probe of a URL is not a probe of the service.** Whatever is in front may
answer, and cheerfully, while the thing behind it is down.
