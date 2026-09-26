# 0005 — A local cluster is the gate; the real one is a consumer's

**Status:** accepted

## Context

A chart and the service it deploys can be tested at three depths, and the
question is which one every pull request pays for.

1. **Rendering.** `helm template`, a golden file, a schema. Fast, catches
   typos and schema drift, and proves nothing about whether the manifests
   work.
2. **A local cluster.** Real operators, real admission, real readiness. A
   database resource becomes a database because a controller made one.
3. **A real deployment.** Identity, provisioning, network policy, the
   platform's own services.

Depth 1 is already the goldens. Depth 3 cannot be a public repository's gate:
it needs credentials the repository must not hold, infrastructure a
contributor does not have, and a fork's pull request must never touch either.

A fourth option was considered and rejected: containers without Kubernetes,
composed by hand. It is faster and it tests the application well, but it
tests none of what a chart claims. Every failure this repository's example is
meant to catch — a resource no controller acts on, a hook that never runs, a
probe the platform reads at a path the chart did not render — is invisible
without a cluster. Two environments would also have to be kept in step, and
the cheap one becomes the one people run.

## Decision

**A local Kubernetes cluster is the gate, and a required check for merges.** It
carries the same operators a deployment carries, and the one stand-in it uses
(an S3 implementation) is addressed by endpoint, so swapping it for a real
store is configuration.

**There is no container-only tier.** One environment, run by CI and by a
contributor, from the same recipe.

**What only a real deployment can prove is run by whoever has one.** A
private repository checks this one out at a pinned version and runs the same
suite against its own infrastructure. The result travels with the pin, which
is the evidence the release contract already asks a consumer's adoption to
carry.

**And the two kinds of repository should not make the same choice here.**

A **public** repository stands up a local cluster, for the reason above: a
fork's pull request must never reach the estate's own infrastructure, and a
contributor has none of it. That constraint is what makes a local cluster the
only honest answer, not a preference.

A **private** repository has no such constraint, and taking the local cluster
anyway costs it the thing it actually needs. Its CI can reach a shared
development cluster — the real identity plane, the real provisioning, the
real network policy — which is exactly the depth a local cluster cannot
reach. Standing up a throwaway cluster to avoid using one that is already
there trades a better test for a slower one.

So: **a public repository's gate is a local cluster; a private
repository's is the shared development cluster.** The suite is the same
either way, which is the point of writing it against a chart and a set of
probes rather than against an environment.

## Consequences

### Good

- A chart is proved against the controller that will act on it. The failures
  that matter — a resource nothing reconciles, a hook that never runs — are
  the ones a renderer cannot see.
- It earned this on the first product put through it. Building the worked
  example, with a green hermetic gate throughout, the box caught five
  defects a renderer accepts:
  - a migration hook that ran before the database its own chart created, so
    the chart had to become two;
  - the same fault one level down, a hook mounting a ConfigMap the release
    had not applied yet;
  - a migration that never created the schemas its tables name, because
    something outside it had always created them before;
  - one credential shared by the migration and the services, which put the
    right to drop a table on the request path;
  - a library writing its own coloured log format past the service's
    logger, which is a broken log contract that every local run looks fine
    with.

  Each is now held by a test. Three of them are impossible to state as a
  rendering assertion at all.
- One environment, so nothing has to be kept in step, and the thing a
  contributor runs is the thing that gates the merge.
- It needs nothing but a container runtime, so a stranger can run it and a
  fork's pull request costs the estate nothing.

### Bad

- **Minutes, not seconds.** About two to stand up, and more for anything that
  waits on a database. That is the price of the depth, and it is why the
  hermetic gate stays separate and fast.
- **It cannot prove the identity plane.** Workload identity, provisioning,
  the platform's own policies: none of it is here, and a chart's
  cloud integration is installed switched off. A repository that believed
  this box proved everything would ship a chart whose only untested part is
  the part that talks to the cloud.
- A container runtime is now a prerequisite for the deeper suite, though not
  for the gate.

### Neutral

- The stand-in for object storage is pinned by digest and is a community
  build. That is a supply-chain dependency like any other, and it is the
  reason the store is configured by endpoint rather than by product.

## Amendment: the box installs no infra chart

This cluster used to carry every operator a worked example's infra chart
needed — CloudNativePG, NATS's own controller — so that chart could be
installed here like any other. That bound the box to one example's platform
choices: a `Cluster` resource, a managed role, a `Stream` custom resource
are `url-shortener-infra`'s decisions, not this box's, and a second example
would either fight them or need its own cluster.

**The box is SERVERS ONLY now — a plain Postgres, NATS with JetStream, an
S3 stand-in, a local registry — and installs no infra-shaped chart.** What
such a chart would have provisioned is an example's own job: a FIXTURE,
under the example's own directory, naming what it creates by the exact
names the example's application chart takes as values. `hack/kind/README.md`
says why; `examples/url-shortener/e2e/fixture` is the worked example.

The infra chart itself is unaffected — it still renders, still has goldens,
still gets validated against the Kubernetes API's own schemas in the
`charts` job. It is simply never installed on a public repository's local
cluster, because installing it would prove one platform's choices on
infrastructure meant to outlive any one example.
