# Glossary

The words these documents use with a narrower meaning than everyday
English. Where a word has two meanings here, both are listed, with the
place each is used.

**Estate.** Everything one organisation runs and the repositories that
describe it: its clusters, its accounts, its identity plane, its private
configuration repository. The contracts are written for *an* estate and
never name one; "a consuming estate" is the estate that adopts a release.
A particular — a hostname, an account, a cluster's name — belongs to an
estate and arrives as an input ([repository.md §7](contracts/repository.md)).

**Platform.** Whatever runs a service and owes it what
[platform.md](contracts/platform.md) lists: a namespace, an identity, the
secrets it names, a route's parent, the stores and streams it finds. The
other side of the seam from the service. The platform is a role, not a
product: a second platform, run by someone else, should satisfy the same
chart without a patch.

**Ring.** A layer of the platform, ordered by scope: what exists per
cluster, per namespace, per install. [platform.md §11](contracts/platform.md)
places every resource of the example in one. An estate may number its
rings; the contracts care only about the **scope**, never the number.

**Tier.** Two uses, both "which kind of the same thing":

- *An install's tier* — the `tier` value of the example's infrastructure
  chart. `test` provisions nothing and runs as the namespace's standing
  identity; `primary` provisions the store, identity and policies the
  install owns ([platform.md §11](contracts/platform.md)).
- *An integration tier* — where the shared CI runs a repository's suites:
  **kind** (a disposable cluster on a hosted runner, for a public
  repository) or **shared** (the estate's own development cluster, for a
  private one). Chosen by the repository's visibility, never by the caller
  ([decision 0005](decisions/0005-kind-is-the-gate.md)).

**Lane.** One build → install → test sequence the integration workflow
runs on a tier. This repository has one, `example`, in
[`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml); "the kind
lane" and "the cluster lane" are that lane on the kind tier.

**Component.** A public repository that ships mechanism for an estate to
consume — charts, images, Go libraries, Pulumi components, CLIs or actions —
and the thing it ships. Held to [component.md](contracts/component.md).

**Service.** A long-running process with a configuration file, probes, a
log stream and a drain on SIGTERM, held to
[service.md](contracts/service.md). A job (such as the example's migration)
is held to the same boundary where it applies. A component may ship a
service; most ship mechanism around someone else's.

**Preset.** A named bundle of a service's adapter or deployment choices that
expands at load time into the keys it stands for, and names only choices
that are implemented ([config.md §9](contracts/config.md)). Not a **profile**:
a bundle that selects a compliance posture (retention, redaction, record
settings) is a profile, chosen by who audits the deployment.

**Stabilizing / stable.** A product's declared state, in its README. A
stabilizing product may ship a breaking change in a minor; a stable one
ships it only in a major ([release.md §2](contracts/release.md)).

**Consumer.** A repository that installs, imports or calls a released
artifact at a pinned version, and whose pin bump carries the evidence that
adopting it changes only what the release announced
([release.md §6](contracts/release.md)). A component's README names its
consumers and the surface each uses ([component.md C8](contracts/component.md)).

**kernel, devel, stage, prod.** The environment names an estate's clusters
commonly carry, and the order a release is promoted through. The contracts
never branch on them — a chart that reads an environment's name has become a
deployment — but the words turn up in consumers' discussions, so:

- **kernel** — the platform's own cluster: the shared services every other
  environment leans on, such as the identity issuer and the secret store.
- **devel** — where changes are tried together: the shared development
  cluster a private repository's integration tier runs against, and where
  engineers install their own copies.
- **stage** — the last environment before production, reached by the same
  promotion and shaped like it.
- **prod** — production.

A contract that needs to say "production" says "the deployment"; one that
needs "a test install" says the install's **tier**.
