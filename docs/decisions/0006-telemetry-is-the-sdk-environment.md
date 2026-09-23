# 0006 — Telemetry is configured by OpenTelemetry's own environment

**Status:** accepted

## Context

[0002](0002-config-file-plus-env.md) says configuration is a file and the
environment carries secrets. Telemetry looked like configuration, so an
`otel` block was added to the service schema: an endpoint, a service name, a
sample ratio.

Nothing read it.

The two services in the estate that actually export telemetry both ignored
the file and used OpenTelemetry's own environment variables, and both wrote a
comment saying why. The specification defines those variables, every
language's SDK reads them without being asked, and every collector's
documentation is written in terms of them. Restating a subset of them in a
schema produces three bad outcomes and no good one:

- **A subset is a ceiling.** The specification has variables for protocol,
  headers, compression, timeouts, samplers, resource attributes and per-signal
  overrides. A schema that names three of them is a schema that has to grow
  every time an operator needs a fourth, and until it does, the operator
  cannot set it.
- **Two ways to say the same thing.** With a block in the file and the
  variables still read by the SDK, precedence becomes a question. Whatever the
  answer, someone debugging at three in the morning has to know it.
- **A second vocabulary.** An operator who knows OpenTelemetry has to learn
  our names for its concepts, and the mapping is ours to document and keep
  true.

There was a second failure in the same area, worth recording because it is
what a switch costs. A Node service exported only when an environment name
matched `production`. Nothing set that variable in any deployment, so the
service ran with a console exporter, writing spans onto the same stream as
its logs, in every environment including the one it was written for.

## Decision

**Telemetry is configured by the OpenTelemetry SDK's own environment
variables**, in every language. The `otel` fragment is removed from the
service schema, and no service reads telemetry settings from its
configuration file.

A service **exports when an endpoint is configured, and does not when it is
not.** There is no environment-name switch, no enable flag, and no debug
exporter chosen by the code.

This is the one exception to [0002](0002-config-file-plus-env.md), and it is
narrow: it applies to telemetry, because there is an existing, specified,
cross-language environment contract for it. No other subsystem has one.

## Consequences

### Good

- An operator configures telemetry the way every document about
  OpenTelemetry says to, including the parts we never thought about.
- The exporter's behaviour is the same in a test cluster and in production,
  differing only in where it points.
- One less schema to keep in step with an upstream specification that moves
  faster than this repository does.

### Bad

- **The rule in [0002](0002-config-file-plus-env.md) now has an exception**,
  and an exception is a thing to explain. Anyone can argue for a second one;
  the test to apply is whether a cross-language specification already defines
  the variables, which is rare.
- **Telemetry settings are not in the file a chart renders and a reviewer
  reads.** They are in the deployment's environment, which is a different
  place to look. A chart that sets them should keep them beside the rest of
  the service's configuration in its values, even though they leave as
  variables.
- Validation of these settings is the SDK's, which mostly means a bad value
  is ignored rather than refused.

### Neutral

- Service name and resource attributes come from the same environment, so a
  deployment that already sets them gets consistent naming for free.
