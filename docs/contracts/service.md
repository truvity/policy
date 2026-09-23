# The service contract

**Normative.** A service that satisfies this contract can be configured,
started, observed, upgraded and stopped by anyone who has met another one,
without reading its source.

It binds at the **process boundary** — a file, a port, a stream, a signal —
and never at a library boundary. That is deliberate: a Go service, a Node
service and a JVM service satisfy it the same way and share no code. There
is nothing to import in order to conform.

Each rule below says what must be true, why, and how it is checked. A rule
with no check is a preference, and does not belong here.

## 1. Configuration is a file; secrets are the environment

A service reads **one configuration file**, whose path it takes as an
argument or a single environment variable, and validates it against a schema
it ships. Everything structural is in that file. Secrets — and only secrets —
arrive as environment variables, each one declared.

Why not the environment for everything: a flat namespace cannot express a
list of things, it has no types, and nothing can validate it before the
process starts. A file can be rendered by whatever deploys the service and
checked against the same schema the service uses, so a typo fails where it
was written rather than in a crash loop. [config.md](config.md) is the whole
rule.

*Checked by:* the schema golden in the service's own tests, and the same
schema applied to what its chart renders.

## 2. The composition root is hand-written

Dependencies are constructed in `main`, in order, with plain constructors,
and passed as arguments. No dependency-injection container, no service
locator, no registry consulted at run time.

A container turns a missing dependency into a run-time failure in an
environment where nobody is watching, and turns the shape of the program
into something only the container can explain.
[0001](../decisions/0001-no-di-containers.md) has the argument and the
evidence.

*Checked by:* the shared lint configuration, which refuses the imports.

## 3. Probes are HTTP, on their own listener

Two endpoints, on a listener separate from the service's own traffic:

| Path | Answers |
|---|---|
| `/health/live` | the process is alive. Nothing else. It must not check a dependency: a liveness probe that fails when a database is slow restarts a healthy process and makes an outage worse. |
| `/health/ready` | this instance can serve traffic now, having checked what it needs to serve it. |

Separate listener, because readiness must be answerable when the service's
own listener is saturated, and because a probe endpoint reachable from
outside is an information leak nobody meant to ship.

A service with a long start-up may also serve `/health/startup`, with the
same shape.

*Checked by:* the chart's golden render, which names the paths, and the
example's tests, which call them.

## 4. Logs are JSON on stderr, at one level

One structured line per event, JSON, on **stderr**. The level comes from a
single `LOG_LEVEL` variable.

**stdout is the program's product; stderr is its commentary.** A command-line
tool's output is its answer, and a caller pipes it somewhere. A job's report
is the same. A service usually produces nothing on stdout at all, and that is
the point: the split is the same one in both cases, so a binary that is a
service today and grows a subcommand tomorrow does not have to move its logs.
Under an orchestrator nothing is lost, because the runtime captures both
streams and tags each line with the one it came from.

The rule that follows is worth stating plainly: **a service writes nothing to
stdout.** A library that prints there is a library that has to be wired or
replaced.

Per-package or per-module log levels are not part of the contract. They
sound useful and are, twice a year, at the cost of a configuration surface
that every service, every chart and every operator has to know about. A
service that needs more detail in one area logs it at a level, not in a
namespace.

**Every library that logs is wired to the service's logger** at the
composition root, like any other dependency. A library with its own format,
its own colours and its own level is a second log stream that looks fine on a
laptop and is unparseable everywhere else — and it will not respect the level
the configuration set, which is how a service ends up ignoring `LOG_LEVEL`
for the half of its output that matters.

*Checked by:* review, and by the example, which wires one such library and
says so.

## 5. Shutdown drains, on SIGTERM

`SIGTERM` starts a drain: stop accepting new work, finish what is in flight,
release what was held, exit 0. A second `SIGTERM` or the orchestrator's grace
period ending is the end of the argument.

The cost of getting this wrong is invisible in every test and obvious in
production: a rolling upgrade becomes a period of dropped requests and
half-finished work, attributed to anything but the deploy.

*Checked by:* the example's tests, which upgrade it while it is serving.

## 6. Version comes from the build

A service reports the version it was built from — the tag, or the commit
when there is no tag — read from the build metadata the toolchain already
embeds. It is not passed in configuration, and it is not a constant someone
edits.

A version that can disagree with the binary is worse than no version at all,
because it is trusted.

*Checked by:* the release, which stamps it, and a test that asserts the
reported version is not the zero value.

## 7. Runtime images contain no build step

A runtime image copies an artifact that CI already built and declares how to
run it. It has no `RUN` line, no package manager, no compiler, no shell
script that fetches something.

Two consequences follow, and both are the point. The image has nothing in it
that can execute at build time, so building it for another architecture
needs no emulation — one job builds every platform, and the cross-platform
build stops being an event. And the image contains only what the artifact
needs, so what is in it is a question with an answer.

*Checked by:* the lint rule that refuses a `RUN` in a runtime image, and by
the release, which builds every platform in one job.

## 8. Services talk over Connect; events fan out

