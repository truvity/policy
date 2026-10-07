---
name: github-ci
description: >-
  The GitHub Actions layer per the Truvity policy contracts — for bringing
  a repository onto Truvity infrastructure: truvity/ci-workflows called by
  COMMIT SHA, recipes run by name with devbox on the runner,
  policy-conformance judging the component contract, one fan-in required
  context, the cluster integration lane, security.yaml (vuln, scheduled,
  never the gate), the tag-driven release (one job, digest-pinned charts,
  stamped versions), auto-release patches, renovate + the devbox parity
  job, the lefthook leak canary. The toolchain and the gate are the
  devbox-just skill's half; the chart is helm-kubernetes's
---

# GitHub CI

The GitHub Actions layer that sits between the gate and the platform: the
workflows that run a repository's recipes, judge it against the component
contract, and cut its releases. **CI never deploys**: a release lands in
a registry, and a separate GitOps control plane renders and syncs it — a
product repository's CI never applies to a cluster.

Division of labour: the toolchain and the gate are the devbox-just skill's
half; the chart is the helm-kubernetes skill's; this skill is everything
under `.github/workflows/`, plus `renovate.json` and `lefthook.yml`.

## Reference

Self-contained detail, shipped with this skill in `references/`:

- `references/workflows.md` — the four workflow skeletons plus `renovate.json` and `lefthook.yml`, as a copy-sheet for migration
- `references/release.md` — the build is the release: digests and ordering, the tag as the sole version authority, adoption proved by render, as-published testing

External source of truth (needs network): [github.com/truvity/policy](https://github.com/truvity/policy) — reference implementation: [`.github/workflows/`](https://github.com/truvity/policy/tree/master/.github/workflows). Normative: [component contract](https://github.com/truvity/policy/blob/master/docs/contracts/component.md) (C1-C17 and its conformance table — what `policy-conformance` judges), [repository contract](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md) §2 (recipes by name), §5 (required checks), §6 (bot bumps), §7 (public repositories), [release contract](https://github.com/truvity/policy/blob/master/docs/contracts/release.md). The shared workflows: [truvity/ci-workflows](https://github.com/truvity/ci-workflows) (the only thing a caller pins) over [truvity/ci-actions](https://github.com/truvity/ci-actions).

## The model

- **One caller, pinned by COMMIT SHA with the tag in a comment** — never a
  tag alone: tags move, and this code runs in every repository of both
  organisations.
- **CI runs recipes by name** (`devbox run -- just <recipe>`), never by
  re-implementing them in a workflow — a check that passes locally and
  fails in CI is a bug in a recipe, not a difference nobody can
  reproduce. Devbox materialises the declared toolchain on the runner
  (`devbox-install-action`, cache on): the same versions as a laptop.
- **`policy-conformance: true`** (+ `strict`) on the gate job: C1-C17 are
  judged, shown in the job summary, and fail the job when a rule fails.
  What it checks mechanically is in the component contract's conformance
  table; the rest is review, and the table says so.
- **One fan-in `check` job** depending on every other job is the single
  required status context — matrix and reusable-workflow jobs report under
  composed names that change when the matrix changes, and requiring one
  wedges every open pull request the moment anything moves.
- **Permissions: `contents: read` at the top.** The caller grants what a
  job needs (a reusable workflow can narrow permissions but never widen
  them).
- **A public repository**: hosted runners only — a fork's code must never
  execute on the estate's own infrastructure — no secrets in CI (work that
  needs the estate runs in a private repository checking this one out at a
  pinned version), and `pull_request_target` never checks out the pull
  request's head.

## The four workflows

- **`ci.yaml`** — the merge gate. `check.yaml` once per lane: the gate
  recipes (`build test lint drift leak-canary`); every recipe that needs
  more than the checkout (a registry, the network) as its OWN job; and the
  cluster tier through `integration.yaml` with `lanes` whose build,
  install and test steps are `devbox run -- just <recipe>` — the workflow
  stands up the box itself, fetching this repository's own `hack/kind/` at
  a pinned release tag, so the recipe a job runs is exactly what a laptop
  runs. Ends with the fan-in `check` job.
