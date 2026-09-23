# Python canon

**Scope:** every Python service, library or tool held to these contracts.

This page starts from nothing. There was no Python in the adopting estate
beyond a handful of single-file scripts with no dependencies at all, so
unlike the Go and Kotlin canons there is nothing to inherit and nothing to
reconcile. It is therefore deliberately short: a row is added when a
component needs it, not in anticipation.

## Canon

| Area | Library | Notes |
|---|---|---|
| Language | Python, with type annotations and a type checker in the gate | annotations that nothing checks are comments that rot |
| Project and dependencies | `uv` | one tool for the interpreter, the resolution and the lock; the lock is committed ([repository.md §3](../contracts/repository.md)) |
| Linting and formatting | `ruff` | one tool for both, so formatting is never a second opinion |
| Testing | `pytest` | |
| Boundary validation | `pydantic` for values that cross a boundary at run time | the configuration file is validated against its JSON Schema by this repository's loader, not by a second description of the same shape — the same rule the Node canon states for its validator |
| Configuration | this repository's loader | [config.md](../contracts/config.md) |
| Logging | `structlog`, emitting JSON to **stderr** at one level | [service.md §4](../contracts/service.md) |
| Telemetry | the OpenTelemetry Python SDK, configured by its own environment | [0006](../decisions/0006-telemetry-is-the-sdk-environment.md) |
| HTTP client | `httpx` | |
| Object store and cloud | the cloud vendor's Python SDK | the S3 API, not the vendor: an endpoint is configuration ([platform.md §4](../contracts/platform.md)) |
| Events | the broker's own Python client | a pull consumer, durable by name, acknowledged only after the work is stored |
| Wiring | a hand-written composition root | no container. [0001](../decisions/0001-no-di-containers.md) |

## What is deliberately absent

**No web framework.** The first Python component held to these contracts is a
stream consumer: it reads from a broker, writes objects to a store, and
serves only its probes, which is a handful of lines on the standard library's
server — [`runtime.py`](../../examples/url-shortener/log/src/url_shortener_log/runtime.py).
Naming a framework before a component needs one is how a canon acquires an
entry nobody chose.

**No `pydantic` yet either**, although the row above names it. It is for
values that cross a boundary at RUN time, and the first component has none:
its configuration is validated against a JSON Schema by this repository's
loader, and the events it archives are not its to describe — an archiver that
imposed a shape on what it stores would drop whatever the publisher added
next. The row stays because the rule is decided; the dependency arrives with
the component that needs it.

**No RPC row yet.** The Connect ecosystem's Python support is younger than
its Go and TypeScript support. When a Python component owns or consumes an
RPC boundary, this row is filled in with the same rule the other canons use —
the schema in the repository, both sides generated at build time — and the
choice is made then, against what exists then.

**No ORM.** Nothing in Python touches a database yet.

## One weakness to know in advance

**Server-side certificate reload is the awkward case.** [service.md
§10](../contracts/service.md) asks a service to reload a rotated certificate
without restarting. A Python client does this naturally, because a context is
built per connection. A Python *server* built on the common frameworks reads
its certificate once at start-up, and there is no portable way to swap it
underneath a running listener.

So a Python component that serves mutual TLS either recycles its workers
within the certificate's lifetime — which is ordinary for a process-per-worker
server and needs no new machinery — or sits behind the terminating proxy
[0007](../decisions/0007-no-mesh-identity-in-process.md) provides for exactly
this case. A component that only makes outbound calls, which the first one
does, is unaffected.

## Images

A runtime image copies an installed environment onto a minimal base and runs
the interpreter directly. No build step in the image, no package manager in
the image, no shell in the image where the base can avoid one
([service.md §7](../contracts/service.md)).

The Go components get this from `ko`, which lays an image around a static
binary. Python has no equivalent, so the equivalent is a script: the
dependency tree is resolved from the committed lock and installed into a
directory, the first-party wheels are built and installed beside it, and the
Dockerfile is a `COPY` and an `ENTRYPOINT`. It is worth keeping honestly
rather than by moving the build into an earlier stage of the same file — a
multi-stage build still executes a package manager while the image is
assembled, which is what makes a cross-architecture build need emulation.

See [`hack/build.sh`](../../examples/url-shortener/log/hack/build.sh) and
[`Dockerfile`](../../examples/url-shortener/log/Dockerfile).

**The base is pinned by tag, not by digest**, which is the same choice the Go
components' base makes, and for a reason worth writing down: the registry's
free tier keeps only the current build, so a pinned digest stops resolving
and breaks every fork's build with a failure whose cause is invisible. What
must not move is the interpreter's MINOR version, and that is pinned where it
can be — the toolchain declares the interpreter that resolves the lock and
builds the wheels, and a mismatch fails the build rather than the deployment.

## Exceptions

A component that needs something not on this list adds a row here first, with
the reason, in the same change that introduces it.
