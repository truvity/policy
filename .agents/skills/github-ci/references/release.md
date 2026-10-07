# The Release: Digests, Versions, Adoption (Truvity policy contract)

What a release publishes, in what order, and what a consumer needs from it. Condensed and self-contained; normative source (needs network): [release contract](https://github.com/truvity/policy/blob/master/docs/contracts/release.md).

## The build is the release

Artifacts are built once, in CI, from the tagged commit, and published from that build. Nothing is built on a laptop and pushed. A release that can be produced by one person's machine is a release nobody else can reproduce.

Images are referenced **by digest** in everything the release publishes, so what a consumer installs cannot be moved under them by a tag being repointed.

That is an ORDERING requirement before it is a preference. A digest exists only once the image is built, so anything that embeds one — a chart, a manifest, a lockfile — is produced **after** the images and **from** them. A release that publishes a chart first publishes one with the field empty.

The ordering is made impossible to get wrong rather than merely written down: **one job** builds and pushes the images and then packages the charts from the file that build produced (`chart-images: goreleaser`, `require-image-digests: true`). And the release refuses outright to package a chart whose image entries are not all digest-pinned.

**Two tools, and nothing of ours between them.** The build tool builds and pushes; the registry tool (`helmctl`, from the fleet's ocictl) reads what was pushed, bakes the digests into the charts and pushes those. A script in this path is a third thing to keep correct. Each language's own compiler still runs as a build hook; that is a build, not a release step: nothing there publishes anything.

**ONE destination per repository.** Public artifacts to a public registry, private to a private one, never both. What varies between a local loop, a CI loop and a release is *where* — one environment variable — and never *what is built*.

## The tag is the sole version authority

Committed chart versions are placeholders (`0.0.0`, the component contract's C1) and package manifests carry one that never moves; the release stamps the tag's version over them. A version committed to a manifest is a second place to forget, and two people can disagree about it. The stamping edit ASSERTS it matched — a substitution that matches nothing succeeds silently, and a package published at the placeholder version cannot be taken back.

## Adoption is proved, not asserted

A consumer adopts a release only when the output it produces is **byte-identical to what runs**, or differs exactly by the change the release announced. That render, or that plan, is the evidence attached to the pull request that moves the pin — in the CONSUMER's repository (the platform control plane's), which is where pins live and where its golden renders are reviewed. What that demands of the product repository is a release whose render CAN be byte-identical: complete values, digest-pinned images, no values a consumer must guess.

Moving from hand-written objects to a chart is one change whose diff is empty. Tightening a default is a separate release, adopted separately, with its diff visible. This is the rule that makes the rest worth anything: without it, "no rendered output changes" in a patch is a claim rather than a fact.

## A published artifact is tested AS PUBLISHED

The artifact a consumer installs is not the one in the tree, and the difference is invisible from inside the repository. The reference repository published a chart that could not render a single Deployment: its image values were empty, because the release built the images in a later job than the one that packaged the chart. Every test passed — each supplied images of its own, as a test naturally does — and the release was green. It was found by installing the published chart onto a cluster, which is the only place the two versions of the artifact differ.

So, for every artifact a repository publishes: there is a test for the as-published case — the chart with no values supplied, the package imported by nothing but its own name, the binary run with no flags. That case is the only one every consumer has. In the reference repository the cluster lane installs the packaged `.tgz` (never the chart's source directory), built by the release's own configuration for one architecture into the box's registry.

## What the tag publishes, and who moves it next

One tag ships everything the product releases — the Go module needs only the tag, images and charts go to the registry, packages to their own destinations — and the platform consumes from there, moving its pins in its own repository with the render as evidence. The product repository's CI never applies anything to a cluster.
