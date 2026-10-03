# 0010 — Data contracts are written once in Pkl and generated

**Status:** accepted. Supersedes [0003](0003-schemas-not-generators.md) for
data contracts only.

## Context

[0003](0003-schemas-not-generators.md) kept the schema and the configuration
type in each language by hand, with a test that regenerates a schema from the
type and diffs it. It refused a generator because the generator it had in
mind read a manifest language and emitted the program: entry points, wiring,
templates. That refusal stands.

What it accepted along the way was a cost it called bad: the same shape is
written twice, in three languages four times, and defaults live in two places.
[0009](0009-charts-pass-config-through-and-share-a-library.md) added a third
and fourth copy without meaning to: the platform shape and each chart's
`values.schema.json` are composed from the same documents again. The drift
tests make the duplication safe, and they have kept it safe. They have not
made it cheap, and every new field is still an edit in several places that
must agree.

0003 left a door open: a generator for the type alone, with the schema still
authored by hand, was "a much smaller thing than what was rejected". The
question this record answers is the one that sentence did not: if the
duplication is paid down at all, why not pay it down at the root, with one
source that every copy is generated from?

The answer was measured rather than argued. A spike (described under
[Evidence](#evidence)) wrote the url-shortener's contracts in Pkl, a
configuration language with typed, constrained, documented classes, and
generated every copy from it. The finding that matters is not that it worked.
It is that the generator is small, and that it did not become the framework
0003 feared, because it was never asked to emit the program.

## Decision

**Data contracts are written once, in Pkl, and everything that restates them
is generated at build time.**

A *data contract* is a shape that more than one party must agree on:

- a service's configuration schema;
- a chart's values (`platform` and `config`);
- a deployment or target fragment.

From the one source the build generates the JSON Schemas (constraints kept),
the chart's `values.schema.json` and its defaults, the type in each language
(Go and Kotlin by Pkl's official generators; TypeScript with zod and Python
with pydantic by generators of ours), and the reference documentation.

**Pkl is a build-time tool and nothing else.** No service, chart, loader or
image contains it or evaluates it at run time. What ships is JSON Schema and
ordinary types.

**Program structure is never generated.** No entry points, no wiring, no
transport, no templates. The one thing a chart renders from a contract is the
`config` block, verbatim, as
[0009](0009-charts-pass-config-through-and-share-a-library.md) already says.
This is 0003's refusal, kept whole: it was always a refusal to generate the
program, and a schema is not the program.

**Generated files are committed and checked.** CI regenerates them and fails
on a diff, in the same pull request that caused it. A reader of a consuming
repository sees the schema and the type without installing anything.

**The generated JSON Schema is the validator, in every language.** A service
validates its file against the schema, as
[0002](0002-config-file-plus-env.md) says, and the generated type gives the
shape only. 0003's pairing of a schema and a type survives; both halves are
now generated, so the test that held a hand-written type to a schema has
nothing left to compare.

### The semantic rules

Pkl, JSON Schema and each language's validators disagree at the edges, and the
contract has to pick one meaning. These are the choices:

- **`null` is not a value for an optional field.** An optional field is
  absent or present with a value. Generated schemas and generators reject
  `null`. Pkl's own evaluation is lenient about it, so generated values never
  contain one.
- **An integral float is an integer.** `20.0` is valid where an integer is
  asked for, as JSON Schema says. pydantic is non-strict for integers.
- **A pattern must not admit a newline.** Regular-expression engines disagree
  on whether `$` matches before a final `\n`, so no pattern may rely on either
  answer, and a probe checks that none admits one.
- **A field with a default is optional.** Pkl cannot say both "required" and
  "has a default", and a default on a required field documented nothing. The
  one existing outlier, the web configuration's `faro.enabled`, becomes
  optional-with-default when it migrates.

### The vocabulary

**There are no ad-hoc constraints.** Every constraint in a contract comes from
a shared vocabulary of constrained aliases (a port, a duration, a host and
port, an image digest, an enum of log levels, and so on). A field that needs a
rule the vocabulary lacks adds it to the vocabulary, where it is reviewed once
and used everywhere.

Pkl's reflection does not expose a constraint, only that one exists. So each
alias states its constraint **twice**: the expression Pkl enforces, and an
annotation the generators read, both built from the same named constants so
that they cannot disagree about a number or a pattern. Where they could still
disagree, **generated probes** catch it: for each alias the boundaries, each
pattern's own valid and invalid examples, every enum member and a non-member,
and a value of the wrong type. A probe must give the same verdict in every
validator, Pkl's included.

The probes earned their place on the first run. `String.matches` in Pkl is a
*full* match; a JSON Schema `pattern` is a *search*. Four aliases accepted
less in Pkl than their schema said. The vocabulary now says what it means.

### Packaging and pinning

Contracts, the vocabulary and the generators live in **a contracts
repository**: a dedicated public repository of independently versioned Pkl
packages, in the manner of Pkl's own pantry, published over HTTPS (Pkl
packages are not OCI artifacts). This repository keeps the rules and the
reasons; the repository of contracts keeps the data they govern.

A consumer pins the **package versions and the Pkl version**. Pkl is
pre-1.0 and breaks between minors, so the pin is per repository and a bump
is a reviewed change. Renovate has no Pkl manager, so pin bumps come from
fleet automation, as they would for any other pin it cannot read.

### Rollout

**Shadow first.** For a time-boxed four to six weeks the generated artifacts
are produced beside the hand-written ones and diffed in CI. **The
hand-written schemas stay authoritative until the switch.** Nothing a
consumer depends on changes in this phase.

**Then one decision: switch or stop.** If the shadow phase shows the
generated artifacts agreeing, and the two costs the spike did not measure
(below) are acceptable, the generated files replace the hand-written ones and
the contracts that describe the hand-written rule are amended. If not, this
record is itself superseded and 0003 stands again for data contracts.

**The product-chart exception in
[0009](0009-charts-pass-config-through-and-share-a-library.md) is removed in
this phase.** A product chart's own values are a data contract too; they are
modelled in the same source rather than left hand-written, and what 0009 calls
the exception ends with it. The rest of 0009, passthrough and the library
chart, is untouched.

### Evidence

The spike is a draft pull request against this repository, titled "spike: Pkl
as the source of the url-shortener config contract" (`truvity/policy`, pull
request 166, branch `spike/pkl-contract`; not for merge). Its raw outputs are
under `spike/pkl/conformance/`. It found:

