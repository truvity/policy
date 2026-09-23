# The twelve factors, mapped

The twelve-factor methodology and its fifteen-factor extension are the common
vocabulary for what these contracts describe. This page maps each factor to
the rule that covers it.

These contracts **do not claim the label**, and
[decision 0008](../decisions/0008-twelve-factor-is-a-map-not-a-label.md) says
why: there are two deliberate deviations, and a badge with two footnotes is
worse than a table. Where a factor and a contract disagree, the contract
wins, and the disagreement is here rather than resolved quietly.

## The twelve

| # | Factor | Here |
|---|---|---|
| I | Codebase | [repository.md §1](../contracts/repository.md): one product per repository, one tag line. |
| II | Dependencies | [repository.md §3](../contracts/repository.md): one manifest, a committed lock, one command to materialise it. Nothing fetched in a shell. |
| III | Config | **Deviates.** [0002](../decisions/0002-config-file-plus-env.md): a file validated against a schema; the environment carries secrets only. See below. |
| IV | Backing services | [platform.md §3, §4, §6](../contracts/platform.md): a store is an endpoint, a stream is found not made, a secret is a name. Swapping one is configuration. |
| V | Build, release, run | [release.md](../contracts/release.md): one tag stamps every artifact; [service.md §7](../contracts/service.md): a runtime image contains no build step. |
| VI | Processes | [service.md §2](../contracts/service.md): a hand-written composition root; state lives in the backing services of IV. |
| VII | Port binding | **Deviates, mildly.** [service.md §3](../contracts/service.md): two listeners, because probes must answer when the main one cannot. See below. |
| VIII | Concurrency | [service.md §9](../contracts/service.md): instances are replaceable and there is more than one; scaling is the deployment's. |
| IX | Disposability | [service.md §5 and §9](../contracts/service.md): drain on `SIGTERM`, with one drain constant the platform honours. |
| X | Dev/prod parity | [0005](../decisions/0005-kind-is-the-gate.md): the gate is a real cluster with the same operators, not a substitute. |
| XI | Logs | [service.md §4](../contracts/service.md): JSON on stderr at one level, as a stream. Routing is the platform's. |
| XII | Admin processes | [service.md §7](../contracts/service.md) and the example's migration: a one-off task is the same image with a different entrypoint. |

## The three additions

| Factor | Here |
|---|---|
| API first | [service.md §8](../contracts/service.md): the schema is in the repository and both sides are generated from it. |
| Telemetry | [service.md §4](../contracts/service.md) and [0006](../decisions/0006-telemetry-is-the-sdk-environment.md): logs as a stream, traces and metrics through the SDK. |
| Authentication and authorisation | [platform.md §7](../contracts/platform.md) for the human edge, [service.md §10](../contracts/service.md) for the transport between services. |

All three were already mandatory here.

## The two deviations, argued

### III — configuration is a file

The environment cannot express a list or a map without a naming convention
with indices in it, cannot be typed, and cannot be validated before the
process starts. A wrong variable is discovered by starting the service, which
under an orchestrator means a crash loop reported as "unhealthy" rather than
"you spelled the key wrong".

A file can be rendered by whatever deploys the service, checked against the
same schema the service uses, reviewed as a diff, and committed as a fixture.
Secrets are the opposite case and stay in the environment, because a file is
printed, mounted and committed.

The factor's real concern — that configuration must not be baked into the
build — is satisfied: the file is supplied at run time and differs per
deployment.

### VII — two ports

The factor says a service is self-contained and exports itself by binding a
port. It does. It binds a second one for probes, because readiness has to be
answerable when the first is saturated, and because a probe endpoint on a
routed port is reachable by people who were never meant to reach it.

This also makes the transport rule in
[service.md §10](../contracts/service.md) narrow: the probes port is the one
exemption from mutual TLS, and it is a port rather than a path.
