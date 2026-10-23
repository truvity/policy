# Chart Design Patterns (Truvity policy contract, interface 15)

Reference: [github.com/truvity/policy](https://github.com/truvity/policy) [`examples/url-shortener/charts/`](https://github.com/truvity/policy/tree/master/examples/url-shortener/charts), reviewed on 2026-10-20 after v1.54.1. Interface 15 was adopted in **v1.48.0**; v1.49.0 added `tls.gatewayFronted`. Current behavior includes cloud-identity admission with an optional account assertion, dashboards, and unconditional cnpg-database 2.31.0 dependency resolution.

## Observed example shape: two charts, one number

| Chart | Contents | Lifetime |
|---|---|---|
| `{product}` (application) | service-lib components and cloud-identity check, migrate Job, optional prober, routes, alerts, dashboards, PDB | per deployment |
| `{product}-infra` | owned stores/stream/identity and install-specific policies; cnpg-database runtime role for a platform-owned Cluster, or password-only resources for its own Cluster | data outlives every deployment |

Both carry the same `delivery.truvity.io/interface` number and release under one tag. Why infra is separate: the migrate hook runs `pre-install` and Helm runs hooks BEFORE release resources — a chart that created its own database could never migrate it.

The registry still requires the end-to-end chart at step 3, but the example stopped publishing it in v1.48.0 while retaining annotation 15. Its prober is an application workload and the suite runs from CI. This inventory describes the code, not an approved exception: report the [contract gap](delivery-interface.md#current-implementation-and-contract-gaps) and correct the contract separately before adopting the discrepant chart set.

## service-lib: the library renders the platform pieces

A one-service chart is a single include:

```gotemplate
{{ include "service-lib.component" (dict "context" $ "name" "echo" "platform" .Values.platform "config" .Values.config) }}
```

That renders the ConfigMap (`config` verbatim), ServiceAccount `<release>-echo`, Deployment, and Service. The library derives, so the chart never writes:

| From | It renders |
|---|---|
| `config.listen.address` | the container's `http` port and the Service's |
| `config.probes.address` | the `probes` port named by liveness/readiness/startup probes — never a number, not in the Service |
| `config.tls.mode` | `permissive`: https port + identity mount; `strict`: one listener; `off`: nothing |
| `config.drain.seconds` | termination grace period (+ pre-stop margin) |
| the config file itself | `checksum/config` pod annotation — a changed file restarts pods |
| `platform.secretFiles` | projected files (mode 0440, read-only) at `config.secrets.root` |
| `platform.telemetry` | OTel env vars; no endpoint = no export |

Dependency mechanics: declared in Chart.yaml (`file://` to the one library in this checkout, a pinned OCI dependency for external consumers), `Chart.lock` committed, archives ignored. Use the library, not standalone templates that imitate it. A chart with no `config.probes` is refused; so is a Service for a component with no `config.listen`.

## Two-block values

These are complete, nonsecret **render-test values** for the reference
`service-example` chart, matching component `echo` above and the composed
schema in [Platform values](platform-values.md).
The neutral registry is a fixture, not a published installable image; a real
release supplies verified image coordinates and a digest.

```yaml
platform:
  replicas: 2
config:
  listen: {address: ":8080"}
  probes: {address: ":7070"}
  log: {level: info}
  drain: {seconds: 20}
  greeting: hello
images:
  echo: {registry: registry.example, repository: example/echo, tag: dev, digest: ""}
```

No `env`/`extraEnv` hatches: a template writing the config file is a second schema, a port beside it a second number — both removed by passing `config` through and deriving.

## Composed schema

For a service chart, commit `values.schema.src.json` naming `platform`, the service's `config` schema and `images`; generate the self-contained `values.schema.json` with `just chart-schemas`. Helm validates offline against the same service schema. Ignore the source in the packaged chart; allow Helm's subchart key. **C16/C17 distinguish product charts**: a multi-component delivery schema may be hand-written and strict; `_components.tpl` derives component inputs in one place and every resulting file is validated against its binary's schema. Do not invent a nonexistent composed source file for that exception.

## Product-chart derivation (`_components.tpl`)

THE ONE PLACE release-wide values map onto components. Multi-component products derive agreement by formula — stream/subject names from namespace + installName, each caller's account from the release name — because a literal in a values file cannot derive from the install. Returns JSON; nothing below it reads a value:

```gotemplate
{{- define "url-shortener.components" -}}
{{- $out := dict -}}
{{- range $name := list "redirect" "urls" "web" "stat" "log" -}}
{{- $arg := dict "root" $ "component" $name -}}
{{- $_ := set $out $name (dict "platform" (include "url-shortener.platform" $arg | fromJson)
                               "config"  (include "url-shortener.config"  $arg | fromJson)) -}}
{{- end -}}
{{- toJson $out -}}
{{- end }}
```

## installName + tier (platform-passed keys)

- `installName` (interface step 2): empty = this release's own name (standalone default); set only when the infra release has a different name — then it must be that release's `installName` exactly. Folds into NATS subjects/durables; DNS-1123-validated at render with `fail`.
- `tier: test|primary` (step 13 passes it to BOTH charts): `test` uses standing resources, while `primary` provisions its owned resources. Infra orders role → binding → ServiceAccount with permanent `Prune=false,Delete=false`; the application only names that account. **This orders objects, not admission**: install infra healthy first, then require the application's `service-lib.identityCheck` for a cloud-identity component. See [identity admission](identity-admission.md); a `Synced` association alone is insufficient.
- `availability: single|high` (step 14): the platform's statement of the CLUSTER, not of the product — `single` (default when the key is absent) runs one replica of every component and renders no budget; `high` runs two and renders a `maxUnavailable: 1` budget. Explicit `replicas.<component>` and `disruption.enabled` still win; a platform with no opinion passes nothing.
- Step 15 requires a runtime `DatabaseRole` without an opt-out. The standalone example's Cluster-managed role is observed behavior that differs from that wording; report the [conformance gap](delivery-interface.md#current-implementation-and-contract-gaps) before adopting it. Long-running workloads get `database.app.*` only; owner credentials stay in migration Jobs, checked by `conformance.MigrationSecretOnlyInJobs`.

## Database-role dependency (example since v1.53.0)

The infra chart always depends on locked **cnpg-database 2.31.0**; there is no
`condition: postgres.platformOwned` and no parent `credential.yaml` template.
`roles: []` renders no subchart resources. A platform-owned Cluster needs an
explicit role entry; a chart-owned Cluster with generated password needs one
with `databaseRole: false`, so two controllers do not own the same role.
With its own Cluster and an external Secret, leave the roles list empty.
See the full input matrix and guard rules in [platform resources](cluster-resources.md).

## Delivery interface (full step table: references/delivery-interface.md)

`annotations: {delivery.truvity.io/interface: "15"}` — the highest step the charts read. Steps only ADD keys (a rename/remove is a breaking change); a platform renders every key of steps 1..N, refuses N+1's, and never compares versions. A platform handed a number higher than it implements refuses to render rather than render short. Steps of note: 2 installName · 3 e2e chart (registry requirement; the example's divergence is recorded above) · 5 per-component tls.mode · 6 SA per component (`<release>-<component>`) · 7 events.tls · 8 database.tls verify-full required · 9 postgres.platformOwned · 11 route.faro buffer limit · 12 alerts (VMRule, off by default) · 13 tier on the app chart · 14 availability · 15 runtime role, migration credential Job-only.

## Labels

Selector labels STABLE (instance + component) — Deployment selector immutability, NP label contract. Full labels add per-component `name`, `part-of`, tenancy. service-lib owns the label shapes; charts consume `service-lib.selectorLabels` etc.

## Feature gates + render-time guards

- Optional resources behind explicit gates (`alerts.enabled`, `route.enabled`, `prober.enabled`, `observability.dashboards.enabled`) and owned cloud resources behind the tier gate. The current app chart renders no NetworkPolicy; the infra chart owns only its install-specific store/check egress, not the namespace baseline.
- `required` for conditional-mandatory values: `{{ required "route.parentRef.name is required when route.enabled is true: a route that names no parent attaches nowhere" .Values.route.parentRef.name }}`
- Strict schemas everywhere the chart defines something: a typo fails the render (`additionalProperties: false`), including conditional `if/then/else` requirements (e.g. alerts remote rendering)

## Migrate hook Job (`templates/migrate.yaml`)

```yaml
annotations:
  "helm.sh/hook": pre-install,pre-upgrade
  "helm.sh/hook-weight": "0"
  "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
backoffLimit: 3
```

`before-hook-creation` (not `hook-succeeded` alone): a prior stuck Job must not block the next release. Client TLS to the database is `verify-full` ONLY — the client proves who answered; no weaker mode exists to choose. Migration runs DDL as the owner credential (`database.migration.passwordSecret` from interface 15 — mounted in this Job and nowhere else); the runtime roles are DDL-incapable.

## Explicit-field discipline (ArgoCD diff hygiene)

Gateway API servers default omitted fields — `route.yaml` sets `parentRef.{group,kind,namespace,sectionName}` and `backendRefs.{group,kind,weight}` explicitly or they "diff forever" under ArgoCD.

## Chart tests (Go)

- `chart_test.go`, `conformance_test.go`, `delivery_interface_test.go`, `tenant_scope_test.go`, `restricted_test.go`, `library_test.go`, `infra_chart_test.go`, `prober_test.go`
- `identity_check_test.go`: hook/account/labels, wave ordering, expected-account forwarding and malformed-value refusals. Dashboard UID/wave assertions are in `chart_test.go`; cnpg-database lock, password-only mode and parent/subchart disagreement checks are in the library/infra tests.
- Shell out to `helm template`, parse YAML, compare golden renders (`testdata/golden/{everything,minimal}.yaml`), assert SEMANTIC contracts: cloud-tier boundaries (the tests enumerate cloud kinds, not every operator CR), interface key acceptance (every step ≤ N accepted, N+1 refused), tenant scope, restricted fields
- Rationale: "every wrong combination still renders valid YAML" — assert semantics, not schemas
- Vendoring: `helm dependency build` resolves service-lib from the lock (archive under `charts/`, gitignored)
- Resolve the infra chart's cnpg-database archive from its own committed lock too. Cold tests and `chart-deps` can download that public OCI dependency; this is allowed locked dependency resolution, not a requirement for cluster credentials.

## Ring split rationale (superseded vocabulary, same scope idea)

"Rings" are now scope layers only (per-cluster / per-namespace / per-install) — the operative keys are `tier` and `postgres.platformOwned`. What survives from the old framing: app chart = plain K8s any tenant can install; infra chart = cluster-specific CRs; the pair shares release name/namespace; contract documented in both Chart.yaml descriptions.
