# 0009 — Charts pass configuration through verbatim; the platform pieces come from a library chart

**Status:** accepted

## Context

[0002](0002-config-file-plus-env.md) made a service's configuration one file
validated against a schema, and [0003](0003-schemas-not-generators.md)
refused to generate anything from that schema. What neither said is how the
chart that deploys the service gets the file.

The worked example's chart answered it by writing the file out: a template per
component, each a hand-written document with the values spliced in, and a
default or a rename here and there. That is a second schema in the form of a
template. It drifted: a key the binary stopped reading kept being rendered, and
a default in the template was a default nobody chose. The chart tests were
added to catch exactly that, and they do, after the fact.

The other half of every service chart was copied. A Deployment, a
ServiceAccount, a Service, the probes, the identity mount, the telemetry
variables, the container ports — the same hundred lines per component, in every
chart, differing in a port number. The port number is the part that went
wrong: the container's `containerPort`, the Service's `port` and the
`listen.address` in the file are three spellings of one fact, written in three
places, and nothing made them agree.

## Decision

**A service chart's values are two blocks and nothing else of its own.**

- `config` is the service's configuration, **exactly its own schema**, and the
  chart renders it into a ConfigMap as `toYaml .Values.config`: no field
  renamed, none added, none defaulted in a template. What the binary reads is
  what the chart was given.
- `platform` is everything the platform provides: the image, replicas,
  resources, the account, the probes, where the identity is mounted, which
  secrets reach the process as files, how it exports
  telemetry. Its shape is
  [schemas/fragments/platform.json](../../schemas/fragments/platform.json),
  written by hand.

**The pieces every service chart renders the same way come from a library
chart**, `service-lib`, published beside the charts that use it, under the same
tag. From `platform` and `config` it renders a Deployment, a ServiceAccount (one
per component, never `default`), a Service, and the ConfigMap. It **derives**
what the file already says rather than taking it twice: the container and
Service ports from `config.listen`, `config.probes` and `config.tls.address`;
the probes' port from the probes listener; the identity mount from
`config.tls`; the grace period from `config.drain.seconds`. A chart cannot
write a port that disagrees with the file, because there is nowhere to write
one. Environment variables are the OpenTelemetry ones
([0006](0006-telemetry-is-the-sdk-environment.md)) and the **declared secrets**
(`platform.secretFiles` since [0012](0012-stabilization-amendments.md)), nothing else
([0002](0002-config-file-plus-env.md)).

**The chart's `values.schema.json` is composed, never written:** the platform
schema, and `config` as a `$ref` to the service's own schema, bundled by a small
Go package (`chartschema`) into one document Helm can validate with offline. A
test holds the committed schema equal to the composed one, so the schema Helm
checks the values with and the schema the binary checks its file with are one
document.

Three rules in the component contract (C15, C16 and C17) carry this, and a
conformance helper checks each: the rendered ConfigMap equals `.Values.config`,
the container ports equal what `config` binds, and the committed schema is the
composed one.

**One case does not pass its values through, and says so.** A *product* chart
wires several components to each other, and what they must agree on is derived
from the release: a stream name from the namespace and the install name, a
caller's account from the release name, the address of one component from
another's transport mode. A literal in a values file cannot be computed from
the release it is installed into, and a value the platform must type in twice
is how two installs find each other's stream. Such a chart builds each
component's `platform` and `config` in **one** template and hands the library
the same two dicts; the library still renders `config` verbatim from what it was
given, and everything the library derives is still derived. What the chart gives
up is that its own values are not the service's schema, and it is held to what
survives the mapping instead. The worked example's application chart is this
case.

## Consequences

### Good

- A service chart is a few lines: its values, its schema source, and one
  `include`. A new service gets rollouts, probes, the identity mount and
  telemetry by not writing them.
- A port, a probe's target and a mount path cannot disagree with the file,
  because they are computed from it.
- A second chart in the same estate reads the same bytes: one rule for "what
  does a pod of ours look like", changed in one place and reviewed in one
  diff.
- A change to the library shows up as a diff in every golden render that uses
  it, which is the right amount of noise for a change to something shared.

### Bad

- **The library is a dependency, and Helm resolves it, not us.** A chart
  depends on `charts/service-lib` by a `file://` path and commits its
  `Chart.lock`; the archive that resolves to is ignored by git. `helmctl package`
  runs `helm dependency build` in the source chart when a declared dependency
  is missing (from the lock, refusing a stale one), so a published chart carries
  the library. The cost is a lock file to refresh (`just chart-locks`) whenever
  the dependency changes, and a `file://` path that is relative to the chart,
  which fixes where in the tree a chart that uses the library may live.
- **The library has opinions.** It renders a gapless rollout, a spread across
  machines, a restricted pod security context and a pre-stop delay, and a chart
  that needs a different Deployment has to say what the library lacks, in the
  library. That is deliberate and it will be argued with.
- **Passthrough makes the values the service's schema,** so a platform that
  today hands a chart a convenient release-wide value (one log level for six
  components) now hands each component its own. Fewer remappings means more
  repetition in the values file.
- **Helm adds a `service-lib` key to the values it validates** for a chart that
  depends on the library, so such a chart's schema must allow an (empty) one.
  It is not a value anyone sets, and it is the one thing in the composed schema
  that is not either of the two blocks.
- The product-chart exception is exactly the kind of exception that grows.
  What keeps it small is that it ends at a dict handed to the library: nothing
  downstream of that template reads a value.

### Neutral

- This does not generate anything. `platform.json` is written by hand and
  the library's templates are written by hand
  ([0003](0003-schemas-not-generators.md)); the composer bundles documents that
  already exist.
- A chart with no library at all still conforms to every other rule in the
  component contract. C15 to C17 apply to a *service* chart, one whose
  workload is a process this contract's service rules describe.