Three shapes, and the rule is which to reach for:

- **A boundary of ownership is an RPC.** One service owns a table, a
  lifecycle, a decision; everything else asks it. The schema lives in the
  repository, with a build configuration beside it, and both client and
  server are generated from it at build time. A `.proto` with nothing
  configured to generate from it is a schema nobody reads and a client
  somebody hand-wrote.

  The **server is one Connect handler**, which serves the gRPC, gRPC-Web and
  Connect protocols on a single listener: the choice belongs to the caller,
  not to the service.

  **In the cluster a client speaks gRPC**, over cleartext HTTP/2. It is the
  protocol every language's tooling is best at, and streaming works without
  thought. A client speaks **Connect over HTTP/1.1, or gRPC-Web**, where
  HTTP/2 trailers cannot survive the path — a browser, or a hop through a
  tunnel or a proxy that downgrades. That is a property of the route, so it
  is decided per edge and written down where the client is built.

  One consequence worth keeping: a Connect unary call is an ordinary HTTP
  POST with a JSON body, so any edge reachable that way can be exercised with
  a command-line HTTP client, which matters more often than it sounds.
- **Fan-out is an event.** When several unrelated things must happen after a
  fact, the fact is published and the consumers are none of the publisher's
  business.
- **A direct read of another service's store is neither**, and is allowed
  only where latency makes the RPC hop genuinely unaffordable. It is a
  coupling with no schema and no owner, and it is written down where it is
  used.

*Checked by:* review, and by the example, which contains one of each,
including the direct read and its justification.

## 9. A rollout replaces instances without a gap

Rule 5 makes a service drain when it is told to stop. This rule is the other
half, and without it the drain is decoration: **what deploys the service must
give it the time it asks for, and must not remove the last healthy instance
to do it.**

- **Two instances, for anything that answers traffic.** One instance cannot be
  replaced without a gap, whatever the strategy says.
- **A disruption budget**, so that draining a machine cannot take the last one
  either. A rollout is not the only thing that moves pods.
- **`maxUnavailable: 0`**: the replacement becomes ready before the incumbent
  is touched.
- **One drain constant, used three times.** The service's own shutdown
  timeout, the grace period the orchestrator grants, and a pre-stop delay are
  one number written once. They disagree by default, and the two ways they
  disagree both look like a network fault: a grace period shorter than the
  timeout kills a draining process, and no pre-stop delay means traffic keeps
  arriving for the moment it takes the routing layer to notice the endpoint
  has gone.
- **Spread across failure domains**, so the budget is not satisfied by two
  instances on one machine.

Replacing all instances at once is allowed where concurrency is genuinely
unsafe — a single writer, an exclusive volume — and then **the reason is
written in the chart**, next to the setting, because the next reader's first
assumption will be that it was an oversight.

*Checked by:* a chart test, which is the only place this can be checked: it
is a property of what is rendered, not of what runs.

## 10. Transport identity belongs to the platform

A service **terminates TLS for its own listener and for nothing else.** It
does not hold a certificate for a hostname it does not answer to, and it
never terminates on behalf of a neighbour.

Where mutual TLS is in force, a workload's identity is **the account it runs
as**, carried in the certificate, and the platform mounts and rotates that
certificate. The service does three things with it: present it, reload it
without restarting, and check a peer's identity against a list it was given.
It does not fetch it, mint it, or keep it in a secret store.

Two identities travel in one certificate and answer different questions: a
name says **where** a service answers, and is checked the ordinary way by
whoever dials it; an account identity says **who** it is, and is what an
allow-list names. Addresses are not identity — a name resolves to whoever
holds it today.

**The probes listener is exempt**, and deliberately: the thing that probes it
presents no identity, so a probe port that demanded one would fail closed on
every node. Rule 3's separate listener is what makes that exemption narrow
instead of a hole.

**The chart's default is off.** A chart is installable by someone who has
none of this, and a default that assumes a platform produces a pod that waits
forever for a volume nobody serves. Turning it on is a decision the platform
makes for every service at once; see [platform.md](platform.md) for what a
platform must provide before it can.

*Checked by:* the chart's golden render with the default, which is
byte-identical to one with no such block at all, and by the example's tests,
which run it on and prove a foreign identity is refused.

## Conformance

| Rule | Mechanism |
|---|---|
| 1. configuration | schema golden; the chart's render validated against the same schema |
| 2. composition root | lint (import ban) |
| 3. probes | chart golden; the example's tests |
| 4. logs | review; the example |
| 5. shutdown | the example's tests |
| 6. version | release stamp; a test |
| 7. images | lint; the release builds every platform in one job |
| 8. communication | review; the example |
| 9. rollout | a chart test |
| 10. transport | chart golden; the example's tests |

Rules 4 and 8 are the two this contract cannot check mechanically today. That
is stated rather than hidden: a conformance table with a gap is honest, and a
gap is a thing that can be closed.

Rule 4 is partly checkable and is not yet checked: a test can assert that a
service produced nothing on stdout while doing its work, and the example
should grow one.
