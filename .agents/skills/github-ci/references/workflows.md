# Workflow Skeletons (Truvity policy, copy-sheet)

Reviewed on 2026-10-21 against the policy caller's **ci-workflows v5.0.0** integration (`d4350d573f67de8f7a145a4de94c392c6b43fd8c`). These are parameterized public-product skeletons, not literal files to paste unchanged. Replace **every** `<sha>` with a verified full commit SHA (keep its version in a comment), `<owner>`/`<repo>`/`<product>` and the default branch with the adopting repository's facts. Keep only jobs/recipes/artifacts it actually has and update the fan-in accordingly. Private callers select the shared private release/tier contract; do not infer its authentication inputs from the public example. Sources: [reference workflows](https://github.com/truvity/policy/tree/master/.github/workflows).

The four pinned v5 workflows declare the input names used below. `ci-actions`
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

## ci.yaml — the merge gate

```yaml
name: CI
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
  # Shared checks cover C1-C13 and C18; test/drift must prove C14-C17.
  # C18 findings fail strict unless covered by a reviewed scoped exemption.
  recipes:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      policy-conformance: true
      policy-conformance-strict: true
      recipes: '["build", "test", "lint", "drift", "leak-canary"]'
      # `vuln` is never here: security.yaml owns it (C10)

  # This reference keeps its package-registry build as a separate lane.
  # Locked public dependency resolution is allowed in gate recipes.
  ts:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      recipes: '["ts"]'

  charts:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      recipes: '["charts"]'

  # The cluster tier: the workflow stands up the box itself (its own kind
  # tier), fetching POLICY's hack/kind at policy-version. Lane commands are
  # the caller's recipes, adapted below from the worked example. The caller
  # grants id-token: write; lanes are build → install → test, each
  # `devbox run -- just <recipe>`.
  cluster:
    uses: truvity/ci-workflows/.github/workflows/integration.yaml@<sha> # vX.Y.Z
    permissions:
      contents: read
      id-token: write
    with:
      policy-version: vX.Y.Z
      timeout-minutes: 30
      lanes: |
        [
          {
            "name": "example",
            "build": "devbox run -- just example-snapshot",
            "install": "devbox run -- just example-fixture && devbox run -- just example-install",
            "test": "devbox run -- just example-smoke && devbox run -- just example-prober"
          }
        ]

  # Stable name for branch protection: matrix/reusable jobs report under
  # composed names, so this fan-in is the ONE required context. It gates
  # merges: everything, the cluster lane included, must be green.
  check:
    needs: [recipes, ts, charts, cluster]
    if: always()
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - run: |
          [ "${{ needs.recipes.result }}" = "success" ] &&
          [ "${{ needs.ts.result }}" = "success" ] &&
          [ "${{ needs.charts.result }}" = "success" ] &&
          [ "${{ needs.cluster.result }}" = "success" ]
```

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

## release.yaml — one tag, one job, digests before charts

```yaml
name: Release
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
      # ONE job: build+push images, then package the charts from the
      # digests that build produced. require-image-digests refuses a chart
      # whose images.* entries are not all digest-pinned.
      chart-images: goreleaser
      image-repo: ghcr.io/<owner>/<repo>
      require-image-digests: true
      charts: '["<product>", "<product>-infra"]'
      chart-root: charts

  # What goreleaser does not build (npm package, wheel, …) attaches to the
  # Release here: a dependency, not a preference. Stamp the tag's version
  # over the committed placeholders with an edit that ASSERTS it matched —
  # a package published at 0.0.0 cannot be taken back.
  packages:
    needs: [release]
    runs-on: ubuntu-latest
    timeout-minutes: 20
    permissions:
      contents: write
      packages: write
    steps:
      - uses: actions/checkout@<sha> # vX
      - uses: jetify-com/devbox-install-action@<sha> # vX
        with:
          enable-cache: "true"
      - name: Stamp the tag version
        env:
          REF_NAME: ${{ github.ref_name }}
        run: |
          set -euo pipefail
          version="${REF_NAME#v}"
          devbox run -- python3 hack/stamp-version.py "$version"
          echo "VERSION=$version" >> "$GITHUB_ENV"
      - name: Publish the additional packages
        env:
          TAG: ${{ github.ref_name }}
          GITHUB_PACKAGES_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: devbox run -- bash hack/publish.sh "$VERSION" "$TAG"
```

The `packages` job is optional: remove it for a repository publishing only
charts/images/modules. `hack/stamp-version.py` and `hack/publish.sh` are
repository-owned scripts, not files supplied by the reusable workflow.
Implement and verify them before retaining this job: stamping fails on an
unmatched placeholder, and publishing has a dry-run before irreversible
operations. The built-in token needs the shown `contents`/`packages` grants;
never copy an estate credential into a public caller.

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

Configure issuer trust and the tagging App's tag-ruleset bypass outside this
file. The workflow prepares a changelog-heading PR when required, tags its
merged commit, and stands aside for a pending hand-cut minor/major heading.
Repositories requiring an approving review set `changelog-heading: never`
under `with` (release contract §3); do not ask the release App to approve its
own PR. A non-breaking-fix classification still needs review.

## renovate.json

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
    },
    {
      "description": "Yarn's release is owned by the devbox: a packageManager bump rewrites yarn.lock to a format the devbox's yarn refuses.",
      "matchManagers": ["npm"],
      "matchDepTypes": ["packageManager"],
      "matchDepNames": ["yarn"],
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
