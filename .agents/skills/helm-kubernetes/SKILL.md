---
name: helm-kubernetes
description: >-
  Helm chart structure and Kubernetes manifests per the Truvity policy
  contract at delivery interface 13: service-lib library chart, two-block
  values, composed schemas, tier, digest pinning, CNPG, EKS Pod Identity,
  NetworkPolicies, chart tests — delivered via ArgoCD. Repository-agnostic;
  reference is the policy example url-shortener
---

# Helm Kubernetes

Helm chart style for service charts delivered via ArgoCD, per the Truvity policy contract.

## Reference

Self-contained detail, shipped with this skill in `references/`:

- `references/chart-patterns.md` — chart design patterns and derivation
- `references/cluster-resources.md` — target-cluster resource inventory
- `references/delivery-interface.md` — the interface number, its rules, all 13 steps
- `references/install-paths.md` — the two install paths (`helm upgrade --install` vs a controller's `helm template` + apply), every GitOps-facing rule with its check, plain-Job and hook behaviour under a controller
- `references/platform-values.md` — the `platform` block key table, the platform menu by chart, a complete composed `values.schema.src.json`

External source of truth (needs network): [github.com/truvity/policy](https://github.com/truvity/policy) — worked example [`examples/url-shortener/charts/`](https://github.com/truvity/policy/tree/master/examples/url-shortener/charts) and library [`charts/service-lib`](https://github.com/truvity/policy/tree/master/charts/service-lib); normative docs: [delivery-interface](https://github.com/truvity/policy/blob/master/docs/contracts/delivery-interface.md), [platform §10](https://github.com/truvity/policy/blob/master/docs/contracts/platform.md), [charts guide](https://github.com/truvity/policy/blob/master/docs/guides/charts.md). Consumer repositories may vendor the example under a different root; adjust paths accordingly.

## Chart Style Essentials (see references for full patterns)

- **Three charts per product**, one interface number: application (components + migrate hook), `-infra` (database + stream — separate LIFETIME: data outlives deployments; hooks run before release resources, so a chart that owned its DB could never migrate it), `-e2e` (in-cluster e2e Job). All released under one tag.
- **service-lib library chart** (`type: library`, `oci://ghcr.io/truvity/charts`, `Chart.lock` committed): renders Deployment, ServiceAccount, Service, probes, mounts, telemetry/client-library env, ConfigMap. A one-service chart's template is a single `include "service-lib.component"` with `(dict "context" $ "name" X "platform" ... "config" ...)`.
- **Two-block values**: `config` = the service's own configuration schema, rendered VERBATIM into its file (binary validates at start-up); `platform` = everything the platform provides (full key table: `references/platform-values.md`); plus the `images` map. No `env`/`extraEnv` hatches — ports/probes/grace DERIVE from config (`listen.address`, `probes.address`, `drain.seconds`); `checksum/config` restarts pods on change.
- **Composed schema**: commit `values.schema.src.json` with `$ref`s (complete example in `references/platform-values.md`), generate `values.schema.json` by bundling every `$ref` under `$defs` — Helm validates against the very schema the binary loads; `.helmignore` the source.
- **Delivery interface**: `annotations: {delivery.truvity.io/interface: "13"}` in every Chart.yaml — one number per product, steps only ADD keys, platform renders keys of every step ≤ N and never compares versions; an unknown-higher number makes the platform refuse to render.
- **Secrets are files, never env**: `platform.secretFiles` → one projected volume (mode 0440, read-only) at `config.secrets.root`; secret names by declaration, no `...Env` fragments.
- **tier `test|primary`** (same key+value passed to BOTH charts): `test` provisions nothing, runs on the namespace's standing identity; `primary` provisions store/identity/policies. Identity SA is rendered by `-infra` AFTER its PodIdentityAssociation (sync-waves: IAM Role 0 → PIA 1 → SA 2; `Prune=false,Delete=false` permanent) — SA admission blocks pods until identity exists; the app chart only NAMES it.
- **installName** (platform-passed, defaults to release name): folds into NATS subjects/durable consumers so two installs can't find each other's stream; DNS-1123-validated at render.
- **Images**: packaging-baked digest-pinned map (`images.{component}.{registry,repository,tag,digest}`), no defaults, release refuses to publish with an empty digest.
- **Guards**: `required` for conditional-mandatory values (`route.parentRef.name`), render-time `fail` on name/validation violations.
- **Migrate hook Job**: `pre-install,pre-upgrade`, weight 0, `hook-delete-policy: before-hook-creation,hook-succeeded`; `database.tls` is `verify-full` ONLY (root CA required).
- **Chart tests** (Go): render + golden testdata + assert SEMANTIC contracts — tier boundaries, delivery-interface key acceptance, tenant scope, restricted fields. "Every wrong combination still renders valid YAML."
- **A chart cannot tell which installer it is in**: no `lookup`, nothing reading `.Release.Is*`/`.Revision`, every hook idempotent on every sync, plain Jobs carrying no `ttlSecondsAfterFinished` (a self-healing controller recreates a self-deleting Job, forever) — `references/install-paths.md`

## Writing a Product Chart

A product chart per the policy contract — components rendered through `service-lib`, two-block values, composed schema, interface number — with infra (DB, exposure platform, AWS) from the platform where it provides it.

```
charts/<product>/
├── Chart.yaml          # service-lib dependency + interface annotation; install order documented
├── values.yaml         # app-generic defaults only — install-specific values come from the platform
├── values.schema.*     # composed from config schema + platform fragment
├── values-<env>.yaml   # deltas
└── templates/          # _components.tpl (the ONE derivation place); migrate hook Job; route
deploy/argocd/          # ArgoCD Applications (per-repository GitOps layout)
```

Decisions to resolve before writing templates:
1. **Exposure**: HTTPRoute on the platform Gateway (`route.enabled`, `parentRef.name` required) — TLS terminates at the Gateway, never chart-managed Ingress/certs
2. **Postgres**: CNPG cluster + DatabaseRole from `-infra` (or platform-provided via `postgres.platformOwned`); client TLS `verify-full` mandatory
3. **Images**: CI builds and writes back digest-pinned coordinates — never floating tags
4. **Stateful deps without a platform CR** (e.g. Redis): platform-provided or kept in-chart (no CNPG equivalent)
5. **Migrations**: pre-install/pre-upgrade hook Job (`before-hook-creation,hook-succeeded`, TTL reaps residue) — never initContainers
6. **Config**: the app's configuration as a `config` block rendered verbatim and validated at start-up (policy ships config loaders with JSON-Schema validation in Go, TypeScript, Python and Kotlin) + `secretFiles` projected files instead of env secrets
7. **service-lib adoption + interface number**: depend on `oci://ghcr.io/truvity/charts/service-lib` (private-consumer path) or render standalone with its grammar; declare the product's interface step

Validation gates:
- `helm lint` + `helm template | kubectl apply --dry-run=server -f -`
- Go chart tests for boundary contracts (tier, interface keys, tenant scope)
- Delivery: the platform (a separate GitOps control plane) renders and syncs; this repository publishes once, digest-pinned — the github-ci skill's half. A plain Job's name folds in the chart's version (spec immutable under re-apply); `job.annotations` is the force/replace escape

## Best Practices

- One `include` per component; cross-component agreement is DERIVED in one `_components.tpl` returning a JSON dict — nothing below it reads a value
- Never put secrets in values, never secrets as env — files via `secretFiles`
- Prod gate = ArgoCD manual sync; in-chart sync-waves handle ordering
- Version placeholders `0.0.0` in Chart.yaml — the release stamps them; a committed version two people can disagree about
- Names derive from ONE formula; two derivations = two things to keep in step
- Semantics over schemas: assert render boundaries in chart tests, not just lint

## Gotchas

- `service-lib` is NOT vendored beside the example — the example chart's `repository: file://../../../../charts/service-lib` resolves only inside the policy repo; consume via OCI or vendor the library beside your own charts
- Selector labels immutable across upgrades — extra labels go in `labels`, never `selectorLabels`
- NATS/JetStream names are GLOBAL: `installName` fold + namespace qualification prevent cross-install stream deletion
- IAM names ≤64 chars AWS cap (reserve suffix room; guard with `fail`)
- ACK CRs unappliable by tenants — cloud resources render `primary` tier only
- Omitted defaulted fields (Gateway API `group/kind/weight`) diff forever under ArgoCD — set explicitly
- Hook Jobs survive `helm uninstall` — `ttlSecondsAfterFinished` reaps residue; `before-hook-creation` avoids stuck prior Jobs
- A platform handed an interface number HIGHER than it implements refuses to render rather than render short

## Implementation Checklist

- [ ] Chart skeleton: service-lib dependency + interface annotation, two-block values, composed schema, packaging-baked images map
- [ ] `_components.tpl` derivation + migrate hook Job; guards in place
- [ ] `helm lint` + Go chart tests pass (boundaries: tier, interface keys, tenant scope)
- [ ] Server dry-run renders clean against the target cluster
- [ ] Delivery read-back: the chart holds the two-install-paths rules (`references/install-paths.md`); Applications are the platform's, never this repository's
