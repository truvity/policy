# Workflow Skeletons (Truvity policy, copy-sheet)

The four workflows plus `renovate.json` and `lefthook.yml`, minimised and parameterised — the shapes every migrating repository starts from. Skeletons only: the reference implementations carry the reasoning in comments ([`.github/workflows/`](https://github.com/truvity/policy/tree/master/.github/workflows), needs network). Replace `<repo>`, the chart lists and the SHA pins (always `@<commit-sha> # vX.Y.Z`, never a tag alone).

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
  # The gate. policy-conformance judges the component contract (C1-C17),
  # shows it in the job summary, and fails this job when a rule fails.
  recipes:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      policy-conformance: true
      policy-conformance-strict: true
      recipes: '["build", "test", "lint", "drift", "leak-canary"]'
      # `vuln` is never here: security.yaml owns it (C10)

  # A recipe that needs more than the checkout — here a registry — is its
  # own job, never part of the gate.
  ts:
    uses: truvity/ci-workflows/.github/workflows/check.yaml@<sha> # vX.Y.Z
    with:
      recipes: '["ts"]'

  # The cluster tier: the workflow stands up the box itself (its own kind
  # tier), fetching THIS repository's hack/kind at a pinned release tag, so
  # the recipes a lane runs are exactly what a laptop runs. The caller
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
            "test": "devbox run -- just example-smoke"
          }
        ]

  # Stable name for branch protection: matrix/reusable jobs report under
  # composed names, so this fan-in is the ONE required context. It gates
  # merges: everything, the cluster lane included, must be green.
  check:
    needs: [recipes, ts, cluster]
    if: always()
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - run: |
          [ "${{ needs.recipes.result }}" = "success" ] &&
          [ "${{ needs.ts.result }}" = "success" ] &&
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
      image-repo: ghcr.io/<owner>/<repo>/<product>
      require-image-digests: true
      charts: '["<product>", "<product>-infra", "charts/service-lib"]'
      chart-root: examples/<product>/charts

  # What goreleaser does not build (npm package, wheel, …) attaches to the
  # Release here: a dependency, not a preference. Stamp the tag's version
  # over the committed placeholders with an edit that ASSERTS it matched —
  # a package published at 0.0.0 cannot be taken back.
  packages:
    needs: [release]
    runs-on: ubuntu-latest
    permissions:
      contents: write
      packages: write
    steps:
      - uses: actions/checkout@<sha> # vX
      - uses: jetify-com/devbox-install-action@<sha> # vX
        with:
          enable-cache: "true"
      - run: devbox run -- bash hack/publish.sh "$VERSION" "$TAG"
```

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

Every override carries a `description` (C7): an override with no reason is one nobody dares remove.

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
