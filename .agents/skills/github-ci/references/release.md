# The Release: Digests, Versions, Adoption (Truvity policy contract)

What a release publishes, in what order, and what a consumer needs from it. Condensed and self-contained; normative source (needs network): [release contract](https://github.com/truvity/policy/blob/master/docs/contracts/release.md).

## Two producers, two independent releases

| Repository | Owns and releases | Does not release |
|---|---|---|
| policy | contracts, schemas, Go packages and service-lib | the product's images/charts, npm package or Python wheel |
| url-shortener | product images and application/infra charts | service-lib; its TS/Python/Kotlin loaders are internal, not published packages |

The split is ADR 0014. Artifacts published from policy up to v1.54.2 remain
available, but that history is not the current publishing plan. One tag stamps
what **that producer** owns, not everything its code depends on. Policy's
library-only release needs no image build or image-digest baking inputs.

## The build is the release

Artifacts are built once, in CI, from the tagged commit, and published from that build. Nothing is built on a laptop and pushed. A release that can be produced by one person's machine is a release nobody else can reproduce.

For an image-bearing product, images are referenced **by digest** in everything
the release publishes, so a tag cannot move what a consumer installs.

That is an ORDERING requirement before it is a preference. A digest exists only once the image is built, so anything that embeds one — a chart, a manifest, a lockfile — is produced **after** the images and **from** them. A release that publishes a chart first publishes one with the field empty.

The ordering is made impossible to get wrong rather than merely written down: **one job** builds and pushes the images and then packages the charts from the file that build produced (`chart-images: goreleaser`, `require-image-digests: true`). And the release refuses outright to package a chart whose image entries are not all digest-pinned.

**Two tools, and nothing of ours between them.** The build tool builds and pushes; the registry tool (`helmctl`, from the fleet's ocictl) reads what was pushed, bakes the digests into the charts and pushes those. A script in this path is a third thing to keep correct. Each language's own compiler still runs as a build hook; that is a build, not a release step: nothing there publishes anything.

**ONE destination per repository.** Public artifacts to a public registry, private to a private one, never both. What varies between a local loop, a CI loop and a release is *where* — one environment variable — and never *what is built*.

## The tag is the sole version authority

Committed chart versions are placeholders (`0.0.0`, the component contract's C1) and package manifests carry one that never moves; the release stamps the tag's version over them. A version committed to a manifest is a second place to forget, and two people can disagree about it. The stamping edit ASSERTS it matched — a substitution that matches nothing succeeds silently, and a package published at the placeholder version cannot be taken back.

This applies to artifacts the repository **owns**. An external dependency
keeps its upstream version and lock: the cited product depends on cnpg-database
**2.31.0** and service-lib **1.54.2**, with policy's Go module **v1.54.2**.
Its identity-check image is another repository's verified tag/digest. Do not
stamp the product's version onto either, rebuild that image, or republish
service-lib merely because the product depends on it. Mirror charts have the
explicit upstream-version exception described by C1/release §1.

The policy module and OCI library are one upstream release and move together
in a grouped, reviewed bump (including Chart.lock); the product's own version
is independent. Embedded `charts.Library` comes from that Go module version,
while rendering resolves the declared archive. An equality test prevents a
product testing one library version and publishing another. The remaining
C15/guide `file://` wording is a recorded contract-source conflict, not a reason
to silently vend a second library or claim an unacknowledged exception.

Changing how a pinned binary is invoked must respect that binary's release:
the optional `-expect-account` flag needs checker v0.34.0 or later. Update the
verified image pin before enabling `identityCheck.expectAccount`; otherwise
the hook fails every retry on an unknown flag. A product release version or
unchanged delivery-interface number is not proof of external binary support.

## Adoption is proved, not asserted

A consumer adopts a release only when the output it produces is **byte-identical to what runs**, or differs exactly by the change the release announced. That render, or that plan, is the evidence attached to the pull request that moves the pin — in the CONSUMER's repository (the platform control plane's), which is where pins live and where its golden renders are reviewed. What that demands of the product repository is a release whose render CAN be byte-identical: complete values, digest-pinned images, no values a consumer must guess.

Moving from hand-written objects to a chart is one change whose diff is empty. Tightening a default is a separate release, adopted separately, with its diff visible. This is the rule that makes the rest worth anything: without it, "no rendered output changes" in a patch is a claim rather than a fact.

## A published artifact is tested AS PUBLISHED

The artifact a consumer installs is not the one in the tree, and the difference is invisible from inside the repository. The reference repository published a chart that could not render a single Deployment: its image values were empty, because the release built the images in a later job than the one that packaged the chart. Every test passed — each supplied images of its own, as a test naturally does — and the release was green. It was found by installing the published chart onto a cluster, which is the only place the two versions of the artifact differ.

For every published artifact test the as-published case: packaged chart, package
imported by its own name, binary with no flags, without test-only image overrides.
The product's protected integration builds its snapshot using release machinery
and installs packaged charts, not source directories. Policy's advisory Go
canary is different evidence: no cluster install and no chart-library override
by default. Do not count it as the product's as-published integration result.

## What the tag publishes, and who moves it next

One tag ships everything the product releases — the Go module needs only the tag, images and charts go to the registry, packages to their own destinations — and the platform consumes from there, moving its pins in its own repository with the render as evidence. Product CI can apply/install ephemeral copies to test that artifact; it does not manage persistent production Applications or deployment values.

## Optional language-package publication (other consumers only)

Neither current reference caller publishes npm/wheel packages. For an adopting
repository that does, this is a **job fragment** added after its release job.
The scripts are owned by that repository and must exist, assert matched
placeholders and provide a dry-run; policy no longer supplies them. Registry
configuration must read the scoped token from the environment, never store it.

```yaml
packages:
  needs: [release]
  runs-on: ubuntu-latest
  timeout-minutes: 20
  permissions:
    contents: write
    packages: write
  steps:
    - uses: actions/checkout@<sha> # this action's own verified pin
    - uses: jetify-com/devbox-install-action@<sha> # this action's own pin
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

## Automatic patches and honest gaps

First/minor/major tags are hand-cut after their headings merge. For automatic
patches, nonempty `Unreleased` entries are headed by a PR before tagging;
dependency-only patches follow C5's exception, and an existing heading is not
rewritten. A pending hand-cut heading takes precedence. With mandatory
approving review use `changelog-heading: never`, as release §3 requires.

The contract limits automatic patches to non-breaking fixes for a stabilizing
product, but its conformance table says the workflow does not yet inspect the
product's state. Review this; a scheduled run can otherwise include a feature
merged after the last tag. The template is not proof that every commit in the
release range qualifies for a patch.
