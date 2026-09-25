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

## Log/trace correlation

**The rule.** Every JSON log record written while a span is current carries
`trace_id` and `span_id` as top-level fields — lower-case hex, the W3C
forms, the names the OpenTelemetry specification itself recommends for
trace context in a log format that is not OTLP. A record written while no
span is current has neither field: **absent, not an empty string and not a
zero value**, so "no trace" and "trace zero" cannot be confused by whatever
reads the line back.

There is no configuration for this. It follows the same rule OTLP logs stay
off for: a service that already exports traces correctly needs nothing
extra turned on to make its log lines findable from one, and a service that
never starts telemetry gets no fields and no error either.

**Use the context-taking log call.** A logger call that is not handed the
request's (or the message's, or the query's) context cannot know which span
was current for it, and gets no fields — even while a span is current
somewhere else in the process. That is deliberate, not a gap: a field that
appeared because *some* span happened to be current would point at the
wrong trace as often as the right one, which is worse than pointing at
none. Go's context-less `Logger.Info` is the clearest case, because nothing
else in the language would stop a mistake there — Python's and
TypeScript's OpenTelemetry contexts are ambient, so a call inside the right
`async`/await chain gets the fields without passing anything explicitly,
but a callback that escaped that chain (a hand-rolled thread, a detached
callback) is the same failure by a different route.

### Where it lives

| Language | | |
|---|---|---|
| Go | an `slog.Handler` wrapping the service's own, in the root `telemetry` package | reads the span from the `context.Context` the `Handle` call receives |
| Python | a `structlog` processor in `truvity_policy` | reads `opentelemetry.trace.get_current_span()`; imported lazily, so a consumer who never took the `telemetry` extra gets a no-op, not an import error |
| TypeScript | wherever the JSON logger already is | reads `@opentelemetry/api`'s active context the same way a span-producing interceptor does |
| Kotlin | the OpenTelemetry Logback MDC instrumentation, wrapping the appender that already writes structured JSON | the Spring Boot starter does not bring this in by itself — it has to be added and wired in `logback.xml` as an appender wrapping the existing one |

## A trace that stays whole

**The rule.** One request is one trace, from the browser to the row it wrote.
Spans that exist but do not connect are the failure this section is about:
every service reports healthy, every store has data, and no request can be
followed across two of them. That is what the example did until it was
looked at from the trace viewer rather than from the exporters.

A trace is joined at exactly the places context crosses a boundary, and each
boundary drops it by default. There are five, and each needs something
written in code — none of them is configuration.

| Boundary | What carries it | What the code must do |
|---|---|---|
| Service to service, RPC | the `traceparent` header | the **caller** starts a client span and injects; the **callee** extracts and starts a server span that is its child |
| Through the broker | the message's headers | the **publisher** injects into the message; the **consumer** extracts and starts a consumer span |
| Into a database | the request's context | pass the request's context to every query, and give queries a span |
| Into an object store | the request's context | the same, for every call the client makes |
| Onto another thread | nothing, by default | hand the context over where the work is queued |

### Trust the caller, or link to it

A server span whose parent came from a header has a choice: be its child, or
only **link** to it and start a trace of its own. The Connect interceptor's
default is the link, and it is right for a service facing the internet, where
any client could otherwise choose which trace the service joins and whether
it is sampled.

A service whose callers are its own platform should trust them
(`otelconnect.WithTrustRemote()` in Go). The example's data service does,
because every caller reaches it through the transport rules and none is
anonymous. Leaving the default in place there is the mistake: every request
becomes two traces, each looking complete, joined only by a link nobody
follows.

### Through a broker

A broker does nothing with the trace context except keep what the publisher
put in the message. Three details, each of which cost a debugging session:

- **The header name is `traceparent`, in lower case.** NATS header names are
  case-sensitive. A publisher that went through an HTTP-style carrier writes
  `Traceparent`, which a consumer in another language looking for the name
  the W3C specification gives will never find, and starts a trace of its own
  without saying so. Inject into a plain map and set the keys directly, and
  have every consumer lower-case what it reads.
- **A consumer of one message is a child of the publisher.** It is the same
  piece of work, later.
- **A batching consumer links; it does not parent.** A component that holds
  hundreds of messages and writes one object cannot make the write the child
  of any one of them. It records a short span per message, each a child of
  its publisher, and one span for the write that **links** to those. That
  puts the archive inside the trace of every request it served without
  pretending the write belongs to just one. A span carries at most 128
  links, so cap the list rather than let a full batch look complete.
- **Show the write in each request's own trace as well.** The write and the
  object-store call beneath it live in the flush's trace, so a viewer
  opened on a request ends at "received" and the write is one more search
  away. Give each message a short `write` child, timed to the flush and
  **linked** to it: the request's trace shows that the record was archived
  and how long the write took, and the link is the way into the real spans.
  Do not try to hang the store call itself under one request; it belongs to
  all of them.

### A thread is a boundary

The current span lives in a thread-local. Work queued to another thread —
an HTTP client's dispatcher, a pool, an async runtime — starts with no span
unless the executor captures the context **where the work is queued**. The
symptom is a client span with no parent, so every call starts a trace of its
own even though a span was current a line earlier. In Kotlin, wrap the
client's executor with `Context.taskWrapping`. Python's `asyncio.to_thread`
and Node's async context copy it for you; a hand-made thread does not.

### Databases and object stores

Both are spans, or the trace ends at the service's edge and "was it the
database?" has no answer. Two rules for the database:

- **Record the statement text, never the arguments.** The text, with its
  placeholders, is a bounded set and says which query was slow. The
  arguments are user data, and a trace store is read by everyone who can
  read any trace.
- **Prefer a small callback of your own to a plugin that imports every
  driver.** The published GORM plugin does, so a service that speaks to one
  database ships the client for four; the example's copy is
  [`gormtrace.go`](../../examples/url-shortener/internal/runtime/gormtrace.go).

An object-store client gets its spans from the SDK's instrumentation; the
example's archiver installs the botocore one before it builds the client.
A missing row is an answer, not a failure — do not mark it an error.

### Where to look

| | |
|---|---|
| RPC, caller (TypeScript) | [`urls.ts`](../../examples/url-shortener/web/src/server/urls.ts), an interceptor |
| RPC, callee (Go) | `otelconnect.NewInterceptor(otelconnect.WithTrustRemote())` in [`cmd/urls/main.go`](../../examples/url-shortener/cmd/urls/main.go) |
| Publish (Go) | [`publisher.go`](../../examples/url-shortener/internal/events/publisher.go) |
| Consume (Kotlin) | `consumed` in [`Stat.kt`](../../examples/url-shortener/stat/src/main/kotlin/com/truvity/example/stat/Stat.kt) |
| Consume and link (Python) | [`tracing.py`](../../examples/url-shortener/log/src/url_shortener_log/tracing.py) |

### Verifying it

**Exporters healthy is not the claim.** Fetch ONE trace by its id from the
trace store and read its tree: every service you expect must be in it, and
every span but the root must have a parent that is also in it. A search
that lists a span per service proves each service exports; only a single
trace with more than one service in it proves they are connected.

Test each boundary where it breaks, not only that a span exists: assert the
callee's span has the caller's span id as its parent, and that the header a
consumer reads is byte-for-byte the name the publisher wrote. The two
failures above — a capitalised header and a parentless client span on
another thread — both passed a "a span was created" test.

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
