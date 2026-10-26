# Two Install Paths (Truvity policy guide)

What `helm upgrade --install` provides that a GitOps controller's `helm template` + apply does not, and the rules that keep a chart portable across both. Persistent Applications and environment values are the platform's; product CI can install ephemeral test releases. Condensed and self-contained; sources: [testing guide](https://github.com/truvity/policy/blob/master/docs/guides/testing.md), [conformance guide](https://github.com/truvity/policy/blob/master/docs/guides/conformance.md), and the example's `charts/conformance_test.go`.

The example/test paths are in [url-shortener][product], not policy. The product consumes a released policy
module and OCI service-lib; policy's current Go gate does not run its chart
render suite.

[product]: https://github.com/truvity/url-shortener/tree/76770607aa08baa58331668c228dacd62a13a89a

## What differs

`helm upgrade --install` can use Helm's Kubernetes connection to query the live
API. **That API access makes `lookup` work, not a stored Helm release record**:
it can find an object on a first install with no previous release. A missing
object returns an empty result; an API/RBAC error can fail template processing.
The release record separately tracks Helm's install/upgrade history and Helm
stamps its release-name annotation on the objects it manages.

A GitOps controller normally renders with offline `helm template` and applies
the output directly. That rendering does not query the Kubernetes API, so
`lookup` returns empty; the controller also creates no Helm release record.
Server-side dry-run can explicitly permit API access even for `helm template`,
so the distinction is rendering mode/connectivity, not the command name alone.
See [Helm's lookup documentation](https://helm.sh/docs/chart_template_guide/functions_and_pipelines/#using-the-lookup-function).

| | `helm upgrade --install` | controller (`helm template` + apply) |
|---|---|---|
| Helm release record | held | none |
| `lookup` | can query live API, without a prior release; missing objects return empty | empty in the ordinary offline render |
| `.Release.IsInstall`/`.IsUpgrade`/`.Revision` | reflect Helm's operation/history | render-time defaults/options, not live deployment history |
| object identity | `meta.helm.sh/release-name` | the chart's own `app.kubernetes.io/instance` label — the ONLY thing identifying a release's objects |
| hooks | install/upgrade phases | mapped to sync phases, run on EVERY sync |

A chart that happens to pass under `helm upgrade --install` must not rely on what only that path provides — which is what the checks below enforce.
The portability prohibition on `lookup` is unchanged by this explanation.

## Every GitOps-facing check, with its mechanism

| Rule | Check |
|---|---|
| a chart names no `lookup`, and nothing depends on `.Release.IsUpgrade`/`.Release.IsInstall`/`.Release.Revision` | static scan of every chart's templates |
| resources are discovered by `app.kubernetes.io/instance`, not Helm's release annotation | render tests require the label; the static scan rejects selection by the annotation |
| no value is generated once inside a template — `randAlphaNum`/`randAlpha`/`randNumeric`/`randAscii`/`genPrivateKey`/`genCA`/`genSelfSignedCert`/`derivePassword` — it regenerates on every render (every sync) and never converges to one value | static scan of every chart's templates |
| a hook is idempotent and safe to run on every sync — a controller maps `post-install`/`post-upgrade` to one phase and runs it on every sync, not only the first install or a version change | review (an honest gap: the conformance table says so rather than implying a check) |
| a plain (non-hook) Job a chart renders does not set `ttlSecondsAfterFinished` — a self-healing controller recreates a Job that deleted itself, running whatever it does again on a loop, forever, at a fixed version | chart test: every Job without a `helm.sh/hook` annotation is rejected if it carries `ttlSecondsAfterFinished` |

## A plain Job under both paths

The product chart renders no suite Job: its current protected workflow runs
the Go suite from a hosted runner against an ephemeral install. A different
harness could create a scoped in-cluster Job, but that is not this snapshot's
workflow. A chart that does render a plain Job needs an explicit replay and
immutable-spec policy that works under both installers:

- **Its name folds in the chart's own version**: a Job's spec is immutable, so re-applying an upgraded chart under the same name would be refused rather than converge. Each release of the chart is therefore a Job nothing before it ever created; the one before it is left for the NEXT version's apply to prune, or for `helm uninstall`.
- **`ttlSecondsAfterFinished` is unset by default** — see the table above.
- **The name alone does not cover a platform changing only the chart's VALUES at a version already deployed** (Kubernetes still refuses that patch as immutable), so a `job.annotations` value — empty by default, rendered on the Job's own metadata only — lets a platform put its own force/replace annotation there for a controller that reads one to decide it may delete and recreate the Job rather than apply in place.

## Hooks under a controller

| Helm | Controller |
|---|---|
| `pre-install`/`pre-upgrade` | PreSync |
| `post-install`/`post-upgrade` | PostSync (ONE phase, every sync) |
| `hook-weight` | `sync-wave` |

The controller cannot distinguish install from upgrade — every operation is a sync. The migrate hook (`pre-install,pre-upgrade`, weight 0, `hook-delete-policy: before-hook-creation,hook-succeeded`, TTL reaping residue) becomes a PreSync resource; and a chart that owned its database could never migrate it, which is the whole two-chart split.

For a cloud-identity component, the library's identity check precedes migration
at weight **-20**, runs as the infra-owned account and uses `restartPolicy: Never`.
It proves admission by retrying with fresh pods. Ordinary resource waves do not
close the binding's eventual-consistency gap. See `identity-admission.md`.
Only a live platform test proves injection; render tests prove template wiring.
Policy's advisory Go canary retains released service-lib unless explicitly
overridden, so it cannot be presented as proof of a new library render or a
live install. The cited product's integration is separate from its fork PR gate.
