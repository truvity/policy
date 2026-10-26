# The Recipe Inventory (Truvity policy contract)

Fixed names, with separate inventories for the producer and product. Normative
source: [repository contract §2](https://github.com/truvity/policy/blob/master/docs/contracts/repository.md).
The policy [Justfile](https://github.com/truvity/policy/blob/master/Justfile) and
product [Justfile][product-justfile] are different implementations, not
interchangeable command inventories.

[product-justfile]: https://github.com/truvity/url-shortener/blob/76770607aa08baa58331668c228dacd62a13a89a/Justfile

## The fixed names, and why they are fixed

`check build test lint drift vuln charts golden leak-canary` — a repository that has the job under another name has made every caller special: anything automating across repositories has to learn its exception, and a person moving between them has to read a recipe file to find out what the linter is called. A repository without one of these jobs does not have the recipe.

The failure is invisible from inside the repository: everything runs, the gate is green, and only a caller from outside notices. The reference repository shipped its chart check as `kubeconform` — an accurate name for the tool and the wrong name for the recipe — and did not notice until the contract was read back against it. List the names against `just --list` rather than assuming.

## Policy producer inventory

Policy's actual gate is `build test lint leak-canary`. Generated loader, RPC
and product-chart artifacts moved out, so its old drift/golden/chart/language
recipes were removed. The contract names jobs consistently and also permits
omitting a recipe for an absent job; do not invent a no-op drift recipe.

| Recipe | Needs | Scope |
|---|---|---|
| `build`, `test` | Go and public modules named by manifests/locks | root Go packages, including library-template structural tests; not the extracted product's render suite |
| `lint` | declared tools | root Go/static checks, Markdown links, whitespace and tool version declarations |
| `leak-canary` | nothing but the checkout | refuses a particular that must never be published |
| `vuln` | the network, on a schedule | news about the world, not about the change under review (C10); its own `security.yaml` workflow |
| `cluster`, `cluster-verify`, `cluster-down` | container runtime | the reusable servers-only kind definition; these do not deploy the product |
| `example-canary` | network and downstream toolchain | advisory downstream Go build/test against this checkout; not in the gate |
| `landscape` | authenticated GitHub API/network | maintenance helper, not a gate prerequisite |

## Product reference inventory

The product's gate is `build test lint drift leak-canary`. Its tests may
download locked modules/chart archives and start a declared local PostgreSQL;
they need no cluster or cloud credential. `POLICY_DIR` resolves policy's Go
module at the version in go.mod, so loaders/tests read its schemas and fixtures
instead of copying an assumed sibling checkout.

| Product recipes | Needs / claim |
|---|---|
| `build`, `test`, `lint` | Go plus the declared Python/Kotlin tools and locks; these are product jobs, not policy root jobs |
| `schemas`, `protos`, `chart-schemas`, `drift` | regenerate product artifacts and fail on a diff; `drift` also resolves chart dependencies |
| `chart-locks`, `chart-deps` | update locks deliberately versus resolve committed locks; OCI service-lib and cnpg-database archives are not committed |
| `golden` | reviewed render changes for application, infra and service-example suites |
| `ts`, `charts` | separate registry-build and API-schema lanes |
| `shadow-generate`, `generate`, `shadow-generated`, `shadow-diff` | networked Pkl shadow; `generate` is a real recipe because discovery via `just --summary` excludes aliases |
| `snapshot` | build/push product images and package charts as a release does; credentialed integration, not a fork PR check |
| `kubeconfig`, `e2e-install`, `e2e-test`, `e2e-uninstall`, `e2e` | protected integration using platform inputs and job identity; `e2e` preserves status and uninstalls after failure |

The old policy `example-snapshot`, `example-fixture`, `example-install`,
`example-smoke`, `example-prober` and `cluster-all` are not current recipes in
either inventory. Do not mechanically move their old caller strings.

## Downstream canary: exact coverage

`just example-canary` invokes policy's `hack/example-canary.sh`. It clones the
public product when needed, copies tracked files into temporary work, supplies
a go.work using the product copy and this policy checkout, and runs
`go build ./... && go test ./...` under the product's devbox environment.
`URL_SHORTENER_DIR` chooses an existing checkout; `URL_SHORTENER_REF` selects
the branch/ref when cloning. An existing checkout is not silently fetched or
reset to that ref. Neither the original checkout nor its lock is rewritten.

By default product charts retain their **released** service-lib dependency.
`URL_SHORTENER_LOCAL_LIB=1` additionally rewrites dependencies in the temporary
copy to policy's local library and drops the temporary locks. It is opt-in
because tests may intentionally assert the released declaration. Default
canary success proves neither this tree's changed library templates nor the
whole `just check`/non-Go loader/integration suite.

## How the gate is wired into CI

- **CI runs recipes by name**, never by re-implementing them in a workflow — a check that passes locally and fails in CI is a bug in a recipe rather than a difference nobody can reproduce. The reference calls the shared `check.yaml` workflow once per lane with a recipe list; cluster lanes call `devbox run -- just <recipe>`.
- **One fan-in job is the stable required context.** It depends on required jobs, not advisory ones. Policy deliberately excludes `example-canary`; requiring it would deadlock changes that must break a downstream pin until adoption. The product's credentialed integration is outside its PR fan-in too. Preserve stable context names.
- **`vuln` is not among the recipes CI requires**, and `just check` does not reach it: the conformance check follows the Justfile's dependency graph — `check` depending on `vuln` directly, through another recipe, or via a body line running `just vuln` are all refusals.
- **Distinguish resolution from infrastructure.** Repository contract §2 explicitly allows downloads named by manifests and locks. The conformance guide's older “network off” wording is stricter than that contract; do not use it to exclude locked OCI dependencies from `check`. Credentials, containers and clusters still belong in separate lanes.
- **Coverage has a boundary.** Policy supplies servers-only `hack/kind/` for consumers choosing that tier. The cited product now uses protected shared-cluster integration; do not describe the old local fixture lane as its current workflow. Unit/render tests and advisory canaries do not prove live identity or policy enforcement.

## Justfile mechanics the reference holds itself to

- `[doc("…")]` on every recipe — `just --list` is the interface; comments say WHY, and what a recipe NEEDS.
- Recipes run from the repo root (`cd dir &&` into subdirectories); multi-step recipes use a shebang with `set -euo pipefail`.
- Handle expected nonzero results explicitly (`if`, `!`, or an accumulated failure flag). The reference's aggregate lint recipe uses `set -uo pipefail` and `fail`; do not prescribe disabling `-e` for every grep — a grep used as an `if` condition already has the intended semantics.
- Parameters or env vars use neutral defaults (`${NS:-shortener}`); never a hardcoded deployment environment, never a particular.
- **One `devbox run` at a time.** Every invocation generates and executes its own copy of `.devbox/gen/scripts/.cmd.sh`; two started together can each be mid-(over)write of that shared file while the other is mid-exec ("text file busy", measured in CI). Background the `just` recipe inside one script instead.
