# The component contract

Version: 1.0 · Effective: 2026-09-29 · Changes: see [CHANGELOG](../../CHANGELOG.md)

**Normative.** What a public repository that ships *mechanism* looks like,
so that a stranger who has read one can install, audit and change the next
without being told anything.

## Purpose

The other contracts here are about a **service**: a process with a
configuration file, probes and a log stream ([service.md](service.md)), the
platform it runs on ([platform.md](platform.md)), and the repository and
release around it ([repository.md](repository.md),
[release.md](release.md)). Most public repositories in the estate are not
services. They ship a chart that installs someone else's server, a Go
module an estate imports, a Pulumi component, a command-line tool, a
GitHub Action, or several of these released together. This contract is what
each of those is held to on top of the repository and release contracts.

It replaces two doctrines that grew beside the shared CI, one in
`truvity/ci-workflows` (`docs/component-contract.md`) and one in
`truvity/ci-plane` (`docs/normalization.md`). They agreed on most things and
contradicted each other on two; this page decides both (see
[Retired forms](#retired-forms)) and is the only copy. A repository links
here rather than restating a rule, because a restated rule is a second
version of it.

Every rule has an ID, **C1** to **C14**, so that a review comment, a
ticket and a failing check can say "C5" instead of paraphrasing. Each says
what must be true, why, and how it is checked. **C1 to C12 are checked
mechanically** by the `policy-conformance` action in `truvity/ci-actions`,
which reads a checkout and reports each rule by its ID; **C13 is review**,
because telling an estate's fact from a neutral default takes a reader, and
**C14 is checked by the chart's own tests** in the repository that ships the
chart.

## Scope

A **component repository** is a public repository in the `truvity`
organisation that ships any of:

- Helm charts;
- container images;
- Go libraries (a module other repositories import);
- Pulumi components;
- command-line tools;
- GitHub Actions or reusable workflows.

A repository that also ships a service is held to both this contract and
[service.md](service.md). A private repository is not in scope; it may
follow these rules, and nothing here requires it to.

The repository and release contracts apply in full. Where this contract
repeats one of their rules, it is to give the rule an ID a machine can
report; the reasoning stays where it is.

---

## C1. Committed chart versions are `0.0.0`

**What.** Every `charts/*/Chart.yaml` commits `version: 0.0.0`, and
`appVersion: 0.0.0` when the repository ships an image. The release
workflow stamps both from the tag.

**Why.** One tag stamps every artifact ([release.md §1](release.md)). A
real version committed in the tree is dead weight that reads as the truth
and is wrong the moment the next tag is cut; a placeholder that never moves
cannot be mistaken for anything. `0.0.0` rather than a pre-release suffix
because a suffix makes `helm` and every semver range treat the chart as a
pre-release, which changes how a consumer's constraint resolves against a
locally packaged copy — and the one value every repository uses is the one
nobody has to look up.

**Conformance.** Every `charts/*/Chart.yaml` parses; `version` is exactly
`0.0.0`; when the repository builds an image, `appVersion` is exactly
`0.0.0`.

## C2. Every chart has a values schema

**What.** Every chart has `values.schema.json`. It is strict:
`additionalProperties: false` at the top level and inside every object the
chart itself defines. It is **open** at every extension point a consumer
legitimately fills — annotations, labels, `resources`, affinity,
tolerations, selectors — which accept any string map or pass through
Kubernetes' own shape.

**Why.** Without a schema a typo in a values file is silently ignored, and
the install runs with the default nobody chose. Strictness is what makes
the typo fail at render. Closing an extension point is the opposite
failure: a chart whose `serviceAccount` block refused unknown keys made
attaching any annotation a fork of the chart.

**Conformance.** `charts/*/values.schema.json` exists for every chart and is
valid JSON Schema. Strictness and openness are proved by C3's fixtures.

**Exception: library charts.** A chart with `type: library` in
`Chart.yaml` renders no manifests of its own — it exports templates an
application chart includes with `{{- include }}` — and takes no values a
consumer sets directly; whatever it reads comes from the including chart's
own, already-schema'd `values.yaml`. It carries no `values.schema.json` and
is skipped by C2. Declare it via `.github/policy-conformance.yaml` (see
[Exemptions](#exemptions)) so the exception is named, not merely absent.

## C3. Every chart has golden renders and a refused fixture

**What.** Every chart has golden renders under `tests/golden/<chart>/` —
at least a minimal case and a case that sets every value — and at least one
negative fixture under `tests/invalid/<chart>/`, one per refusal (a schema
rule or a render-time `fail`), each otherwise valid so it fails for its one
reason. `unknown-key.yaml` is always one of them.

A repository that tests its charts in Go instead (this one does, under
`examples/url-shortener/charts/`) satisfies C3 when those tests compare
committed renders byte for byte and render at least one values file the
chart must refuse, and assert that it is refused.

**Why.** A golden is the only way a reviewer sees what a change does to the
output, and the only way "no rendered output changes" in a patch
([release.md §2](release.md)) is a fact rather than a claim. A refusal with
no fixture is a refusal that quietly stops working: nothing fails when the
schema loosens.

**Conformance.** For every chart: `tests/golden/<chart>/` holds at least
one file and `tests/invalid/<chart>/` holds at least one file; or the
repository's Go chart tests reference a directory of rejected values files
and a golden directory.

## C4. The leak canary is in the gate

**What.** `hack/leak-canary.sh` exists, reads **tracked files only**
(`git ls-files`), and `just check` runs it. The canonical copy is this
repository's [`hack/leak-canary.sh`](../../hack/leak-canary.sh); a
repository vendors it and may narrow or drop a pattern only with a comment
in the script's header saying why the matched text is mechanism.

**Why.** A public repository's history cannot be unpublished
([repository.md §7](repository.md)). The canary is the mechanical floor of
that rule. The traps every copy has met, so the next one does not:

- **Hex that looks like an account ID.** A 12-digit run inside a commit
  SHA matched an unanchored pattern. The pattern is anchored on word
  boundaries; `go.sum` holds public dependency data by definition.
- **Generated state.** A recursive walk descended into a gitignored
  directory whose hashes matched. Tracked files are the right scope.
- **Mechanism that is the matched text.** A path Kubernetes mounts
  credentials at, a module that composes ARNs from caller inputs: the
  repository says so in the header rather than weakening the pattern for
  everyone.
- **Commit messages.** The canary cannot read history. A commit that fixed
  a pattern quoted a real account ID in its message as the example — and
  landed in every repository that vendors the script. Quote a placeholder.

**Conformance.** The file exists and is executable; the `check` recipe in
`Justfile` depends on `leak-canary`, which runs it.

## C5. The CHANGELOG has one heading per tag

**What.** `CHANGELOG.md` exists at the root. Its version headings are
`## vX.Y.Z`, optionally followed by ` — YYYY-MM-DD`: **one per tag**,
newest first. An optional `## Unreleased` sits on top, and there is only
one. A heading exists for the latest tag. Bullets are prose written for a
consumer ([release.md §3](release.md)); a breaking one starts with
**Breaking:** and names the step to take. A tag whose only change was a
dependency bump still has its heading, and says so.

**Why.** A consumer reading a pin bump needs to find the version they are
moving to, and every other version between the two. A heading that covers
several versions, or several `Unreleased` sections that each shipped in a
different tag, make that search a guess — and a missing heading reads as
"nothing changed" whether or not that is true. One heading per tag is the
only rule a machine can check.

**Conformance.** The file exists; every `## ` heading that is not
`## Unreleased` matches `^## v\d+\.\d+\.\d+( — \d{4}-\d{2}-\d{2})?$`;
there is at most one `## Unreleased` and it is first; the headings are in
descending version order; the latest `v*` tag has a heading.

## C6. The toolchain names a version for every tool

**What.** `devbox.json` pins every package to a version. No `latest`, no
bare name.

**Why.** "Latest" means two checkouts of the same commit build with
different tools, which turns a reproducible build into a coincidence and a
failure on one machine into an argument ([repository.md §3](repository.md)).
A version that should move is moved by the bot, in a pull request, where
the move is visible and revertable.

**Conformance.** Every entry of `packages` in `devbox.json` names a version
(`name@X…` in the list form, a non-empty version other than `latest` in the
map form).

## C7. Dependency bumps come from the shared configuration

**What.** `renovate.json` is

```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["github>truvity/ci-workflows"]
}
```

plus overrides that each carry a `description` saying why this repository
differs.

**Why.** The shared preset is where the estate's lessons about bumps live
— which monorepos move as one, that a 0.x minor is breaking, that `go.sum`
needs a tidy. A repository that writes its own configuration misses the
next lesson; an override with no reason is one nobody dares remove
([repository.md §6](repository.md)).

**Conformance.** The file exists; `extends` contains
`github>truvity/ci-workflows`; every entry of `packageRules` carries a
`description`.

## C8. The README answers a newcomer's questions, in order

**What.** `README.md` opens with the title, one sentence saying what this
is, and the table of what ships and where it is published. Then these
`## ` headings, in this order (others may sit between them):

1. `Who it is for` — the platform it assumes, and what it deliberately
   does not install.
2. `The model` — the two or three nouns a reader needs and how they relate.
3. `Install and a worked example` — the install line (C12) and a values
   file or snippet with **neutral values** (`example.com`,
   `eu-example-1`, the cloud's own documented placeholder account) that
   renders or compiles as written.
4. `Consumers` — who uses this and through which surface (chart, Go
   module, action, CLI). The one place a consuming repository is named;
   never a version, an environment or a cluster.
5. `Neighbours` — the repositories a reader must know about, one line
   each, with the boundary between them.
6. `Documentation` — links into `docs/`.
7. `The rule that makes this repository public` — mechanism only, and that
   `hack/leak-canary.sh` enforces it.
8. `Status` — what is true today. A "pending", "not yet" or "next" is
   checked against the releases and the tree, and deleted or made true.
9. `Development` — `devbox shell`, `just check`, how to regenerate
   goldens.
10. `Releasing` — the release mode in a sentence or two: whether
    auto-release is armed, and that minors and majors are manual.
11. `Licence`.

`docs/` holds the reference: `adoption.md` (install order, the zero-diff
gate, every breaking upgrade), `safety.md` (each refusal and the failure
that earned it), `reference.md` (every value, flag, input and output), and
whatever the repository genuinely needs beyond that. A repository's design
rules are this contract; a `docs/doctrine.md` links here rather than
restating it.

**Why.** A reader deciding whether this is theirs to use asks the same
questions in the same order every time: what is it, is it for me, how do I
see it work, who already relies on it, what else must I know, where is the
reference, can I trust it, how do I change it. A README in that order
answers them without a table of contents, and a reader who has read one
knows where to look in the next.

`Consumers` exists because a component's blast radius is invisible from
inside it. A change that renames a value is a major for every consumer
listed and a patch for none, and the author cannot tell which without the
list. Naming the consuming repository and the surface — not the version
it pins, which lives with the consumer's own pin — is the smallest thing
that answers the question.

**Conformance.** The file exists and contains each of the eleven headings
as a `## ` line, in this order.

## C9. The licence is MIT, at the root

**What.** `LICENSE` exists at the root and is the MIT licence.

**Why.** A licence claim in a README or a package manifest is not a grant;
the file is. A repository without one is, legally, not open for anyone to
use, whatever it says about itself.

**Conformance.** `LICENSE` exists and its text is the MIT licence.

**Exception: a fork of a non-MIT upstream.** A repository that is a fork
of, or vendors the source of, an upstream project under a licence other
than MIT keeps that licence — a derivative work cannot unilaterally
relicense itself, whatever this contract would otherwise prefer. Declare
it via `.github/policy-conformance.yaml` (see [Exemptions](#exemptions));
the reason names the upstream project and its licence.

## C10. Vulnerability scanning is its own workflow, never the gate

**What.** A repository with a `go.mod` has
`.github/workflows/security.yaml`, which runs the `vuln` recipe through the
shared `check.yaml` workflow in `truvity/ci-workflows`, on push, on pull
requests and on a daily schedule. `vuln` is **not** one of the recipes
`ci.yaml` requires, and `just check` does not depend on it.

**Why.** A new advisory is news about the world, not about the change
under review. In the gate, a standard-library advisory with no released fix
turns every pull request red on a finding nobody can act on — which is how
a repository learns to ignore its gate. On its own schedule the same finding
is reported daily, visibly, and blocks nothing it should not.

**Conformance.** When `go.mod` exists: the workflow file exists and runs the
`vuln` recipe; the recipe list in `.github/workflows/ci.yaml` does not name
`vuln`; the `check` recipe's dependencies do not include `vuln`.

## C11. Image names never repeat the repository

**What.** Images are published as `ghcr.io/truvity/<repo>/<component>`,
where `<component>` is a role (`server`, `runner`, `operator`, `broker`)
or the component's own name — never the repository's name again.
`gemaal/server` is right; `ci-cache/ci-cache` is wrong. Charts are flat,
`ghcr.io/truvity/charts/<chart>`, tagged with the bare version.

**Why.** The repository segment already says whose image it is. A repeat
carries no information, and it breaks the moment a second image joins the
first: one of them is then named after the repository and the other after
what it does, and a reader cannot tell which is which. With `ko`, the name
comes from the command's directory — its `repositories:` key is inert when
`base_import_paths` is set — so the directory is named for the role.

A repository that ships several images under one registry prefix may have
one component legitimately named after the repository itself — a
repository shipping a writer, a query service and a server under one
prefix is not wrong for one of them matching the repository's own name.
The rule is about the degenerate case: a *sole* image whose full path is
`<repo>/<repo>`, with nothing published beside it.

**Conformance.** The image repositories named by `.goreleaser.yaml`, `.ko.yaml`
and the release workflow's `image-repo` input do not end in
`<repo>/<repo>`, unless that image has a sibling published under the same
prefix.

## C12. Install instructions pin a version

**What.** Every install line the repository publishes — README, `docs/`, a
chart's own README — names a version: `go install …@vX.Y.Z`,
`helm install … --version X.Y.Z`, `uses: …@<sha> # vX.Y.Z`. Never
`@latest`, never an unversioned chart.

**Why.** An instruction that installs "latest" installs something
different every week, so the worked example beside it stops matching what
it installs, and two people following the same page get two versions. A
pinned line is also what makes the next consumer's upgrade a pin bump they
can review.

**Conformance.** No tracked Markdown file contains `@latest` in an install
command, and no `helm install`/`helm upgrade` line with an `oci://` chart
omits `--version`.

## C13. Estate facts are inputs, never defaults

**What.** Nothing the consuming estate owns has a default in `values.yaml`,
a module's options, or a CLI's flags: no region, no account, no namespace
other than the chart's own release namespace, no `nodeSelector`, no
toleration, no internal hostname, no registry host. The keys exist; their
defaults are empty or absent; the schema or the documentation says where
one is required.

**Why.** A default that names one estate's fact installs correctly for that
estate and silently wrongly for every other: a chart defaulting a region
creates the bucket in the wrong one, and nobody notices until the data is
somewhere it should not be. It is also a leak — the value is the estate's
([platform.md §1](platform.md), [§9](platform.md)). An empty default fails
loudly where the value was needed, which is the failure a stranger can
diagnose.

**Conformance.** Review. A reviewer reads every `values.yaml` and every
flag default for a value only one estate would choose. The leak canary
(C4) catches the mechanical half: account IDs, registry hosts, internal
domains.

---

## C14. Each component runs as its own ServiceAccount

**What.** A chart that deploys more than one workload runs every component
(and any migration or other Job) as its own Kubernetes ServiceAccount. No two
components share one. A component that needs rights outside the cluster
keeps its own account for them; the account is never shared to reach them.
Where a chart wires one component to another over mutually authenticated
transport, it grants each caller by ITS account, never a shared one.

**Why.** A workload identity is a namespace plus a ServiceAccount (a SPIFFE
ID is `spiffe://<trust domain>/ns/<namespace>/sa/<account>`). A shared
account gives every component holding it the same identity, so an allow-list
cannot admit one and refuse another: mTLS authorisation degrades to "anyone
in this release". Distinct accounts are also what keep a migration's rights
off the request path.

**Conformance.** The chart's own render test: no two Deployments or Jobs
render the same `serviceAccountName`, in the default render and in a fully
set one, and the render itself refuses a values file that makes two
components collide (`examples/url-shortener`:
`TestNoTwoWorkloadsShareAServiceAccount`). The migration hook is not
exempt. Review for a chart in another repository.

---

## Exemptions

A rule can be wrong for a repository's *kind* without being wrong in
general — a library chart has no values of its own to schema (C2); a fork
of a non-MIT upstream cannot relicense itself (C9). When that happens, the
rule is amended here, in its own pull request, with the reason (see "When
the contract is wrong" in this repository's `CLAUDE.md`), and the
repository declares the exception mechanically so it stays visible rather
than merely absent.

A repository declares an exception in `.github/policy-conformance.yaml`:

```yaml
exempt:
  C2:
    reason: library chart takes no values
    charts: [example-routes]
  C9:
    reason: fork of an Apache-2.0 upstream; cannot relicense
```

`reason` is required and reviewed like any other change. `charts:`, where
the rule is chart-scoped (C1, C2, C3), names which charts the exception
covers; a rule with no `charts:` line is exempted for the whole
repository. This is not the `policy-conformance` action's `skip:` input —
that silences a whole rule for one CI run and demands a reason on every
invocation, for a rule that genuinely cannot be checked here yet (C13,
always) or a temporary gap being tracked elsewhere. An exemption is
committed, permanent until the exception is removed, and answers to a
named rule and reason rather than a blanket skip.

An exempted rule is not silently `PASS`: `policy-conformance` reports it
as `EXEMPT` (C9) or narrows exactly which sub-check the exemption
suppresses (C1, C2 by chart; C5, the missing-heading-for-the-latest-tag
check only — format, order and duplicate headings still run), so the job
summary shows a reviewed exception rather than indistinguishable
conformance.

**A repository is not free to invent its own exception.** An exemption
file naming a rule this section does not list, or a reason that does not
match the kind of case above, is itself a gap: fix the rule text first,
here, then adopt it.

---

## What the rules assume

These are not numbered, because a checkout cannot show them; they are how a
component is built and released, and they are held by the shared workflows
rather than by each repository.

- **One tag stamps every artifact**, and a consumer pins one version per
  repository ([release.md §1](release.md)). A Go module at v2 or later
  carries `/vN` in its path before the tag is pushed.
- **The release is the shared workflow.** `release.yaml` is a thin caller of
  `truvity/ci-workflows`' `release-public.yaml`; a hand-rolled release
  needs a written reason. Images are built by goreleaser (with `ko`,
  distroless, unless the image *is* the product); charts are packaged and
  pushed by `helmctl` from `truvity/ocictl`, deterministically, so the same
  content is the same registry digest; a chart pins its own images by digest
  and the release refuses to package one that does not
  ([release.md §5](release.md)).
- **Who cuts a tag.** Automation cuts patches only — immediately for a
  security fix, weekly when merged bumps have moved the default branch —
  and only once `vars.AUTO_RELEASE` arms it. Every first release, minor and
  major is a person's tag ([release.md §4](release.md)).
- **Promotion is pull-based.** A producer releases and stops. A consumer
  moves its own pin, and that pull request carries the zero-diff render as
  its evidence ([release.md §6](release.md)). No producer pushes a pin into
  a consumer.
- **The shared-workflow pin is a commit SHA with the tag in a comment**,
  moved by the bot, never by hand.
- **Hosted runners only**, enforced by the shared workflows' refusal of a
  self-hosted runner for a public caller ([repository.md §7](repository.md)).
- **A chart is agnostic about how a workload gets its cloud identity.** It
  exposes `serviceAccount.{create,name,annotations}` and grants nothing;
  whether Pod Identity, IRSA or something else binds the account is the
  platform's ([platform.md §2](platform.md)).

## Retired forms

Retired on **2026-09-29**, when this contract replaced the two doctrines it
folds in. A repository still carrying one fails the rule named.

| Retired | Replaced by | Rule |
|---|---|---|
| `version: 0.0.0-dev` in a committed `Chart.yaml` (the ci-plane doctrine's form; five charts used it, seventeen used `0.0.0`) | `0.0.0` | C1 |
| grouped CHANGELOG headings — several `## Unreleased` sections, a suffixed `## Unreleased — <topic>`, or one heading covering several versions | one `## vX.Y.Z` per tag, one `## Unreleased` | C5 |
| "a patch cut for dependency bumps alone has no CHANGELOG heading" (the ci-workflows doctrine) | every tag has a heading | C5 |
| "a public repository never names its consumers" (the ci-workflows doctrine) | a `Consumers` section naming repository and surface, never version | C8 |
| a per-repository `docs/doctrine.md` restating design rules | a link to this contract | C8 |

The road not taken on consumers, because git will not keep it: keeping the
consumer map only in the consuming estate was the previous rule, and it
kept versions out of public text, which this contract still does. What it
lost was the blast radius — an author could not see who a rename would
break. Naming the repository and the surface, and nothing else, recovers
that without publishing anything a consumer's own pin does not already
say.

## Applies to

The scope is the rule in [Scope](#scope); this list is that rule applied on
2026-09-29, for the conformance action's first run. A repository created
later is in scope by the rule, not by being added here.

| Repository | Ships |
|---|---|
| `access-roster` | charts, images, Go module, CLI, action |
| `amazon-eks-pod-identity-webhook` | chart, image |
| `argocd-ecr-updater` | chart, image |
| `audit` | chart, images, Go module |
| `ci-actions` | actions |
| `ci-cache` | chart, image, CLI |
| `ci-plane` | charts, images |
| `ci-workflows` | reusable workflows, renovate preset |
| `cloudflare` | charts, image, Go module |
| `cnpg-cluster` | charts |
| `gateway` | charts |
| `gemaal` | chart, image, Go module |
| `github-structure` | Go module |
| `nats-auth-callout` | chart, image |
| `observability` | charts, image |
| `ocictl` | CLIs, charts |
| `openbao` | charts, Go module |
| `policy` | contracts, Go module, packages, example charts and images |
| `tailscale` | chart, Go module |
| `workstation` | CLIs |

## Conformance

| Rule | Mechanism |
|---|---|
| C1 chart versions | `policy-conformance`: every `Chart.yaml` reads `0.0.0` |
| C2 values schema | `policy-conformance`: `values.schema.json` beside every `Chart.yaml`, unless a library chart is [exempted](#exemptions) |
| C3 goldens and refusals | `policy-conformance`: `tests/golden/<chart>/` and `tests/invalid/<chart>/` are non-empty, or the Go chart tests name both |
| C4 leak canary | `policy-conformance`: the script exists and `check` depends on `leak-canary` |
| C5 CHANGELOG | `policy-conformance`: heading grammar, order, one `Unreleased`, the latest tag present |
| C6 devbox pins | `policy-conformance`: no `latest` in `devbox.json` |
| C7 renovate | `policy-conformance`: extends the shared preset; every override has a `description` |
| C8 README | `policy-conformance`: the eleven headings, in order |
| C9 licence | `policy-conformance`: `LICENSE` is MIT, unless a fork is [exempted](#exemptions) |
| C10 security workflow | `policy-conformance`: `security.yaml` exists with `go.mod`; `vuln` in neither `ci.yaml` nor `check` |
| C11 image names | `policy-conformance`: no image repository ends in `<repo>/<repo>` with nothing published beside it |
| C12 pinned installs | `policy-conformance`: no `@latest`, no unversioned `oci://` install |
| C13 estate facts | review; the leak canary catches the mechanical half |
| C14 ServiceAccount per component | the chart's own render test; review elsewhere |
