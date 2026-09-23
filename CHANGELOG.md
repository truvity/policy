# Changelog

What changed for someone consuming this repository, newest first. A version
missing from this file changed nothing a consumer can see — a dependency bump
and nothing else — and its GitHub Release lists the commits.

## v0.1.0

Not yet released. The first version will carry:

- **The repository skeleton.** Devbox toolchain, the `just check` gate, the
  leak canary on every commit and in CI, and hosted-runner-only CI.
- **The service contract and the configuration contract**, with the three
  decisions they rest on: hand-wired composition roots, a configuration file
  with secrets in the environment, and a schema with a hand-written type
  rather than a code generator.
- **The canon**: the Go library list, the Node and TypeScript list, a Kotlin
  stub written from the first JVM service rather than before it, the pinned
  toolchain versions with the reason each is a pin, and the build tools.
- **The repository and release contracts**: one product per repository, a
  gate that needs nothing but the checkout, fixed documentation paths, and
  what a public repository is held to on top; one tag stamping every
  artifact, what a version means read from the consumer's side, and the rule
  that adoption is proved by a byte-identical render rather than asserted.
- **The configuration schemas and the Go loader.** Seven shared fragments and
  the service envelope, embedded in the module so that validation needs no
  network; `config.Load`, which validates before it decodes and names the key
  that failed rather than the file; `config.Secret`, which reads the variable
  a configuration names and never the value it carries; and `conformance`,
  which holds a configuration type and a rendered chart to the same schema.
- **The TypeScript loader**, `@truvity/policy`: the same three calls as the Go
  one, carrying the same schemas, tested against the same fixtures and
  wording its refusals the same way, so that a misconfiguration reads
  identically whichever runtime refused it.
