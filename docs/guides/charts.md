# Writing a service chart

**The rule.** A service chart's values are two blocks: `config`, which is the
service's own configuration schema and is rendered into its file verbatim, and
`platform`, which is everything the platform provides. The Deployment, the
ServiceAccount, the Service and the ConfigMap come from the library chart,
`service-lib`; the chart's `values.schema.json` is composed from the two schemas
rather than written.
[contracts/component.md](../contracts/component.md) C15 to C17 are normative;
[0009](../decisions/0009-charts-pass-config-through-and-share-a-library.md)
argues it.

**Why.** A template that writes the configuration file is a second schema, and
a port written beside the file is a second number for one fact. Passing `config`
through and deriving the ports from it removes the places a chart can disagree
with its binary.

## The smallest chart

[`examples/url-shortener/charts/testdata/service-example`](../../examples/url-shortener/charts/testdata/service-example/)
is a whole service chart. Its template is one `include`:

```yaml
{{ include "service-lib.component" (dict "context" $ "name" "echo" "platform" .Values.platform "config" .Values.config) }}
```

and its values are the two blocks (plus the `images` map a release stamps):

```yaml
platform:
  replicas: 2
config:               # exactly echo.schema.json
  listen: {address: ":8080"}
  probes: {address: ":7070"}
  log: {level: info}
  drain: {seconds: 20}
  greeting: hello
```

That renders a ConfigMap holding `config` as it came, a ServiceAccount named
`<release>-echo`, a Deployment, and a Service on port 8080. Nothing in the chart
says 8080 or 7070 except the file.

## What the library derives, so you do not write it

| From | It renders |
|---|---|
| `config.listen.address` | the container's `http` port and the Service's |
| `config.probes.address` | the `probes` port, which the liveness, readiness and (optional) startup probes name, never a number; it is not in the Service |
| `config.tls.mode` | `permissive`: an `https` port from `config.tls.address`, in the container and the Service; `strict`: one listener, nothing added. Either mounts the identity |
| `config.tls.certFile` and the other two | checked to be under `platform.tls.mountPath`, and refused if not |
| `config.drain.seconds` | the grace period: that, plus `platform.drain.preStopSeconds`, plus a margin |
| the file itself | the `checksum/config` annotation, so a changed file restarts the pods |
| `platform.secretFiles` | one file per declared secret name, from a Secret and key, in one projected volume mounted at `config.secrets.root` (mode 0440); the configuration must say `secrets: {source: file, root: ...}`. A secret is never an environment variable |
| `platform.telemetry` | OpenTelemetry's own variables, and no endpoint means do not export |

A component with no `config.probes` is refused, and so is a Service for one
with no `config.listen`. The full list of what `platform` takes, with what each
defaults to, is its schema,
[`schemas/fragments/platform.json`](../../schemas/fragments/platform.json).

## The schema is composed

Commit a `values.schema.src.json` that says what the blocks are:

```json
{
  "type": "object",
  "additionalProperties": false,
  "required": ["config", "images"],
  "properties": {
    "service-lib": { "type": "object", "additionalProperties": false,
                     "properties": { "global": { "type": "object" } } },
    "platform": { "$ref": "https://github.com/truvity/policy/schemas/fragments/platform.json" },
    "config": { "$ref": "echo.schema.json" },
    "images": { "...": "one entry per component, as the release stamps them" }
  }
}
```

then run `just chart-schemas`, which writes `values.schema.json` beside it by
bundling every `$ref` under `$defs`, keyed by its own `$id`, so that Helm
validates offline and the `config` it checks is the very schema the binary
loads. Commit both files. Two things to know:

- Helm puts a key named for every sub-chart into the values it validates, so a
  chart that depends on the library allows an empty `service-lib`, as above.
- The source is not part of the chart: list it in `.helmignore`.

## The library is a dependency

The library lives at the repository root, [`charts/service-lib`](../../charts/service-lib).
A chart declares it by path and commits the lock:

```yaml
dependencies:
  - name: service-lib
    version: 0.0.0
    repository: file://../../../../charts/service-lib   # relative to the chart
```

`just chart-locks` runs `helm dependency update` for the charts that have one
and writes `Chart.lock`; commit it. What the lock resolves to, `charts/*.tgz`
inside the chart, is ignored by git. Nothing else is needed to publish:
`helmctl package` runs `helm dependency build` in the source chart when the
dependency is missing, from the committed lock, and refuses a stale one, so the
published chart carries the library. `just chart-deps` does the same locally
and fails on a stale lock (it is part of `drift`).

Tests render the chart where the repository does: they lay the chart and the
library out at their real relative paths and run `helm dependency build`
(see `chartDir` in the example's `chart_test.go`).

Release the library beside the charts that use it by listing it in the release
workflow's `charts` as a path from the repository root, `charts/service-lib`
(a plain entry is a name under `chart-root`); it is a `type: library` chart with
a `values.yaml` of `{}`, which the release tool needs in order to bake in a
manifest.

## The tests a chart owes

Call these from the chart's own tests, on a default render and one that sets
every value ([`conformance`](../../conformance/chart.go)):

| Helper | Fails when |
|---|---|
| `ConfigMapEqualsConfig` | the file in the ConfigMap is not `.Values.config`, parsed |
| `ChecksumFollowsConfig` | the pods would not restart when the file changes |
| `PortsEqualConfig` | a container or Service port is not what `config` binds, or a probe is not on the probes port |
| `EnvIsDeclared` | an environment variable is neither telemetry nor one the chart allows, or is read from a Secret |
| `SecretsAreFiles` | the secrets volume is not projected with mode 0440 and mounted read-only |
| `NoEnvSecretFields` | a configuration document has a key ending `Env` |
| `ChartSchemaIsComposed` | the committed schema is not what its source composes to |
| `ValidDocument` | the rendered file is refused by the service's own schema |

[`library_test.go`](../../examples/url-shortener/charts/library_test.go) does
all of them on the example, and also holds the refusals (an identity file
outside its mount, a secret with no key, the `default` account).

## A product chart is the one exception

url-shortener is five components wired to each other, and what they must agree
on is derived from the release: the stream and subject names from the namespace
and the install name, each caller's account from the release name, the address
of `urls` from its transport mode. A literal in a values file cannot be computed
from the release it is installed into, so the chart builds each component's
`platform` and `config` from product-level values, in
[`templates/_components.tpl`](../../examples/url-shortener/charts/url-shortener/templates/_components.tpl)
and nowhere else, and hands the library the two dicts. The library does not
know the difference: it renders the `config` it was given verbatim and derives
the ports, the probes, the mounts and the checksum from it, and the same
helpers check them.

If your chart has one service, none of that applies: hand the library
`.Values.platform` and `.Values.config`.

## Traps

**Do not give `config` a default in a template.** A `default` in a template is a
value nobody chose and no schema describes; put it in `values.yaml`, where the
schema and a reviewer both see it.

**`platform.env` is not for configuration.** It exists for the environment a
platform *client library* reads (a database client's connection variables). A
service takes no other structural input than its file
([0002](../decisions/0002-config-file-plus-env.md)); a variable that carries a
service setting is a second way to configure it, and `EnvIsDeclared` is how a
test notices.

**Zero is a value.** A pre-stop delay of `0`, no replicas and a surge of `0` are
each legitimate and each easy to turn into a default with Sprig's `default`.
The library checks for the key; a chart that does its own defaulting should too.

**A passing render is not the published chart.** The published chart is the
packaged one, with the library inside it and the images stamped; render that,
with nothing else supplied, before calling a release done
([release.md §7](../contracts/release.md)).

0010 ends the exception, and its first phase has begun: the product chart's
values, with the other charts', are also written in Pkl and generated beside
this chart's hand-written schema, which stays authoritative until the switch
([pkl-shadow.md](pkl-shadow.md)).
