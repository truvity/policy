# The configuration contract

**Normative.** One typed configuration per binary, described by a schema
that both the binary and whatever deploys it are held to.

The failure this prevents is specific and common: a configuration key is
renamed in the code, the deployment still sets the old one, nothing
complains, and the service runs with a default nobody chose. The chart was
right yesterday and is wrong today, and the only signal is behaviour.

## 1. One schema, two readers

Each binary ships a JSON Schema describing its configuration. Two things are
held to it:

- **the binary**, which validates the file before it builds anything; and
- **the chart** (or whatever renders the deployment), whose rendered
  configuration is validated against the same schema in its own tests.

Because the two read the same file, a key the binary does not know is a test
failure in the pull request that added it, not a surprise in a cluster.

## 2. The schema is the source; the type is checked against it

The schema is authored, not generated as an afterthought, and the
configuration type in each language is written by hand. A test regenerates a
schema from the type and compares it to the committed one, so the two cannot
drift.

Why not generate the type from the schema: generated configuration types
grow a generator, the generator grows options, and the options grow a
language. Every service then depends on that language being installed,
current and correct. A hand-written struct beside a schema, with a test that
they agree, costs one test.
[0003](../decisions/0003-schemas-not-generators.md) is the decision.

## 3. Strictness

Everything the service defines is strict: `additionalProperties: false` on
every object the schema itself describes. A typo must fail, and a key that
means nothing must be impossible to set.

Pass-through regions — a block handed verbatim to a platform that has its own
schema — stay open, and say in the schema description who validates them.

## 4. Shared shapes are shared

Configuration that means the same thing in more than one service is a
fragment in [`schemas/fragments/`](../../schemas), referenced with `$ref`: how a listener is
described, how a log level is set, how an object store is addressed, how a
database is reached. A service that invents its own spelling of a shared
shape makes every tool that reads configuration into a special case.

The fragments are versioned with the repository: a `$ref` names a released
version, and moving to a newer one is a change a reviewer sees.

## 5. Secrets are not in the file

A secret is an environment variable, declared in the schema as a *name* the
service reads rather than a value it takes. Configuration files are rendered
into config maps, logged when someone debugs a deployment, and committed as
test fixtures; secrets must survive all three being true.

A service never logs a secret's value, and never includes one in an error. An
error says which key was wrong, not what it contained.

## 6. Failure is at start-up, and says where

**The shared envelope holds what EVERY component has** — somewhere to report
health, a log level, a shutdown budget — and nothing else. A listener is not
one of those: a job exits, and a consumer answers nothing. Neither is a
transport identity, for the same reason.

That line is easy to put in the wrong place, and putting it wrong is cheap
to do and expensive to notice: a field every component carries and only some
can use is a field a deployment sets and watches do nothing. This repository
has put it wrong twice — once with a telemetry block that was deleted, and
once with a listener that the counter carried for no reason but to satisfy a
test. A component declares what it actually has.

A service validates its whole configuration before it opens a listener or
connects to anything, and refuses to start on the first failure, naming the
path that failed (`store.endpoint`, not "invalid config"). Half-starting with
a bad configuration is how a service ends up serving with a default nobody
chose.

## Conformance

| Rule | Mechanism |
|---|---|
| 1. one schema, two readers | the chart's tests validate the rendered configuration against the binary's schema |
| 2. no drift | a test regenerates the schema from the type and diffs it against the committed file |
| 3. strictness | a negative fixture per schema: an unknown key must fail |
| 4. shared shapes | review, and the fragment `$ref`s in the schema |
| 5. secrets | review; the loader has no way to read a secret from the file |
| 6. start-up failure | a test that starts the binary with each invalid fixture |

The [Go loader](../../config) and the [TypeScript loader](../../ts) implement
rules 1, 5 and 6 so that a service does not have to, and
[`conformance`](../../conformance) is what a chart's tests use to be held to
the same schema.

The two are tested against **the same fixtures**, in the same directory. That
is the point rather than an economy: two loaders that claim to implement one
contract must refuse the same documents and say something a person can act on
when they do. A fixture only one of them sees is a contract that exists
twice. They load, validate, decode, and stop. They are deliberately
not a framework: no lifecycle, no dependency wiring, no HTTP, no reflection
over the environment.
