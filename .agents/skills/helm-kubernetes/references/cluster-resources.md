# Target-Cluster Resource Inventory (policy example `url-shortener-infra`)

Resources the platform provides and the charts wire to. Infra resources render on the `primary` tier only; `test` provisions nothing.

## Exposure: Gateway API, NOT Ingress

```
HTTPRoute → route.parentRef.{name (required), group, kind, namespace, sectionName} — all explicit
```
- `route.enabled: false` keeps CI/local installs route-less; hostname required when enabled
- Alerts/telemetry can add their own rules (`route.faro`, step 11) with a policy targeting that rule alone
- Sets `backendRefs.{group,kind,weight}` explicitly (ArgoCD diff hygiene)

## Postgres: CNPG (CloudNativePG)

- Per-install `Cluster` + `DatabaseRole` in the `-infra` chart; read-write service `{release}-pg-rw` (`database.host` is a value only an install can give — schema demands non-empty)
- Role separation: OWNER (DDL/migration credential) vs RUNTIME (DDL-incapable); grants come from app migrations (`SET ROLE`), not role membership
- Client TLS is `verify-full` ONLY (interface step 8): server certificate must chain to the given root AND name the host dialled; `database.tls.rootCA.configMapName` + `.key` required — a platform whose database serves no certificate cannot move the product to it. Host dialled FQDN-qualified (`<host>.<ns>.svc.<clusterDomain>`)
- `postgres.platformOwned` (step 9): platform renders the database; the chart renders the runtime role only
- Credentials via `Password` + `ExternalSecret` (ESO) — never chart-rendered Secret literals
- Storage sizing gotchas: WAL ≥2Gi (max_slot_wal_keep_size ~50% + PG default max_wal_size 1GB collide with 1Gi); EBS cannot shrink — right-sizing is recreate-only (fresh initdb + new `serverName` generation, must not archive into predecessor's dir)

## Messaging: NATS JetStream via NACK CRs

- `Stream` (jetstream.nats.io) rendered by `-infra`; controller connects to the platform broker
- Stream/subject/durable names fold in `installName` + namespace — names are GLOBAL on shared NATS; qualification prevents cross-install stream deletion. `test` tier shares the cluster's standing broker (two test installs on it get different names)
- Events auth (step 7): publisher authenticates with its workload identity — `events.tls.{enabled, serverName, caConfigMap}`

## AWS: ACK + EKS Pod Identity (IRSA retired)

- CRs in `-infra`/`cloud.yaml`: `s3.services.k8s.aws Bucket`, `iam.services.k8s.aws Policy/Role`, `eks.services.k8s.aws PodIdentityAssociation`
- Identity ordering (step 13, sync-waves): IAM Role 0 → PodIdentityAssociation 1 → identity ServiceAccount 2, the SA carrying `Prune=false,Delete=false` permanently; ArgoCD's ACK health waits for `ACK.ResourceSynced`. Rationale: EKS injects Pod Identity only at pod creation; SA admission refuses product pods until the SA exists
- `cloud.serviceAccountAnnotations` mirrors `serviceAccount.app.annotations` onto the identity account
- Per-component ServiceAccounts `<release>-<component>` (step 6); the prober's `tls.peers` names components' own accounts, not a shared one
- Least privilege down to the action actually issued; `permissionsBoundary` + tags where the deploy role demands them
- Persistent data: `services.k8s.aws/deletion-policy: retain` on primary-tier Buckets + preserve tags

## Tenancy tiers

| Tier | Cloud resources | Identity |
|------|----------------|----------|
| `primary` | own Cluster/Stream/Bucket/Policy/Role/PIA chain | identity SA rendered by `-infra` after its PIA |
| `test` | NONE rendered (standing namespace resources; shared broker/Postgres) | namespace's standing SA |

- The tier gate is a security boundary, not tidiness: ACK CRs are unappliable by tenants and could mint cloud objects — enforced by chart tests (`tenant_scope_test.go`)

## NetworkPolicies

- Vanilla `NetworkPolicy` (Ingress) + EKS Auto Mode `ApplicationNetworkPolicy` (`networking.k8s.aws/v1alpha1`, FQDN egress `domainNames`) — rendered by `-infra`
- Additive with the per-namespace tenant baseline; install-level flows: CNPG ingress 5432 from install pods + cnpg-system operator, gateway ingress to exposed backends, egress 443 to `*.amazonaws.com` (S3/STS)
- Label contract: `instance: {install}` + `component: {component}` (selector-stable)
- Gate behind `networkPolicies.enabled` (clusters without the EKS CRDs)

## Platform-injected vs project-declared values

The platform passes install-specific values BY NAME (delivery interface; full menu tables in `references/platform-values.md`): database host + roots, secrets by name, broker, bucket, tier, route coordinates, otel, tls. The chart carries only app-generic defaults. Never hardcode install-specific coordinates — "a nonexistent route → an UNPROTECTED app route".
