# The delivery interface

Version: 1.0 · Effective: 2026-10-02 · Changes: see [CHANGELOG](../../CHANGELOG.md)

**Normative.** [platform.md §10](platform.md#10-what-a-platform-passes-by-name)
is the menu of what a platform may hand a chart. This is the order in which
the menu grew, and the one number a chart gives a platform so that the
platform hands it only what it can take.

## 1. The problem

A chart's schema refuses a key it does not know. That is the point of a
strict schema, and it is also why a platform cannot hand every pin every key
it has: a pin from before a key existed fails the render, the failure is a
comparison error the deploying tool caches, and whatever is waiting for that
deployment to become healthy waits for something that cannot happen.

Platforms have answered this with a version comparison per key ("the
release that took `installName` is 0.8.0"). Twelve keys is twelve version
strings per product, each one a copy of a fact the chart already knows.

## 2. The interface number

A product's charts declare the **interface** they read as one whole number,
in `Chart.yaml`:

```yaml
annotations:
  delivery.truvity.io/interface: "10"
```

The rules:

1. **The number only grows, and a step only adds.** Interface *N* accepts
   every key of interfaces 1 to *N*, and refuses the keys that step *N+1*
   adds. A step never renames or removes a key an earlier step added; a chart
   that must stop accepting one is a breaking change and gets a major version.
2. **A chart declares the highest step it reads**, and its schema is held to
   it: accepts every key of steps 1 to *N*, refuses step *N+1*'s.
3. **One number per product.** The infrastructure, application and end-to-end
   charts of a product are released under one tag and pinned together, so all
   three carry the same number: the highest step any of them reads. A chart
   that reads none of a step's keys simply never sees them.
4. **A platform reads the number as data.** It writes it next to the product's
   pin, renders the keys of every step up to it, and never compares a version
   to decide whether a key is allowed. A platform that is handed a number it
   does not know (higher than the highest step it implements) refuses to
   render rather than render short.
5. **No annotation means the pin predates it.** The annotation first ships in
   the release named in the CHANGELOG. For an earlier release the number is
   the one in the table in section 4, for that product; a platform that
   checks its own data against the charts uses the table for those pins and
   the annotation for the rest.

A step is a **capability the platform may pass**, not a requirement that it
does: a platform without a workload identity passes no `tls` block whatever
the interface says. The interface says what a pin can take; the platform's
own facts say what it has.

## 3. The steps

The key names are [platform.md §10](platform.md#10-what-a-platform-passes-by-name)'s.
"Chart" says which of the product's charts reads the keys.

| Step | Adds | Chart | Meaning |
|---|---|---|---|
| 1 | the explicit interface | all | Every address by name: `tier`, `cloud.*` (infra), `postgres.*` (infra), `events.*`, `database.host`, `.owner`, `.app` (application), `archive.bucket.*`, `serviceAccount.app.name`, `route.enabled`, `.hostname`, `.parentRef`, `.surfaces`, `otel.*`, `tls.mode`, `.trustDomain`, `.grantRequest`, `.peers`, `identityProviders`, `access`, `product`. The floor: a chart that reads less than this is not on the interface. |
| 2 | `installName` | infra, application | One shared name from which the pair derives its event stream, subjects and consumers, so two releases of different names agree on the stream. |
| 3 | the end-to-end chart | end-to-end | The third chart exists and takes `appRelease`, `mode`, `database.*`, `events.*`, `archive.*`, `serviceAccount`, `job.annotations`, `otel`, `prober.*`, and `traces.url`, `.tokenExchange.*`, `.caConfigMap`. |
| 4 | `tls` on the end-to-end chart | end-to-end | The prober can present a workload identity (`tls.mode`, `.trustDomain`, `.grantRequest`, `.peers`) and `prober.serviceAccount.name`. |
| 5 | `tls.components.<component>.mode` | application, end-to-end | Transport identity per component (`off`, `permissive`, `strict`), so a component can run `strict` while the rest stay `permissive`. |
| 6 | a ServiceAccount per component | application, end-to-end | Each component runs as `<release>-<component>` (C14), and the prober's `tls.peers` names the answering components' own accounts, not the release's shared one. |
| 7 | `events.tls` | application | The publisher authenticates to the broker with its workload identity: `events.tls.enabled`, `.serverName`, `.caConfigMap`. |
| 8 | `database.tls` | application | The client verifies the database server's certificate: `database.tls.mode: verify-full` and `.rootCA.configMapName`, `.key`. From the release that makes `verify-full` the only mode (the first release of the platform PostgreSQL client), the block is required, and a platform whose database serves no certificate cannot move the product to it. |
| 9 | `postgres.platformOwned` | infra | The platform renders the database; the chart renders the runtime role only. |
| 10 | `web.faro`, `route.faro` | application | Browser telemetry: the page's runtime configuration, and the public rule a browser reports through. |
| 11 | `route.faro.requestBufferLimit` | application | The gateway's request body limit on the telemetry rule, rendered by the chart as a policy that targets that rule alone, so a platform no longer renders it beside the chart. |
| 12 | `alerts` | application | The product's own alert rules as a `VMRule` (`alerts.enabled`, off by default): `alerts.remote.enabled`, `.clusterName`, `.namespace` render the rule object alone, for an install evaluated on another cluster (nothing else renders and no other value is required), and `alerts.ruleLabels`, `.alertLabels`, `.interval` and one block of thresholds per rule. A platform no longer writes the product's rules beside the chart. |
| 13 | `tier` on the application chart | application, infra | The identity ServiceAccount lives in the `-infra` chart: on the primary tier the `-infra` chart renders the workload's AWS identity ServiceAccount after its PodIdentityAssociation (sync-waves: IAM Role 0, PodIdentityAssociation 1, ServiceAccount 2; ArgoCD's ACK health waits for ACK.ResourceSynced), carrying `argocd.argoproj.io/sync-options: Prune=false,Delete=false` permanently; the product chart does not render that ServiceAccount on the primary tier and only references it. Why: EKS injects Pod Identity credentials only at pod creation; Kubernetes' ServiceAccount admission refuses product pods until the ServiceAccount exists, so no pod starts before its identity. The one key this adds: from interface 13 the platform passes `tier` to the application chart as well, the same value (`test` or `primary`) it passes the `-infra` chart (the application chart's default is `test`); the `-infra` chart's optional `cloud.serviceAccountAnnotations` mirrors `serviceAccount.app.annotations` onto the account it renders. Nothing else is passed. |

## 4. The first release of each step

The numbers a platform writes for a pin that predates the annotation. A
product's number at a release is the highest step whose release is at or
below it.

| Step | url-shortener | dms |
|---|---|---|
| 1 | 0.4.0 | 0.40.0 |
| 2 | 0.8.0 | 0.40.0 |
| 3 | 1.28.1 (the chart is first published at 1.26.1; 1.28.1 is the first release a platform can render it from: `traces.*` arrived in 1.28.0, `job.annotations` in 1.28.1) | not yet |
| 4 | 1.29.1 | not yet |
| 5 | 1.30.0 | not yet |
| 6 | 1.31.0 | not yet |
| 7 | 1.32.0 | not yet |
| 8 | 1.33.0 | 0.46.0 (`database.tls`, verify-full; wins over its legacy `pg.tls`) |
| 9 | 1.34.0 | 0.46.0 (`postgres.platformOwned: true` with the embedded subchart off by `pgSubchart.enabled: false`; unset, dms-infra renders its own Cluster) |
| 10 | 1.35.0 | not yet |
| 11 | 1.39.0 | not yet |
| 12 | 1.40.0 | not yet |
| 13 | 1.43.0 | not yet |

"Not yet" is a step the product's charts do not read: its number stops below
it, and a platform renders none of that step's keys for it. The numbers are
cumulative, so a product that does one of a later step's things under its own
keys (a database client that verifies the server certificate, a ServiceAccount
per component) still cannot declare that step until it has read every step
before it. A product's first release of a step is written as `next` in the pull
request that adds the reading, and replaced by the release's version when it is
cut.

A product's current number is in its charts' `Chart.yaml`; the examples
under `examples/` declare it and a test holds them to this document.

## 5. Adding a step

A release that adds a key a platform may pass:

1. adds the row to section 3 with the next number, and the release to
   section 4, in the same pull request;
2. moves the annotation of the product's charts to that number;
3. says so in the CHANGELOG under `### Contracts`, because every platform
   that implements the interface must now know a step it did not.

A platform that implements this interface needs a release that knows the new
step before a product can declare it; that ordering is the reason a platform
refuses a number it does not know instead of rendering what it can.

**How to check.** The three charts of a product carry the annotation, as one
whole number, and it names a step in section 3 (`charts/chart_test.go`).
