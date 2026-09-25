# Changelog

What changed for someone consuming this repository, newest first. A version
missing from this file changed nothing a consumer can see — a dependency bump
and nothing else — and its GitHub Release lists the commits.

## Unreleased

- **The archiver's write now shows inside each request's trace.** The write
  and its S3 call are one span in one trace (a batch cannot be any single
  request's child), so a trace opened on a redirect ended at "received" and
  gave no sign the event had been archived. Each message now also gets a
  short `archive.write` child, timed to the flush and linked to it. The
  guide says why the S3 call itself stays under the flush.

- **`logging-and-telemetry.md` gains "A trace that stays whole"**, with
  pointers from the events, RPC and object-storage guides and a new item in
  the conformance review. Spans that exist per service but do not connect
  pass every exporter's health check, so the guide names the five places
  context is dropped (an RPC, a broker, a database, an object store, a
  thread hand-off), the two decisions that are not configuration (trust the
  caller or link to it; parent a single message but *link* a batch), the
  lower-case `traceparent` header NATS needs, and the check that proves it:
  fetch one trace by id and read its tree.

- **The example's traces are one graph, not one fragment per service.**
  Every service was exporting spans and no request could be followed
  across them, because each hop began a trace of its own. Five joins,
  each a place context was being dropped:

  - **web to urls**: the web server's client now makes a client span per
    call and puts the trace context on the request. urls **trusts** the
    incoming parent (`otelconnect.WithTrustRemote()`); the default is to
    only *link* to an untrusted caller, which is right for a service
    facing the internet and wrong for one whose callers are its own
    platform.
  - **redirect to the broker to stat and log**: the publisher writes
    `traceparent` into the message in lower case (NATS header names are
    case-sensitive, and an HTTP-style carrier would have written
    `Traceparent` where no other language looks). The Kotlin consumer
    continues the trace; the Python archiver records a span per message
    and one flush span that *links* to them, because a write of hundreds
    of records cannot be the child of any one.
  - **stat to urls**: the outbound call runs on another thread, and the
    current span lives in a thread-local, so the client span had no
    parent. The HTTP client's executor now captures the context where the
    call is enqueued.
  - **database and object store**: every query is a span under its
    request (values are not recorded: they are the URLs people
    shorten), and every S3 call is a span under the flush.

  The header case and the thread hand-off are each covered by a test that
  fails without the change.

- **`platform.md` §11: two charts, and who installs each.** The split
  between the infrastructure chart and the application one has a second
  consequence that rule 6 does not state — the infrastructure chart is
  **per install**, so anything running per *cluster* cannot supply what
  it supplies. There is no "the install" at that level, and a store
  minted there serves the deployment while leaving every test install
  with nothing.

  Written down because it was got wrong: an attempt to move those
  resources to a per-cluster provisioning stack covered one tier and
  silently broke the other two. The section also records why a chart
  takes a **tier** (an engineer's namespace often cannot create custom
  resources at all, so a chart that always mints them is one they cannot
  install), and why generating a password is the wrong instinct — it
  invents a provider that only the deployment has.

- **`guides/conformance.md`** gains the checklist that follows from it,
  for a repository whose service owns a database or a stream.

- **§11 places every resource this example has**, in a table, by scope —
  because the abstract rule is easy to agree with and hard to apply. Two
  entries answer questions that kept coming back: the **store is at
  namespace scope** even though each install writes to its own prefix
  (which is what lets one namespace serve an engineer's copy and a CI run
  without either provisioning anything), and **nothing in the
  infrastructure chart names a vendor** — a database and a stream are
  custom resources some operator reconciles, while "bucket" is one
  cloud's word, and this chart has to install on a laptop too.

## Unreleased

- **The infrastructure chart provisions the objects an install owns**, and
  takes a `tier` that decides whether it provisions them at all. A `test`
  install mints nothing and runs as the namespace's standing identity; a
  `primary` install makes its own store and the identity that reaches it,
  from names it is given rather than names it derives.

  Those resources had been moved into a stack that runs per cluster. The
  reason that was wrong is scope: there is no "the install" at cluster
  scope, so it served the deployment and left every engineer's copy and
  every CI run with nothing.

- **`platform.md`'s credential advice is corrected.** It recommended "a
  credential the database operator issues for the role it already
  manages"; the operator issues no such thing, and its managed roles take
  a password or no password. The recommendation stands and now says what
  it costs: who owns the CA, why one arrangement makes you take over
  replication's identity, and why splitting the two CAs breaks verifying
  the server. The example still uses a password, because the issuer is a
  platform's to provide.

- **The default is unchanged and the default render is byte-identical.**
  `tier` defaults to `test`, so nothing here is reachable until a
  platform asks for it by name.

- **`platform.md` §11 no longer says a chart may not name a vendor.** The
  bar is that it RENDERS without that cloud, not that it installs on one,
  and the tier is what keeps that honest.

## Unreleased — packaging

- **v0.4.6's `url-shortener-infra` chart cannot be installed. Use v0.4.7.**
  It reached the registry carrying a top-level `images:` map it never
  declares, and its schema sets `additionalProperties: false` — which
  Helm checks before any template runs. So the chart refuses *every*
  install, including one that passes no values at all:

  ```
  - at '': additional properties 'images' not allowed
  ```

  The chart source was never wrong. The packaging tool gave every chart
  in a release every image the build produced, which a repository
  publishing one chart never notices and this one, publishing two, did.
  Fixed upstream in the tool, so nothing here changed but the version of
  it that CI runs.

  Worth keeping as an example of the shape: the release packaged, pushed
  and went green, and the only signal was somebody trying to install the
  result. Rule 7 — *a published artifact is tested as published* — is in
  `contracts/release.md` because of this class, and the test that now
  guards it renders the **packaged** artifact, since rendering the
  source tree cannot see a defect that packaging introduces.

## v0.4.6 — 2026-09-24

- **A client can authenticate to the broker.** The configuration contract
  has had `nats.tokenFile` all along, described as something the platform
  mounts — and nothing mounted it. Against a broker that authenticates
  its clients, every component that touches the stream failed at connect
  with `Authorization Violation`, which reads like a wrong password
  rather than a missing mount.

  `events.auth.audience` is an **audience**, not a credential: the chart
  projects a ServiceAccount token for it, the pod cannot forge one, and
  nothing here or in a values file is a secret. Empty renders no volume
  at all, which is a broker that admits anonymous clients — what a local
  one does.

- **The route declares the fields the API server defaults.** `group`,
  `kind` and `weight` on a backend reference are filled in if omitted, so
  a chart that leaves them out renders a route that never matches what is
  stored: a permanent difference, in every renderer that compares the
  two, for a route nobody changed.

  Declared rather than ignored. Telling a comparer to skip those fields
  silences the real changes underneath them, and declaring a default is
  not duplication — it is saying which value this chart wants, where a
  reader can see it.

## v0.4.5 — 2026-09-24

- **Every component exports telemetry**, in all four languages, read from
  OpenTelemetry's own environment (decision 0006). The chart carries an
  `otel` block in its values and renders `OTEL_*`; nothing reads
  telemetry from a configuration file and no schema changed.

  **No endpoint means export nothing** — not "export to localhost and
  retry forever", which is what an SDK left to its defaults does. The
  chart sets the exporters to `none`, so a laptop, a test and a cluster
  with no collector all do the same thing, and no component carries an
  enable flag. That flag is the failure 0006 records: a service that
  exported to a console in production because nothing set the
  environment name its code tested.

  **OTLP logs are off.** A node agent already collects stdout into the
  same store under the same namespace, so an exporter buys a second copy
  of what is there — and logs that exist only over OTLP vanish exactly
  when the exporter is what broke.

  In Python and TypeScript the starter is an **optional extra**: the
  loader is what every consumer takes, and a service reading a
  configuration file should not be made to carry an SDK it never starts.

  Go also exports **runtime metrics**, which cost nothing and make an
  empty store unambiguous — without a series that is always present,
  "nothing is arriving" and "this service is quiet" look identical.

## v0.4.4 — 2026-09-24

- **A route can name its parent's KIND.** It could only ever attach to a
  Gateway, which is the API's default and silently wrong on a cluster
  that serves routes from something else. `route.parentRef.kind` and
  `.group` are the platform's to set, like the name and namespace beside
  them.

  Silently, because there is no good signal: a route whose parent does
  not exist is **Accepted** — the status says so and goes on saying so —
  and the service answers 404 with every pod healthy. The only other tell
  is the listener reporting zero attached routes, which nobody watches.
  Found in a cluster, by the 404.

## v0.4.3 — 2026-09-24

- **v0.4.2 did not publish.** Its Go images went to `ghcr.io/truvity`
  rather than to this example's repository, because `KO_DOCKER_REPO`
  silently overrides a `repositories:` named in the release
  configuration — and it failed only afterwards, on an SBOM written to a
  repository nobody meant to use. The images are built the way the old
  script built them now: the destination in `KO_DOCKER_REPO`, the
  command's own name appended.

  Neither `goreleaser check` nor a snapshot build can see this: a
  snapshot publishes those images to `ko.local` whatever repository is
  named. It is the rule in `release.md` §7 catching its own author.

## v0.4.2 — 2026-09-24

- **The release is two tools and no scripts of ours.** GoReleaser builds
  and pushes every image and records what it pushed; helmctl reads that
  and bakes the digests into the charts. `publish-images.sh` and
  `chart-manifest.py` are both gone — the second of them reproduced a
  schema ocictl owns, inside the repository other repositories copy.

  **One job**, so the ordering cannot be got wrong: the charts are
  packaged from a file that does not exist until the images are pushed.
  v0.4.1 fixed that ordering; this removes the possibility of it.

- **One configuration, three loops.** The image destination, the tag and
  the GitHub-release switch are taken from the environment, so a local
  loop, a CI loop and a release differ in three values and never in what
  is built. One destination per repository — public to a public registry,
  private to a private one, never both.

- **The multi-architecture rule is a test, not a shell assertion.** It was
  a check that inspected images after pushing them, which could only fail
  once a release had happened and could not see the likeliest mistake — a
  platform quietly dropped from the list. It now fails in the gate, and it
  is joined by two more: no Dockerfile may execute anything while the
  image is assembled (which is what makes cross-building a file copy), and
  no registry may be written into the release configuration.

  The build context is the repository root for every image, because the
  release stages files into the build tool's context keeping their paths
  and a Dockerfile can only be written for one context. One `COPY` line
  both builds agree on beats two that can drift.

## v0.4.1 — 2026-09-24

- **The release pins every image by digest, and refuses to publish a chart
  that is not.** The build writes down what it pushed, the chart is
  packaged from that file, and `--require-image-digests` rejects a chart
  with an entry left blank. The image values changed shape to
  `images.<component>.{registry,repository,tag,digest}` — the shape that
  check reads. A chart spelling them any other way passes the check with
  nothing to check, which is worse than not running it.

  The jobs are in the other order now: images first, then the charts
  packaged from their digests. A digest exists only once the image is
  built, so a release that published charts first was always going to
  publish empty ones.


- **The published chart could not render a single Deployment.** Its own
  values said *the release stamps a digest per component here*, and the
  release does not: it publishes the charts and the images in the same run,
  and a digest only exists once the image is built. So the chart reached the
  registry with `digests: {}` and `tag: ""`, and every install of it failed
  at the first `image:` with `no image for web`.

  Nothing in this repository could see it. Every chart test supplied a tag
  or a set of digests, so all of them passed against a chart no consumer
  could use. It was found by installing the published artifact, which is the
  only place the difference exists.

  A published chart now falls back to its own **appVersion** — the version
  its release stamped, and the one thing such a chart always knows about the
  images built beside it. `image.digests` remains, and a deployment that
  needs a rollback to reach an exact image still sets it; what changed is
  that a chart with neither installs instead of refusing.

  There is a test for the published case now, and it fails against the old
  helper with the same message the cluster produced.

- **`platform.md` §10 corrected on the same point.** It claimed a release
  stamps digests into the published chart. It does not, and saying so made
  a chart that cannot be installed look like the intended shape. A platform
  still fills neither field; a *deployment* may pin digests, and that is a
  different actor making a stronger promise about one install.

## v0.4.0 — 2026-09-24

- **`platform.md` §10: what a platform passes a chart, by name.** Rules 1 to
  9 say what a chart may ask for; nothing said what a platform hands it, and
  the gap between those two is where a repository ends up satisfying the
  whole service contract and still being undeployable.

  Found by taking these charts to a real delivery layer and discovering that
  the values it renders and the values these charts read have zero keys in
  common — not a spelling difference, two unrelated interfaces. One side
  passes addresses; the other derives them from a naming convention. The
  second renders perfectly and installs on exactly one platform, and nothing
  says so until the second platform tries.

  Two tables, one per chart, each row naming the decision it answers. The
  test is rule 1's: hand the table to a platform that shares no naming
  convention with the first, and a row it cannot fill is a convention
  wearing a value's clothes.

- **The infrastructure chart takes the platform's decisions.**
  `postgres.labels`, `postgres.scheduling`, `postgres.backup`,
  `postgres.serverTLS` and `events.account`, named for the decision rather
  than for the operator's field — so a platform running a different database
  operator can still say *these instances belong on that pool*. Every one
  defaults to nothing: installing this on an empty cluster renders exactly
  what it rendered before.

  Archiving is a **name**, not a description. The archive is a resource the
  platform made, with its own retention and credential model; a chart that
  described one would be describing the wrong one on every platform but the
  one it was written against, and the difference is only visible when
  somebody tries a restore.

  `events.account` and `events.url` are alternatives rather than a pair. An
  account carries both the broker and the identity, so a server list beside
  one is the chart arguing with the broker about an answer the broker
  already has — and that argument is resolved silently.

- **Rule 6 is checked by a test now, not by reading.** Both charts render in
  the suite and neither may produce the other's kinds: no workload from the
  chart with a separate lifetime, nothing the application chart should be
  finding rather than making. That ordering was discovered by building it
  the other way first, and nothing but a test would notice it being undone.

  Embedding the second chart paid immediately: the platform's labels were
  being written *above* the chart's own rather than merged, producing a
  duplicate `app.kubernetes.io/instance` line. Helm renders it, the API
  server keeps the last, and which one that is depends on the order a
  template happens to write them in.

## v0.3.0 — 2026-09-24

- **The charts are published.** The image repository has pointed at ghcr
  since the first commit and the images have been published since v0.2.0;
  the charts that install them were published nowhere, so nothing could
  install this. Both of them, because they are a pair: Helm runs every hook
  before anything else in the same release, so the chart that migrates a
  database cannot be the chart that creates it.

- **The chart check is called `charts`, which is what the repository
  contract says it is called.** It shipped as `kubeconform` — an accurate
  name for the tool and the wrong name for the recipe. The contract fixes
  these names precisely so that a person moving between repositories, and
  anything automating across them, does not have to read a recipe file to
  find out what the chart check is called here; a repository that has the
  job under another name has made every caller special.

  Found by reading the contract back against the repository that publishes
  it. The failure is invisible from the inside — everything runs, the gate
  is green, and only a caller from outside notices — so the conformance
  guide now says to list the fixed names against `just --list` rather than
  assume them.

## v0.2.0 — 2026-09-24

- **Which cluster a repository tests against is not a preference.** A public
  repository stands up a local one because it is forced to: a fork's pull
  request must never reach your infrastructure, and a contributor has none of
  it. A private repository has no such constraint, and taking the local
  cluster anyway costs it the thing it actually needs — its CI can reach a
  shared development cluster, which has the identity plane, the provisioning
  and the network policy a local one cannot. Standing up a throwaway cluster
  to avoid one that is already there trades a better test for a slower one.
  The suite does not change either way, which is the point of writing it
  against a chart and a set of probes rather than against an environment.

- **Chart goldens, and the second one is the one that earns its keep.** A
  `minimal` render records what the defaults produce, so a changed default is
  a diff in a review rather than a surprise in a cluster. An `everything`
  render sets every value to something other than its default, so a template
  that stopped READING one shows up — which nothing else in the package would
  catch, because a test that asserts a particular key says nothing about the
  other four hundred lines. The failure names the first line that moved,
  rather than printing a thousand.

- **The renders are validated against the Kubernetes API's own schemas.**
  That is a question the chart tests do not ask: they check that a rendered
  file is one the BINARY accepts, and would pass just as happily for a
  Deployment with a misspelled field — because the API server ignores one
  rather than refusing it. Proved by misspelling one. It reads the committed
  goldens, so what is validated is the render a reviewer actually read.

- **The example has a front end, and it is the fourth language.** It serves a
  page and asks the service that owns the tables; it writes nothing and holds
  no database credential, which is the ownership rule seen from the consuming
  side. Its deployment has no password block at all while the two Go services
  do — the difference is visible in the chart, which is where it should be.

- **One code generator for TypeScript, not two.** connect-es v2 builds a
  client from the service descriptor `protoc-gen-es` already emits, so the
  separate Connect plugin the earlier line needed is gone.

- **The runtime image carries no `node_modules`.** The server is bundled into
  one file, which is not only tidiness: the dependency on this repository's
  own loader is a symlink in a checkout, and a symlink is not a thing that
  can be copied into a container. Two things had to be got right for the
  bundle to run — a dependency reached through its CommonJS build calls
  `require` for a Node builtin and an ES module has none, so the bundle
  starts with one; and generated imports must end in `.ts`, because the
  server runs TypeScript directly and Node resolves the path it is given.

- **A fourth loader, in Kotlin**, read against the same fixtures as the other
  three. An empty file parses to a MISSING node on the JVM rather than a null
  one, and checking only the second let it reach the validator — which
  reported "unknown found, object expected". Accurate, and no help at all to
  somebody looking at a blank file.

- **The counter is a Spring Boot service now, and the Go one is deleted.**
  The example has four languages in it and the chart still does not know
  which is which: the same probes on the same paths, the same drain, the same
  account, the same configuration file validated against the same schema.
  Three things had to be got right for that to be true, and each was wrong
  first:

  A framework that serves no traffic still has to stay running. With the web
  application turned off entirely the process started, consumed once and
  exited — and the probe listener never started either, so the symptom was a
  pod reporting "drained" a second after it reported "consuming". The main
  listener is disabled and the management one is not.

  The configuration file is read ONCE, by the composition root. A bean that
  re-read it would have to rediscover the path, and the first version did
  exactly that and found nothing, because the path arrives as an argument
  and a bean has none.

  And the refusal happens before the framework starts. A configuration this
  service will not accept should be one line on stderr, not a framework
  stack trace about a bean that could not be created — which is the same
  refusal with the answer buried in it.

- **The JVM presents an identity as a client**, with TLS 1.3, a certificate
  reloaded when the platform replaces it, and the peer admitted by the
  ACCOUNT in its certificate rather than by the address that answered. Proved
  under strict mutual TLS on a cluster, in the same gate as the Go and
  Python components.

- **The Connect generator for Kotlin is a JAR, not a binary**, so the build
  writes a launcher around it. Every other generator here is a static binary
  the environment manifest pins; this one is a JVM artifact, and pinning it
  in the manifest would mean pinning a jar as if it were a binary while
  pretending the JVM is not already on the machine.

- **The example has an ownership boundary, and all three shapes are now in
  it on purpose.** A new service owns the URL tables; the counter asks it
  instead of writing them. The counter's configuration has no `database`
  block at all, and a test asserts that it does not — the ownership rule
  usually shows up as an absence rather than as a line of code. What used to
  be a database password with write rights on a table it did not own is now
  an address.

  The contrast is the interesting part and it is documented rather than
  tidied away: an RPC for a boundary of ownership, an event for fan-out, and
  a direct read on the redirect path because that is the hot path and a
  second network hop on it is not worth what it buys.

- **One handler serves Connect, gRPC and gRPC-Web on one port**, so the
  protocol is the caller's choice. The smoke test proves both ends of that:
  the counter's call arrives as gRPC, and the same boundary answers an
  ordinary `curl` POST with a JSON body. The second is worth a test because
  it is the difference between a boundary anyone can ask a question of and
  one that needs a generated client.

- **The schema is a file with a linter on it, and the generated code is
  committed.** A field renumbered by hand is a wire incompatibility that no
  compiler catches, because both sides are regenerated from the same file in
  the same commit and agree with each other perfectly. Generation at build
  time was the alternative, and its first casualty is the editor, which
  cannot resolve a symbol that does not exist yet.

- **`listen` leaves the shared envelope**, for the same reason the transport
  block never joined it: a job exits and a consumer answers nothing, so a
  listener is not something every component has. This was found rather than
  reasoned about — the counter had been carrying a `listen` it never read,
  purely so that a test comparing its type to its schema would pass. A field
  every component carries and only some can use is a field a deployment sets
  and watches do nothing, which is the third time this repository has met
  that shape.

- **A client presents an identity too, and the gate now proves it.** The
  counter is the example's first in-cluster RPC client, so it is the first
  component that needs a certificate without serving one. Three things came
  out of making that work on a cluster rather than on paper:

  A chart's own internal callers are the chart's to grant. The allow-list
  value is for callers from OUTSIDE the release — leaving the internal one
  to an operator means a chart whose default configuration cannot talk to
  itself, and the error names a certificate rather than a list nobody
  filled in.

  `permissive` is meaningless for a component that serves nothing. There is
  no second listener to put anywhere, and rendering that mode produced a
  crash loop complaining about a listener address on a component with none.
  A client either presents an identity or it does not.

  And the two allow-lists are different questions. "Who may call this
  service" and "whose answer will this client accept" look alike enough to
  share a value, and must not: a client that checked only the certificate
  chain would accept any workload in the trust domain that happened to
  answer on that address.

- **The example's cluster scripts name the cluster they talk to.** Not
  whatever context is current — `kind create cluster` points the current
  context at whatever it just made, so a second box created in another
  terminal silently moves every command in the script. The symptom is
  "namespace not found" for a namespace that is right there, in the cluster
  you thought you were talking to. It also means the scripts cannot be
  aimed at a real cluster by accident.

- **The deprecated `h2c` wrapper is gone.** Cleartext HTTP/2, which a gRPC
  client needs when there is no TLS to negotiate over, is `Protocols` on the
  standard library's server and transport since Go 1.24. One fewer dependency
  on each side.

## v0.1.0 — 2026-09-24

The first version. It carries:

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
- **The transport is provable on the cluster, both ways.** The chart takes a
  `tls` block: an ephemeral volume the platform mounts an identity into, the
  in-cluster permission a workload needs to ASK for its own certificate, and
  a second port under `permissive` because one listener cannot be both. With
  the default `off`, the render carries no trace of any of it, which a test
  asserts by name rather than by golden.

  A new cluster step proves what only a cluster can: a caller on the list is
  served, a caller holding a REAL identity that is not on the list is closed
  at the handshake, and the service says which account it refused. All three
  halves are asserted — the second alone would pass for a service that
  refuses everyone, and without the third a refusal is indistinguishable from
  a service that is simply broken.
- **Verification is by identity, not by name.** A platform's workload
  certificate carries an identity and usually no host name, so a client
  builds the chain against the trust bundle and reads the identity out of the
  leaf itself. That means turning the standard library's own verification
  off, which looks alarming and is not: the comment sits next to the flag,
  because the next reader's first instinct will be to delete it. Two tests
  hold the property the flag would otherwise destroy — a server outside the
  trust bundle is refused, and so is one that chains correctly but runs as an
  account the client was not told to trust.
- **A pod security context, and the group is the point.** A driver writes
  what it mounts owned by root, so a process running as anyone else cannot
  read its own certificate. It surfaces as a permission error on a
  certificate authority file, or a complaint that a certificate is malformed
  — neither of which mentions identity, and both only once the transport is
  on.
- **The leak canary no longer fires on Kubernetes' own secret path.** Its
  parameter-store pattern matched `/var/run/secrets/`, which is where a pod's
  own credentials are mounted and is therefore in any manifest that reads
  one. A pattern that fires on the most common path convention in the
  ecosystem makes nobody safer: it teaches the next person to rename their
  mount to get past it, and the one after that to stop reading the output. It
  still catches a real parameter path, which is proved rather than assumed.
- **The local cluster can issue workload identities**, and asserts the one
  thing that makes them mean anything. cert-manager, its identity driver and
  that driver's approver, over a self-signed authority.

  **cert-manager's own approver is turned off, deliberately**, and this is
  the finding the box was built to produce. It approves every request for an
  authority it knows, so with it on the driver's approver never gets a say
  and any account that may create a request receives ANY identity it asks
  for — including its neighbour's. Nothing fails: certificates mount,
  services connect, every log line says success, and the attestation is
  decoration.

  Measured, not reasoned about: an account called `alice` submitted a request
  naming another account by hand and was issued a certificate for it. With
  the approver off the same request sits inert and nothing is issued. The
  verification step now asserts the flag, because the two states are
  indistinguishable from every other angle, and the platform contract now
  asks a platform to demonstrate the REFUSAL rather than the issuance.

  The consequence, found by CI rather than by thinking: turning that approver
  off turns it off for **everything**, including the authority's own
  bootstrap. A self-signed root expressed as a certificate needs its request
  approved like any other, and nothing was left to approve it, so the box
  never finished standing up. The box's root is a generated fixture now. A
  real deployment answers this with a policy engine; a throwaway root does
  not need one.
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
- **A third loader, in Python**, read against the SAME fixtures as the other
  two. That is the point rather than a detail: a contract with one
  implementation is a library, and a contract whose implementations are
  tested against different inputs is two contracts wearing one name — a key
  one loader refuses and another accepts is a configuration that passes a
  chart's test and crashes the service.
- **A Python transport helper**, and the difference it cannot hide. Python's
  `ssl` module has no verification callback, so a peer cannot be admitted or
  refused during the handshake the way the Go package does it: the chain is
  verified by the library and the ACCOUNT is checked immediately afterwards,
  by the caller. The failure that shape invites is invisible — a service
  that builds the context correctly and never makes that call verifies a
  certificate chain and admits anybody holding one, with every other test
  still passing. The test for it is a stranger holding a genuine certificate
  from the same authority, in the same trust domain, for an account nobody
  granted.
- **The example's fourth component is not written in Go**, and almost
  nothing changes. It consumes the request records the redirect service
  publishes and archives them as NDJSON in an object store, and its
  deployment is twenty lines that never mention the language: the same
  probes on the same port, the same drain, the same account, and its
  configuration file validated by the same chart test as the other three. A
  platform that had to know which language a workload was written in would
  be a platform every new language has to be added to.
- **The `bucket` fragment has a consumer**, so "a store is an endpoint, not
  a vendor" is exercised rather than stated: name, region, endpoint, path
  style, certificate authority, and credentials by NAME. The local cluster
  points it at its own object store and the component cannot tell.
- **A batch is named by its first stream sequence and nothing else.** A
  batch is acknowledged only after its object is written, so a failed write
  means the same records are redelivered — and a redelivery begins at the
  same sequence, so it overwrites its own partial attempt rather than
  leaving a second copy beside it. A failed write KEEPS the batch, because a
  consumer that acknowledged what it had not stored would lose it for good.
- **A runtime image with no build step, in a language that has no `ko`.**
  The dependency tree is resolved from the committed lock and installed
  OUTSIDE the image; the Dockerfile is a copy and an entry point. A
  multi-stage build is not the same thing — it still runs a package manager
  while the image is assembled, which is exactly what makes a
  cross-architecture build need emulation.
