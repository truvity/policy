---
name: github-ci
description: >-
  Use when migrating or reviewing .github/workflows, Renovate and release
  integration under the Truvity policy contracts — for bringing
  a repository onto Truvity infrastructure: truvity/ci-workflows called by
  COMMIT SHA, recipes run by name with devbox on the runner,
  policy-conformance and runner-agnostic YAML, one fan-in required
  context, the cluster integration lane, security.yaml (vuln, scheduled,
  never the gate), the tag-driven release (one job, digest-pinned charts,
  stamped versions), auto-release patches, renovate + the devbox parity
  job, the lefthook leak canary. The toolchain and the gate are the
  devbox-just skill's half; the chart is helm-kubernetes's
---

# GitHub CI

The GitHub Actions layer that sits between the gate and the platform: the
workflows that run a repository's recipes, judge it against the component
contract, and cut its releases. **Persistent delivery is GitOps-owned**:
releases land in a registry and the platform moves its pins, values and
Applications. Product CI does install and test ephemeral copies in an
integration cluster; it does not become a second production reconciler.

Division of labour: the toolchain and the gate are the devbox-just skill's
half; the chart is the helm-kubernetes skill's; this skill is everything
under `.github/workflows/`, plus `renovate.json` and `lefthook.yml`.

## Reference

Self-contained detail, shipped with this skill in `references/`:

- [Workflows](references/workflows.md) — four workflow skeletons, optional package publishing, Renovate and lefthook
- [Release](references/release.md) — digest ordering, external image pins, adoption evidence and as-published testing

Reviewed on 2026-10-21 against the checkout after policy v1.54.2.
The current reference caller pins ci-workflows **v5.0.0** at
`d4350d573f67de8f7a145a4de94c392c6b43fd8c`. Choose and verify a supported SHA at
migration time; the `<sha>` placeholders in the copy-sheet are not runnable pins.

External source of truth (needs network): [github.com/truvity/policy](https://github.com/truvity/policy) — reference implementation: [`.github/workflows/`](https://github.com/truvity/policy/tree/master/.github/workflows). Normative: [component contract](https://github.com/truvity/policy/blob/master/docs/contracts/component.md) (C1-C18 and each rule's conformance mechanism), [repository contract](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md) §2 (recipes by name), §5 (required checks), §6 (bot bumps), §7 (public repositories), [release contract](https://github.com/truvity/policy/blob/master/docs/contracts/release.md). The shared workflows: [truvity/ci-workflows](https://github.com/truvity/ci-workflows) over [truvity/ci-actions](https://github.com/truvity/ci-actions).

## The model

- **One caller, pinned by COMMIT SHA with the tag in a comment** — never a
  tag alone: tags move, and this code runs in every repository of both
  organisations.
- **CI runs recipes by name** (`devbox run -- just <recipe>`), never by
  re-implementing them in a workflow — a check that passes locally and
  fails in CI is a bug in a recipe, not a difference nobody can
  reproduce. Devbox materialises the declared toolchain on the runner
  (`devbox-install-action`, cache on): the same versions as a laptop.
- **`policy-conformance: true` and `policy-conformance-strict: true`** enable
  shared **C1-C13 and C18** checks; C13 still has review-only parts. C18 is an
  **error** in strict mode: [runner portability](references/workflows.md#c18-runner-agnostic-yaml).
  **C14-C17 require product-owned chart/render/schema/lock/drift tests**.
  Live identity, namespace policy and database behavior need platform tests.
- **One fan-in `check` job** depending on every other job is the single
  required status context — matrix and reusable-workflow jobs report under
  composed names that change when the matrix changes, and requiring one
  wedges every open pull request the moment anything moves.
- **Permissions: `contents: read` at the top.** The caller grants what a
  job needs (a reusable workflow can narrow permissions but never widen
  them).
- **A public repository**: hosted runners only — a fork's code must never
  execute on the estate's own infrastructure — no estate secrets in CI (work that
  needs the estate runs in a private repository checking this one out at a
  pinned version), and `pull_request_target` never checks out the pull
  request's head. Publishing can use the built-in, permission-scoped GitHub
  token as shown in the reference workflow.

## The four workflows

Skeletons for all four: [Workflows](references/workflows.md). What each is for:

- **`ci.yaml` — the merge gate.** `check.yaml` once per lane: the gate
  recipes; infrastructure-needing or deliberately separate checks as their own
  jobs; the cluster tier through `integration.yaml`, whose kind box comes from
  the **policy repository's** `hack/kind/` at `policy-version` (never copied
  into the adopting repository) while the lane commands are the caller's
  recipes. Ends with the fan-in `check` job.
