---
name: helm-kubernetes
description: >-
  Use when migrating or reviewing Helm charts and Kubernetes manifests under the
  Truvity policy contracts. Covers cloud-identity admission checks and the
  example at delivery interface 15: service-lib library chart, two-block
  values, composed schemas, tier, availability, runtime database role,
  digest pinning, CNPG, EKS Pod Identity, chart tests — delivered via
  ArgoCD. Repository-agnostic; reference is the policy example
  url-shortener
---

# Helm Kubernetes

Helm chart style for service charts delivered via ArgoCD, per the Truvity policy contract.

## Reference

Self-contained detail, shipped with this skill in `references/`:

- `references/chart-patterns.md` — product vs service charts, layout, derivation, checks
- `references/cluster-resources.md` — database dependencies, dashboards, tenancy, environment and secret ownership
- `references/delivery-interface.md` — the number and all 15 steps, with historical caveats
- `references/identity-admission.md` — the required check, image ownership, selectors, egress and coverage limits
- `references/install-paths.md` — Helm and render/apply compatibility
- `references/platform-values.md` — component block, delivery menu and composed schema example

Siblings: `devbox-just` owns the gate and toolchain; `github-ci` owns workflow wiring and release.

## Snapshot

Reviewed 2026-10-26: policy `v1.54.5` owns contracts, schemas and service-lib;
[url-shortener][product] owns the worked charts/tests. The checker image
`truvity/k8s` **v0.34.0+** is required before using `identityCheck.expectAccount`.
Not pin authority: if this date predates the checkout's CHANGELOG head,
re-verify pins before relying on them. References defer here.

[product]: https://github.com/truvity/url-shortener/tree/76770607aa08baa58331668c228dacd62a13a89a
[product-charts]: https://github.com/truvity/url-shortener/tree/76770607aa08baa58331668c228dacd62a13a89a/charts

External sources: [product charts/tests][product-charts], policy's [library](https://github.com/truvity/policy/tree/master/charts/service-lib), [delivery interface](https://github.com/truvity/policy/blob/master/docs/contracts/delivery-interface.md), [platform](https://github.com/truvity/policy/blob/master/docs/contracts/platform.md), [chart guide](https://github.com/truvity/policy/blob/master/docs/guides/charts.md), and [extraction decision 0014](https://github.com/truvity/policy/blob/master/docs/decisions/0014-the-worked-example-lives-in-its-own-repository.md). Product-relative paths below belong to url-shortener, not this checkout.

## When sources disagree

Contracts are normative; templates and changelogs describe observed behavior.
Record requirement, observation and conformance gap separately, and propose a
contract correction **before** adopting unacknowledged behavior (AGENTS.md);
an example or green test is not an exception.

## Chart Style Essentials

Each rule says what must be true; the references carry the full patterns. The
application/infra split, identity check and namespace prerequisites are separate
rules, not new interface steps to invent.

### Ownership and lifetime

| Rule | What must be true |
|---|---|
| **Application/infra split** | State and cloud identity MUST use a separate release with a longer lifetime; a migration hook cannot depend on resources its own release has not created (platform §11); the observed shape is in `references/chart-patterns.md`. |
| **Environments are the platform's** | Per-environment values (`tier`, `availability`, database host, secret names, routes) and Argo CD Applications live in GitOps. The product ships generic defaults, schemas and nonsecret fixtures; no real `values-<env>.yaml` files. C13 checks selected defaults; deployment-file ownership stays review. |
| **Secrets are named files** | The platform or `-infra` owns Secret objects; the application takes names and keys only. `platform.secretFiles` projects them read-only at `config.secrets.root`, mode 0440, `secrets.source: file`; `ssm`/`openbao` serve platforms without Kubernetes Secrets. Checks: `SecretsAreFiles`, `EnvIsDeclared`, `NoEnvSecretFields`. |
| **Runtime vs owner credentials** | Interface 15 requires a runtime `DatabaseRole` with no opt-out; long-running workloads get `database.app.*` only, and owner credentials are migration-Job-only under `database.migration.*` (`conformance.MigrationSecretOnlyInJobs`). The example's Cluster-managed role is a known gap recorded in `references/delivery-interface.md`, not an accepted alternate rule. |
| **Observability and limits** | Dashboards are product-owned, off by default without the operator; the platform supplies datasource/folder coordinates, folder wave 0 precedes dashboard wave 1. Workloads declare requests/limits; a JVM sizes its heap from its memory limit; see `references/cluster-resources.md`. |
| **Library ownership** | The product resolves OCI service-lib at its policy Go-module version, with grouped bumps and an equality test; only policy releases it. C15 and older guide text still mandate co-located `file://`/`0.0.0`; record the source conflict in `references/delivery-interface.md`, not a silent amendment. |

