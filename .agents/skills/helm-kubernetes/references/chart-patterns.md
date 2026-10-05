# Chart Design Patterns (Truvity policy contract, interface 13)

Reference: [github.com/truvity/policy](https://github.com/truvity/policy) [`examples/url-shortener/charts/`](https://github.com/truvity/policy/tree/master/examples/url-shortener/charts) — the current contract (delivery interface 13, first released at example 1.43.0): service-lib library, two-block values, composed schemas, delivery interface, tier, digest pinning, chart tests.

## Product shape: three charts, one number

| Chart | Contents | Lifetime |
|---|---|---|
| `{product}` (application) | components via service-lib, migrate hook Job, HTTPRoute, alerts, PDB | per deployment |
| `{product}-infra` | CNPG Cluster + DatabaseRole, NATS Stream, Bucket/Policy/Role/PIA, ExternalSecret, identity SA | data outlives every deployment |
| `{product}-e2e` | in-cluster e2e Job + prober | per test run |

All three carry the same `delivery.truvity.io/interface` number and release under one tag. Why infra is separate: the migrate hook runs `pre-install` and Helm runs hooks BEFORE release resources — a chart that created its own database could never migrate it.

## service-lib: the library renders the platform pieces

A one-service chart is a single include:

```yaml
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

Dependency mechanics: declared in Chart.yaml (`file://` in the policy repo, `oci://ghcr.io/truvity/charts` for consumers), `Chart.lock` committed, archive under `charts/` gitignored, resolved at package time. A chart with no `config.probes` is refused; so is a Service for a component with no `config.listen`.

## Two-block values

```yaml
platform:            # everything the platform provides — key table in references/platform-values.md
  replicas: 2
config:              # exactly the service's own config schema — rendered verbatim
  listen: {address: ":8080"}
  probes: {address: ":7070"}
images:              # stamped by the release, digest-pinned, no defaults
  web: {registry: "", repository: "", tag: "", digest: ""}
```

No `env`/`extraEnv` hatches: a template writing the config file is a second schema, a port beside it a second number — both removed by passing `config` through and deriving.

## Composed schema

Commit `values.schema.src.json` naming the blocks (`platform` → platform fragment `$ref`, `config` → the service's own schema, `images`), generate `values.schema.json` by bundling every `$ref` under `$defs` (any JSON-Schema bundler resolving the `$ids` works offline; the policy repo's recipe is `just chart-schemas`; complete example in `references/platform-values.md`). Helm then validates offline against the very schema the binary loads at start-up. The source is `.helmignore`d. Helm injects a key per sub-chart, so an empty `service-lib:` object is allowed.

## Product-chart derivation (`_components.tpl`)

THE ONE PLACE release-wide values map onto components. Multi-component products derive agreement by formula — stream/subject names from namespace + installName, each caller's account from the release name — because a literal in a values file cannot derive from the install. Returns JSON; nothing below it reads a value:

```yaml
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
- `tier: test|primary` (step 13 passes it to BOTH charts): `test` provisions nothing, runs as the namespace's standing identity; `primary` provisions the store, identity and policies. The identity ServiceAccount is rendered by `-infra` AFTER its PodIdentityAssociation — sync-waves IAM Role 0 → PIA 1 → SA 2, carrying `argocd.argoproj.io/sync-options: Prune=false,Delete=false` permanently. Rationale: EKS injects Pod Identity only at pod creation and SA admission refuses product pods until the SA exists — no pod starts before its identity. The app chart never renders that SA on primary, only names it.

## Delivery interface (full step table: references/delivery-interface.md)

`annotations: {delivery.truvity.io/interface: "13"}` — the highest step the charts read. Steps only ADD keys (a rename/remove is a breaking change); a platform renders every key of steps 1..N, refuses N+1's, and never compares versions. A platform handed a number higher than it implements refuses to render rather than render short. Steps of note: 2 installName · 3 e2e chart · 5 per-component tls.mode · 6 SA per component (`<release>-<component>`) · 7 events.tls · 8 database.tls verify-full required · 9 postgres.platformOwned · 11 route.faro buffer limit · 12 alerts (VMRule, off by default) · 13 tier on the app chart.

## Labels

Selector labels STABLE (instance + component) — Deployment selector immutability, NP label contract. Full labels add per-component `name`, `part-of`, tenancy. service-lib owns the label shapes; charts consume `service-lib.selectorLabels` etc.

## Feature gates + render-time guards

- Optional resources behind `enabled` gates + tier gates (`alerts.enabled`, `route.enabled`, `networkPolicies.enabled`)
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

`before-hook-creation` (not `hook-succeeded` alone): a prior stuck Job must not block the next release. Client TLS to the database is `verify-full` ONLY — the client proves who answered; no weaker mode exists to choose. Migration runs DDL as the owner credential, runtime roles are DDL-incapable.

## Explicit-field discipline (ArgoCD diff hygiene)

Gateway API servers default omitted fields — `route.yaml` sets `parentRef.{group,kind,namespace,sectionName}` and `backendRefs.{group,kind,weight}` explicitly or they "diff forever" under ArgoCD.

## Chart tests (Go)

- `chart_test.go`, `conformance_test.go`, `delivery_interface_test.go`, `tenant_scope_test.go`, `restricted_test.go`, `library_test.go`, `infra_chart_test.go`, `e2e_chart_test.go`
- Shell out to `helm template`, parse YAML, compare golden renders (`testdata/golden/{everything,minimal}.yaml`), assert SEMANTIC contracts: tier boundaries (test tier renders ZERO cloud objects — a security boundary, tenants cannot apply CRDs), interface key acceptance (every step ≤ N accepted, N+1 refused), tenant scope, restricted fields
- Rationale: "every wrong combination still renders valid YAML" — assert semantics, not schemas
- Vendoring: `helm dependency build` resolves service-lib from the lock (archive under `charts/`, gitignored)

## Ring split rationale (superseded vocabulary, same scope idea)

"Rings" are now scope layers only (per-cluster / per-namespace / per-install) — the operative keys are `tier` and `postgres.platformOwned`. What survives from the old framing: app chart = plain K8s any tenant can install; infra chart = cluster-specific CRs; the pair shares release name/namespace; contract documented in both Chart.yaml descriptions.
