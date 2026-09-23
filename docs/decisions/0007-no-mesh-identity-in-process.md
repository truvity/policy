# 0007 — No service mesh; a workload's identity is its account, terminated in process

**Status:** accepted

## Context

Traffic between services inside a cluster was cleartext, isolated by network
policy alone. Network policy answers "may this address reach that address",
which is not the same question as "who is calling". An address is whoever
holds it today.

Mutual TLS answers the second question, and the decision is not whether to
have it but where it terminates. Three shapes were considered.

**A service mesh.** A data plane on every node or in every pod, transparently
encrypting and authenticating, with identity derived from the workload's
account. Of the meshes available, exactly one runs on both kinds of cluster
in use — a managed cloud offering with its own network plugin, and a bare
cluster whose plugin is a different project entirely. Making it work on the
second requires turning off that plugin's exclusivity, disabling one of its
data-path features, and accepting that its layer-7 policy and the mesh's
layer-4 path do not compose; there is also an open ordering defect in which
the mesh's plugin runs before its own agent and both stay unready. A second
mesh is open source but its supported release line is a vendor subscription.
A third is the network plugin's own, and needs that plugin, which the managed
cluster does not run.

**A terminating proxy beside every workload.** This removes the per-language
work: the application listens on a loopback address in cleartext and the
proxy owns the port. It is a mesh's data plane without its control plane. It
costs a second container in every pod, with its own image to pin and scan, a
second configuration to render and drift, explicit addressing per upstream in
place of interception, and a decision about how the caller's identity is
forwarded to an application that can no longer see it.

**In process.** The certificate is mounted; the service presents it, reloads
it, and checks its peers. Three small functions per language.

The other half of the problem is where identity comes from. A certificate
whose identity a workload *asserts* is worth nothing: a pod that can ask for
a certificate naming a neighbour's account has defeated the scheme. What
makes it sound is that the runtime attests the identity — the node's agent
hands a requesting component the pod's *own* account token, the request is
made as that account, and an approving component refuses any request whose
identity is not the one the requester holds. The workload is never asked what
it is.

## Decision

**No service mesh.**

A workload's identity is **the account it runs as**, carried in a certificate
that a platform component mounts into the pod and rotates. The identity is
attested by the runtime, not asserted by the workload, and an approver
refuses a mismatch. The private key lives in the pod's ephemeral volume and
never in a secret.

**Termination is in process**, through a helper in this repository's package
for each language, doing three things: load and reload, present as a client,
and verify a peer's identity against an allow-list. A **terminating proxy
beside the workload is the escape hatch for a container we do not build**,
declared in the chart as an ordinary sidecar, never injected by admission
machinery: a container that a reviewer cannot see in the rendered manifest is
a container nobody reviews.

Three things stay outside this scheme, each for its own reason:

- **The probes listener**, because the thing that probes it presents no
  identity.
- **The database**, which already authenticates its clients with certificates
  its own operator issues and maps to roles by a different field. It is
  working mutual TLS; replacing it with this one buys nothing.
- **Third-party servers that cannot do it**, which take the proxy or a
  written exception.

## Consequences

### Good

- Identity is the account, which is also what authorisation, secrets and
  cloud roles already key on. One notion of who a workload is.
- No data plane: nothing added to the packet path, nothing to debug between
  two services that are not talking, no interaction with either cluster's
  network plugin.
- The same design works on a bare cluster with no cloud behind it, which is
  the test this repository applies to every rule.
- The key is never in a secret, so a workload that can read secrets in its
  namespace still cannot read a neighbour's key.

### Bad

- **Every language implements it.** Three functions each, and they are not
  identical: one runtime rebuilds a context per connection, another reloads a
  bundle, a third takes callbacks. Server-side reload is weakest in the
  runtime whose common servers load a certificate once at start, which is
  why a server in that language either recycles its workers inside the
  certificate's lifetime or takes the proxy.
- **A mesh's other gifts are forgone**: authorisation by identity without
  code, per-hop telemetry, and coverage of workloads we do not build. The
  first two are not current needs; the third is why the proxy exists.
- **Every third-party component is configured separately** — the broker, the
  cache, the database each in their own vocabulary.
- **Two ports during a migration.** Without a proxy, one listener cannot be
  both cleartext and encrypted, so a service being migrated serves both and
  its clients move one at a time.

### Neutral

- Nothing in a service changes if a mesh is later adopted: a service that
  presents an identity and verifies its peers is a service a mesh can take
  over from. This decision is reversible in the direction that matters.
- The decision is the platform's to enable. A chart's default is off, and a
  deployment that has not built the platform side is not non-conforming.
