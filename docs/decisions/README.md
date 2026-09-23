# Decisions

Why the contracts read the way they do.

A decision here is one that a stranger adopting these contracts would also
face. A decision that applies to a single deployment belongs with that
deployment.

Decisions are **never edited after acceptance**. A decision that turns out to
be wrong is superseded by a later one that links back to it, so that the
record of what was believed, and when, survives. That matters more than
tidiness: most bad rules are re-proposed by someone who was not there, and a
superseded decision is the only thing that answers them.

Each one states the context, the decision, and the consequences — including
the ones that are bad. A decision record with no negative consequences is a
sales pitch.

| # | Decision |
|---|---|
| [0001](0001-no-di-containers.md) | Application graphs are hand-wired; dependency-injection containers are not used |
| [0002](0002-config-file-plus-env.md) | Configuration is a file; the environment carries secrets |
| [0003](0003-schemas-not-generators.md) | A schema and a hand-written type, not a code generator |
| [0004](0004-policy-is-public.md) | This repository is public |
| [0005](0005-kind-is-the-gate.md) | A local cluster is the gate; the real one is a consumer's |
