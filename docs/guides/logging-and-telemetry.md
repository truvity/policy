# Logging and telemetry

**The rule.** One structured stream of JSON on **stderr**, at one level from
one variable. Traces and metrics go through the OpenTelemetry SDK, configured
by its own environment. [service.md §4](../contracts/service.md) and
[0006](../decisions/0006-telemetry-is-the-sdk-environment.md).

**Why stderr.** stdout is the program's product; stderr is its commentary. A
command-line tool's output is its answer and a caller pipes it somewhere; a
job's report is the same; a service usually produces nothing on stdout at
all. The split is identical in all three cases, so a binary that grows a
subcommand does not have to move its logs. Under an orchestrator nothing is
lost, because the runtime captures both streams and tags each line.

## Where to look

| Language | Logger | Wired at |
|---|---|---|
| Go | `slog` with a JSON handler on stderr, in [`internal/runtime/runtime.go`](../../examples/url-shortener/internal/runtime/runtime.go) | each `main`, before anything else is constructed |
| TypeScript | a JSON logger to stderr | the composition root; the framework's own logger replaced at start-up |
| Kotlin | the framework's binding, configured for JSON on stderr | |
| Python | `structlog`, JSON to stderr, in [`runtime.py`](../../examples/url-shortener/log/src/url_shortener_log/runtime.py) | `run`, before anything else is constructed |

## Every library that logs is wired to your logger

This is the part that gets skipped, and it is the part that matters. A
library with its own format, its own colours and its own destination is a
second log stream: it looks fine on a laptop, is unparseable in a pipeline,
and — worse — it will not respect the level the configuration set.

The example's database layer is exactly this case. See
`runtime.GormLogger` in
[`internal/runtime/gormlog.go`](../../examples/url-shortener/internal/runtime/gormlog.go):
the library's messages become a field under a constant message, because a
message that varies per call cannot be grouped, counted or alerted on, and
its statements go at debug so a service at the default level does not log
every query.

## Telemetry

**Configured by OpenTelemetry's own environment variables**, in every
language, and by nothing else. There is no `otel` block in a configuration
file; there was one, and nothing read it.

**Export when an endpoint is set.** Not when an environment name matches
something. A service that exported only when a variable said `production`,
in an estate where no deployment set that variable, wrote its spans to a
console exporter everywhere — onto the same stream as its logs.

### Starting it

Each language's starter is beside its loader, and takes nothing:

| | |
|---|---|
| Go | `telemetry.Start(ctx)` → a shutdown to defer |
| Python | `telemetry.start()` → a shutdown to call in a `finally` |
| TypeScript | `await start()` → a shutdown to await while draining |
| Kotlin | the OpenTelemetry Spring Boot starter, on the classpath |

In Python and TypeScript it is an **optional extra**
(`truvity-policy[telemetry]`, optional peers) — the loader is what every
consumer takes, and a service reading a configuration file should not be
made to carry an SDK it never starts.

**Flush on the way out.** The spans describing a shutdown are the ones
somebody wants when they ask why it shut down, and they are the first to
be lost. In Go that needs a context of its own: the one you have is
already cancelled by then, and a flush on a cancelled context sends
nothing.

### What a chart sets, and what it must not

A chart renders the SDK's variables and **nothing else**:

```
OTEL_SERVICE_NAME, OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_PROTOCOL,
OTEL_TRACES_EXPORTER, OTEL_METRICS_EXPORTER, OTEL_LOGS_EXPORTER,
OTEL_TRACES_SAMPLER, OTEL_TRACES_SAMPLER_ARG, OTEL_RESOURCE_ATTRIBUTES
```

**No endpoint means `none` on all three exporters**, set by the chart, not
decided by the code. An SDK left to its default exports to localhost and
retries forever, which is what a laptop, a test and a cluster with no
collector all get otherwise. Deciding it in the chart is also why no
component carries an enable flag.

**`service.name` is a log STREAM field.** Stable for the life of the pod,
low cardinality, and never a request id, a tenant or a version. Two
components must not share one.

**OTLP logs stay off.** Where a node agent already collects stdout — which
is the usual arrangement — an exporter buys a second copy of what is
already stored, and logs that exist only over OTLP vanish exactly when the
exporter is the thing that broke. Turn them on only for records that must
carry a span id, and send only those.

### Verifying

**A 200 from the exporter is not evidence.** It means a collector queued
the batch. Ask the STORES: a log line, a span for the service, and the
metric series. Until then you know the sender did not error, which is a
different claim.

**Export something that is always there.** Runtime metrics cost nothing
and make an empty store unambiguous — without a series that is always
present, "nothing is arriving" and "this service is quiet" look identical,
and no query can tell you which.

## Traps

**A hardcoded level is a service that cannot be debugged without a release.**

**Per-package levels are deliberately not in the contract.** They are useful
about twice a year, at the cost of a configuration surface every service,
chart and operator has to know. A service needing more detail in one area
logs it at a level, not in a namespace.

**No logger configured at all is the quiet failure.** The language's default
handler ships: usually text, usually on stderr, with no level control and no
structure. It looks like logging and satisfies none of this rule.

**Nothing on stdout.** If a library prints there, wire it or replace it.

**A span name is a dimension.** Name spans by ROUTE, never by path: a path
carries the short key, the id, the customer, and naming spans by it is how
a trace store runs out of memory.

**Only some resource attributes become metric labels.** Typically the
cluster, the namespace and the environment; the rest land on an info
series and nowhere else. Anything you plan to filter a metric by must be a
METRIC attribute, not a resource attribute — and you cannot see the
difference from the sending side.

**A rejected series can still answer 200.** A write with too many labels
may be dropped by the store and acknowledged to the sender. It is the
easiest way to emit nothing while every dashboard says the pipeline is
healthy, so verify against the store's own ignored-rows counter, sampled
twice, rather than against your own success rate.

**Spans are read by everyone who can read any.** A trace store usually
cannot be scoped per tenant the way logs and metrics can. Keep credentials
and personal data out of span names and attributes — that is a rule for
the code, because nothing downstream will enforce it.
