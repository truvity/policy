# Guides

Not normative. The contracts say what must be true; these say how, and point
at the file in the worked example where it is done.

Each guide has the same shape: the rule in a paragraph, why it is that way in
another, a table of where to look per language, and the traps — the failures
that look like something else.

## By aspect

| Guide | Covers |
|---|---|
| [configuration.md](configuration.md) | one file, one schema, secrets by name |
| [rpc.md](rpc.md) | which boundary is an RPC, one handler for three protocols |
| [identity-and-secrets.md](identity-and-secrets.md) | the account a workload runs as, and how a secret reaches it |
| [events.md](events.md) | publishing and consuming, and how a client authenticates |
| [probes-and-rollout.md](probes-and-rollout.md) | the two endpoints, and replacing an instance without a gap |
| [logging-and-telemetry.md](logging-and-telemetry.md) | one stream, one level, and where traces go |
| [exposure.md](exposure.md) | the route, its parent, and why its rules are named |
| [releases-and-images.md](releases-and-images.md) | one tag, every platform, no build step in the image |
| [testing.md](testing.md) | what the gate proves, and what only a cluster can |
| [transport-security.md](transport-security.md) | mutual TLS, whose identity, and the three modes |
| [twelve-factor.md](twelve-factor.md) | the factors mapped, and the two deviations argued |
| [conformance.md](conformance.md) | the checklist, and what CI checks for you |

## Arriving with their components

The example is built one component per language, and a guide lands when the
component that proves it does. Writing one earlier would mean describing code
nobody has run.

| Guide | Arrives with |
|---|---|
| object storage | the log component, which is the first to read a bucket |
| keys and signing | the first component that signs something |
| migrating from a framework | written from the change that removes one |

## The languages

| Language | Canon | In the example |
|---|---|---|
| Go | [canon/go.md](../canon/go.md) | the migration, the redirect service, the counter |
| TypeScript | [canon/node.md](../canon/node.md) | the loader package; the front end follows |
| Kotlin | [canon/kotlin.md](../canon/kotlin.md) | the counter, once it is rewritten |
| Python | [canon/python.md](../canon/python.md) | the [loader](../../python/) and the [archiver](../../examples/url-shortener/log/) |

A cell reading "follows" means the rule holds for that language and the
example does not yet demonstrate it. That is stated rather than left blank,
because a blank cell reads as "not applicable" and none of these are.
