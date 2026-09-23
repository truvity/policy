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
- **The example's chart**, and the test that makes the configuration contract
  real: what the chart renders is validated with the schema the BINARY
  validates against at start-up, so the two cannot drift in the direction
  that matters. Five negative fixtures, one per refusal, each failing for its
  own reason.
- **`AGENTS.md`**, the entry point for a reader with no other context:
  what to read first, the four gate checks that surprise people, the writing
  rules, and a second set of instructions for bringing another repository to
  this shape — the survey-before-you-change order, what a survey of real
  services usually finds, and what never to do. Mirrored under whatever
  filename a particular tool looks for.
- **A Python canon**, starting from nothing because there was nothing to
  inherit. Deliberately short: no web framework, no RPC row and no ORM until
  a component needs one, and one weakness written down in advance — a Python
  server cannot swap a certificate underneath a running listener, so it
  recycles its workers or takes the proxy.
- **The Kotlin canon is no longer a stub.** It inherits the JVM stack the
  adopting estate already runs rather than choosing a lighter one, because
  the alternative is two JVM stacks and the newer one wins every later
  argument. The Kubernetes shape is not inherited; it comes from the service
  contract.
- **TypeScript is two lines, split by job.** The 7.0 compiler type-checks
  several times faster and ships no programmatic API, so type checking is
  7.0 and anything that drives the compiler stays on 6.0. Emit was never the
  compiler's job anyway. The framework's build command and its
  code-generating plugins are out of the canon, with the workaround for the
  first and the reason there is none for the second.
- **Fixed recipe names** in the repository contract, so that moving between
  repositories does not mean reading a recipe file to find the linter, and
  **every toolchain entry names a version** — "latest" makes a reproducible
  build a coincidence.
- **Mutual TLS, with the identity the platform gives.** A new `tls` fragment
  and a `transport` package that does three things and no more: load the
  mounted certificate and reload it when it changes, present it as a server
  and as a client, and admit a peer by the ACCOUNT it runs as rather than by
  the address it calls from. It never fetches or mints a certificate, because
  a workload that did would be asserting an identity rather than presenting
  one it was given.

  Three modes. `off` is the default and always will be, so a chart stays
  installable by someone whose platform provides none of this. `permissive`
  serves both on two ports, so an edge migrates one side at a time. `strict`
  serves only the authenticated port.

  Eleven tests over real handshakes, including the two that matter: an
  unlisted peer is closed at the handshake with the reason on the SERVER and
  a bare refusal to the caller, and a rotated certificate is picked up
  without a restart. Both mutation-checked.
- **A guide per aspect**, each with the rule, why it is that way, a table of
  where to look per language, and the traps — the failures that look like
  something else. Configuration, identity and secrets, events, probes and
  rollout, logging and telemetry, exposure, releases and images, and testing,
  indexed by [`docs/guides/README.md`](docs/guides/README.md). The remaining
  five arrive with the components that prove them, and the index says which,
  because a guide written before the code is a description of nothing.
- **The example proves the rollout rule instead of violating it.** Two
  instances of everything routed, a disruption budget, `maxUnavailable: 0`,
  spread across machines, and ONE drain number used three times — the
  service's own timeout in its configuration file, the grace period
  Kubernetes grants, and a pre-stop delay. A new `drain` fragment carries
  the first of those, so the number the chart renders is the number the
  process uses.
- **The example names the account it runs as**, and names two: the migration
  creates tables and grants rights, the services read and write rows, and
  one account for both puts the migration's rights on the request path. The
  chart still grants nothing — it names accounts and leaves their
  annotations open, so a platform binds them by whichever mechanism that
  cluster uses.
- **The route's rule is named**, because a policy attaches to a rule by name
  and a policy whose target names no rule is not refused — it is simply not
  attached, and the route keeps serving without it.
- **Six new chart tests**, each mutation-checked: a rollout with no gap, a
  grace period that outlasts the drain, every route rule named, every
  workload naming an account, the migration and the services on different
  accounts, and — the third instance of one trap — everything a pre-install
  hook references being a hook itself. The first version of that last test
  passed while the install hung, because the account and the job share a
  name and it keyed on the name alone.
- **A platform contract**, `docs/contracts/platform.md`: the other side of
  the seam. What a service asks of whatever runs it — names never values, an
  account and its annotations rather than a grant mechanism, secrets as
  Kubernetes Secrets, a store as an endpoint, a key operation with more than
  one provider, streams it finds rather than makes, a route whose parent it
  is given and whose rules are named — and what a platform owes back. Written
  so that a second platform, run by someone else, can satisfy it without a
  patch to any chart.
- **Logs move to stderr.** stdout is the program's product and stderr its
  commentary, which is the same split for a service, a job and a
  command-line tool, so a binary that grows a subcommand does not have to
  move its logs. The rule that follows is that a service writes nothing to
  stdout, and that every library which logs is wired to the service's logger.
- **The RPC rule says which protocol, not just which library.** The server is
  one Connect handler serving all three protocols; a client in the cluster
  speaks gRPC over cleartext HTTP/2, and speaks Connect over HTTP/1.1 or
  gRPC-Web only where HTTP/2 trailers cannot survive the path. A schema with
  nothing configured to generate from it is a client somebody hand-wrote.
- **Two new rules in the service contract.** A rollout replaces instances
  without a gap — two instances, a disruption budget, no unavailable
  replicas, and one drain constant used by the code, the grace period and a
  pre-stop delay alike. And transport identity belongs to the platform: a
  service presents an identity it is given, reloads it, and checks its peers,
  while the probes listener is exempt and the chart's default is off.
- **Three decisions.** Telemetry is configured by OpenTelemetry's own
  environment, which is the one exception to "configuration is a file" and
  removes the `otel` fragment nothing ever read. There is no service mesh:
  identity is the account a workload runs as, attested by the runtime rather
  than asserted by the workload, and terminated in process. And the twelve
  factors are a map to read these contracts by rather than a label to claim,
  with the two deviations argued instead of footnoted.
- **`nats` takes a `tokenFile`, not a credentials file.** It is the
  workload's own account token, re-read on every reconnect so a rotation
  needs no restart, and the broker asks an authorisation service who the
  bearer is rather than trusting what the client claims.
- **`bucket` takes a `ca`**, because a store inside somebody's own network is
  the ordinary case and is not signed by a public root.
- **The example runs.** Two releases, because a migration hook cannot wait
  for a database its own release creates: one for the database and the
  stream, one for the application. Two database roles with two credentials,
  because the migration creates tables and the services must not be able
  to. And a smoke test that asks the only question rendering cannot: a
  redirect is served, an event crosses the broker, and a counter another
  service owns moves by exactly the number of requests made.
- **The cluster tier is now a pull-request gate**, which is what decision
  0005 said it would become once something consumed it. Five defects it
  caught while the example was built are listed there.
- **The `nats` fragment now describes a CONNECTION only**, and a new
  `nats-consumer` fragment describes what a consumer binds to. The first real
  consumer is what showed that a publisher carrying a `consumer` field it
  never reads is a field somebody will eventually set.
