# The platform contract

**Normative.** [service.md](service.md) says what a service does at its own
process boundary. This says what it asks of whatever runs it, and what that
platform owes back.

It exists because the two are separate jobs done by separate people, often in
separate repositories, and because the seam between them is where portability
is won or lost. A chart that names a cloud, a cluster or an organisation has
stopped being a chart and become a deployment. This contract is the list of
things a chart may take, and what a platform must provide to satisfy them.

The test of every rule here is the same: **a second platform, run by someone
else, should be able to satisfy it without a patch to the chart.** Where that
is not yet true, it is written down.

## 1. A chart takes names, never values

Anything the platform owns is passed to a chart as an **identifier** — the
name of a secret, an account, a bucket, a stream, a route's parent. Never the
thing itself.

A chart that takes a password renders it into the release's stored manifest,
where everyone who can read a release can read the password. A chart that
generates one produces a different password every time it is rendered, which
is worse. A chart that takes a *name* can be rendered by anyone, reviewed in
a pull request, and committed.

## 2. Workload identity is an account plus annotations

A chart declares the **account a workload runs as** — whether to create it,
what to call it, and what annotations to put on it — and nothing more. The
service's code uses its platform's ambient credentials and names no
mechanism.

Platforms grant differently, and more than one mechanism is normal within a
single estate: an account may be bound to a cloud role by a controller, or by
an annotation the platform's own admission machinery understands, and which
one is in use is a property of the cluster rather than of the workload. A
chart that hard-codes either can only be installed on half the clusters it
should run on, and the failure appears as a permissions error a long way from
the cause.

**What the platform owes:** that the account named in the chart resolves to
the rights the service needs, by whatever mechanism, before the workload
starts.

## 3. Secrets arrive as Kubernetes Secrets, and nothing else

A service reads a **Secret**. It does not talk to a secret store, hold a
store credential, or know which store exists.

The objects that fetch from a store, or push into one, belong to the
platform, or to the *infrastructure* release that provisions a service's
dependencies — never to the application chart. The application chart takes
the name of the Secret and the key inside it.

This is what keeps a store swappable. A service that reads a parameter store
directly cannot be run anywhere that store is absent, and cannot be tested
without credentials for it.

**What the platform owes:** the named Secret, populated, before the workload
starts, and its rotation.

## 4. An object store is an endpoint

A service that reads or writes objects takes a bucket name, a region, an
optional endpoint, a path-style flag, an optional certificate authority, and
credentials **either by variable name or not at all** — the last meaning the
ambient identity of rule 2.

The vendor is not configuration. An API is; one implementation of it runs in
a cloud, another runs in a test cluster, and a service that can only reach
one of them cannot be tested without it. The certificate authority is there
because a store inside somebody's network is normal and is not signed by a
public root.

See [`schemas/fragments/bucket.json`](../../schemas/fragments/bucket.json).
A deployment that requires a capability beyond the API — object locking, a
particular encryption mode — states it as a requirement of that *profile*,
not as an assumption baked into the code.

## 5. A key operation is an interface with more than one provider

Signing, sealing and envelope encryption are declared as a provider and its
settings, with at least these three: a **local** key from a mounted secret, a
**remote transit** service, and a **cloud key service**.

Local is not optional, and it is not only for tests: it is what makes the
gate runnable and what a second platform starts with. An implementation that
exists only against one cloud's key service cannot be proved anywhere else,
which means in practice that it is proved nowhere until it reaches
production.

## 6. Streams and databases are things the service finds, not things it makes

A service **connects to** a stream, a topic, a database. It does not create
them, and it does not own their retention, their partitions or their
lifecycle. Those belong to the infrastructure release, where they can outlive
any one version of the application and be reasoned about by whoever operates
the estate.

The ordering consequence is real: a migration that runs as part of a release
cannot create the database that release also creates.

There is a sharper version of the same rule, which has now cost three
debugging sessions on one chart. **Whatever runs before a release must only
reference things that also run before it.** A task that runs first and names
an ordinary resource of the same release does not fail on the task — it fails
on the *pod*, which is never created, with a message about a missing
reference in an event nobody is watching, while the install sits at "in
progress" until it times out. The database was the first instance, a
configuration file the second, and the account the workload runs as the
third. It is worth a test, because every instance looks like a hang rather
than an error.

## 7. Exposure is a route with a parent the chart is given

A chart that is reachable from outside renders **one route**, whose parent is
a value, and whose rules are **named**.

It renders no gateway, no listener, no certificate and no DNS record. Those
belong to the platform's edge, are shared between services, and are described
in a catalogue the platform owns. A chart that rendered its own would compete
with that catalogue and win intermittently.

Rules are named because a policy — for sign-in, for CSRF, for rate limiting —
attaches to a rule *by name*. A policy whose target names no rule that exists
does not fail loudly: it is simply not attached, and the route keeps serving
without it. That is the failure mode this rule exists to make impossible, and
it is worth a test of its own.

**What the platform owes:** a parent that admits the route, and the sign-in
or authorisation policy attached to the named rule.

## 8. Transport identity, and what it takes to turn it on

[service.md](service.md) rule 10 says a service presents an identity the
platform gives it. A platform that wants to turn that on must provide, at
minimum:

- a way to obtain a certificate whose identity is **attested by the
  platform**, not asserted by the workload — that is, derived from the
  account the pod actually runs as, by something the pod cannot lie to;
- **something that refuses a request for an identity other than the
  requester's**, and nothing else that approves requests behind its back;
- an authority that will sign that identity;
- a trust bundle, distributed to every workload that must verify peers;
- rotation, without restarting workloads;
- whatever in-cluster permission the workload's own account needs to ask for
  its certificate, since the request is made **as that account**.

The second item is the one that is missed, and missing it is invisible.
Certificate machinery commonly ships with a component that approves every
request for an authority it knows about. Left in place beside an attesting
approver, it answers first: any account that may ask for a certificate
receives **any identity it asks for**, including its neighbour's. Nothing
fails. Certificates mount, services connect, every log line reports success,
and the attestation is decoration.

A platform claiming this capability should be able to demonstrate the
refusal, not the issuance. The issuance proves nothing.

Until all four exist, the setting stays off, and the chart renders exactly
what it renders today. A platform that cannot yet do this is not
non-conforming; it has not adopted an optional capability.

## 9. What the platform decides, and a chart never does

| Decision | Whose |
|---|---|
| Which mechanism binds an account to a cloud role | the platform's, per cluster |
| Which secret store, and how secrets arrive | the platform's |
| Which object store implementation | the deployment's, as configuration |
| Whether transport identity is on | the platform's, for every service at once |
| What a route's parent is, and which policies attach | the platform's catalogue |
| How many instances, and what the budget is | the deployment's, within rule 9 of the service contract |
| Image tags and digests | the release's |

A chart's defaults are the ones that let a stranger install it on an empty
cluster and get something that runs. Every value above has a default that
asks for nothing.

## Conformance

| Rule | Mechanism |
|---|---|
| 1. names not values | chart test: no rendered secret material |
| 2. account and annotations | chart golden |
| 3. secrets as Secrets | review; lint (no store SDK in an application) |
| 4. object store | schema |
| 5. key providers | the local provider runs in the gate |
| 6. found, not made | review; the install order in the example |
| 7. exposure | chart golden; a negative fixture for an unattached policy |
| 8. transport | the default render is byte-identical without it |

Rules 3 and 6 are checked by review today. Both are mechanically checkable —
an import ban for the first, and for the second a test that the application
chart renders no resource whose kind belongs to the infrastructure one — and
both should be.
