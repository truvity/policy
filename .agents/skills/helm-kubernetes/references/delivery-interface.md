# The Delivery Interface (Truvity policy contract)

One number per product so a platform hands a chart only the keys it can take. Condensed and self-contained; normative source (needs network): [docs/contracts/delivery-interface.md](https://github.com/truvity/policy/blob/master/docs/contracts/delivery-interface.md).

## The number

```yaml
annotations:
  delivery.truvity.io/interface: "15"
```

Rules:

1. **The number only grows; a step only adds keys.** Interface *N* accepts every key of steps 1..N and refuses step *N+1*'s. A rename or removal is a breaking change and gets a major version.
2. **A chart declares the highest step it reads**, and its schema is held to it.
3. **One number per product**: the registry names infrastructure, application and end-to-end charts released under one tag with the same number. The current example ships application + infra only; that disagreement is recorded below, not treated as a relaxed requirement.
4. **The platform reads the number as data**: renders every key of steps ≤ N, never compares per-key versions. A number higher than the platform implements → refuses to render rather than render short.
5. **No annotation means the pin predates it** (the policy repo keeps a first-release table per product).

A step is a capability the platform MAY pass, not a requirement that it does — a platform without workload identity passes no `tls` block whatever the number says.

## The steps

These summarize normative requirements. Differences in the worked example
are observations in the gap table below, not alternative requirements.

| Step | Adds | Chart | Meaning |
|---|---|---|---|
| 1 | the explicit interface | all | every address by name: `tier`, `cloud.*`, `postgres.*` (infra), `events.*`, `database.host`/`.owner`/`.app`, `archive.bucket.*`, `serviceAccount.app.name`, `route.enabled`/`.hostname`/`.parentRef`/`.surfaces`, `otel.*`, `tls.mode`/`.trustDomain`/`.grantRequest`/`.peers`, `identityProviders`, `access`, `product`. The floor. |
| 2 | `installName` | infra, application | one shared name from which the pair derives stream, subjects and consumers |
| 3 | the end-to-end chart | end-to-end | third chart: `appRelease`, `mode`, `database.*`, `events.*`, `archive.*`, `serviceAccount`, `job.annotations`, `otel`, `prober.*`, `traces.*` |
| 4 | `tls` on the e2e chart | end-to-end | prober can present a workload identity; `prober.serviceAccount.name` |
| 5 | `tls.components.<c>.mode` | application, e2e | transport identity per component (`off`/`permissive`/`strict`) |
| 6 | a ServiceAccount per component | application, e2e | `<release>-<component>`; prober's `tls.peers` names components' own accounts |
| 7 | `events.tls` | application | publisher authenticates to the broker with workload identity: `.enabled`, `.serverName`, `.caConfigMap` |
| 8 | `database.tls` | application | `verify-full` ONLY + `.rootCA.configMapName`/`.key` required — the client always verifies the server |
| 9 | `postgres.platformOwned` | infra | platform renders the database; the chart renders the runtime role only |
| 10 | `web.faro`, `route.faro` | application | browser telemetry: page runtime config + the public rule a browser reports through |
| 11 | `route.faro.requestBufferLimit` | application | chart renders the gateway's body-limit policy targeting that rule alone |
| 12 | `alerts` | application | product's own rules as a `VMRule` (`alerts.enabled`, off by default); `alerts.remote.*` renders the rule object alone for cross-cluster evaluation |
| 13 | `tier` on the application chart | application, infra | identity SA rendered by `-infra` AFTER its PodIdentityAssociation (sync-waves: IAM Role 0 → PIA 1 → SA 2, `Prune=false,Delete=false` permanent); platform passes `tier` to BOTH charts, same value (app default `test`); `-infra`'s `cloud.serviceAccountAnnotations` mirrors `serviceAccount.app.annotations` |
| 14 | `availability` | application | the platform's statement of the CLUSTER's redundancy, one word: `single` (default when absent — one replica, no budget that blocks a drain) or `high` (at least two replicas, a `maxUnavailable: 1` budget). The chart sizes its replicas and PodDisruptionBudget from it; an explicit `replicas.<component>` or `disruption.enabled` still wins; a platform no longer carries a per-product `replicas` stanza |
| 15 | the database runtime role by default; migration credentials under their own key | application, infra | the product connects as a separate RUNTIME role (DML on its own schema, no DDL); the infra chart always renders a DatabaseRole and refuses the role opt-out. Long-running workloads get `database.app.role`/`database.app.passwordSecret` only. The owner's credential: `database.migration.passwordSecret` (with `.role`, `.passwordKey`, defaulted), passed only to a product declaring `postgres.migrations: true`, mounted ONLY in migration Jobs — `conformance.MigrationSecretOnlyInJobs` fails other manifests naming it. `database.owner.*` is deprecated for it |

## Current implementation and contract gaps

Reviewed on 2026-10-20 against the rebased checkout after v1.54.1. The
example's charts declare 15, but an annotation and passing tests are not
proof that every normative requirement is satisfied.

| Normative requirement / source | Observed implementation | Known gap |
|---|---|---|
| Step 15: infra always renders a `DatabaseRole` | platform-owned mode does; standalone mode declares a managed role inside its Cluster and forbids a second DatabaseRole | a least-privilege role exists, but its representation differs from the required kind; semantic similarity is not an accepted exception |
| Step 3 and rule 3: an e2e chart is part of the product's chart set | the example stopped publishing it in v1.48.0, moved the prober into the app chart and runs its suite from CI | the chart-set requirement and the interface-15 example disagree |
| First-release table: steps 14–15 are “not yet” adopted | the charts declare 15 and the changelog records adoption in v1.48.0 | registry release metadata needs correction |
| Platform §10: cnpg-database values default to one live install's role | current values use empty roles and guards require explicit entries in the relevant modes | the documented defaults no longer describe the published example's inputs |

**Handling rule:** report the conflict with its source paths and render/release
evidence. Propose a separate contract correction before adopting behavior the
contract does not acknowledge, as AGENTS.md requires. Keep the implementation
inventory useful for diagnosis, but do not use a template, changelog or green
render test to silently override a normative rule. Until resolved, describe
the discrepancy as a conformance gap rather than claiming full conformance.

Platform rule 11 and decision 0013 additionally require an admission-time
identity-check hook; this is not step 16. Likewise, the example's locked
cnpg-database dependency, optional dashboards and `identityCheck.expectAccount`
do not advance its annotation. The account assertion still requires a checker
image that understands it (v0.34.0+); an unchanged interface number is not a
binary-compatibility check.
See [identity admission](identity-admission.md) and [platform resources](cluster-resources.md).

## Adding a step

1. Add the step row + the first-release version in the same pull request (policy repo).
2. Move the charts' annotation to the new number.
3. CHANGELOG under `### Contracts` — every implementing platform needs a release that knows the step first; that ordering is why a platform refuses an unknown number instead of rendering what it can.