### Values and schema

| Rule | What must be true |
|---|---|
| **Two-block values** | `config` is the service's own configuration, rendered VERBATIM into its file (the binary validates at start-up); `platform` is everything the platform provides (key table in `references/platform-values.md`); plus `images`. No `env`/`extraEnv` hatches — ports, probes and grace DERIVE from config; `checksum/config` restarts pods on change. |
| **Composed schema** | Commit `values.schema.src.json` and its generated bundle, then check drift (C17). A multi-component product chart is C16's named exception: a strict hand-written delivery-values schema and one derivation template, whose output still validates against each binary's schema. |
| **Delivery interface** | Declare `delivery.truvity.io/interface` as the highest step read (example: 15). Steps only ADD keys; the platform renders supported steps and refuses an unknown higher number; the declaration must match schema/render tests, not a copied number. An external library dependency is not another product chart; the step list is in `references/delivery-interface.md`. |
| **`availability: single\|high`** | Step 14: the platform's statement of the CLUSTER, one word the chart sizes itself from. `single` (default when absent): one replica, no budget that blocks a drain; `high`: at least two replicas and a `maxUnavailable: 1` budget. Explicit `replicas.<component>` or `disruption.enabled` wins; no per-product `replicas` stanza. |
| **`installName`** | Platform-passed, defaults to the release name; folds into NATS subjects and durable consumers with namespace qualification so two installs cannot find each other's stream; DNS-1123-validated at render. |
| **Images** | The release bakes its own component digests into `images.{component}` and refuses unpinned entries. Third-party tooling such as `images.identity-check` carries its owner's tag+digest; this repository does not rebuild or publish it. |
| **Guards** | `required` for conditional-mandatory values (`route.parentRef.name`); render-time `fail` on name/validation violations; strict schemas (`additionalProperties: false`), including conditional `if/then/else`. |

### Ordering and identity

| Rule | What must be true |
|---|---|
| **`tier: test\|primary`** | Step 13 passes the same value to both charts. For cloud identity the infra release renders role → binding → ServiceAccount in increasing waves, retains `Prune=false,Delete=false` on the account, and must be healthy before the application. A `Synced` binding can still be invisible at pod admission: waves alone are not proof. |
| **Identity check** | A cloud-identity component requires `service-lib.identityCheck`: a pre-install/pre-upgrade hook rendered as that component's account, no deployer opt-out. `restartPolicy: Never` retries with a NEW pod. The example exposes optional `expectAccount` (empty by default) to check the target AWS account; it needs checker **v0.34.0+**, pinned by tag+digest. Dedicated labels and infra-owned egress keep it out of Service/PDB selectors; details in `references/identity-admission.md`. |
| **Migrate hook Job** | `pre-install,pre-upgrade`, weight 0, `hook-delete-policy: before-hook-creation,hook-succeeded`; `database.tls` is `verify-full` ONLY (root CA required); the migration runs as the owner credential, mounted nowhere else. |
| **Two install paths** | A chart cannot tell which installer it is in: no `lookup`, nothing reading `.Release.Is*`/`.Revision`, every hook idempotent on every sync, plain Jobs carry no `ttlSecondsAfterFinished` (a self-healing controller recreates a self-deleting Job, forever); details in `references/install-paths.md`. |

