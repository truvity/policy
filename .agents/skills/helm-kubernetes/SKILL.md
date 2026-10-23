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

- [Chart patterns](references/chart-patterns.md) — product vs service charts, derivation, checks
- [Platform resources](references/cluster-resources.md) — database dependencies, dashboards, tenancy, environment and secret ownership
- [Delivery interface](references/delivery-interface.md) — the number and all 15 steps, with historical caveats
- [Identity admission](references/identity-admission.md) — the required check, image ownership, selectors, egress and coverage limits
- [Install paths](references/install-paths.md) — Helm and render/apply compatibility
- [Platform values](references/platform-values.md) — component block, delivery menu and composed schema example

Reviewed on 2026-10-20 against the rebased checkout after policy v1.54.1
(product source through `37cd9b4`). The worked example declares
interface 15; declare only the highest step a migrating product actually supports.
The application/infra split, identity admission check and namespace prerequisites
are separate rules, not new interface steps to invent.

External source of truth (needs network): [github.com/truvity/policy](https://github.com/truvity/policy) — worked example [`examples/url-shortener/charts/`](https://github.com/truvity/policy/tree/master/examples/url-shortener/charts) and library [`charts/service-lib`](https://github.com/truvity/policy/tree/master/charts/service-lib); normative docs: [delivery-interface](https://github.com/truvity/policy/blob/master/docs/contracts/delivery-interface.md), [platform §10](https://github.com/truvity/policy/blob/master/docs/contracts/platform.md), [charts guide](https://github.com/truvity/policy/blob/master/docs/guides/charts.md). Consumer repositories may vendor the example under a different root; adjust paths accordingly.

## When sources disagree

Contracts are normative; templates and changelogs describe observed behavior.
Record the requirement, observation and conformance gap separately. Report a
conflict and propose a separate contract correction **before** adopting behavior
the contract does not acknowledge (AGENTS.md, "When the contract is wrong").
An example, green test or bundled summary is not an exception to that rule.

## Chart Style Essentials (see references for full patterns)

- **Split application and infrastructure when there is something to provision**: database/stream state has a separate lifetime; a migration hook cannot depend on ordinary resources its own release has not created. A component with cloud identity MUST use the two-release split (platform §11). The example ships application + `-infra` under one tag; its optional prober is an application Deployment, while the e2e suite runs from CI, outside-in or as an ephemeral Job. It no longer publishes an e2e chart.
- **service-lib library chart** (`type: library`, `oci://ghcr.io/truvity/charts`, `Chart.lock` committed): renders Deployment, ServiceAccount, Service, probes, mounts, telemetry/client-library env, ConfigMap. A one-service chart's template is a single `include "service-lib.component"` with `(dict "context" $ "name" X "platform" ... "config" ...)`.
- **Two-block values**: `config` = the service's own configuration schema, rendered VERBATIM into its file (binary validates at start-up); `platform` = everything the platform provides (full key table: `references/platform-values.md`); plus the `images` map. No `env`/`extraEnv` hatches — ports/probes/grace DERIVE from config (`listen.address`, `probes.address`, `drain.seconds`); `checksum/config` restarts pods on change.
- **Composed service schema**: commit `values.schema.src.json` and its generated bundle, then check drift (C17). A multi-component product chart is C16's named exception: it may have a strict hand-written delivery-values schema and derive component inputs in one template; the resulting files still validate against each binary's schema.
- **Delivery interface**: the example's product charts declare `delivery.truvity.io/interface: "15"`; an external library dependency is not another product chart. Steps only ADD keys; the platform renders supported keys up to the declared step and refuses an unknown higher number. A declaration must match schema/render tests, not a copied number.
- **Environments are the platform's**: real per-environment values (`tier`, `availability`, database host, secret names, routes) and Argo CD Applications live in the GitOps repository. The product ships generic defaults, schemas and nonsecret render fixtures. Do not add real `values-<env>.yaml` deployments to the product; test fixture files are allowed. C13 checks selected estate defaults; ownership of deployment files remains review.
- **Kubernetes secrets are named files**: the platform or `-infra` creates/populates Secret objects and store-fetching resources; the application takes names and keys only. `platform.secretFiles` projects them read-only at `config.secrets.root`, mode 0440, with `secrets.source: file`. `ssm`/`openbao` are configuration-source capabilities for platforms without Kubernetes Secrets, not alternative store access for a Kubernetes service. Check `SecretsAreFiles`, `EnvIsDeclared` and `NoEnvSecretFields`.
- **tier `test|primary`** goes to both charts. For cloud identity the infra release renders role → binding → ServiceAccount in increasing waves, retaining `Prune=false,Delete=false` on the account; install it and wait for it to be healthy first. The application only references that account. A `Synced` binding can still be invisible at pod admission: waves alone are not proof.
- **Require `service-lib.identityCheck` for a cloud-identity component**: the application derives `cloudIdentity` and renders the pre-install/pre-upgrade hook as that component's account; no deployer opt-out. `restartPolicy: Never` retries with a NEW pod. Optional `expectAccount` checks the target AWS account; the example exposes `identityCheck.expectAccount`, empty by default. It needs the `truvity/k8s` checker image **v0.34.0+**, pinned by tag+digest before using the flag. Dedicated labels and infra-owned egress keep the check out of Service/PDB selectors. Details and tests: [identity admission](references/identity-admission.md).
- **`availability: single|high`** (step 14): the platform's statement of the CLUSTER, one word the chart sizes itself from — `single` (default when absent): one replica, no budget that blocks a drain; `high`: at least two and a `maxUnavailable: 1` budget. An explicit `replicas.<component>` or `disruption.enabled` still wins; a platform no longer carries a per-product `replicas` stanza.
- **The database runtime role is required** (step 15): the normative text requires a `DatabaseRole` with no opt-out. The observed standalone example instead declares a Cluster-managed role: this is a [known conformance gap](references/delivery-interface.md#current-implementation-and-contract-gaps), not an accepted alternate rule. Long-running workloads get `database.app.*` only; owner credentials remain migration-Job-only under `database.migration.*`, checked by `conformance.MigrationSecretOnlyInJobs`.
- **installName** (platform-passed, defaults to release name): folds into NATS subjects/durable consumers so two installs can't find each other's stream; DNS-1123-validated at render.
- **Images**: the release bakes its own component digests into `images.{component}` and refuses unpinned entries. Third-party tooling such as `images.identity-check` already carries its owner's tag+digest; this repository does not rebuild or publish it.
- **Guards**: `required` for conditional-mandatory values (`route.parentRef.name`), render-time `fail` on name/validation violations.
- **Migrate hook Job**: `pre-install,pre-upgrade`, weight 0, `hook-delete-policy: before-hook-creation,hook-succeeded`; `database.tls` is `verify-full` ONLY (root CA required).
- **Chart tests** (Go): render + golden testdata + assert SEMANTIC contracts — tier boundaries, delivery-interface key acceptance, tenant scope, restricted fields. "Every wrong combination still renders valid YAML."
- **A chart cannot tell which installer it is in**: no `lookup`, nothing reading `.Release.Is*`/`.Revision`, every hook idempotent on every sync, plain Jobs carrying no `ttlSecondsAfterFinished` (a self-healing controller recreates a self-deleting Job, forever) — `references/install-paths.md`
- **Product observability and resource limits**: optional dashboards ship with the product chart, off by default without the operator; the platform supplies datasource/folder coordinates. Folder wave 0 precedes dashboard wave 1; both allow cross-namespace import and the dashboard UID includes cluster scope. Workloads declare requests/limits; a JVM sizes its heap from its memory limit. See [platform resources](references/cluster-resources.md).

## Writing a Product Chart

A product chart renders components through `service-lib` and tests its supported
delivery interface. Service charts compose their schemas; product charts keep
their reviewed mapping and strict delivery schema. The platform provides shared
infrastructure; the infra release provisions what this install actually owns.

```
charts/<product>/
├── Chart.yaml          # service-lib dependency + supported interface annotation
├── Chart.lock          # commit the resolved dependency lock, not charts/ archives
├── values.yaml         # app-generic defaults only — install-specific values come from the platform
├── values.schema.*     # composed for a service; strict delivery schema for a product
├── dashboards/         # optional product-owned dashboard JSON
└── templates/          # _components.tpl (the ONE derivation place); components, migrate hook, prober, route, alerts, PDB
```

Decisions to resolve before writing templates:
1. **Exposure**: HTTPRoute on the platform Gateway (`route.enabled`, `parentRef.name` required) — TLS terminates at the Gateway, never chart-managed Ingress/certs
2. **Postgres**: the example always resolves locked `cnpg-database` **2.31.0**; `roles` defaults empty. Platform-owned mode requires one explicit runtime role entry. With its own Cluster and generated password, the entry must set `databaseRole: false` so only password resources render; with an external Secret, leave roles empty. Parent/subchart values must agree. See the ownership matrix in [platform resources](references/cluster-resources.md).
   This inventories the example; resolve its recorded standalone-role contract
   gap before adopting that departure as a migration pattern.
3. **Images**: baked at packaging — ONE release job builds and pushes the images, then packages the chart from the digests they produced; never floating tags
4. **Other stateful dependencies**: use platform-provided/shared resources or provision per-install state in the infra release; do not fold that state into the application release to avoid an ordering problem
5. **Migrations**: pre-install/pre-upgrade hook Job (`before-hook-creation,hook-succeeded`, TTL reaps residue) — never initContainers
6. **Config**: the app's configuration as a `config` block rendered verbatim and validated at start-up (policy ships config loaders with JSON-Schema validation in Go, TypeScript, Python and Kotlin) + `secretFiles` projected files instead of env secrets
7. **Library and namespace prerequisites**: resolve `service-lib` from a committed lock, not hand-written replacement workloads (C15). The platform supplies namespace labels, Pod Security, baseline policy, quota, limits and role spine before workloads start (tenancy contract); an application chart does not provision that namespace kit.

Validation gates:
- `just test` for schema, lock, render/golden and refusal checks; `just charts` for API-schema validation (a separate lane). A server dry-run is an additional cluster check, not part of the credential-free gate.
- Go chart tests for boundary contracts (tier, interface keys, tenant scope)
- Test the release-packaged artifact in the kind lane. Cloud admission, IAM and policy enforcement still need a platform test; a chart golden cannot prove them. Persistent delivery remains the platform's job; CI can install ephemeral test releases.

## Best Practices

- One `include` per component; cross-component agreement is DERIVED in one `_components.tpl` returning a JSON dict — nothing below it reads a value
- Never put secrets in values, never secrets as env — files via `secretFiles`
- When the platform syncs is the platform's business; the chart's ordering travels in-chart (sync-waves, hook weights)
- Version placeholders `0.0.0` in Chart.yaml — the release stamps them; a committed version two people can disagree about
- Names derive from ONE formula; two derivations = two things to keep in step
- Semantics over schemas: assert render boundaries in chart tests, not just lint

## Gotchas

- `service-lib` is NOT vendored beside the example — the example chart's `repository: file://../../../../charts/service-lib` resolves only inside the policy repo; consume via OCI or vendor the library beside your own charts
- Selector labels immutable across upgrades — extra labels go in `labels`, never `selectorLabels`
- NATS/JetStream names are GLOBAL: `installName` fold + namespace qualification prevent cross-install stream deletion
- IAM names ≤64 chars AWS cap (reserve suffix room; guard with `fail`)
- Cloud store/identity resources render on `primary`; operator-CR access needs platform grants. A `test` infra render is not guaranteed to contain zero CNPG/NATS resources — the kind lane uses fixtures instead of installing it
- Omitted defaulted fields (Gateway API `group/kind/weight`) diff forever under ArgoCD — set explicitly
- Hook Jobs survive `helm uninstall` — `ttlSecondsAfterFinished` reaps residue; `before-hook-creation` avoids stuck prior Jobs
- A platform handed an interface number HIGHER than it implements refuses to render rather than render short
- `postgres.labels` does not propagate through the `cnpg-database` subchart: put role labels in `cnpg-database.roles[0].labels`. Helm replaces the roles list; it cannot derive its values from the parent templates.
- A gateway-fronted cleartext workload cannot be `strict`: `tls.gatewayFronted` and `transport.CheckMode` refuse it. Keep transport off in chart defaults; let the platform enable a supported mode.
- An older identity-check image cannot recognize `-expect-account`; every retry fails. Move the verified image pin before enabling `identityCheck.expectAccount`, and quote the platform-supplied account string so leading zeros survive YAML.

## Implementation Checklist

- [ ] Chart skeleton: locked service-lib dependency, tested interface annotation, service schema composition or reviewed product mapping, digest-pinned images
- [ ] `_components.tpl` derivation + migrate hook Job; guards in place
- [ ] `helm lint` + Go chart tests pass (boundaries: tier, interface keys, tenant scope)
- [ ] Cloud identity: infra healthy first; guarded account, check hook, isolated selectors/egress; checker v0.34.0+ before an optional expected-account assertion
- [ ] DB ownership modes: locked cnpg-database, explicit matching roles when needed, `databaseRole: false` for password-only generation; render comparison preserves names/spec/Secret
- [ ] Optional dashboards: platform coordinates, unique UIDs, cross-namespace import, folder before dashboard
- [ ] `availability` sizes replicas + PDB (defaults: one replica, no budget under `single`); no owner credential outside the migration Job (`conformance.MigrationSecretOnlyInJobs` at interface ≥ 15)
- [ ] Real environment values and Applications stay in GitOps; only generic defaults and nonsecret test fixtures in the product; Kubernetes secrets use the file source
- [ ] Delivery read-back: the chart holds the two-install-paths rules (`references/install-paths.md`); Applications are the platform's, never this repository's
