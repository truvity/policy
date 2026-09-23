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
