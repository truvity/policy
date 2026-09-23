# 0004 — This repository is public

**Status:** accepted

## Context

These contracts describe how a service is configured, started, observed and
released. They contain no secrets and no particulars: every environment,
cluster, account and hostname they touch is an input with a neutral default,
because that is rule one of the contracts themselves.

The repository could still have been private. The argument for private is
that a contract is an internal matter and publishing invites having to
explain it. The arguments against turned out to be stronger:

- **A private repository is a token in every consumer.** Teams in more than
  one organisation consume these contracts, and a private dependency means
  an installation token, its rotation, and a build that fails for a stranger
  who clones the example.
- **The example must run.** The worked example is the proof that the
  contracts are complete, and an example that only the authoring estate can
  run proves that only the authoring estate can satisfy them.
- **Public is a forcing function on the rules.** The discipline that nothing
  may name a particular is the same discipline that makes a contract
  portable. A private repository would have let a hostname sit in a document
  for a year, and with it the assumption that everyone reading has that
  host.
- **The components these contracts govern are already public.** Their
  documentation would otherwise reference a repository their readers cannot
  open.

## Decision

**This repository is public, under MIT.** Every particular is an input. A
canary enforces the mechanical part of that on every commit and in CI, and
the rest is a review rule that covers documents, tests, commit messages and
pull request text.

Its CI runs on hosted runners only and holds no credentials.

## Consequences

### Good

- Anyone can clone the example and run it, which is what makes it evidence.
- No token, no installation, no rotation for any consumer.
- The rule against particulars is enforced rather than remembered, and it
  keeps the contracts portable as a side effect.

### Bad

- **History cannot be unpublished.** A particular that reaches a push is
  public permanently; a rewrite changes the hashes and not what was already
  fetched. This is the whole reason the canary runs on commit rather than
  only in CI.
- Prose costs more. Every incident that earned a rule has to be told without
  its particulars, which takes longer to write and reads less vividly.
- Issues, discussions and pull requests are visible, so a half-formed idea is
  visible too.

### Neutral

- The consuming estate's own configuration stays private, which is where
  particulars belonged anyway.
