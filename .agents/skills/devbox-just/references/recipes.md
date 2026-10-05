# The Recipe Inventory (Truvity policy contract)

The fixed recipe names, what each recipe of the reference repository needs, and how the gate is wired into CI. Condensed and self-contained; normative sources (need network): [repository contract §2](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md), the reference [Justfile](https://github.com/truvity/policy/blob/master/Justfile), [conformance guide](https://github.com/truvity/policy/blob/master/docs/guides/conformance.md).

## The fixed names, and why they are fixed

`check build test lint vuln charts golden leak-canary` — a repository that has the job under another name has made every caller special: anything automating across repositories has to learn its exception, and a person moving between them has to read a recipe file to find out what the linter is called. A repository without one of these jobs does not have the recipe.

The failure is invisible from inside the repository: everything runs, the gate is green, and only a caller from outside notices. The reference repository shipped its chart check as `kubeconform` — an accurate name for the tool and the wrong name for the recipe — and did not notice until the contract was read back against it. List the names against `just --list` rather than assuming.

## What each recipe of the reference needs

The gate (`check`) is `build`, `test`, `lint`, `drift`, `leak-canary`, and needs nothing but the checkout — it may download what the manifests and lock files name, because anyone with the checkout can fetch those too. Everything below the line is its own recipe AND its own CI job, named so nobody is surprised by what it needs.

| Recipe | Needs | Why it is (or is not) in `check` |
|---|---|---|
| `build`, `test` | nothing but the checkout | the gate: a first `go build`/`uv sync` fetches what the locks name |
| `lint` | nothing but the checkout | every static check, including the ones that are not a linter |
| `drift` | nothing but the checkout | regenerates committed generated code (`schemas`, `protos`, `chart-schemas`, `chart-deps`), then `git diff --exit-code` |
| `leak-canary` | nothing but the checkout | refuses a particular that must never be published |
| `vuln` | the network, on a schedule | news about the world, not about the change under review (C10); its own `security.yaml` workflow |
| `charts` | the network | fetches the schema for a custom resource |
| `ts` | a package registry | installs what the gate does not depend on |
| `cluster`, `cluster-verify`, `cluster-and-snapshot` | a container runtime | the local cluster; in CI, the shared integration workflow's kind tier |
| `example-snapshot` | a container runtime + the box's registry | packages exactly as a release does — a published artifact is tested as published |
| `example-fixture`, `example-install`, `example-smoke`, `example-e2e-chart`, `example-prober` | the running cluster | the cluster lane; `cluster-all` is the whole tier from nothing |
| `golden` | nothing but the checkout | regenerates committed renders — read the diff BEFORE running: a golden updated without being read records whatever happened |
| `shadow-generate`, `shadow-generated`, `shadow-diff` | the network | fetch Pkl and the contracts' packages from GitHub |
| `generate` | the network | the name the fleet's pin-bump job runs — a recipe, not an alias, because `just --summary` does not list aliases |

## How the gate is wired into CI

- **CI runs recipes by name**, never by re-implementing them in a workflow — a check that passes locally and fails in CI is a bug in a recipe rather than a difference nobody can reproduce. The reference calls the shared `check.yaml` workflow once per lane with a recipe list; cluster lanes call `devbox run -- just <recipe>`.
- **One fan-in job is the single required status context.** Matrix and reusable-workflow jobs report under composed names that change when the matrix changes, so requiring one directly wedges every open pull request the moment anything moves — and renaming a required job wedges every open pull request.
- **`vuln` is not among the recipes CI requires**, and `just check` does not reach it: the conformance check follows the Justfile's dependency graph — `check` depending on `vuln` directly, through another recipe, or via a body line running `just vuln` are all refusals.
- **A gate that needs more is not the gate.** The first reviewer question in the conformance guide: does `just check` pass from a clean clone, with nothing but the checkout? If it needs a registry, a container or a cluster, the rest of the review is guesswork.

## Justfile mechanics the reference holds itself to

- `[doc("…")]` on every recipe — `just --list` is the interface; comments say WHY, and what a recipe NEEDS.
- Recipes run from the repo root (`cd dir &&` into subdirectories); multi-step recipes use a shebang with `set -euo pipefail`.
- The one exception: a check that greps for ABSENCE runs with no `-e` and its own `fail` flag — a grep that finds nothing exits non-zero, and under `set -e` the gate fails silently precisely when it should pass.
- Parameters are env vars with neutral defaults (`NS:-shortener`); never a hardcoded environment, never a particular.