- **19 of 20 JSON Schemas identical** to the hand-written ones. The one
  difference is the `faro.enabled` case above.
- **7 generated chart `values.schema.json` pass `helm lint`.** They differ
  from the composed ones only in that a shared sub-schema is inlined instead
  of referenced; the constraints are the same.
- **214 fixtures through 13 validators** (four JSON Schema engines on the
  hand-written and on the generated schemas, zod, pydantic, Pkl itself, and
  Pkl decoded into the Go and Kotlin classes): **209 unanimous.** The five
  splits come from the three semantic gaps decided above, newline, `null` and
  integral float. There were **no disagreements between a hand-written and a
  generated schema.**
- **The generators total 768 lines (565 of code)**, against a budget of about
  three thousand.
- **About 1.6 seconds** to generate, warm.

Two caveats limit what this shows. The spike **bootstrapped its Pkl from the
existing JSON**, so "identical" proves the generator and vocabulary lose
nothing, not that a person writing Pkl first would land on the same schemas.
And two costs were **not measured**: what the generation step adds to a
consumer's build, and what it costs a newcomer to learn to author a contract.
They are the shadow phase's job.

## Consequences

### Good

- A shape is written once. A new field is one edit; the schema, the types, the
  defaults, the chart's values schema and the reference documentation follow,
  and cannot disagree.
- Constraints, which a hand-written type in most languages cannot carry, are
  in the schema every language validates with, and the probes hold the
  validators to one meaning.
- A deployment's own values can be written in the same typed language and fail
  at build time, with the rule that failed named.
- 0003's real objection, a generator that becomes a framework, is met by what
  the generator is: a few hundred lines, driven by one source, emitting data
  shapes and nothing else.

### Bad

- **A new language and a new toolchain to author in.** Pkl is a build-time
  dependency of every repository that authors a contract, and it is
  pre-1.0. The pin is per repository, and an upgrade is a change in each.
  Whether a newcomer can pick it up quickly is unmeasured.
- **Go and Kotlin types carry no constraints.** The official generators emit
  plain types; a range, a pattern and a cross-field rule are not in them. Nor
  does a required field with a default survive there. Kotlin's generator
  cannot emit a union at all. In those languages the schema is the only
  validator, as it is today, and a type that decodes is not thereby valid. We
  accept this, track upstream, and **do not write our own Go or Kotlin
  generators**. The Go binding also lags Pkl's releases, which constrains
  which Pkl version a consumer can pin.
- **Each constraint is written twice** in the vocabulary, as an expression and
  an annotation. The shared constants and the probes make that safe, not free.
- **A second repository, and a second set of pins.** Contracts are versioned
  independently of this repository, a consumer pins them, and the pin bump is
  fleet automation rather than a tool the ecosystem provides.
- **The spike does not prove the switch.** Everything above was measured on
  one estate's contracts, bootstrapped from the schemas they replace. That is
  why the rollout has a stop.

### Neutral

- 0003 is not wrong for what it described; it is superseded because its
  neutral note, a generator for the type alone, turned out to be the smaller
  half of a better answer. For anything that is not a data contract, and for
  the program, it still holds.
- RPC schemas keep generating clients and servers from the protocol's own
  generators, as 0003 already allowed.
