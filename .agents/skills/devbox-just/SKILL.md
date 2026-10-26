---
name: devbox-just
description: >-
  Use when setting up or reviewing devbox.json and Justfile during a repository
  migration: one locked, versioned toolchain; `just check` as the gate (with
  drift where generated code exists); fixed recipe names CI runs by name.
  Distinguishes the policy producer's Go-focused gate from the url-shortener
  product reference.
---

# Devbox and Just

How a repository declares its toolchain (one locked manifest, materialised by one command) and its fixed recipe names, per the Truvity policy repository contract.

## Reference

Self-contained detail, shipped with this skill in `references/`:

- `references/toolchain.md` — pins, grouped updates, the yarn/corepack failure and the limits of the checks
- `references/recipes.md` — fixed names, locked dependencies, prerequisites and CI wiring

Siblings: `github-ci` owns workflow wiring; `helm-kubernetes` owns charts.

## Snapshot

Reviewed 2026-10-26 against policy `v1.54.5` (source `c24f403`) and
[url-shortener][product]. Policy produces contracts/Go modules/library; the
product owns the example, non-Go loaders and generated-code lanes (ADR 0014);
paths in each reference name their owner. Not pin authority: if this date
predates the checkout's CHANGELOG head, re-verify pins before relying on them.

[product]: https://github.com/truvity/url-shortener/tree/76770607aa08baa58331668c228dacd62a13a89a

External source of truth (needs network): [github.com/truvity/policy](https://github.com/truvity/policy) — reference implementation: the repository's own [`devbox.json`](https://github.com/truvity/policy/blob/master/devbox.json) and [`Justfile`](https://github.com/truvity/policy/blob/master/Justfile). Normative: [repository contract](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md) §2 (the gate) and §3 (the declared toolchain), [component contract](https://github.com/truvity/policy/blob/master/docs/contracts/component.md) C6 (pins) and C10 (vuln), [canon/toolchain.md](https://github.com/truvity/policy/blob/master/docs/canon/toolchain.md), [conformance guide](https://github.com/truvity/policy/blob/master/docs/guides/conformance.md). Consumer repositories may vendor this skill under a different root; adjust paths accordingly.

## The Gate, and the Fixed Recipe Names

**`just check` is the gate**: policy runs `build test lint leak-canary`; the
product also runs `drift` because it commits generated artifacts. A missing
job needs no recipe; never add a fake no-op. The gate needs **no credentials,
no container runtime and no cluster** — it may download what the manifests and
locks name, because anyone with the checkout can. A gate that needs a cluster
is a gate people run once a week and then argue with.

**Everything heavier is its own recipe AND its own CI job**, named so
nobody is surprised by what it needs. Policy's cluster helpers need a runtime;
its networked `example-canary` is advisory and `vuln` has its own workflow.
The product has separate `ts`, `charts` and integration lanes.

**Recipe names are fixed** (repository.md §2), because a person — and
anything automating across repositories — should not have to read a recipe
file to find out what the linter is called here:

| Recipe | Does |
|---|---|
| `check` | the applicable gate jobs; includes `drift` where generated code exists |
| `build` | compiles everything |
| `test` | the tests that need no credentials, no container and no cluster |
| `lint` | every static check, including the ones that are not a linter |
| `drift` | regenerates committed generated code; fails if it moved |
| `leak-canary` | refuses a particular that must never be published |
| `vuln` | known vulnerabilities — its own scheduled workflow, **never** part of `check` (C10: an advisory with no released fix would turn every pull request red on a finding nobody can act on) |
| `charts` | renders and validates every chart |
| `golden` | regenerates committed renders; fails if they moved |

The naming failure and reference inventory are in `references/recipes.md`.

**CI runs recipes by name**, never re-implemented in a workflow. Fan-in and required-context rules are `github-ci`'s and in `references/recipes.md`.

## devbox.json — the One Declared Toolchain

Shape and full example: `references/toolchain.md`. The rules:

- **Every entry names a version** (C6, checked by `policy-conformance`): no
  `latest`, no bare name, no empty string — two checkouts of one commit must
  build with the same tools. The bot (renovate extends
  `github>truvity/ci-workflows`, C7) moves a pin in a pull request and aligns
  its followers in the same change.
- **For a Node/Yarn consumer**, disable the Node plugin's corepack shims and
  check its `packageManager` against its declared Yarn. Those are product
  checks: policy installs neither tool; `hack/toolchain-check.py` only checks
  that each package entry names a version.
- **The reference's `env.PATH` prepends `bin/`**: checked-in wrapper scripts
  for tools invoked through a `go.mod` tool directive or a pinned,
  checksum-verified tool (`bin/helmctl`). A product's pinned Pkl wrapper
  belongs to that product. Ignore outputs and name script exceptions.
- **The init_hook installs lefthook**, tolerated to fail: the one check
  worth paying for on every commit is the leak canary. Onboarding is
  `devbox shell` (or direnv) then `just --list` — nothing else to remember.
- **`devbox.lock` is committed; `.devbox/` is gitignored.** Language
  dependencies are NOT declared here: `go.mod`, the uv lock (from
  `pyproject.toml`) and the yarn lockfile own those — nix provides the
  runtime and the tools the language's own manager cannot.

## Justfile

The shared mechanics — `[doc("…")]`, WHY/NEEDS comments, `cd`, `set -euo
pipefail`, handled nonzero exits, regenerate-then-diff `drift`, committed
chart locks, neutral defaults, no particulars, and the CI rule that one
`devbox run` at a time owns `.devbox/gen/scripts/.cmd.sh` — are in
`references/recipes.md`.

## Best Practices

- When a tool is missing, add a line to the manifest. Never a fetch in a
  shell, and never a PATH someone exported.
- In a Python product, tooling (ruff, mypy, pytest) comes from uv's dev groups
  (`uv run --frozen …`), not from nix — the interpreter is the manifest's
  job, the dependencies are the lockfile's.

## Gotchas

- Corepack shadowing, duplicate Gradle toolchains and executable/version
  parity checks are covered in `references/toolchain.md`.
- A machine without the toolchain (an automation that only edits
  dependency pins) must not fail every commit on a missing command — the
  lefthook skip guard is `! command -v just`.

## Implementation Checklist

- [ ] every `devbox.json` entry names a version (C6 — `policy-conformance` checks it mechanically)
- [ ] `devbox.lock` committed; `.devbox/` ignored; chart dependencies resolve from committed locks, with cold-checkout downloads accounted for
- [ ] for a Node/Yarn consumer: disable corepack shadowing and check the package-manager pin
- [ ] recipe names match the fixed table; `just --list` shows a doc line for each
- [ ] `just check` passes from a clean clone needing nothing but the checkout (the conformance guide's first reviewer question)
- [ ] `vuln` unreachable from `check` — the conformance check follows the recipe graph, including body lines
- [ ] CI runs recipes by name; one fan-in job is the required status context
