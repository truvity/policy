---
name: github-ci
description: >-
  Use when migrating or reviewing .github/workflows, Renovate and releases:
  SHA-pinned ci-workflows, producer/product gates, advisory canaries, C18
  portability, artifact ownership, conformance, updater ownership, patch
  automation and the leak canary. The gate is devbox-just's; charts are
  helm-kubernetes's.
---

# GitHub CI

The GitHub Actions layer between the gate and the platform: workflows run a
repository's recipes, judge it against the component contract, and cut its
releases. **Persistent delivery is GitOps-owned**: the registry holds artifacts
and the platform moves its pins, values and Applications. Product CI tests
ephemeral copies in an integration cluster. This skill owns `.github/workflows/`,
`renovate.json` and `lefthook.yml`.

## Reference

Self-contained detail, shipped with this skill in `references/`:

- `references/workflows.md` — producer/product caller variants, protected integration, Renovate and lefthook
- `references/release.md` — digest ordering, external image pins, adoption evidence and as-published testing

Siblings: `devbox-just` owns the gate and toolchain; `helm-kubernetes` owns charts.

## Snapshot

Reviewed 2026-10-26 against policy `v1.54.5` and [url-shortener][product]. Both
callers pin ci-workflows **v5.5.0** at `f0f75f715a4feef8ab1e4e6222e84708bb32f6cd`;
verify every placeholder SHA before use. Not pin authority: if this date
predates the checkout's CHANGELOG head, re-verify pins before relying on them.

[product]: https://github.com/truvity/url-shortener/tree/76770607aa08baa58331668c228dacd62a13a89a

External source of truth (needs network): [github.com/truvity/policy](https://github.com/truvity/policy) — reference implementation: [`.github/workflows/`](https://github.com/truvity/policy/tree/master/.github/workflows). Normative: [component contract](https://github.com/truvity/policy/blob/master/docs/contracts/component.md) (C1-C18 and each rule's conformance mechanism), [repository contract](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md) §2 (recipes by name), §5 (required checks), §6 (bot bumps), §7 (public repositories), [release contract](https://github.com/truvity/policy/blob/master/docs/contracts/release.md). The shared workflows: [truvity/ci-workflows](https://github.com/truvity/ci-workflows) over [truvity/ci-actions](https://github.com/truvity/ci-actions).

## The model

- **One caller, pinned by COMMIT SHA with the tag in a comment** — tags move,
  and this code runs in every repository of both organisations.
- **CI runs recipes by name**, never re-implemented — a check that passes
  locally but fails in CI is a recipe bug. Devbox materialises the declared
  toolchain on the runner (`devbox-install-action`, cache on).
- **`policy-conformance: true` and `policy-conformance-strict: true`** enable
  shared **C1-C13 and C18**; C13 has review-only parts and C18 is an **error**
  in strict mode (runner portability in `references/workflows.md`).
  **C14-C17 require product-owned chart/render/schema/lock/drift tests**; live
  identity, namespace policy and database behavior need platform tests.
- **One fan-in `check` job** is the single required status context. Matrix and
  reusable-workflow jobs report composed names that change; requiring one wedges
  every open pull request. Advisory jobs stay outside; policy's canary is one.
- **Permissions: `contents: read` at the top.** The caller grants what a job
  needs; a reusable workflow can narrow but never widen permissions.
- **Public PR checks:** hosted, read-only, no estate identity;
  `pull_request_target` never executes an untrusted head. Publishing uses the
  scoped built-in token; protected integration has a separate OIDC grant
  authorized by receiver-side trust, never a static key.

## Workflows and ownership

- **Policy CI:** shared `build test lint leak-canary`; the fan-in requires only
  that caller. `example-canary` tests the separate product against this Go
  checkout through go.work and is advisory; local service-lib substitution is
  opt-in, so default success does not prove new chart templates.
- **Product PR CI:** shared gate including drift, plus `ts`, `charts` and
  shadow jobs; hosted, read-only, no cluster credentials. **Protected
  integration** is separate: trusted pushes/tags/dispatch, job OIDC, platform
  inputs, `snapshot` then `e2e-*`, unconditional cleanup. Receiver-side trust
  limits the accepted repository/ref/workflow; an editable `if:` is not that
  boundary.
- **`security.yaml` — the `vuln` recipe alone**, on push, on pull requests and
  daily; in no other workflow and unreachable from `just check` (C10).
- **Release:** policy publishes its Go module and `service-lib`, no images or
  language packages; the product publishes its own images and two charts in one
  ordered job and does not republish the library. The tag stamps owned
  artifacts; dependencies retain upstream pins. `references/release.md`.
- **`auto-release.yaml` — PATCH releases only** (release.md §4): weekly, or
  immediately for a push whose merged pull request carries the `security`
  label. Doubly guarded (issuer configured AND `vars.AUTO_RELEASE`), no app key
  committed — the job exchanges its identity token for one narrowed to this
  repository. Needs the tagging App's ruleset bypass; with mandatory review use
  `changelog-heading: never` (release §3) and review the schedule's range so an
  untagged feature cannot ship as the next patch.

## Renovate and the parity job

- `renovate.json` extends `github>truvity/ci-workflows` (C7); every override
  carries a `description` saying why this repository differs.
- **The devbox parity job owns `devbox.json`, `devbox.lock` and the `go`
  directive together** — Renovate leaves all three alone, or lock, directive
  and manifest disagree while each looks fine.
- Confirm fleet parity enrollment before copying those ignores; the preset
  alone is not evidence another updater will move the pins.
- For a Node product, Yarn's `packageManager` moves with devbox: a newer yarn
  rewrites the lockfile to a format the devbox's yarn refuses.
- Base images pinned by tag AND digest (goreleaser's `kos:`, `.ko.yaml`) need
  a custom regex manager; the dockerfile manager moves `:latest@sha256:`.
- A service-lib consumer groups its policy Go-module pin, OCI chart version
  and lock update; tests hold the two policy versions equal.

## The hook

`lefthook.yml` runs the leak canary on every commit — the one check worth
paying for before a push. It is skipped on a machine without `just`, so a
dependency-pin automation does not fail every commit on a missing command.

## Gotchas

- A gate recipe needing credentials/containers/a cluster — the first
  contributor without them finds out; public locked downloads are allowed.
- Cancelling a release mid-publish burns the version; `concurrency` with
  `cancel-in-progress: false`, and a manifest `0.0.0` cannot be taken back.
- Granting estate identity to fork-triggered checks: keep protected
  integration's OIDC grant and receiver trust separate; scope the publishing
  token through declared permissions.
- Copying producer-only or historical commands into a product caller: choose
  the owning repository's recipes, and keep optional publishing jobs only where
  that repository publishes those packages.

## Implementation Checklist (migration order)

- [ ] devbox.json + Justfile gate standing (`devbox-just` skill); `just check` hermetic from a clean clone
- [ ] `ci.yaml` + `security.yaml`: applicable recipes, strict conformance, stable fan-in, advisory canaries excluded; `vuln` scheduled only (C10)
- [ ] `renovate.json` extends the shared preset; the parity job owns devbox + `go`; every override described (C7)
- [ ] `lefthook.yml`: leak canary pre-commit (C4)
- [ ] product integration is separate from fork PRs, with scoped OIDC trust and cleanup
- [ ] release only owned artifacts; for image-bearing charts build before packaging, with verified digests and as-published evidence
- [ ] `auto-release.yaml`: patches only, doubly guarded
- [ ] C1-C13 and C18 shared checks pass; C18 exceptions are reviewed/path-scoped; C14-C17 tests and review/live evidence recorded
