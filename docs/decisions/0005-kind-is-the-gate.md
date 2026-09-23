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

**A local Kubernetes cluster is the gate.** It carries the same operators a
deployment carries, and the one stand-in it uses (an S3 implementation) is
addressed by endpoint, so swapping it for a real store is configuration.

**There is no container-only tier.** One environment, run by CI and by a
contributor, from the same recipe.

**What only a real deployment can prove is run by whoever has one.** A
private repository checks this one out at a pinned version and runs the same
suite against its own infrastructure. The result travels with the pin, which
is the evidence the release contract already asks a consumer's adoption to
carry.

## Consequences

### Good

- A chart is proved against the controller that will act on it. The failures
  that matter — a resource nothing reconciles, a hook that never runs — are
  the ones a renderer cannot see.
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
