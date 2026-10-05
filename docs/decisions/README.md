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

| # | Status | Decision |
|---|---|---|
| [0001](0001-no-di-containers.md) | accepted | Application graphs are hand-wired; dependency-injection containers are not used |
| [0002](0002-config-file-plus-env.md) | accepted; amended by 0011, 0012 | Configuration is a file; the environment carries secrets (secret delivery and reloading amended by 0011) |
| [0003](0003-schemas-not-generators.md) | superseded by 0010 for data contracts | A schema and a hand-written type, not a code generator (superseded for data contracts by 0010) |
| [0004](0004-policy-is-public.md) | accepted | This repository is public |
| [0005](0005-kind-is-the-gate.md) | accepted | A local cluster is the gate; the real one is a consumer's |
| [0006](0006-telemetry-is-the-sdk-environment.md) | accepted | Telemetry is configured by OpenTelemetry's own environment |
| [0007](0007-no-mesh-identity-in-process.md) | accepted | No service mesh; identity is the account, terminated in process |
| [0008](0008-twelve-factor-is-a-map-not-a-label.md) | accepted | The twelve factors are a map to read this by, not a label to claim |
| [0009](0009-charts-pass-config-through-and-share-a-library.md) | accepted | Charts pass configuration through verbatim; the platform pieces come from a library chart |
| [0010](0010-data-contracts-are-written-in-pkl.md) | accepted | Data contracts are written once in Pkl and generated; the program never is (supersedes 0003 for data contracts) |
| [0011](0011-config-source-versions-and-secret-delivery.md) | accepted; amended by 0012 | Configuration is immutable per instance, versioned (a binary reads N and N-1), and found the same way on every platform; secrets arrive by name as a variable, a file or a source read at start-up (amends 0002) |
| [0012](0012-stabilization-amendments.md) | accepted | Documents carry a product group; secrets are named and resolved through one source; a service may read a policy document; presets are defined; a stabilizing product may break in a minor (amends 0002, 0011) |
