# The Delivery Interface (Truvity policy contract)

One number per product so a platform hands a chart only the keys it can take. Condensed and self-contained; normative source (needs network): [docs/contracts/delivery-interface.md](https://github.com/truvity/policy/blob/master/docs/contracts/delivery-interface.md).

## The number

```yaml
annotations:
  delivery.truvity.io/interface: "13"
```

Rules:

1. **The number only grows; a step only adds keys.** Interface *N* accepts every key of steps 1..N and refuses step *N+1*'s. A rename or removal is a breaking change and gets a major version.
2. **A chart declares the highest step it reads**, and its schema is held to it.
3. **One number per product**: infra, application and e2e charts release under one tag and carry the same number.
4. **The platform reads the number as data**: renders every key of steps ≤ N, never compares per-key versions. A number higher than the platform implements → refuses to render rather than render short.
5. **No annotation means the pin predates it** (the policy repo keeps a first-release table per product).

A step is a capability the platform MAY pass, not a requirement that it does — a platform without workload identity passes no `tls` block whatever the number says.

## The steps

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

## Adding a step

1. Add the step row + the first-release version in the same pull request (policy repo).
2. Move the charts' annotation to the new number.
3. CHANGELOG under `### Contracts` — every implementing platform needs a release that knows the step first; that ordering is why a platform refuses an unknown number instead of rendering what it can.
