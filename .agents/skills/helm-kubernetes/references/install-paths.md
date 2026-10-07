# Two Install Paths (Truvity policy guide)

What `helm upgrade --install` provides that a GitOps controller's `helm template` + apply does not, and every rule that keeps a chart unable to tell which one it is in. The Applications are the platform's — a separate GitOps control plane renders and syncs; a product repository's CI never applies to a cluster. This is the chart-side half. Condensed and self-contained; normative sources (need network): [testing guide](https://github.com/truvity/policy/blob/master/docs/guides/testing.md), [conformance guide](https://github.com/truvity/policy/blob/master/docs/guides/conformance.md).

## What differs

An install with `helm upgrade --install` holds a Helm release record. That record is what makes `lookup` return something, `.Release.IsUpgrade` and `.Release.IsInstall` tell the truth, and `helm install`/`upgrade` stamp the `meta.helm.sh/release-name` annotation on every object.

A consumer's own platform commonly does something else entirely: a GitOps controller renders the chart with `helm template` and applies the output directly, with **no Helm release record at all**.

| | `helm upgrade --install` | controller (`helm template` + apply) |
|---|---|---|
| Helm release record | held | none |
| `lookup` | returns the live object | always empty |
| `.Release.IsInstall`/`.IsUpgrade`/`.Revision` | truthful | meaningless |
| object identity | `meta.helm.sh/release-name` | the chart's own `app.kubernetes.io/instance` label — the ONLY thing identifying a release's objects |
| hooks | install/upgrade phases | mapped to sync phases, run on EVERY sync |

A chart that happens to pass under `helm upgrade --install` must not rely on what only that path provides — which is what the checks below enforce.

## Every GitOps-facing check, with its mechanism

| Rule | Check |
|---|---|
| a chart names no `lookup`, and nothing depends on `.Release.IsUpgrade`/`.Release.IsInstall`/`.Release.Revision` | static scan of every chart's templates |
| no value is generated once inside a template — `randAlphaNum`/`randAlpha`/`randNumeric`/`randAscii`/`genPrivateKey`/`genCA`/`genSelfSignedCert`/`derivePassword` — it regenerates on every render (every sync) and never converges to one value | static scan of every chart's templates |
| a hook is idempotent and safe to run on every sync — a controller maps `post-install`/`post-upgrade` to one phase and runs it on every sync, not only the first install or a version change | review (an honest gap: the conformance table says so rather than implying a check) |
| a plain (non-hook) Job a chart renders does not set `ttlSecondsAfterFinished` — self-heal on recreates a Job that deleted itself, running whatever it does again on a loop, forever, at a fixed version | chart test: every Job without a `helm.sh/hook` annotation is rejected if it carries `ttlSecondsAfterFinished` |

## A plain Job under both paths

A chart's e2e Job should carry no `helm.sh/hook` annotations at all — a plain Job, applied the same way by both paths:

- **Its name folds in the chart's own version**: a Job's spec is immutable, so re-applying an upgraded chart under the same name would be refused rather than converge. Each release of the chart is therefore a Job nothing before it ever created; the one before it is left for the NEXT version's apply to prune, or for `helm uninstall`.
- **`ttlSecondsAfterFinished` is unset by default** — see the table above.
- **The name alone does not cover a platform changing only the chart's VALUES at a version already deployed** (Kubernetes still refuses that patch as immutable), so a `job.annotations` value — empty by default, rendered on the Job's own metadata only — lets a platform put its own force/replace annotation there for a controller that reads one to decide it may delete and recreate the Job rather than apply in place.

## Hooks under a controller

| Helm | Controller |
|---|---|
| `pre-install`/`pre-upgrade` | PreSync |
| `post-install`/`post-upgrade` | PostSync (ONE phase, every sync) |
| `hook-weight` | `sync-wave` |
| pre/post-delete, pre/post-rollback, `helm test` | not supported at all |

The controller cannot distinguish install from upgrade — every operation is a sync. The migrate hook (`pre-install,pre-upgrade`, weight 0, `hook-delete-policy: before-hook-creation,hook-succeeded`, TTL reaping residue) becomes a PreSync resource; and a chart that owned its database could never migrate it, which is the whole two-chart split.