### Rendering and checks

| Rule | What must be true |
|---|---|
| **service-lib** | `type: library`, `oci://ghcr.io/truvity/charts`, `Chart.lock` committed. It renders Deployment, ServiceAccount, Service, probes, mounts, telemetry/client-library env and ConfigMap; a one-service chart is a single `include "service-lib.component"` with `(dict "context" $ "name" X "platform" ... "config" ...)`. |
| **Derivation** | One `include` per component; cross-component agreement is DERIVED in one `_components.tpl` returning a JSON dict — nothing below it reads a value. |
| **Chart tests (Go)** | Render + golden testdata + assert SEMANTIC contracts: tier boundaries, delivery-interface key acceptance, tenant scope, restricted fields. "Every wrong combination still renders valid YAML." |
| **Validation gates** | Product: `just test` (render/schema/lock) and `just charts` (API schemas). Policy: Go unit tests and advisory `just example-canary`. Product integration installs packaged charts in its protected workflow. Neither schema tests nor the canary prove live admission/IAM/policy enforcement. |

## Writing a Product Chart

A product chart renders components through `service-lib` and tests its supported
delivery interface; service charts compose schemas, product charts keep their
reviewed mapping. Resolve exposure (Gateway HTTPRoute only), Postgres ownership,
digest-pinned images, per-install state, migration hooks, verbatim config and
locked library/namespace prerequisites before writing templates. The ownership
matrix, decisions and layout are in `references/cluster-resources.md` and
`references/chart-patterns.md`.

## Best Practices

- When the platform syncs is the platform's business; the chart's ordering travels in-chart (sync-waves, hook weights).
- Version placeholders `0.0.0` in Chart.yaml — the release stamps them; a committed version two people can disagree about.

## Gotchas

- Selector labels immutable across upgrades — extra labels go in `labels`, never `selectorLabels`.
- IAM names ≤64 chars AWS cap (reserve suffix room; guard with `fail`).
- Cloud store/identity resources render on `primary`; operator-CR access needs platform grants. A `test` infra render is not guaranteed to contain zero CNPG/NATS resources; a chosen local tier needs appropriate fixtures/operators.
- Omitted defaulted fields (Gateway API `group/kind/weight`) diff forever under ArgoCD — set explicitly.
- `postgres.labels` does not propagate through the `cnpg-database` subchart: put role labels in `cnpg-database.roles[0].labels`. Helm replaces the roles list; it cannot derive its values from parent templates.
- A gateway-fronted cleartext workload cannot be `strict`: `tls.gatewayFronted` and `transport.CheckMode` refuse it. Keep transport off in chart defaults; let the platform enable a supported mode.
- An older identity-check image cannot recognize `-expect-account`; every retry fails. Move the verified image pin before enabling `identityCheck.expectAccount`, and quote the platform-supplied account string so leading zeros survive YAML.

## Implementation Checklist

- [ ] Chart skeleton: locked service-lib dependency, tested interface annotation, service schema composition or reviewed product mapping, digest-pinned images; `_components.tpl` derivation + migrate hook Job + guards
- [ ] `helm lint` + Go chart tests pass (boundaries: tier, interface keys, tenant scope)
- [ ] Cloud identity: infra healthy first; guarded account, check hook, isolated selectors/egress; checker v0.34.0+ before an optional expected-account assertion
- [ ] DB ownership modes: locked cnpg-database, explicit matching roles when needed, `databaseRole: false` for password-only generation; render comparison preserves names/spec/Secret
- [ ] Optional dashboards: platform coordinates, unique UIDs, cross-namespace import, folder before dashboard
- [ ] `availability` sizes replicas + PDB (defaults: one replica, no budget under `single`); no owner credential outside the migration Job
- [ ] Real environment values and Applications stay in GitOps; only generic defaults and nonsecret fixtures in the product; Kubernetes secrets use the file source; the chart holds the two-install-paths rules
