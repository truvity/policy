# Releases and images

**The rule.** One tag stamps every artifact of a release. A runtime image
contains no build step, and one job builds every platform.
[release.md](../contracts/release.md) and
[service.md §7](../contracts/service.md).

**Why.** A version that can disagree with the binary is worse than no version
at all, because it is trusted. And a build step inside an image means the
image is built differently depending on when it was built, which is the
property a release exists to remove.

## Where to look

| Concern | Where |
|---|---|
| how images are built | [`examples/url-shortener/.ko.yaml`](../../examples/url-shortener/.ko.yaml) |
| the version the binary reports | `runtime.Version` in [`internal/runtime/runtime.go`](../../examples/url-shortener/internal/runtime/runtime.go) |
| how the chart pins an image | the image helper in `charts/url-shortener/templates/_helpers.tpl` |
| what the toolchain pins | [`canon/toolchain.md`](../canon/toolchain.md), [`canon/build-tools.md`](../canon/build-tools.md) |

## Per language

| Language | Image is | Built by |
|---|---|---|
| Go | the compiled binary on a minimal base | a tool that cross-compiles and pushes, with no daemon and no emulation |
| TypeScript | a copy of the built bundle onto a runtime base | the bundler, then a copy |
| Kotlin | a copy of the built artifact onto a JRE base | the build tool, then a copy |
| Python | a copy of the installed environment onto a minimal base | the package manager, then a copy |

The pattern is the same in all four: **something else builds, the image
copies.** No compiler, no package manager and no shell in the runtime image
where the base can avoid one.

## The version

Read from the build metadata the toolchain already embeds — not passed in
configuration, and not a constant someone edits. A test asserts it is not the
zero value, because the common failure is a binary built outside the release
path reporting nothing at all.

## Multi-platform without emulation

Every image is built for more than one processor architecture, and **one job
builds them all**. Emulating one architecture on another is slow enough that
people stop running the build, and the tool for each language cross-compiles
natively instead.

## Traps

**A digest, not a tag, in a deployment.** A tag can move under a running
service, and an image that can move is one nobody can roll back to. The chart
takes either and refuses both being empty.

**A failed release burns the version.** Where the tag is pushed before the
artifacts are built, a build failure leaves a tag with no release. Cut the
next patch; do not move the tag, because anything that already fetched it
will never fetch it again.

**Archives must be uniform across platforms** where a release tool builds
them, and the refusal arrives late — after every cross-compile has succeeded.
Assert it in the gate rather than discovering it in a tagged run.
