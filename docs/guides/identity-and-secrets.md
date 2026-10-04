# Identity and secrets

**The rule.** A chart names the account its workloads run as and puts
annotations on it. A service reads a Kubernetes Secret and never talks to a
secret store. [platform.md §2 and §3](../contracts/platform.md).

**Why.** Platforms bind accounts to outside rights by different mechanisms,
and more than one is normal within a single estate — which one is in use is a
property of the *cluster*, not of the workload. A chart that hard-codes one
can only be installed on half the clusters it should run on, and the failure
arrives as a permissions error a long way from the cause. A service that
reads a store directly cannot run where that store is absent and cannot be
tested without credentials for it.

## Where to look

| Concern | Where |
|---|---|
| the accounts | [`charts/url-shortener/templates/serviceaccount.yaml`](../../examples/url-shortener/charts/url-shortener/templates/serviceaccount.yaml) |
| naming them per workload | the `serviceAccountName` line in each workload template, resolved by `url-shortener.componentServiceAccountName` |
| a secret reaching a process | `url-shortener.dbPasswordVolume` in `_helpers.tpl` (a mounted file the database client re-reads), and `config.Secret` in [`config/`](../../config/) for a secret read from a variable |

In code, nothing names a mechanism: the cloud SDK's ambient credential chain
finds whatever the platform bound to the account.

## Two accounts, not one

The example runs its migration as a different account from its services, for
the same reason it gives them different database credentials: the migration
creates tables and grants rights, the services read and write rows. One
account for both jobs puts the migration's rights on the request path, and a
test asserts they differ.

## Per-component identity

Every component runs as its own account, always: `redirect`, `urls`, `web`,
`stat` and `log` (default `<release>-<component>`, renamable under
`serviceAccount.components.<component>.name`), plus the migration's own. With
the transport on, each has its own SPIFFE identity,
`spiffe://<trustDomain>/ns/<namespace>/sa/<account>`, which is namespace plus
account: a shared account would make two components indistinguishable to an
allow-list. There is no shared mode, and the render refuses two components
(or one and the migration) resolving to the same name. This is rule
[C14](../contracts/component.md#c14-each-component-runs-as-its-own-serviceaccount).

**The cloud binding does not move.** `log` is the one component that needs
rights outside the cluster (the archive bucket). Its account defaults to
`serviceAccount.app.name` when that is set, so the account a platform already
bound a cloud role to is still log's, and `serviceAccount.app.annotations`
go on that account alone. Nothing else runs as it: the other four components
no longer run as the app account. If a platform bound anything else by that
name (for example broker permissions for `redirect` and `stat`, which
connect with a projected token of their own account), bind their new names.
Set `serviceAccount.create: false` where the platform creates the accounts.

**The identity account is created after its cloud binding.** An account that
a cloud identity is bound to has an ordering problem: EKS injects the identity's
credentials only when a pod is created, and Kubernetes admits a pod only when
its account exists, so an account created BEFORE the binding lets the pod start
with no rights. So on the primary tier the infrastructure chart, which creates
the binding, also renders the account (`cloud.serviceAccount`), in sync-wave 2
after the role (wave 0) and the association (wave 1), and the application chart
(`tier: primary`, which the platform passes to both) does not render it and only
names it. There are no switches; `tier: test` renders it from the application
chart as before. Waves order the objects of ONE Application; the guarantee
across two is Kubernetes' own refusal of a pod whose account is missing.
The account carries `argocd.argoproj.io/sync-options: Prune=false,Delete=false`
permanently: a recreated account is a new object, and the token a running pod
holds names the old one. It has the name and labels the application chart
rendered (its `app.kubernetes.io/instance` label is `installName`, which must be
the application release's name) and `cloud.serviceAccountAnnotations`, which
should mirror `serviceAccount.app.annotations`.

**The chart's own allow-lists follow its call graph.**

| Component | Admits (in the release) | Accepts an answer from |
|---|---|---|
| `urls` | `web`, `stat` | (serves only) |
| `redirect` | nobody | (serves only) |
| `web` | (calls only) | `urls` |
| `stat` | (calls only) | `urls` |
| `log` | (no tls block) | (none) |

Callers from outside the release stay in `tls.peers.<component>`.

**Grant the e2e chart the right names.** The prober's and the suite Job's
`tls.peers` (who may ANSWER them) must name `<release>-urls` and
`<release>-redirect` (or the names given under `serviceAccount.components`).
Their own accounts still go in the application chart's `tls.peers.urls` and
`tls.peers.redirect`.

## How a secret arrives

The chart takes the **name** of a Secret and the key inside it, and renders
an environment variable that reads from it. The configuration file names the
*variable*. Nothing renders a value.

The objects that *fill* that Secret — pulling from a store, or pushing into
one — belong to the platform or to the infrastructure release. An application
chart that renders one has taken on the platform's job and will disagree with
it.

## Traps

**A chart that generates a password is worse than one that takes it.** It
produces a different value on every render, so the rendered output is not
reproducible and a second install is a different service.

**An ordinary resource a pre-install task needs is the hang you will spend an
afternoon on.** Whatever runs before the release can only reference things
that also run before it. The account is the third thing that has bitten this
way here; the symptom is a pod that is never created and an install that sits
at "in progress" until it times out.

**Static credentials next to a bound account are a contradiction.** If the
pod already has an identity the platform bound, fetching a long-lived key
pair into it removes exactly the property that identity provides.

**Leave the annotations open even if you only run one kind of cluster
today.** An empty map costs nothing; a hard-coded annotation costs a
migration.
