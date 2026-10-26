# Platform Values: Block Schema and Menu (Truvity policy contract)

What the `platform` block accepts (the schema service-lib reads) and what a platform passes each chart by name. Self-contained; normative sources (need network): [schemas/fragments/platform.json](https://github.com/truvity/policy/blob/master/schemas/fragments/platform.json), [docs/contracts/platform.md §10](https://github.com/truvity/policy/blob/master/docs/contracts/platform.md).

Example-specific delivery keys and the echo fixture are in
[url-shortener][product-charts], which consumes policy/service-lib v1.54.2; the
product's schema bundle/tests use that adopted version, so do not assume a
latest producer checkout is its pin.

[product-charts]: https://github.com/truvity/url-shortener/tree/76770607aa08baa58331668c228dacd62a13a89a/charts

## The `platform` block (per component)

Nothing here is the service's own configuration — that is the `config` block, rendered verbatim. Strict throughout: `additionalProperties: false`.

| Key | Shape | Default / rule |
|---|---|---|
| `image` | `{registry, repository*, tag, digest}` | falls back to top-level `images.<component>`; digest pattern `^(sha256:[0-9a-f]{64})?$` |
| `imagePullPolicy` | `Always\|IfNotPresent\|Never` | `IfNotPresent` |
| `replicas` | int ≥ 0 | library default 1, with explicit zero preserved; the product chart maps `availability` to this field and honors explicit replica overrides |
| `strategy` | `{maxUnavailable, maxSurge}` | 0 / 1 — gapless: replacement READY before the incumbent is touched |
| `resources` | object | open passthrough (Kubernetes' own shape) |
| `podSecurity` | `{runAsUser, runAsGroup, fsGroup}` | all 65532; `fsGroup` matters — CSI writes mounts owned by root |
| `serviceAccount` | `{create, name, annotations}` | name defaults `<release>-<component>`; `default` refused; `create: false` where the platform renders the accounts |
| `service` | `{enabled}` | defaults to whether `config.listen` exists |
| `probes` | `{liveness, readiness, startup}` | probe = `{path, periodSeconds, initialDelaySeconds, timeoutSeconds, successThreshold, failureThreshold}`; liveness is the process ONLY — a probe that checks a dependency restarts a healthy process |
| `drain` | `{preStopSeconds}` | 5; grace period DERIVED from `config.drain.seconds` + this |
| `tls` | `{csiDriver, mountPath, mount}` | workload-identity mount; `csiDriver` required once mounted (platform's, no default); `mountPath` `/var/run/identity`; `mount` defaults to `config.tls.mode` permissive/strict |
| `telemetry` | `{serviceName, endpoint, protocol, tracesSampler, sampleRatio, resourceAttributes}` | OTel env vars, never config keys; no endpoint = no export |
| `secretFiles` | `{<name>: {secretName*, key*}}` | names/keys only in values; material is projected as read-only FILES, mode 0440, under `config.secrets.root` with `secrets.source: file` |
| `env` | `[{name*, value}]` | CLIENT-LIBRARY env only (e.g. a database client's connection vars); never the service's config, never secrets |
| `volumes` / `volumeMounts` | arrays | Kubernetes-shape passthrough for client-library mounts (trust bundle, password file) |
| `config` | `{fileName, mountPath, pathFlag, pathEnv}` | how the file reaches the process: `<component>.yaml`, `/etc/<chart name>`, `-config`; ONE of `pathFlag`/`pathEnv`, never both |
| `configMap` | `{annotations}` | hook-resource support: a pre-install job cannot mount a ConfigMap the release has not created |

## What a platform passes, by name

The full menu, not a checklist: a chart's schema accepts only the keys it reads — accepting a key it does nothing with quietly promises a behaviour it does not have; refusing it fails the render loudly.

Real per-environment assignments and Applications are authored in GitOps.
Nonsecret fixture values remain in the product repository's tests. A schema
accepting a key is not evidence that the platform supplies it: verify both ends
at the product's supported interface step. Product-specific optional keys below
do not create a new numbered step by themselves.

### To the infrastructure chart

| Value | Answers |
|---|---|
| `postgres.instances`, `.storage` | how much database |
| `postgres.labels`, `.annotations` | what reads metadata here (e.g. a controller's prune policy) |
| `postgres.platformOwned` | platform renders the database; chart renders the runtime role |
| `cnpg-database.clusterName`, `.roles` | unconditional locked dependency, roles empty by default; platform-owned mode requires one role, own-Cluster password generation requires one with `databaseRole: false`, own-Cluster external Secret requires empty roles; parent/subchart names, labels and generation settings must agree |
| `postgres.scheduling` | which pool a database may run on |
| `postgres.backup.objectStoreName`, `.serverName` | which archive, and who this is inside it |
| `postgres.serverTLS.secretName`, `.caSecretName` | what the server presents — two Secrets the chart must NOT create |
| `postgres.runtimePasswordSecret` | name of the runtime credential Secret, populated externally or generated once; step 15 requires a DatabaseRole. The standalone example's Cluster-managed role differs from that wording; see the contract-gap record in `delivery-interface.md` |
| `events.storage`, `.replicas`, `.maxAge` | how durable the stream is |
| `events.account` OR `events.url` | who this is to the broker, or which broker — alternatives, not a pair |

### To the application chart

| Value | Answers |
|---|---|
| `database.host` | where the database the other chart made is |
| `database.app.role`, `database.app.passwordSecret` | the RUNTIME credential — DDL-incapable, the only one long-running workloads ever get (step 15) |
| `database.migration.passwordSecret` | the owner's credential (`.role`/`.passwordKey` defaulted), for the schema migration alone: passed only to a product declaring `postgres.migrations: true`, mounted only in a migration Job — replaces `database.owner.*` (deprecated) |
| `database.tls.rootCA.configMapName`, `.key`, `database.clusterDomain` | the root CA ConfigMap (required — the client always verifies); `.tls.mode` accepted only as `verify-full` |
| `events.url` | which broker |
| `archive.bucket.*` | which store, as an endpoint |
| `serviceAccount.app.name`, `.annotations` | who the workload is to the cloud |
| `route.enabled`, `.hostname`, `.parentRef` (+ `.surfaces[]`) | exposure on the platform Gateway |
| `availability` | `single` or `high` — the CLUSTER's redundancy, not the product's; the chart sizes replicas and its disruption budget from it (step 14) |
| `images.identity-check` | already pinned third-party image for the required admission check; command/image owned by `truvity/k8s`, not built by the product release |
| `identityCheck.expectAccount` | example's optional quoted account string, empty by default; forwarded to the library's `expectAccount` and requires identity-check v0.34.0+ before enablement |
| `observability.dashboards.*` | example-specific optional product dashboards: enablement, `clusterFolderUID`, `datasources.metrics`, `datasources.logs`, optional `instanceSelector`; supplied by the platform, not baked-in deployment coordinates |
| `identityProviders.<name>.issuer_url`, `.client_id_list` | issuers a chart that verifies bearer tokens ITSELF trusts (only if it does) |
| `access.issuer`, `.audience`, `.signOutUrl` | gateway-auth coordinates (only if behind such a proxy) |

## Two source-of-truth traps to avoid

- The **library** does not read product `availability`; `_components.tpl` maps
  it to `platform.replicas`, while the product's PDB template reads the same
  decision. Do not document a library default as if it sizes product redundancy.
- The cnpg-database **subchart** cannot read the parent `postgres` block or
  derive its values after templates run. Since the example's v1.53.0 adoption
  its roles list defaults empty, not to one live install. State the complete
  applicable entry explicitly; `databaseRole: false` is required when the parent
  Cluster already manages the role. Render guards reject disagreement and
  duplicate owners. See `cluster-resources.md`.

The expected-account value is another platform input, not a new numbered
delivery-interface step. A blank value sends no command flag; a nonempty one
must retain its digits as a YAML string. Upgrade the checker image before using
it, or an old binary fails every hook attempt on the unknown flag.

The Kubernetes source for `secretFiles` is `file`, not `ssm` or `openbao`.
Those configuration-source options serve platforms without Secret objects.
Store provisioning and rotation belong to the platform/infra side; secret
material never goes in values, ConfigMaps, chart defaults or workflow YAML.

## Composed `values.schema.src.json` — complete example

Two blocks and the image map; the chart adds nothing between them:

This source schema is the product's `charts/testdata/service-example` chart's: its local
`echo.schema.json` defines the service configuration and requires `listen`
and `greeting`. Use it with the complete echo render values in `chart-patterns.md`,
not a differently named image entry. The committed `values.schema.json`
bundles these references for offline Helm validation.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "service-example chart values",
  "description": "Two blocks and the image map: `platform` is the platform schema, `config` is the service's own schema, and the chart adds nothing between them.",
  "type": "object",
  "additionalProperties": false,
  "required": ["config", "images"],
  "properties": {
    "service-lib": {
      "type": "object",
      "additionalProperties": false,
      "properties": { "global": { "type": "object" } },
      "description": "Not a value: Helm puts a key named for every sub-chart into the values it validates."
    },
    "platform": { "$ref": "https://github.com/truvity/policy/schemas/fragments/platform.json" },
    "config": { "$ref": "echo.schema.json" },
    "images": {
      "type": "object",
      "additionalProperties": false,
      "required": ["echo"],
      "properties": {
        "echo": { "$ref": "https://github.com/truvity/policy/schemas/fragments/platform.json#/properties/image" }
      }
    }
  }
}
```

Generate `values.schema.json` in the product with `just chart-schemas`, using
the shared schemas at its policy pin; bundle refs under `$defs` and ignore the
source when packaging. Policy provides the composer/schema, not this root
recipe anymore. Helm validates offline against the binary's same schema.
