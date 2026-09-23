# The local cluster

A Kubernetes cluster on your machine, carrying the same operators a
deployment carries, so a chart is proved against the controller that will act
on it rather than against a renderer.

```sh
just cluster        # create or upgrade it, then check it is usable
just cluster-smoke  # prove every operator ACTS
just cluster-down   # remove it
```

About two minutes to stand up from nothing, and seconds when it already
exists. The smoke test adds two or three, most of it waiting for a database.

## What is in it, and why

| Component | Why it is the real thing |
|---|---|
| CloudNativePG | a chart that renders a database resource is only proved when an operator turns it into a database |
| NATS with JetStream, and its controller | a stream resource applied with no controller is accepted, stored, and never becomes a stream |
| Gateway API CRDs | a route needs them to exist at all. No controller: nothing here needs one acting on a route, and installing an implementation is minutes spent proving somebody else's software |
| An S3 implementation | the one stand-in, because there is no operator to prove. It is an endpoint, not a product: the schemas address a store by endpoint, region and path style, so swapping it for a real bucket is configuration |

Versions live in one file, [`versions.env`](versions.env), so "which version
is the box on" has an answer that is not a grep through three scripts.

## The two scripts, and why they are two

[`up.sh`](up.sh) installs. [`verify.sh`](verify.sh) asks whether each thing is
**usable**, which is a different question: `helm --wait` already reported
success, and an operator whose CRD is missing or a server running without the
limit it needs both report healthy pods.

[`smoke.sh`](smoke.sh) is the third question and the one that matters: does
an operator **act**. It creates a database, a stream and a bucket, waits for
each to become real, and removes them. Everything lives in a namespace of its
own, so the box is left as it was found.

## What the smoke test already caught

The box shipped with JetStream enabled and no memory store configured. The
server was healthy, the controller connected, and every memory stream was
refused with "insufficient memory resources available". A chart asking for
one would have failed here and worked in production, which is the opposite of
what a test environment is for.

The fix exposed a second trap: JetStream's store limits are read at start-up
and are not hot-reloadable, so an upgrade that changes them leaves the server
running on the limit it booted with. `up.sh` restarts the server, and
`verify.sh` asserts the limit the server is **running** with rather than the
one its configuration holds.

Both are the reason `smoke.sh` exists. Neither is visible from a manifest.

## What is deliberately absent

- **A gateway implementation.** The CRDs are enough to render and apply a
  route. Proving that traffic flows is a different test, and it belongs where
  the traffic is.
- **Authentication on the broker.** The box has one tenant and no secrets.
  A deployment that authenticates its broker configures it; the schema has
  the field.
- **Anything cloud.** No identity, no managed services, no roles. A chart is
  installed here with its cloud integration switched off, and the
  configuration that integration needs is proved where it exists.

That last absence is the honest boundary of this box: it proves the charts
and the code, and it cannot prove the identity plane. What can only be proved
against a real deployment is run by whoever has one, against their own.
