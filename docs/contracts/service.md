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

## 4. Logs are JSON on stdout, at one level

One structured line per event, JSON, on stdout. Nothing writes to stderr but
a crash. The level comes from a single `LOG_LEVEL` variable.

Per-package or per-module log levels are not part of the contract. They
sound useful and are, twice a year, at the cost of a configuration surface
that every service, every chart and every operator has to know about. A
service that needs more detail in one area logs it at a level, not in a
namespace.

*Checked by:* review, and by the example, which is what people copy.

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
  lifecycle, a decision; everything else asks it. The wire protocol is
  Connect over HTTP, with the schema in the repository and the client and
  server generated from it at build time. A unary call is an ordinary HTTP
  POST, so it can be made with a command-line HTTP client, which matters
  more often than it sounds.
- **Fan-out is an event.** When several unrelated things must happen after a
  fact, the fact is published and the consumers are none of the publisher's
  business.
- **A direct read of another service's store is neither**, and is allowed
  only where latency makes the RPC hop genuinely unaffordable. It is a
  coupling with no schema and no owner, and it is written down where it is
  used.

*Checked by:* review, and by the example, which contains one of each,
including the direct read and its justification.

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

Rules 4 and 8 are the two this contract cannot check mechanically today. That
is stated rather than hidden: a conformance table with a gap is honest, and a
gap is a thing that can be closed.
