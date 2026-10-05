# Platform Values: Block Schema and Menu (Truvity policy contract)

What the `platform` block accepts (the schema service-lib reads) and what a platform passes each chart by name. Self-contained; normative sources (need network): [schemas/fragments/platform.json](https://github.com/truvity/policy/blob/master/schemas/fragments/platform.json), [docs/contracts/platform.md §10](https://github.com/truvity/policy/blob/master/docs/contracts/platform.md).

## The `platform` block (per component)

Nothing here is the service's own configuration — that is the `config` block, rendered verbatim. Strict throughout: `additionalProperties: false`.

| Key | Shape | Default / rule |
|---|---|---|
| `image` | `{registry, repository*, tag, digest}` | falls back to top-level `images.<component>`; digest pattern `^(sha256:[0-9a-f]{64})?$` |
| `imagePullPolicy` | `Always\|IfNotPresent\|Never` | `IfNotPresent` |
| `replicas` | int ≥ 0 | 1; a rollout-surviving service runs more than one |
| `strategy` | `{maxUnavailable, maxSurge}` | 0 / 1 — gapless: replacement READY before the incumbent is touched |
| `resources` | object | open passthrough (Kubernetes' own shape) |
| `podSecurity` | `{runAsUser, runAsGroup, fsGroup}` | all 65532; `fsGroup` matters — CSI writes mounts owned by root |
| `serviceAccount` | `{create, name, annotations}` | name defaults `<release>-<component>`; `default` refused; `create: false` where the platform renders the accounts |
| `service` | `{enabled}` | defaults to whether `config.listen` exists |
| `probes` | `{liveness, readiness, startup}` | probe = `{path, periodSeconds, initialDelaySeconds, timeoutSeconds, successThreshold, failureThreshold}`; liveness is the process ONLY — a probe that checks a dependency restarts a healthy process |
| `drain` | `{preStopSeconds}` | 5; grace period DERIVED from `config.drain.seconds` + this |
| `tls` | `{csiDriver, mountPath, mount}` | workload-identity mount; `csiDriver` required once mounted (platform's, no default); `mountPath` `/var/run/identity`; `mount` defaults to `config.tls.mode` permissive/strict |
| `telemetry` | `{serviceName, endpoint, protocol, tracesSampler, sampleRatio, resourceAttributes}` | OTel env vars, never config keys; no endpoint = no export |
| `secretFiles` | `{<name>: {secretName*, key*}}` | projected FILES, mode 0440, under `config.secrets.root`; never env, never in values |
| `env` | `[{name*, value}]` | CLIENT-LIBRARY env only (e.g. a database client's connection vars); never the service's config, never secrets |
| `volumes` / `volumeMounts` | arrays | Kubernetes-shape passthrough for client-library mounts (trust bundle, password file) |
| `config` | `{fileName, mountPath, pathFlag, pathEnv}` | how the file reaches the process: `<component>.yaml`, `/etc/<chart name>`, `-config`; ONE of `pathFlag`/`pathEnv`, never both |
| `configMap` | `{annotations}` | hook-resource support: a pre-install job cannot mount a ConfigMap the release has not created |

## What a platform passes, by name

The full menu, not a checklist: a chart's schema accepts only the keys it reads — accepting a key it does nothing with quietly promises a behaviour it does not have; refusing it fails the render loudly.

### To the infrastructure chart

| Value | Answers |
|---|---|
| `postgres.instances`, `.storage` | how much database |
| `postgres.labels`, `.annotations` | what reads metadata here (e.g. a controller's prune policy) |
| `postgres.platformOwned` | platform renders the database; chart renders the runtime role |
| `postgres.scheduling` | which pool a database may run on |
| `postgres.backup.objectStoreName`, `.serverName` | which archive, and who this is inside it |
| `postgres.serverTLS.secretName`, `.caSecretName` | what the server presents — two Secrets the chart must NOT create |
| `postgres.runtimePasswordSecret` | where the service's password is |
| `events.storage`, `.replicas`, `.maxAge` | how durable the stream is |
| `events.account` OR `events.url` | who this is to the broker, or which broker — alternatives, not a pair |

### To the application chart

| Value | Answers |
|---|---|
| `database.host` | where the database the other chart made is |
| `database.owner.passwordSecret`, `database.app.passwordSecret` | the two credentials (owner = DDL/migration, app = DDL-incapable runtime) |
| `database.tls.rootCA.configMapName`, `.key`, `database.clusterDomain` | the root CA ConfigMap (required — the client always verifies); `.tls.mode` accepted only as `verify-full` |
| `events.url` | which broker |
| `archive.bucket.*` | which store, as an endpoint |
| `serviceAccount.app.name`, `.annotations` | who the workload is to the cloud |
| `route.enabled`, `.hostname`, `.parentRef` (+ `.surfaces[]`) | exposure on the platform Gateway |
| `identityProviders.<name>.issuer_url`, `.client_id_list` | issuers a chart that verifies bearer tokens ITSELF trusts (only if it does) |
| `access.issuer`, `.audience`, `.signOutUrl` | gateway-auth coordinates (only if behind such a proxy) |

## Composed `values.schema.src.json` — complete example

Two blocks and the image map; the chart adds nothing between them:

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

Generate `values.schema.json` by bundling every `$ref` under `$defs` (the policy repo's recipe is `just chart-schemas`; any JSON-Schema bundler resolving the `$ids` works offline). `.helmignore` the source. Helm validates offline against the very schema the binary loads at start-up.
