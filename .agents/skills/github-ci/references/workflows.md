# Workflow Skeletons (Truvity policy, copy-sheet)

Both callers use **ci-workflows v5.5.0**, `f0f75f715a4feef8ab1e4e6222e84708bb32f6cd`;
this file owns that pin.
These are role-specific skeletons: replace every placeholder with that
dependency's verified SHA, actual artifact/recipe names and the default branch.
Private consumers select the supported private workflow/tier inputs; do not
invent authentication inputs from a public example.

The pinned shared workflows declare the input names used below. `ci-actions`
remains behind them; consumers do not add direct action pins just to follow its
implementation update. The shared-workflow SHA above applies only to those
workflow calls: checkout/devbox-install actions need their own verified SHAs.

## C18: runner-agnostic YAML

The component contract now has C1-C18. Shared `policy-conformance` mechanically
checks **C1-C13 and C18**; C14-C17 remain the product's own tests. C18 scans
tracked workflow/action YAML, excluding comments and test/fixture/golden/vendor
directories, and reports `C18 <file>:<line>`. A finding is an **error** and fails
strict mode, not an informational warning.

| Forbidden form | Portable replacement |
|---|---|
| an in-cluster service address, with or without a cluster-domain suffix | an endpoint input supplied through the estate's organization variables |
| an expression branching on `runner.environment` | probe the capability actually needed and use the same behavior across runner pools |
| static AWS credentials in env, scripts, same-named secrets or `aws-access-key-id`-family inputs | the runner's ambient workload identity or the job's OIDC token |

The credential names are `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and
`AWS_SESSION_TOKEN`. Explicitly clearing one (`AWS_ACCESS_KEY_ID: ""`) is not a
credential. The built-in, permission-scoped GitHub publishing token in the
package example is not a static AWS credential either.

For **existing code with a reviewed migration exception**, record the reason
and narrow paths in `.github/policy-conformance.yaml`, for example:

```yaml
exempt:
  C18:
    reason: >-
      The existing legacy action still requires runner-local endpoint resolution;
      migrate it to an organization-variable endpoint input.
    paths:
      - .github/actions/legacy/action.yaml
```

This is an exception example, not a default to add to every repository. Paths
must name the reviewed scope; repair other findings. Do not turn strict mode
off or skip the whole rule to hide a remaining finding. The exception remains
visible as an exemption, not proof of conformance. Sources: [C18](https://github.com/truvity/policy/blob/master/docs/contracts/component.md#c18-workflow-and-action-yaml-is-runner-agnostic)
and the contract's exemption rules.

## Policy producer CI

This variant runs in policy, where the downstream canary script exists. The
minimal canary below uses its anonymous clone path; the actual caller performs
two explicit checkouts. It is advisory and not in the required fan-in.

```yaml
name: CI (producer)
on:
  pull_request:
    branches: [master]
  push:
    branches: [master]
permissions:
  contents: read
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
jobs:
  recipes:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      policy-conformance: true
      policy-conformance-strict: true
      recipes: '["build", "test", "lint", "leak-canary"]'
      cache-paths: |
        ~/.cache/go-build
        ~/go/pkg/mod
      cache-key-files: |
        **/go.sum
        devbox.lock
  example-canary:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@<sha> # v7.0.1
        with:
          persist-credentials: false
      - uses: jetify-com/devbox-install-action@<sha> # v0.15.0
        with:
          enable-cache: "true"
      - run: devbox run -- just example-canary
  check:
    needs: [recipes]
    if: always()
    runs-on: ubuntu-latest
    steps:
      - run: '[ "${{ needs.recipes.result }}" = "success" ]'
```

The canary runs downstream Go build/test via go.work, not every product lane.
`URL_SHORTENER_LOCAL_LIB=1` additionally swaps the temporary copy's chart
dependency to this checkout; default success does not prove changed library
templates. The producer's [canary script](https://github.com/truvity/policy/blob/c24f403/hack/example-canary.sh)
defines that coverage; the sibling devbox-just skill carries the full recipe inventory.

## Product PR CI

The product owns generation and multi-language jobs. Its fork PR gate has a
read-only token, no id-token grant and no cluster access. Keep only applicable
jobs and list all required ones in the fan-in; add the shadow lane if used.

```yaml
name: CI (product)
on:
  pull_request:
    branches: [master]
  push:
    branches: [master]
permissions:
  contents: read
concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true
jobs:
  recipes:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # v5.5.0
    with:
      policy-conformance: true
      policy-conformance-strict: true
      recipes: '["build", "test", "lint", "drift", "leak-canary"]'
      cache-paths: |
        ~/.cache/go-build
        ~/go/pkg/mod
        ~/.gradle/caches
      cache-key-files: |
        **/go.sum
        **/build.gradle.kts
        **/settings.gradle.kts
        devbox.lock
  ts:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      recipes: '["ts"]'

  charts:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      recipes: '["charts"]'

  shadow:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # v5.5.0
    with:
      recipes: '["shadow-generated", "shadow-diff"]'
  check:
    needs: [recipes, ts, charts, shadow]
    if: always()
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - run: |
          [ "${{ needs.recipes.result }}" = "success" ] &&
          [ "${{ needs.ts.result }}" = "success" ] &&
          [ "${{ needs.charts.result }}" = "success" ] &&
          [ "${{ needs.shadow.result }}" = "success" ]
