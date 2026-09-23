# Build tools

One build system, one package manager and one container build method per
ecosystem. The list is short on purpose: every entry is something a
contributor must have working before they can change one line.

| Ecosystem | Build system | Package manager | Container image |
|---|---|---|---|
| Go | `just` recipes over the Go toolchain | Go modules | `ko`, from the compiled binary |
| Node and TypeScript | `just` recipes over the package manager's scripts | Yarn 4 | a `COPY` of the built bundle onto a runtime base |
| Kotlin and the JVM | Gradle, wrapped in `just` recipes | Gradle | a `COPY` of the built artifact onto a JRE base |
| Helm charts | `just` recipes over `helm` | — | — |

## Why `just` in front of everything

Not because the underlying tools are inadequate, but because CI and a
contributor must run **the same thing**. A recipe is that thing: CI runs
recipes by name and nothing else, so a check that passes locally and fails in
CI is a bug in a recipe rather than a difference in how they were invoked.

It also means the gate is nameable. `just check` is the whole contract of a
repository's CI, and it must need nothing but the checkout: no network, no
containers, no cluster. Anything heavier is its own recipe, run as its own
job, and named so that nobody is surprised by what it needs.

## Container images: the build never happens in the image

Every row above builds the artifact **in CI** and copies it into the image.
No image has a `RUN` line, a package manager or a compiler in it.

That is the rule in [service.md](../contracts/service.md), and the reason is
worth repeating here because this is where people reach for the other shape:
an image with nothing executable at build time can be assembled for any
architecture without emulation. One job builds every platform. The moment a
`RUN` appears, a cross-platform build needs an emulator, the emulator is slow
enough to need a fleet of builders, and the fleet needs maintenance — all of
it downstream of one line in a Dockerfile.

`ko` is the Go row because it cross-compiles and assembles the image itself,
with no daemon at all.

## What is deliberately absent

- **A monorepo build graph.** Repositories here are one product each; a task
  graph across projects is a tool for a problem this shape does not have.
- **A second package manager per ecosystem.** The second one always arrives
  for a single dependency and stays forever.
- **Anything that generates the build.** [0003](../decisions/0003-schemas-not-generators.md).
