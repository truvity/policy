# Go canon

The approved library for each area. Anything not listed is a choice nobody
has had to make yet; make it, then add the row.

**Scope:** every Go service or library held to these contracts.

**Direct imports only.** A transitive dependency on a retired library is not
a violation — it is somebody else's dependency, and chasing it produces
`replace` directives and forks. The rule is about what a repository imports
itself.

## Canon

| Area | Library | Notes |
|---|---|---|
| CLI | `urfave/cli/v3` | |
| HTTP server | `gofiber/fiber/v3`, with `huma/v2` on top where an OpenAPI surface is wanted | |
| HTTP client | stdlib `net/http` | |
| RPC | `connectrpc.com/connect` | one handler serves Connect, gRPC and gRPC-Web; in the cluster a client speaks gRPC ([service.md §8](../contracts/service.md)). `google.golang.org/grpc` is tolerated as the transport inside a third-party SDK, never for a server of our own |
| YAML | `go.yaml.in/yaml/v3` | the maintained continuation of the archived `gopkg.in/yaml.v3`; a drop-in |
| JSON Schema | `santhosh-tekuri/jsonschema/v6` for validation, `invopop/jsonschema` to derive a schema from a type in a drift test | the second is a test dependency, not a run-time one |
| JWT and JOSE | the HTTP framework's own middleware where it fits; `lestrrat-go/jwx` underneath and everywhere else | `golang-jwt` and direct `go-jose` retire at next touch |
| Logging | stdlib `log/slog` | JSON to stderr at one level. Every library that logs is wired to it at the composition root ([service.md §4](../contracts/service.md)) — the ORM in particular has its own format, its own colours and its own level, and ignores the one the configuration set |
| Telemetry | OpenTelemetry Go SDK | configured by its own environment ([0006](../decisions/0006-telemetry-is-the-sdk-environment.md)). `prometheus/client_golang` only where a third-party component bakes it in |
| Retry and backoff | `cenkalti/backoff/v5` | |
| Circuit breaking, bulkheads, rate limiting | **no canon yet** | see below |
| Testing | `stretchr/testify`, `neilotoole/slogt/v2`, `pgregory.net/rapid` | rapid where the property is the point |
| Errors | stdlib wrapping | |
| Database | driver `jackc/pgx/v5`; `gorm.io/gorm` where an ORM is wanted | different layers, not alternatives |
| Key-value and cache | `redis/go-redis/v9` | the protocol client; `alicebob/miniredis/v2` is the test fake |
| Object store and cloud | `aws-sdk-go-v2` | the S3 API, not necessarily the vendor: an endpoint is configuration ([config.md](../contracts/config.md)) |
| Kubernetes | `client-go`; `controller-runtime` in an operator only | |
| Wiring | **a hand-written composition root** | no container. [0001](../decisions/0001-no-di-containers.md) |
| Configuration | a file plus a schema, through this repository's loader | [config.md](../contracts/config.md), [0002](../decisions/0002-config-file-plus-env.md) |
| Common | `google/uuid` | |

## The empty row is deliberate

A previous version of this canon named a single resilience library covering
retry, circuit breaking, timeouts, bulkheads and rate limiting. It was a good
library. Weeks later **no repository had imported it**, while an ordinary
backoff library was in real use in several — so the canon's most confident
row was its least true one.

A canon entry nobody follows costs more than a missing entry, because it
teaches readers that the document describes intentions rather than practice,
and that doubt spreads to the rows that are true. So the row that is in use
says what is in use, and the row that has no adopter says it has none. The
first repository that genuinely needs a circuit breaker chooses one, and
fills the row in the same change.

## Exceptions

An exception is **named, in this document, with its reason**. Two exist as
patterns rather than as a list:

1. **A framework with its own opinions, adopted wholesale.** An operator
   built on the Kubernetes builder ecosystem follows that ecosystem —
   its logger, its metrics registry, its YAML library — because half-adopting
   a framework is worse than either choice. The exemption covers the whole
   repository and is stated in its README.
2. **A third-party SDK's transport.** An SDK that speaks gRPC brings gRPC.
   That is tolerated transitively and never imported directly.

Anything else is a change to the table above.
