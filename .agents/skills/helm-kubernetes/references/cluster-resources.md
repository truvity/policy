# Target-Cluster Resource Inventory (policy example `url-shortener-infra`)

Reviewed on 2026-10-20 against the rebased checkout after policy v1.54.1.
This is the worked example's resource inventory, not a claim that every
platform runs these operators.
Sources: [platform contract](https://github.com/truvity/policy/blob/master/docs/contracts/platform.md),
[tenancy contract](https://github.com/truvity/policy/blob/master/docs/contracts/tenancy.md),
the example's charts and their render tests. The tier gates the cloud resource
chain; database ownership and credential generation are separate decisions.
The kind tier supplies fixtures instead of installing this infra chart.

## Exposure: Gateway API, NOT Ingress

```
HTTPRoute → route.parentRef.{name (required), group, kind, namespace, sectionName} — all explicit
```
- `route.enabled: false` keeps CI/local installs route-less; hostname required when enabled
- Alerts/telemetry can add their own rules (`route.faro`, step 11) with a policy targeting that rule alone
- Sets `backendRefs.{group,kind,weight}` explicitly (ArgoCD diff hygiene)

## Postgres: CNPG (CloudNativePG)

- The infra chart always resolves **cnpg-database 2.31.0** from its committed Chart.lock; the dependency has no ownership-mode condition. Its `roles` list now defaults **empty**. The parent renders neither a DatabaseRole nor Password/ExternalSecret templates of its own; it still renders a standalone Cluster when `postgres.platformOwned: false`.
- Role separation: OWNER (DDL/migration credential) vs RUNTIME (DDL-incapable); grants come from app migrations (`SET ROLE`), not role membership
- Client TLS is `verify-full` ONLY (interface step 8): server certificate must chain to the given root AND name the host dialled; `database.tls.rootCA.configMapName` + `.key` required — a platform whose database serves no certificate cannot move the product to it. Host dialled FQDN-qualified (`<host>.<ns>.svc.<clusterDomain>`)
- `database.host` and the root CA ConfigMap are platform inputs; the application does not reconstruct a platform's database hostname convention. Workloads receive runtime `database.app.*`; only the migration Job receives `database.migration.*`.
- Credentials may be provided by the platform or generated once via Password + ExternalSecret in infra; never template random password material into a Secret. In the subchart path the generated ExternalSecret has refresh interval `"0"`, so adopting the subchart must preserve the existing Secret/password.

### Ownership modes and parent/subchart agreement

**Normative requirement:** delivery-interface step 15 says the infra chart
always renders a `DatabaseRole`. **Observed implementation:** the standalone
example declares its runtime role in the Cluster instead and forbids a second
DatabaseRole. **Known conformance gap:** this differs from the required kind;
the matrix below inventories code, not an approved exception. Report it and
propose a separate contract correction before adopting that departure. See
the [contract-gap record](delivery-interface.md#current-implementation-and-contract-gaps).

Helm resolves subchart values before parent templates run. The platform states
the applicable `cnpg-database` entry explicitly alongside `postgres`; the
parent's `templates/guard.yaml` rejects disagreements.

| Cluster owner | `postgres.runtimePassword.generate` | `cnpg-database.roles` | Rendered result |
|---|---|---|---|
| platform (`postgres.platformOwned: true`) | false | exactly one runtime role with `passwordSecret`; `databaseRole` absent/true | subchart DatabaseRole references the external Secret; no password generator |
| platform | true | exactly one role with `generatedPassword`; `databaseRole` absent/true | subchart DatabaseRole, Password and once-only ExternalSecret |
| this infra chart (`postgres.platformOwned: false`) | true | exactly one role with **`databaseRole: false`** and `generatedPassword` | parent Cluster declares the managed role; subchart renders only Password/ExternalSecret |
| this infra chart | false | **empty** | parent Cluster declares the managed role and references the externally populated Secret; subchart renders nothing |

For an explicit entry:

- `.name` matches the resolved `postgres.runtimeRole`, including tenant scoping;
- labels include the chart's name and actual infra release's instance label;
  platform role/password-resource labels go in this entry, not `postgres.labels`;
- `generatedPassword.secretName` and `.length` match the parent when generation
  is enabled; otherwise platform-owned mode uses `passwordSecret` with no generator;
- in platform-owned mode `cnpg-database.clusterName` must equal
  `<infra-release>-pg`. This **example guard** is not permission for an application
  to reconstruct its database host rather than accept the platform's value;
- Helm replaces the roles list, so supply the complete applicable entry.

**Adoption change since v1.53.0:** the old default role for one live install is
gone. Even that install now supplies a role entry in platform-owned mode, and
an own-Cluster install that generates a password adds its password-only entry.
`charts/testdata/infra-generated-password.yaml` is the latter's neutral fixture.
The platform contract's menu still describes the older live-install defaults.
That documented-input mismatch is also a [known gap](delivery-interface.md#current-implementation-and-contract-gaps):
record current values, guards and changelog evidence, and propose a separate
contract correction before adopting discrepant defaults as policy.

`TestTheInfraChartPinsCnpgDatabase` checks the version/lock and unconditional
dependency. Infra tests check both ownership modes, empty roles, password-only
generation, preserved once-only refresh and refusal of mismatches, including
the duplicate-owner case. The example's `hack/prove-infra-render.sh` compares
objects across refs with supplied real values; keep those values outside this
public repository. Preserve names, Secret keys and `refreshInterval: "0"` so
adoption does not rotate a password or introduce a second role owner.

### Database health and retry wiring

The current example pins `truvity/cnpg/v2` **v2.32.0** in go.mod. This is the
application's client version, distinct from the cnpg-database **chart** pin.
Use its `clients/go/pgclient`: connection parts arrive through `PGHOST`,
`PGPORT`, `PGDATABASE`, `PGUSER`, `PGSSLMODE=verify-full`, `PGSSLROOTCERT`,
`CNPG_CLIENT_PASSWORD_FILE` and `PGAPPNAME`, not a connection URL. The password
file is re-read on each new connection; the CA is directory-mounted, not subPath.
Where the platform supplies client certificates, the client reads `PGSSLCERT`
and `PGSSLKEY` without an application-specific connection-string layer.

Readiness is the client's `Health`: `SELECT 1` through the actual Service/TLS/
password path with a two-second bound. Liveness remains process-only. Safe
writes and idempotent request reads use `Database.Do`; the incrementing click
counter does not, since replaying a committed increment counts twice.

Since client v2.30.0, request retry uses **the library default**, not the removed
example `requestRetry`: no try cap, 0.2-to-2-second back-off, **45-second budget**.
The caller's own deadline can end it sooner. Startup still has separate
three-minute patience (0.5-to-10-second back-off); `CNPG_CLIENT_RETRY_*` tuning
wins when supplied. These are client-library env variables, not a second
service configuration system. Store/runtime tests cover retry decisions; a
render does not prove a live database switchover.

## Messaging: NATS JetStream via NACK CRs

- `Stream` (jetstream.nats.io) rendered by `-infra`; controller connects to the platform broker
- Stream/subject/durable names fold in `installName` + namespace — names are GLOBAL on shared NATS; qualification prevents cross-install stream deletion. `test` tier shares the cluster's standing broker (two test installs on it get different names)
- Events auth (step 7): publisher authenticates with its workload identity — `events.tls.{enabled, serverName, caConfigMap}`

## Cloud identity: ACK + EKS Pod Identity in the example

- CRs in `-infra`/`cloud.yaml`: `s3.services.k8s.aws Bucket`, `iam.services.k8s.aws Policy/Role`, `eks.services.k8s.aws PodIdentityAssociation`
- Infra orders IAM Role 0 → PodIdentityAssociation 1 → ServiceAccount 2 with permanent `Prune=false,Delete=false`. Install infra healthy first, then the app renders `service-lib.identityCheck` for a cloud-identity component. `Synced` can precede webhook visibility; the check retries with a new pod, not a same-pod restart. Full contract, image ownership and check-specific egress: [identity admission](identity-admission.md).
- `cloud.serviceAccountAnnotations` mirrors `serviceAccount.app.annotations` onto the identity account
- Per-component ServiceAccounts `<release>-<component>` (step 6); the prober's `tls.peers` names components' own accounts, not a shared one
- Least privilege down to the action actually issued; `permissionsBoundary` + tags where the deploy role demands them
- Persistent data: `services.k8s.aws/deletion-policy: retain` on primary-tier Buckets + preserve tags

The binding mechanism is not hard-coded in service code. The identity-check
command supports IRSA as well; do not turn an example's Pod Identity choice
into a claim that every platform retired other mechanisms.

## Tenancy tiers

| Tier | Cloud store/identity resources | Database/stream and local testing |
|------|----------------|----------|
| `primary` | Bucket/Policy/Role/PIA and infra-owned identity ServiceAccount | database resources follow ownership mode; Stream follows the broker values |
| `test` | no cloud store/role/association chain | the infra render can still contain CNPG/NATS kinds; kind supplies standing servers through the example fixture instead |

- The tier gate is a cloud-provisioning boundary, not proof that no custom
  resource renders. Infra tests check the cloud kinds per tier. Operator-CR
  rights are platform grants through the role spine; do not assume namespace
  `admin` grants those rights or that kind validates their reconciliation.

## NetworkPolicies

The current application chart renders no NetworkPolicy. The infrastructure
chart adds app-specific egress to its store and separate identity-check
egress. Namespace default-deny/shared allows and platform-owned database
policy remain the platform's. Do not copy that baseline into the product to
work around an install ordering problem. Match check labels separately from
workload selectors; policy enforcement needs a real platform test.

## Namespace prerequisites (tenancy contract)

Before a workload starts, the platform supplies namespace labels, Pod Security
level/version and protected deletion policy; default-deny ingress/egress with
shared and intra-namespace allows; ResourceQuota/LimitRange; the admin/edit/read
role spine and identity bindings. Departures from Pod Security defaults need a
recorded reason and tightening rolls out warn-first. A namespace's default
ServiceAccount carries no external rights.

The product may add its own allowed flows but cannot create/relabel the
namespace, replace the baseline/quota/limits or grant itself cluster-wide
rights. The tenancy contract's §8 is a render-review checklist; the platform
must test its own admission refusals and baseline enforcement. A green kind
lane does not prove those platform guarantees.

## Dashboards and resource sizing

The product owns dashboard JSON and optional operator objects; the platform
owns the operator, shared Grafana, parent folder and datasource identities.
In the example `observability.dashboards.enabled` defaults false and renders
nothing in `alerts.remote` mode. Enablement requires `clusterFolderUID` and
both `datasources.metrics` and `.logs`; `instanceSelector` is overridable.

- GrafanaFolder wave **0**, GrafanaDashboard wave **1**: a dashboard applied
  before its folder can fail without an operator retry.
- Both resources set `allowCrossNamespaceImport: true` when matching the
  shared Grafana in another namespace.
- The JSON dashboard UID is `<clusterFolderUID>-url-shortener-overview`, not
  a globally fixed product string; `spec.uid` stays unset. The example limits
  the folder input to 17 characters to keep that derived UID within 40.
  Other products calculate their own suffix budget.
- `TestTheChartRendersItsDashboardsOnlyWhenAsked` covers gates, required inputs,
  both cross-namespace flags, UID derivation and ordering. Live operator/Grafana
  permissions remain a platform integration test.

Declare workload memory limits as well as requests, based on measurements
rather than copying the example's numbers. The example JVM uses
`jvmOptions.stat` → `JAVA_TOOL_OPTIONS` with heap-as-a-share-of-container-limit
and exit-on-OOM; resource values and that option are configurable. A namespace
LimitRange is a fallback, not evidence the product's heap fits its limit.

## Platform-injected vs project-declared values

The platform passes install-specific values BY NAME (delivery interface; full menu tables in `references/platform-values.md`): database host + roots, secrets by name, broker, bucket, tier, availability, route coordinates, otel, tls. The chart carries only app-generic defaults. Never hardcode install-specific coordinates — "a nonexistent route → an UNPROTECTED app route".

Real environment values are authored in the GitOps control-plane repository,
not deployment files in the product repository. The product owns generic
defaults and **nonsecret test fixtures**, which exercise the same values
interface without naming a real deployment. C13 enforces selected estate
defaults and explicitly distinguishes fixtures from defaults; it is not a
universal filename ban. Review ownership as well as file names.

Secret objects and store-fetching integrations belong to the platform or
appropriate infra release; the app takes Secret names and keys. On Kubernetes,
`platform.secretFiles` is a read-only projected volume at `config.secrets.root`
using the **file** source. Do not interpret config's `ssm`/`openbao` options as
permission for Kubernetes workloads to bypass Secret objects. The platform
provides population and rotation; values and fixture ConfigMaps carry names,
never credentials.
