# Kotlin and JVM canon

Written from the JVM service the adopting estate already runs, not from a
survey of what is popular. That order is deliberate: a canon assembled from a
survey describes nobody's service; one assembled from a service that had to
work describes the choices that actually came up.

The first JVM component held to these contracts is a stream consumer, and it
inherits this list rather than proposing a second one.

## Canon

| Area | Library | Notes |
|---|---|---|
| Language | Kotlin, JVM 21 target, `-Xjsr305=strict` | strict JSR-305 so platform types from Java libraries are nullable, which is where the null errors are |
| Build | Gradle, Kotlin DSL | a version catalog once there is a second module to share versions with |
| Application framework | Spring Boot | the estate's existing JVM service is a Spring Boot service; a second framework for one component is how two become permanent |
| HTTP client | the framework's client | |
| RPC | Connect's Kotlin library, as a **client** | see the constraint below |
| JSON | Jackson, with the Kotlin and JSR-310 modules | not kotlinx.serialization: the framework's own conversion, its error handling and every Java library in reach already speak Jackson |
| Persistence | the framework's JPA integration over Hibernate | not Exposed; the same reason |
| Migrations | Flyway | versioned SQL, applied by the migration job of the service contract, never at start-up |
| Security | the framework's OAuth2 resource server | a JWT is verified against the issuer's keys, never parsed by hand |
| Configuration | a file plus a schema, through this repository's loader | [config.md](../contracts/config.md), [0002](../decisions/0002-config-file-plus-env.md) |
| Logging | the framework's SLF4J binding, configured to emit JSON to **stderr** at one level | [service.md §4](../contracts/service.md) |
| Telemetry | the OpenTelemetry Java agent or SDK, configured by its own environment | [0006](../decisions/0006-telemetry-is-the-sdk-environment.md) |
| Transport identity | the framework's SSL bundles, with reload on update | see below |
| Testing | JUnit 5, and the framework's test support | |
| Cloud | the cloud vendor's v2 SDK | an endpoint is configuration ([platform.md §4](../contracts/platform.md)) |

## Why this inherits rather than chooses

Every row above is what the estate's existing JVM service already uses. The
alternative — a lighter server, a Kotlin-native serialiser, a Kotlin-native
SQL library — would be defensible for a new service in isolation and is not
defensible here: it would mean two JVM stacks, one of which nobody on the
team has run in production, and the second one always wins the next argument
by being newer.

The Kubernetes shape is **not** inherited. It comes from
[service.md](../contracts/service.md), and a Spring Boot service satisfies it
like any other: one configuration file validated against a schema, secrets
from the environment, probes on their own listener at `/health/live` and
`/health/ready`, JSON logs on stderr, a drain on `SIGTERM`, a version read
from the build, and a runtime image with no build step in it — a jar copied
onto a JRE base, every platform in one job.

[0001](../decisions/0001-no-di-containers.md) applies at the boundary it
applies everywhere: the framework may wire itself however it likes, and the
service's own dependencies are constructed where they can be read.

## Transport identity

[service.md §10](../contracts/service.md) asks three things of a service:
present a mounted identity, reload it without restarting, and check a peer's
against a list. Spring Boot's SSL bundles do the first two — a bundle points
at the mounted files and reloads them when they change, which is what makes
a one-hour certificate lifetime survivable without a rollout.

The third is a few lines: the peer's identity is an entry in the
certificate's subject alternative names, read from the authenticated
principal and compared against the allow-list from the configuration file.

## One constraint worth knowing in advance

The Connect ecosystem's Kotlin library is a **client** library: it generates
clients for the Connect, gRPC and gRPC-Web protocols, and no servers. A JVM
service that consumes an RPC boundary is well served by it. A JVM service
that *owns* one serves gRPC with a JVM gRPC stack, which Connect clients in
every other language reach over their gRPC transport, and which a browser
reaches through a gateway's gRPC-Web filter.

That is a real constraint on where a JVM service sits in a topology, and it
is better known before the first one is written than after.

## Exceptions

A service that needs something not on this list adds a row here first, with
the reason, in the same change that introduces it. A row added afterwards is
a migration that already happened.
