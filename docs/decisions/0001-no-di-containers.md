# 0001 — Application graphs are hand-wired

**Status:** accepted

## Context

Services built on a dependency-injection container register providers and
resolve them lazily at run time. The appeal is real: a new dependency is one
registration, and nothing in between has to know about it.

What that costs, measured on services that were built this way and then
taken apart:

- **A missing dependency becomes a run-time failure.** The program compiles.
  It fails when the graph is resolved, which is after the process has
  started, in whatever environment resolved it first.
- **The shape of the program is not in the program.** Reading `main` tells
  you a container is built. What is in it, and in what order, is spread
  across registration calls and only the container can say.
- **Lifecycle hooks silently have no caller.** Taking two such services apart
  surfaced two shutdown bugs that had been latent: a connection pool that was
  never closed on shutdown, and a background refresher that was never
  stopped. Both had correct code. Nothing called it, and nothing could have
  noticed.
- **It spreads.** A container in the entry point needs the whole graph to be
  in the container, so packages grow registration functions, and those are
  importable, so they are imported.

The counter-argument is that a hand-written composition root is verbose and
that wiring mistakes are not caught by unit tests. The first is true. The
second is true of containers too, and worse: the container defers the same
mistake to run time.

## Decision

**Application object graphs are constructed by hand, in `main`, with plain
constructors, in dependency order.** Dependency-injection containers,
service locators and run-time registries are not used in services held to
these contracts.

Where several listeners share singletons, one composition-root value owns
them and is built once; sharing is a field, not a lookup.

This is a rule about *application* graphs. A framework that a service is
built on may do what it likes internally: the rule is that the service's own
dependencies are visible in its own entry point.

## Consequences

### Good

- The graph is a readable sequence of constructor calls. What is constructed,
  in what order, with what, is the text of `main`.
- A missing or mistyped dependency is a compile error.
- Shutdown has an owner. Everything constructed in the root is released by
  the root, in reverse, and a hook with no caller is visible as such.
- Nothing needs a container to be understood, so nothing needs the container
  to be testable.

### Bad

- **Graph construction in `main` is only partly testable.** An argument
  dropped or two constructors reordered fails no unit test. Integration tests
  are the net, and that is a real gap, honestly the strongest argument the
  other way.
- It is more verbose at the call site than a registration.
- A service with a genuinely large graph has a long `main`. Long and flat
  beats short and indirect, but it is long.

### Neutral

- No run-time behaviour follows from the wiring style itself.
- A service in a language whose framework provides its own injection (as
  several Node frameworks do) keeps it; the rule is not a campaign against a
  language's idiom, and the framework boundary is where it stops.
