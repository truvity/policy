# The local cluster

A Kubernetes cluster on your machine, carrying SERVERS ONLY — a plain
Postgres, NATS with JetStream, an S3 stand-in, and a local registry. No
operator, and nothing here names anything about any example: a chart that
renders a CNPG `Cluster` or a JetStream `Stream` is proved only at the level
a renderer can reach — see
[0005-kind-is-the-gate.md](../../docs/decisions/0005-kind-is-the-gate.md)'s
"Bad" list for what that trades away.

```sh
just cluster        # create or upgrade it, then check it is usable
just cluster-down   # remove it
```

About a minute to stand up from nothing, seconds when it already exists.

## Why servers only

This box used to carry the CloudNativePG operator, NATS's own controller
(NACK), cert-manager and its SPIFFE driver: a chart that rendered a
`Cluster` or a `Stream` custom resource was proved against the controller
that would act on it in a real deployment.

That coupled the box to ONE example's platform. A `Cluster` resource, a
managed role, a `Stream` custom resource — these are `url-shortener`'s own
chart's choices, made by the `url-shortener-infra` chart, which binds an
application to a specific platform (see that chart's own header comment).
Installing it here would make this box prove one example's platform
choices and nothing else's, on infrastructure every future example
would have to fight or ignore.

So the box stopped being a platform. It is SERVERS: a database, a broker,
an object store, a registry — the things ANY chart's fixture can reach by
endpoint, regardless of which operator a real deployment reconciles that
endpoint through. What an infra-shaped chart would have provisioned is now
an example's own job, under its own directory, naming what it creates by
the names its application chart takes — see
[`examples/url-shortener/e2e/fixture`](../../examples/url-shortener/e2e/fixture)
for the worked example.

## What is in it, and why

| Component | Why it is here |
|---|---|
| Postgres | a plain server: a database and two roles, by SQL, the same connection contract every chart's fixture reaches |
| NATS with JetStream | a plain server: a stream, a publish, a consume, by the client protocol every chart's fixture reaches |
| An S3 implementation | the one stand-in this box has ever needed one for: no vendor runs a licensed object store for a laptop. It is an endpoint, not a product — a schema addresses a store by endpoint, region and path style, so swapping it for a real bucket is configuration |
| A local registry | kind's own documented recipe (https://kind.sigs.k8s.io/docs/user/local-registry/): every node resolves `localhost:5001` to it, so a chart's `images.*.registry` value can be proved against a real push and pull rather than `kind load` |

What pushes to that registry is `.goreleaser.yaml` itself, run by
[`hack/example-snapshot.sh`](../example-snapshot.sh) for one architecture
instead of every one — the same release configuration a tag builds,
pointed at this box instead of a real registry, never a second build
definition kept beside it. See
[`docs/guides/testing.md`](../../docs/guides/testing.md) for why the
example is installed from the packaged chart that produces, and not from
its source directory.

Versions live in one file, [`versions.env`](versions.env), so "which version
is the box on" has an answer that is not a grep through three scripts.

## The two scripts

[`up.sh`](up.sh) installs, idempotently — re-running it upgrades an existing
cluster in place rather than starting over, which is what makes it a
development loop rather than only a CI step.

[`verify.sh`](verify.sh) asks each server a REAL question, not whether its
pod is `Running`: a SQL query, a publish read back through JetStream, an
object put and read back, an image pushed from the host and pulled by a
node. A healthy pod answers none of these on its own — this is the same
lesson the box's earlier, operator-carrying design learned the hard way
(below), just asked of servers instead of controllers now that there is no
controller to ask it of separately.

## What this used to catch, and still would

The box shipped, in its earlier design, with JetStream enabled and no
memory store configured. The server was healthy, a controller connected,
and every memory stream was refused with "insufficient memory resources
available". A chart asking for one would have failed here and worked in
production, which is the opposite of what a test environment is for.

The fix exposed a second trap: JetStream's store limits are read at
start-up and are not hot-reloadable, so an upgrade that changes them leaves
the server running on the limit it booted with. `up.sh` restarts the
server after every upgrade for exactly this reason, and `verify.sh`'s
round trip — not a `varz` limit it never checks any more — is what would
catch the server refusing to hold it.

## What is deliberately absent

- **Every operator.** No CNPG, no NACK, no cert-manager. A chart that
  describes an operator-managed resource is proved as far as a renderer
  and the Kubernetes API's own admission can prove it (see the `charts`
  job and `hack/kubeconform.sh`), and no further, on this box.
- **Workload identity.** Transport identity testing (mTLS, SPIFFE) moved
  off this box entirely; it belongs to a different tier, one with a real
  certificate authority behind it. `examples/url-shortener/hack/identity-smoke.sh`
  still exists and is not yet run anywhere — a later task ports it to
  where it now belongs.
- **A gateway implementation, and its CRDs.** Nothing here renders a route
  by default, and nothing that does needs a controller acting on one.
- **Anything cloud.** No managed services, no roles, no cloud objects. A
  chart is installed here with its cloud integration switched off, and the
  configuration that integration needs is proved where it exists.

That absence is the honest boundary of this box: it proves what a chart
RENDERS and what a plain server ANSWERS, and it cannot prove an operator's
reconciliation, an identity plane, or a cloud integration. What only a real
deployment can prove is run by whoever has one.
