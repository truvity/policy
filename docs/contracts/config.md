# The configuration contract

Version: 1.1 · Effective: 2026-10-05 · Changes: see [CHANGELOG](../../CHANGELOG.md)

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

> **Forward note.** [0010](../decisions/0010-data-contracts-are-written-in-pkl.md)
> supersedes 0003 for data contracts: the schema and the type in each language
> are to be generated from one Pkl source, in a shadow phase beside the
> hand-written ones. This section and the "no drift" rule stay in force, and the
> hand-written schemas stay authoritative, until a later change switches.

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

**A secret is a declared name, delivered by environment variable, a mounted
file, or a declared secret source resolved at start-up; never a value in the
configuration file.** The schema declares the *name* the service reads — the
variable, the file's path, the source's identifier — and never a field that
takes the value. Configuration files are rendered into config maps, logged
when someone debugs a deployment, and committed as test fixtures; secrets
must survive all three being true.

Why three ways and not one. The environment was the only one, and it does not
survive every platform:

- **A serverless platform caps the environment.** A function's variables
  share one small, fixed budget, and a service with a handful of credentials
  and its telemetry settings runs out of it. There is no larger budget to ask
  for.
- **Some secrets are per client.** A credential for each party a service
  deals with is a set that grows with the business, not with the release, and
  a variable per client is a deployment change per client.
- **A file can be re-read; a variable cannot.** A rotated secret mounted as a
  file reaches a running process. A variable is fixed for the life of the
  process, so rotating it is a restart.

A *declared secret source* is the service reading a secret store itself,
once, before it serves anything, from an identifier the configuration names.
It is for a platform that has no secret object to deliver through; where the
platform has one, [platform.md §3](platform.md) is the rule and the secret
arrives as a variable or a file.

A service never logs a secret's value, and never includes one in an error. An
error says which key was wrong, not what it contained.

The shared fragments hold the same line in their patterns where a value can
smuggle a secret in: a database URL that carries a password —
`user:password@host`, or a `password` or `sslpassword` parameter — is
refused, and the error names `url` without quoting it. A `passfile`
parameter is not refused: it names a file, which is a delivery this rule
allows.

## 6. Failure is at start-up, and says where

**The shared envelope holds what EVERY component has** — somewhere to report
health, a log level, a shutdown budget, which version of its document it is
(rule 7) — and nothing else. A listener is not
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

The file is read **once**, at start-up, and is not re-read: a running
instance's configuration does not change, and changing it means new
instances. What does change under a running instance — a rotated credential,
the state the service keeps in its own store — is read where it lives, when
it is used, and is not configuration. A reload is a second code path that
runs only in production, under load, half-way through a request; a new
instance runs the one path every test ran.
[0011](../decisions/0011-config-source-versions-and-secret-delivery.md) is
the decision.

## 7. A document says which version it is; a binary reads two

A configuration document **may** carry, at its root,

```yaml
apiVersion: <group>/<kind>/v<N>
```

— a group (a DNS-like name the kind belongs to), the kind of document, and a
version. **Absent means v1**, so every document written before this rule is a
v1 document and none has to change. The envelope
([`schemas/service.json`](../../schemas/service.json)) declares the key and
its form; a kind's own schema may narrow it to its own `const`.

**Each version has its own authored schema** (rule 2 holds for each), and the
loader chooses by the version the document states, **before** it validates
anything:

| The document says | The loader |
|---|---|
| N, the version the binary is written against | validates against N, decodes |
| N-1, and the binary declares it reads N-1 | validates against N-1, converts with the binary's upgrade, validates the result against N, decodes |
| newer than N | refuses: `apiVersion: v3 is newer than this binary reads (v2, v1)` |
| older than the oldest the binary reads, or absent when that is v2 or later | refuses, naming the key |
| another kind, or not of the form | refuses, naming the key |

**A binary accepts N and N-1, and no more.** N-1 is what lets a version change
be two ordinary deployments instead of one synchronised one: the binary that
reads N ships first and keeps reading the N-1 file it is given, then the file
moves to N. A rollback runs the same steps backwards — the file first, then
the binary — because the binary before it does not read N. Two and not every
version, because a binary that read every version would carry every
conversion forever, and nobody could ever say which documents are still in
use.

**The conversion is the service's**, declared with the kind it converts, and
works on the document rather than on a type: it is handed the N-1 document,
already valid against N-1, and returns an N one. The loader then sets
`apiVersion` to N and validates the result against N's schema, so a
conversion that produces something N does not accept is refused at start-up,
named as the conversion's, rather than decoded.

A document validated against a schema it was not written for is the failure
this prevents. Without the version, a v2 file mounted beside a v1 binary is
checked against v1's schema, and either fails with errors about keys that are
perfectly good in v2, or — worse — passes, because the keys that changed
meaning kept their names.

## Conformance

| Rule | Mechanism |
|---|---|
| 1. one schema, two readers | the chart's tests validate the rendered configuration against the binary's schema |
| 2. no drift | a test regenerates the schema from the type and diffs it against the committed file |
| 3. strictness | a negative fixture per schema: an unknown key must fail |
| 4. shared shapes | review, and the fragment `$ref`s in the schema |
| 5. secrets | review; the loader has no way to read a secret from the file; the database fragment's negative fixtures (a password in the URL's user information, and in its query) |
| 6. start-up failure | a test that starts the binary with each invalid fixture |
| 6. read once | review — unchecked |
| 7. versions | the shared case table [`config/testdata/versions/cases.json`](../../config/testdata/versions/cases.json): every version path and every refusal |

The [Go loader](../../config) and the [TypeScript loader](../../ts) implement
rules 1, 5, 6 and 7 so that a service does not have to, and
[`conformance`](../../conformance) is what a chart's tests use to be held to
the same schema. The [Python](../../python) and [Kotlin](../../kotlin)
loaders implement rules 1, 5 and 6, and only the plain loader's half of rule
7: they read v1 and refuse a document that names a later version. That is a
gap, stated rather than hidden: a service in either language cannot yet read
two versions.

The Go and TypeScript loaders also say **where the file is** (rule 1 of
[service.md](service.md)): `config.PathFrom(os.Args[1:], "<APP>_CONFIG")` in
Go and `configPath(process.argv.slice(2), "<APP>_CONFIG")` in TypeScript,
both held to the shared table
[`config/testdata/path-from.json`](../../config/testdata/path-from.json).

The two are tested against **the same fixtures**, in the same directory. That
is the point rather than an economy: two loaders that claim to implement one
contract must refuse the same documents and say something a person can act on
when they do. A fixture only one of them sees is a contract that exists
twice. They load, validate, decode, and stop. They are deliberately
not a framework: no lifecycle, no dependency wiring, no HTTP, no reflection
over the environment.
