# 0008 — The twelve factors are a map to read this by, not a label to claim

**Status:** accepted

## Context

The twelve-factor methodology, and the fifteen-factor extension of it, are
the common vocabulary for what these contracts describe. Someone arriving
here has almost certainly read one of them, and an agent working from these
documents has certainly been trained on both.

The question is whether to adopt the label. Against it: **these contracts
deviate from the twelve in two places on purpose**, and one of the two is the
factor most people remember.

- **Configuration is a file, not the environment.** Factor III says store
  configuration in the environment. [0002](0002-config-file-plus-env.md) says
  a file, validated against a schema, with the environment carrying secrets
  and nothing else. The reasoning is in that record; the short version is
  that the environment cannot express structure, cannot be typed, and cannot
  be validated before the process starts.
- **A service binds two ports.** Factor VII says export services via port
  binding, and is usually read as one. [service.md](../contracts/service.md)
  rule 3 puts probes on a listener of their own, because readiness must be
  answerable when the main listener is saturated, and because a probe
  endpoint on a routed port is reachable by people who were never meant to
  reach it.

Claiming the label and then explaining two exceptions is worse than not
claiming it: the exceptions are what a reader needs, and they arrive as
footnotes to a badge.

The fifteen-factor additions are a different case. All three — API first,
telemetry, authentication and authorisation — are already mandatory here,
and were before anyone checked the correspondence.

## Decision

**These contracts do not claim to be twelve-factor or fifteen-factor.** They
carry a guide, [twelve-factor.md](../guides/twelve-factor.md), that maps each
factor to the rule that covers it, states plainly where the contracts differ
and why, and points at the record that argues each difference.

The map is documentation, not a conformance target. **Where a factor and a
contract disagree, the contract wins**, and the disagreement is written down
rather than resolved in favour of the more famous document.

## Consequences

### Good

- A reader who knows the vocabulary can find any factor's answer in one
  table, including the two answers that are "not this, and here is why".
- The deviations are argued where they are visible, instead of being
  discovered by someone who assumed the label meant what it usually means.
- Nothing has to be bent to keep a claim true. The claim was the only thing
  that would have made "configuration is a file" awkward to state.

### Bad

- **A question that could be answered with one word now takes a page.** "Are
  you twelve-factor?" is a real question with a real purpose, often from
  someone assessing a system quickly, and the honest answer is longer than
  the useful one.
- The guide is a third place describing the same rules, after the contracts
  and the canon, and it can fall out of step with them. It is short for that
  reason, and it links rather than restates.

### Neutral

- The three fifteen-factor additions needed no change, which is mild evidence
  that the contracts were derived from operating services rather than from
  the earlier document.