- **`security.yaml`** — the `vuln` recipe, through the same shared
  workflow, on push, on pull requests and on a daily schedule. In no other
  workflow's recipe list, and unreachable from `just check` (C10: a new
  advisory is news about the world; one with no released fix would turn
  every pull request red on a finding nobody can act on).
- **`release.yaml`** — tag-triggered, calling `release-public.yaml`. ONE
  job builds and pushes the images, then packages the charts from the
  digests that build produced (`chart-images: goreleaser`,
  `require-image-digests: true`, `charts` + `chart-root`) — the ordering
  is made impossible to get wrong rather than merely written down. The tag
  is the sole version authority: committed versions are placeholders
  (C1), stamped at release with an edit that asserts it matched. A second
  job attaches what goreleaser does not build, and its concurrency never
  cancels — a half-published artifact set is worse than a queued wait, and
  a version that failed halfway is burned.
- **`auto-release.yaml`** — PATCH releases only (release.md §4), when
  merged dependency bumps have moved the default branch past the latest
  tag: weekly, or immediately for a push whose merged pull request carries
  the `security` label. Doubly guarded (the organisation's issuer
  configured AND `vars.AUTO_RELEASE`), and no app key committed — the job
  exchanges its own identity token at the issuer for a token narrowed to
  this repository.

## Renovate and the parity job

- `renovate.json` extends `github>truvity/ci-workflows` (C7); every
  override carries a `description` saying why this repository differs.
- **The devbox parity job owns `devbox.json`, `devbox.lock` and the `go`
  directive together** — Renovate leaves all three alone, or the lock and
  the directive and the manifest disagree while each looks fine.
- Yarn's `packageManager` moves with devbox, never on its own: a bump to
  a newer yarn rewrites the lockfile to a format the devbox's yarn
  refuses.
- Base images pinned by tag AND digest (goreleaser's `kos:`, `.ko.yaml`)
  need a custom regex manager — the dockerfile manager already moves a
  `:latest@sha256:` digest.

## The hook

`lefthook.yml` runs the leak canary on every commit — the one check worth
paying for before a push, skipped on a machine without `just` so an
automation that only edits dependency pins does not fail every commit on
a missing command.

## Gotchas

- Pinning a shared workflow by tag instead of SHA — the tag moves, and
  every repository's CI changes in a way no pull request reviewed.
- A recipe in `ci.yaml`'s gate list that needs more than the checkout —
  the gate is not the gate, and the first contributor without the
  credential finds out.
- Cancelling a release mid-publish (a version that failed halfway is
  burned — `concurrency` with `cancel-in-progress: false`).
- A matrix or reusable-workflow job set as a required context — its
  composed name changes with the matrix and wedges every open pull
  request.
- A version committed to a manifest: the release stamps the placeholder,
  and a package published at `0.0.0` cannot be taken back.
- Secrets in a public repository's CI — there are none to have; the work
  runs in a private repository at a pinned version.

## Implementation Checklist (migration order)

- [ ] devbox.json + Justfile gate standing (`devbox-just` skill); `just check` hermetic from a clean clone
- [ ] `ci.yaml`: `check.yaml` by SHA — gate recipes, `policy-conformance: true` + strict; out-of-gate recipes their own jobs; cluster lane through `integration.yaml`; fan-in `check` job as the one required context
- [ ] `security.yaml`: `vuln` on a schedule, in nothing else (C10)
- [ ] `renovate.json` extends the shared preset; the parity job owns devbox + `go`; every override described (C7)
- [ ] `lefthook.yml`: leak canary pre-commit (C4)
- [ ] `release.yaml`: one job, digest-pinned charts, stamped versions, an as-published test for every artifact
- [ ] `auto-release.yaml`: patches only, doubly guarded
- [ ] C1-C17 green in the job summary — and the review-only rows read, not assumed
