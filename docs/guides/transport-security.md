# Transport security

**The rule.** In-cluster traffic is mutually authenticated. A workload's
identity is **the account it runs as**, carried in a certificate the platform
mounts and rotates. A service presents it, reloads it, and admits peers by
identity rather than by address.
[service.md §10](../contracts/service.md) is normative;
[0007](../decisions/0007-no-mesh-identity-in-process.md) argues it and says
why there is no service mesh.

**Why not addresses.** A network policy answers "may this address reach that
one". An address resolves to whoever holds it today, and in a cluster that is
a moving target. An account is what authorisation, secrets and cloud roles
already key on, so identity here is the same notion as identity everywhere
else.

## Where to look

| Language | Package | Notes |
|---|---|---|
| Go | [`transport/`](../../transport/) | `Load`, then `Server()` and `Client()` give `*tls.Config` |
| TypeScript | follows | a secure context, replaced on rotation |
| Kotlin | follows, with the counter | the framework's SSL bundles, reload on update |
| Python | [`python/src/truvity_policy/transport.py`](../../python/src/truvity_policy/transport.py) | `load`, then `client_context()` / `server_context()`. The account check is **`verify_peer` after the handshake**, because `ssl` has no verification callback |

The configuration is the [`tls`](../../schemas/fragments/tls.json) fragment.

## What the platform must attest

**A certificate whose identity the workload asserts is worth nothing.** If a
pod can ask for a certificate naming a neighbour's account, the scheme is
decoration.

What makes it sound is that the *runtime* attests the identity: the node's
agent hands the requesting component the pod's **own** account token, the
request is made as that account, and an approver refuses any request whose
identity is not the one the requester holds. The workload is never asked what
it is. [platform.md §8](../contracts/platform.md) lists what a platform must
provide before any of this can be turned on.

## The identity is not a name

A platform's workload certificate carries an **identity** and usually no host
name at all. So service-to-service verification asks "is the thing answering
the account I was told to trust", and does not ask "did I reach the address I
meant to". The second is the weaker question: an address resolves to whoever
holds it today, and the first makes it redundant.

This has a consequence that looks alarming in code and is not. The standard
library insists on checking the name, so a client that verifies by identity
has to turn the library's verification **off** and do both halves by hand:
build the chain against the trust bundle, then read the identity out of the
leaf. Skipping either half would be the mistake the flag's name warns about;
skipping the name is the point.

See `Client` in [`transport/`](../../transport/), where the comment says
exactly this next to the flag, because the next reader's first instinct will
be to delete it.

## The three modes

| Mode | Serves |
|---|---|
| `off` | cleartext only |
| `permissive` | both, on **two ports** |
| `strict` | the authenticated port only |

`permissive` is two ports because one listener cannot be both in every
runtime. An edge then migrates in three commits with no coordinated window:
the server adds its authenticated port, its clients move to it, the server
drops the cleartext one.

**The chart's default is `off`, and stays that way.** A chart is installable
by someone whose platform provides none of this, and a default that assumes a
platform produces a pod waiting forever for a volume nobody serves. Turning
it on is a decision a platform makes for every service at once.

## Two exemptions, and they are narrow

**The probes listener.** Whatever probes it presents no identity, so a probe
port demanding one fails closed on every node. This is why
[service.md §3](../contracts/service.md) puts probes on a listener of their
own: the exemption is a port, not a path, and a port is easy to reason about.

**A database that already does this.** Where a database authenticates its
clients with certificates its own operator issues and maps to roles by a
different field, that is working mutual TLS. Replacing it with this scheme
buys nothing.

A third-party server that cannot do any of it takes a terminating proxy in
its pod, or a written exception naming the reason. Both are decisions, and
both are recorded where the conformance guide says exceptions go.

## Traps

**A certificate read once authenticates fine until it rotates.** These live
about an hour. A process that cached the first one works through a whole
afternoon of testing and fails everywhere at once, with an expiry error and
nothing pointing at the line that read it. Re-read when the file changes.

**A rotation is not atomic across two files.** A read that catches it
half-done must keep serving the previous certificate, which is still valid,
rather than refusing every connection for the moment it takes to finish.

**The group owns the mount, and forgetting it costs an afternoon.** A driver
writes what it mounts owned by root; a process running as anyone else cannot
read its own certificate. The symptom is a permission error on a certificate
authority file, or a complaint that a certificate is malformed — neither of
which mentions identity, and both of which appear only once the transport is
turned on. Set the pod's group to the user the image runs as.

**Refuse loudly on the server, quietly to the caller.** The caller is told
that it was refused and nothing more; telling it *which* rule rejected it
describes the allow-list to whoever is probing. The reason belongs in the
server's log, where an operator reads it — and a test should assert both
halves, or a service refusing everyone for an unrelated reason looks correct.

**An empty allow-list must admit nobody.** It is the right default for a
service nobody has been granted, and it is the one default that has to fail
closed.

**Prove the refusal, never the issuance.** The dangerous failure is a
platform where identities are issued correctly *and anyone can ask for
anyone's*. Certificate tooling commonly ships a component that approves every
request for an authority it knows; leave it beside an attesting approver and
it answers first. Everything works, and the identity means nothing.

This was measured rather than imagined. In the local cluster, before the
blanket approver was turned off, an account called `alice` submitted a
request naming another account's identity by hand and was issued a certificate for it,
with no error anywhere. With it off, the same request sits inert and no
certificate is produced. The box asserts the approver is off, because the
difference between the two is invisible from every other angle.

**The workload's own account needs permission to ask.** The request is made
*as* the workload's account — that is the attestation — so that account needs
whatever in-cluster permission creating the request requires. A pod that
cannot ask simply never starts, with the reason in an event nobody is
watching.

**Trust domain before account.** Two clusters can each have a `shop`
namespace and an `api` account. The account alone is not the identity; the
trust domain is what makes it one.
