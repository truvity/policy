# The Recipe Inventory (Truvity policy contract)

The fixed recipe names, what each recipe of the reference repository needs, and how the gate is wired into CI. Condensed and self-contained; normative sources (need network): [repository contract §2](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md), the reference [Justfile](https://github.com/truvity/policy/blob/master/Justfile), [conformance guide](https://github.com/truvity/policy/blob/master/docs/guides/conformance.md).

## The fixed names, and why they are fixed

`check build test lint drift vuln charts golden leak-canary` — a repository that has the job under another name has made every caller special: anything automating across repositories has to learn its exception, and a person moving between them has to read a recipe file to find out what the linter is called. A repository without one of these jobs does not have the recipe.

The failure is invisible from inside the repository: everything runs, the gate is green, and only a caller from outside notices. The reference repository shipped its chart check as `kubeconform` — an accurate name for the tool and the wrong name for the recipe — and did not notice until the contract was read back against it. List the names against `just --list` rather than assuming.

## What each recipe of the reference needs

The gate (`check`) is `build`, `test`, `lint`, `drift`, `leak-canary`, and needs no credentials, container runtime or cluster. It may download what the manifests and lock files name. The inventory below is the reference implementation as reviewed on 2026-10-20; helper recipes need not each become a separate CI job.

| Recipe | Needs | Why it is (or is not) in `check` |
|---|---|---|
| `build`, `test` | declared tools and locked public dependencies | gate recipes; chart tests resolve `cnpg-database` from OCI when absent |
| `lint` | nothing but the checkout | every static check, including the ones that are not a linter |
| `drift` | declared tools and locked chart dependencies | runs `schemas`, `protos`, `chart-schemas`, `chart-deps`, then checks generated files with `git diff --exit-code`; `chart-deps` resolves archives, not generated source |
| `chart-deps` | local library and public OCI registry on a cold checkout | builds the application and infra dependencies from `Chart.lock`; rejects a stale declaration |
| `chart-locks` | registry access for the OCI dependency | intentionally updates both locks after a dependency change; review and commit the locks, ignore `charts/` archives |
| `leak-canary` | nothing but the checkout | refuses a particular that must never be published |
| `vuln` | the network, on a schedule | news about the world, not about the change under review (C10); its own `security.yaml` workflow |
| `charts` | the network | fetches the schema for a custom resource |
| `ts` | a package registry | installs what the gate does not depend on |
| `cluster`, `cluster-verify`, `cluster-and-snapshot` | a container runtime | the local cluster; in CI, the shared integration workflow's kind tier |
| `example-snapshot` | a container runtime + the box's registry | packages exactly as a release does — a published artifact is tested as published |
| `example-fixture`, `example-install`, `example-smoke`, `example-prober` | the running cluster | the cluster lane; `cluster-all` is the whole tier from nothing |
| `golden` | tools and resolved chart dependencies | regenerates all three render suites: application, infra and service-library example; inspect the render change before accepting new goldens |
| `shadow-generate`, `shadow-generated`, `shadow-diff` | the network | fetch Pkl and the contracts' packages from GitHub |
| `generate` | the network | the name the fleet's pin-bump job runs — a recipe, not an alias, because `just --summary` does not list aliases |

The infra chart's cnpg-database dependency is now unconditional, even if its
`roles: []` renders no objects. Resolve its archive from the lock before a
render; do not skip `chart-deps` just because the install supplies an external
Secret or its own Cluster manages the role. Template output and dependency
resolution are different prerequisites.

## How the gate is wired into CI

- **CI runs recipes by name**, never by re-implementing them in a workflow — a check that passes locally and fails in CI is a bug in a recipe rather than a difference nobody can reproduce. The reference calls the shared `check.yaml` workflow once per lane with a recipe list; cluster lanes call `devbox run -- just <recipe>`.
- **One fan-in job is the single required status context.** Matrix and reusable-workflow jobs report under composed names that change when the matrix changes, so requiring one directly wedges every open pull request the moment anything moves — and renaming a required job wedges every open pull request.
- **`vuln` is not among the recipes CI requires**, and `just check` does not reach it: the conformance check follows the Justfile's dependency graph — `check` depending on `vuln` directly, through another recipe, or via a body line running `just vuln` are all refusals.
- **Distinguish resolution from infrastructure.** Repository contract §2 explicitly allows downloads named by manifests and locks. The conformance guide's older “network off” wording is stricter than that contract; do not use it to exclude locked OCI dependencies from `check`. Credentials, containers and clusters still belong in separate lanes.
- **Coverage has a boundary.** The kind lane supplies servers and an example fixture, not cloud controllers or admission machinery. It does not prove IAM, cloud identity injection, or network-policy enforcement; those need a platform's shared-cluster test.

## Justfile mechanics the reference holds itself to

- `[doc("…")]` on every recipe — `just --list` is the interface; comments say WHY, and what a recipe NEEDS.
- Recipes run from the repo root (`cd dir &&` into subdirectories); multi-step recipes use a shebang with `set -euo pipefail`.
- Handle expected nonzero results explicitly (`if`, `!`, or an accumulated failure flag). The reference's aggregate lint recipe uses `set -uo pipefail` and `fail`; do not prescribe disabling `-e` for every grep — a grep used as an `if` condition already has the intended semantics.
- Parameters or env vars use neutral defaults (`${NS:-shortener}`); never a hardcoded deployment environment, never a particular.