```

Hosted caches are declarative `cache-paths`/`cache-key-files` inputs; without a
cache bucket the shared setup uses GitHub's cache. Do not put a cluster-local
cache address or runner-kind branch into workflow YAML (C18).

## Protected product integration

The [complete product workflow][integration]
runs on trusted default-branch pushes, release tags and manual dispatch, never
fork PRs or PR-target/workflow-run/comment triggers. It uses hosted runners,
scoped package/OIDC permissions and platform repository variables, prepares
registry/build tooling and kubeconfig, then runs `snapshot`, `e2e-install`,
`e2e-test` and `e2e-uninstall` through devbox. Cleanup is an `always()` step;
integration runs queue with cancellation disabled.

An editable `if:` is courtesy, not authorization. Receiver-side OIDC trust
must constrain the accepted repository, workflow, event and ref. Persistent
production Applications still belong to GitOps; this installs ephemeral tests.
The shared `integration.yaml` kind tier remains available to consumers that
choose it, but the old policy example-lane strings are not the current product
workflow. Do not copy the root producer's advisory job into its required gate.

## security.yaml — vuln, scheduled, never the gate

```yaml
name: Security
on:
  push:
    branches: [master]
  pull_request:
    branches: [master]
  schedule:
    - cron: "0 6 * * *" # daily
permissions:
  contents: read
jobs:
  govulncheck:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      recipes: '["vuln"]'
```

## Policy producer release

The Go module and library are owned here; no application images or language
packages are published. This is the producer variant, not a product template.

```yaml
name: Release (producer)
on:
  push:
    tags:
      - "v*"
permissions:
  contents: read
# Never cancel a running publish — a version that failed halfway is burned.
concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false
jobs:
  release:
    # The caller grants: a reusable workflow can narrow, never widen.
    permissions:
      contents: write
      packages: write
    uses: truvity/ci-workflows/.github/workflows/release-public.yaml@<sha> # vX.Y.Z
    with:
      charts: '["charts/service-lib"]'
```

## Product release

An image-bearing product builds images before packaging its own charts. Do not
add service-lib to its artifact list. The reference product's loaders are
internal and not published; optional package-publishing patterns belong in
`release.md`, not in either current caller.

```yaml
name: Release (product)
on:
  push:
    tags: ["v*"]
permissions:
  contents: read
concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false
jobs:
  release:
    permissions:
      contents: write
      packages: write
    uses: truvity/ci-workflows/.github/workflows/release-public.yaml@<sha> # v5.5.0
    with:
      chart-images: goreleaser
      image-repo: ghcr.io/<owner>/<repo>
      require-image-digests: true
      charts: '["<product>", "<product>-infra"]'
      chart-root: charts

```

`charts/service-lib` appears in policy's own release because policy owns it.
A consumer resolves it through a dependency lock and publishes only its own
charts. Similarly `images.identity-check` is a pinned `truvity/k8s` image;
leave its tag/digest intact rather than rebuilding it as a product artifact.
If enabling the chart's optional `identityCheck.expectAccount`, first move that
image to the verified v0.34.0+ pin: the release's own tag does not imply that an
external binary understands a newly rendered command flag.

## auto-release.yaml — patches only, doubly guarded

```yaml
name: Auto Release
on:
  schedule:
    - cron: "41 5 * * 1"
  workflow_dispatch:
  push:
    branches: [master]
permissions:
  contents: read
  id-token: write # the token exchange, in the shared workflow's job
jobs:
  tag:
    if: vars.AUTO_RELEASE == 'true' && vars.ACCESS_ROSTER_ISSUER != ''
    uses: truvity/ci-workflows/.github/workflows/auto-release.yaml@<sha> # vX.Y.Z
    with:
      token-source: access-roster
      access-roster-issuer: ${{ vars.ACCESS_ROSTER_ISSUER }}
      github-app: <the organisation's tagging app>
```

Never a minor, a major or a first release — those are a person's tag. A push whose merged pull request carries the `security` label (or is a `fix:`) releases immediately; ordinary bumps wait for the weekly schedule.
This is an enrolled-producer pattern: the cited product cuts tags by hand and
does not arm automatic patch releases.

Configure issuer trust and the tagging App's tag-ruleset bypass outside this
file. The workflow prepares a changelog-heading PR when required, tags its
merged commit, and stands aside for a pending hand-cut minor/major heading.
Repositories requiring an approving review set `changelog-heading: never`
under `with` (release contract §3); do not ask the release App to approve its
own PR. A non-breaking-fix classification still needs review.

## Policy Renovate and product additions

```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["github>truvity/ci-workflows"],
  "description": "devbox.json and devbox.lock are ignored because the devbox parity job owns them: it moves the devbox packages, the lockfile and the go directive together in one pull request.",
  "ignorePaths": ["devbox.json", "devbox.lock"],
  "packageRules": [
    {
      "description": "The go directive is owned by the devbox parity job: it aligns go.mod with the Go devbox ships, in the same pull request as devbox.lock.",
      "matchManagers": ["gomod"],
      "matchDepNames": ["go"],
      "enabled": false
    }
  ]
}
```

Every override carries a `description` (C7). The ignores above assume the
fleet devbox parity job is enabled for the repository: verify that enrollment
before disabling Renovate for those files. Otherwise the manifest and Go/Yarn
followers stop receiving updates. Base-image regex managers are needed only
if the repository has the corresponding GoReleaser/ko fields.

A Node product additionally coordinates Yarn with its own devbox updater. A
service-lib consumer groups `github.com/truvity/policy`, `service-lib` and
`ghcr.io/truvity/charts/service-lib` into one non-automerged bump, including
Chart.lock. The cited product's render tests enforce equal module/chart
versions. These are product rules, not extra entries in policy's current file.

## lefthook.yml

```yaml
pre-commit:
  jobs:
    - name: leak canary
      run: just leak-canary
      # A machine without the toolchain (an automation that only edits
      # dependency pins) would fail every commit on a missing command.
      skip:
        - run: "! command -v just"
```

[integration]: https://github.com/truvity/url-shortener/blob/76770607aa08baa58331668c228dacd62a13a89a/.github/workflows/integration.yaml