- **`security.yaml` — the `vuln` recipe alone**, on push, on pull requests
  and daily; in no other workflow's recipe list and unreachable from
  `just check` (C10).
- **`release.yaml` — tag-triggered** (`release-public.yaml`; private
  products select `release-private.yaml`). ONE job builds and pushes the
  images, then packages the charts from the digests that build produced —
  the ordering is made impossible to get wrong. The tag is the sole version
  authority: committed versions are placeholders, stamped by an edit that
  asserts it matched; external images keep their owners' pins and version
  compatibility rules ([Release](references/release.md)). Publishing never
  cancels — a version that failed halfway is burned.
- **`auto-release.yaml` — PATCH releases only** (release.md §4): weekly, or
  immediately for a push whose merged pull request carries the `security`
  label. Doubly guarded (issuer configured AND `vars.AUTO_RELEASE`), no app
  key committed — the job exchanges its identity token for one narrowed to
  this repository. Needs the tagging App's ruleset bypass; with a mandatory
  approving review use `changelog-heading: never` (release §3), and review
  the schedule's range so an untagged feature cannot silently ship as the
  next patch.

## Renovate and the parity job

- `renovate.json` extends `github>truvity/ci-workflows` (C7); every
  override carries a `description` saying why this repository differs.
- **The devbox parity job owns `devbox.json`, `devbox.lock` and the `go`
  directive together** — Renovate leaves all three alone, or the lock and
  the directive and the manifest disagree while each looks fine.
- Confirm fleet parity enrollment before copying those ignores; the shared
  Renovate preset alone is not evidence that another updater will move the pins.
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

- A gate recipe needing credentials/containers/a cluster — the first
  contributor without them finds out. Public locked chart downloads are allowed.
- Cancelling a release mid-publish (a version that failed halfway is
  burned — `concurrency` with `cancel-in-progress: false`).
- A version committed to a manifest: the release stamps the placeholder,
  and a package published at `0.0.0` cannot be taken back.
- Estate credentials in a public repository's CI — that work runs in a
  private repository at a pinned version; the built-in publishing token is
  scoped through the workflow's declared permissions.
- Calling `hack/publish.sh` with unset `VERSION`/`TAG` or no registry token.
  The copy-sheet includes tag derivation, asserted stamping and the scoped
  built-in GitHub token; these repository scripts must actually exist.

## Implementation Checklist (migration order)

- [ ] devbox.json + Justfile gate standing (`devbox-just` skill); `just check` hermetic from a clean clone
- [ ] `ci.yaml`: `check.yaml` by SHA — gate recipes, `policy-conformance: true` + strict; out-of-gate recipes their own jobs; cluster lane through `integration.yaml`; fan-in `check` job as the one required context
- [ ] `security.yaml`: `vuln` on a schedule, in nothing else (C10)
- [ ] `renovate.json` extends the shared preset; the parity job owns devbox + `go`; every override described (C7)
- [ ] `lefthook.yml`: leak canary pre-commit (C4)
- [ ] `release.yaml`: one job, digest-pinned charts, stamped versions, an as-published test for every artifact
- [ ] `auto-release.yaml`: patches only, doubly guarded
- [ ] C1-C13 and C18 shared checks pass; C18 exceptions are reviewed/path-scoped; C14-C17 tests and review/live evidence recorded
