---
name: devbox-just
description: >-
  Use when setting up or reviewing devbox.json and Justfile during a repository
  migration. Covers one locked, versioned toolchain, the Justfile as the
  gate (`just check` — build, test, lint, drift, leak-canary — needing
  nothing but the checkout) and FIXED recipe names that CI runs by name.
  Repository-agnostic; reference implementation is the policy repository's
  own devbox.json and Justfile
---

# Devbox and Just

How a repository declares its toolchain (one manifest, locked, materialised
by one command) and its commands (a Justfile whose recipe names are fixed),
per the Truvity policy repository contract.

## Reference

Self-contained detail, shipped with this skill in `references/`:

- [Toolchain](references/toolchain.md) — pins, grouped updates, the yarn/corepack failure and the limits of the checks
- [Recipes](references/recipes.md) — fixed names, locked dependencies, prerequisites and CI wiring

Reviewed on 2026-10-20 against the rebased checkout after policy v1.54.1.
Version examples are excerpts from `devbox.json`, not a second pin authority.
The recent parity update changed `devbox.lock` resolutions without changing
the manifest's version declarations; inspect both when reviewing toolchain drift.

External source of truth (needs network): [github.com/truvity/policy](https://github.com/truvity/policy) — reference implementation: the repository's own [`devbox.json`](https://github.com/truvity/policy/blob/master/devbox.json) and [`Justfile`](https://github.com/truvity/policy/blob/master/Justfile). Normative: [repository contract](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md) §2 (the gate) and §3 (the declared toolchain), [component contract](https://github.com/truvity/policy/blob/master/docs/contracts/component.md) C6 (pins) and C10 (vuln), [canon/toolchain.md](https://github.com/truvity/policy/blob/master/docs/canon/toolchain.md), [conformance guide](https://github.com/truvity/policy/blob/master/docs/guides/conformance.md). Consumer repositories may vendor this skill under a different root; adjust paths accordingly.

## The Gate, and the Fixed Recipe Names

**`just check` is the gate**: `build`, `test`, `lint`, `drift`,
`leak-canary`. It needs **no credentials, no container runtime and no
cluster** — it may download what the manifests and lock files name (a
first `go build` fetches modules, a first `uv sync` fetches packages),
because anyone with the checkout can fetch those too. A gate that needs a
cluster is a gate people run once a week and then argue with.

**Everything heavier is its own recipe AND its own CI job**, named so
nobody is surprised by what it needs. In this repository `ts` and `charts`
are separate lanes; `cluster*` needs a container runtime; `vuln` runs in
the security workflow. Network access alone does not disqualify a gate
recipe: resolving public dependencies named by a committed lock is allowed.
The infra chart's `cnpg-database` archive is such a dependency.

**Recipe names are fixed** (repository.md §2), because a person — and
anything automating across repositories — should not have to read a recipe
file to find out what the linter is called here:

| Recipe | Does |
|---|---|
| `check` | the gate: `build`, `test`, `lint`, `drift` and `leak-canary` |
| `build` | compiles everything |
| `test` | the tests that need no credentials, no container and no cluster |
| `lint` | every static check, including the ones that are not a linter |
| `drift` | regenerates committed generated code; fails if it moved |
| `leak-canary` | refuses a particular that must never be published |
| `vuln` | known vulnerabilities — its own scheduled workflow, **never** part of `check` (C10: an advisory with no released fix would turn every pull request red on a finding nobody can act on) |
| `charts` | renders and validates every chart |
| `golden` | regenerates committed renders; fails if they moved |

A missing job needs no recipe; an existing job uses the fixed name.
The naming failure and reference inventory are in [Recipes](references/recipes.md).

**CI runs recipes by name**, never by re-implementing them in a workflow:
a check that passes locally and fails in CI is then a bug in a recipe
rather than a difference nobody can reproduce. One fan-in job depending
on everything else is the single required status context (§5) — a matrix
or reusable-workflow job reports under a composed name and wedges every
open pull request the moment it moves.

## devbox.json — the One Declared Toolchain

Shape and full example: [Toolchain](references/toolchain.md). The rules:

- **Every entry names a version** (C6, checked mechanically by
  `policy-conformance`): no `latest`, no bare name, no empty string — two
  checkouts of the same commit must build with the same tools. A pin that
  should move is moved by the dependency bot (renovate extends
  `github>truvity/ci-workflows`, C7) in a pull request, with the followers
  it drags with it aligned in the same change.
- **`nodejs` sets `disable_plugin: true`**: the plugin's corepack shims go
  on PATH ahead of everything devbox installed and download their own yarn,
  so three declarations can disagree while every command succeeds (see
  Gotchas). `packageManager` stays in `package.json` for whoever does not
  use devbox, and is CHECKED against the `yarn-berry` pin — the reference
  repository's `hack/toolchain-check.py`, run by `just lint`.
- **The reference's `env.PATH` prepends `bin/`**: checked-in wrapper scripts for tools
  invoked through a `go.mod` tool directive or a pinned, checksum-verified
  download (`bin/helmctl`, `bin/pkl` in the reference). `.gitignore`
  covers `bin/*` and names its exceptions.
- **The init_hook installs lefthook**, tolerated to fail: the one check
  worth paying for on every commit is the leak canary. Onboarding is
  `devbox shell` (or direnv) then `just --list` — nothing else to remember.
- **`devbox.lock` is committed; `.devbox/` is gitignored.** Language
  dependencies are NOT declared here: `go.mod`, the uv lock (from
  `pyproject.toml`) and the yarn lockfile own those — nix provides the
  runtime and the tools the language's own manager cannot.

## Justfile

The mechanics the reference holds itself to — `[doc("…")]` on every recipe,
comments that say WHY and what a recipe NEEDS, `cd` into subdirectories,
shebangs with `set -euo pipefail`, handled nonzero exits, `drift` as
regenerate-then-`git diff --exit-code`, committed chart locks, neutral
defaults and no particulars — are in [Recipes](references/recipes.md).

The one mechanic that bites in CI: lanes call `devbox run -- just <recipe>`,
and every invocation generates and executes its own copy of
`.devbox/gen/scripts/.cmd.sh`. Never background a second `devbox run` beside
a live one — two started together can each be mid-(over)write of that shared
file while the other is mid-exec ("text file busy", measured in CI).
Background the `just` recipe inside one script instead.

## Best Practices

- When a tool is missing, add a line to the manifest. Never a fetch in a
  shell, and never a PATH someone exported.
- Python tooling (ruff, mypy, pytest) comes from the uv lock's dev groups
  (`uv run --frozen …`), not from nix — the interpreter is the manifest's
  job, the dependencies are the lockfile's.
- Run `just check` before pushing; it is exactly what CI runs, needing
  nothing CI does not have.
- When a tool version moves, move its followers in the same pull request
  (the yarn pair above, a `go.mod` directive with the Go triad) — and let
  the bot do the moving.

## Gotchas

- Corepack shadowing, duplicate Gradle toolchains and executable/version
  parity checks are covered in [Toolchain](references/toolchain.md).
- Product CI installs and tests ephemeral releases with kubectl/Helm. Persistent
  environment configuration and application delivery belong to the GitOps platform.
- A machine without the toolchain (an automation that only edits
  dependency pins) must not fail every commit on a missing command — the
  lefthook skip guard is `! command -v just`.

## Implementation Checklist

- [ ] every `devbox.json` entry names a version (C6 — `policy-conformance` checks it mechanically)
- [ ] `devbox.lock` committed; `.devbox/` ignored
- [ ] chart dependencies resolve from committed locks, with cold-checkout downloads accounted for
- [ ] `nodejs` sets `disable_plugin`; `packageManager` agrees with the `yarn-berry` pin
- [ ] recipe names match the fixed table; `just --list` shows a doc line for each
- [ ] `just check` passes from a clean clone needing nothing but the checkout (the conformance guide's first reviewer question)
- [ ] `vuln` unreachable from `check` — the conformance check follows the recipe graph, including body lines
- [ ] CI runs recipes by name; one fan-in job is the required status context
