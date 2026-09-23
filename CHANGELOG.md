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
- **The import ban**, as a block a repository copies into its own lint
  configuration: no dependency-injection container, no configuration-mapping
  library, and the libraries the canon retired. Each entry names what to use
  instead, because a lint error that only says "no" gets suppressed rather
  than fixed.
- **A conformance guide**: what CI checks for you, what a reviewer checks,
  and the two rules that are checked by eye.
- **The local cluster**: a recipe that stands up Kubernetes with the same
  operators a deployment carries, a check that asks whether each thing is
  usable rather than merely installed, and a smoke test that proves an
  operator ACTS — a database becomes a database, a stream becomes a stream,
  a bucket becomes a bucket. About two minutes from nothing.
- **The worked example**, first three components: a migration job, the
  redirect service and the click counter, hand-wired against the contracts
  with no framework behind them. Each binary has one configuration file, one
  schema, and a test that the two describe the same fields.
- **The `nats` fragment now describes a CONNECTION only**, and a new
  `nats-consumer` fragment describes what a consumer binds to. The first real
  consumer is what showed that a publisher carrying a `consumer` field it
  never reads is a field somebody will eventually set.
