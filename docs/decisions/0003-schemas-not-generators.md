# 0003 — A schema and a hand-written type, not a code generator

**Status:** accepted

## Context

Once configuration is a schema-validated file ([0002](0002-config-file-plus-env.md)),
the obvious next step is to generate from the schema: the configuration type
in each language, the deployment's own schema, the defaults, the
documentation, and — following the same logic one step further — the wiring
that consumes it.

That step was designed in detail before it was rejected. The design was a
generator reading component manifests and emitting entry points, typed
configuration, deployment templates and build configuration for several
languages. It would have worked.

What it costs is not the generator; it is what the generator becomes. A
generator needs a manifest language. The manifest language needs a schema of
its own, defaults, conditionals and escape hatches, because real services
differ. Everything it emits must then be regenerated when it changes, so
every consuming repository is pinned to a version of it, and an upgrade is a
coordinated change across all of them. The generator becomes the framework
that the rest of these contracts exist to avoid — and this time it is one
nobody else has ever used.

The alternative costs one test.

## Decision

**The schema is authored by hand. The configuration type in each language is
written by hand. A test regenerates a schema from the type and diffs it
against the committed schema.** Nothing generates wiring, entry points or
deployment templates.

Small loaders are published per language. They load a file, validate it
against a schema, decode it into the caller's type, and stop. They contain no
lifecycle, no dependency wiring and no transport.

Generated code is still used where the thing generated is a *protocol*: an
RPC schema generates clients and servers, because the wire format is the
contract and two implementations of it by hand will disagree. The difference
is that the protocol's generator is somebody else's, used by thousands of
people, and what it emits is not the shape of the program.

## Consequences

### Good

- The type a service uses is ordinary code: readable, documented in its own
  language, greppable, and debuggable.
- No generator to install, pin, version or upgrade; no manifest language to
  learn.
- A drift test is a few lines and fails in the pull request that caused it.
- Adding a language means writing a small loader, not teaching a generator to
  emit that language.

### Bad

- **The same shape is written twice** — once as a schema, once as a type —
  and in three languages, four times. The drift test makes the duplication
  safe, not free.
- A large configuration is tedious to add a field to.
- Defaults live in two places unless care is taken: the schema states them,
  the type must agree. The drift test covers exactly this.

### Neutral

- If the duplication ever becomes the dominant cost, a generator can be
  introduced for the type alone, with the schema still authored by hand. That
  is a much smaller thing than what was rejected here, and this decision does
  not forbid it — it forbids generating the program.
