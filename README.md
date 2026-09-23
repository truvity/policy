# policy

The engineering contracts a service is held to: how it is configured, how it
starts and stops, how it is built and released — and one worked example that
runs them.

A contract here is short, normative and testable. The reasoning lives beside
it as a decision record, the allowed libraries as a canon, and the whole
thing is proven by an example service in this repository rather than by
assertion.

## What ships

| Artifact | Where | Status |
|---|---|---|
| the contracts, canons, guides and decisions | `docs/` | being assembled |
| configuration schemas (JSON Schema) | [`schemas/`](schemas/README.md) | shipped |
| the Go configuration loader and conformance helpers | `config/`, `conformance/`, module `github.com/truvity/policy` | shipped |
| the TypeScript loader | `ts/`, package `@truvity/policy` | shipped |
| the shared lint configuration | `lint/` | planned |
| a local cluster recipe for the example's tests | `hack/kind/` | planned |
| the worked example, a URL shortener | `examples/url-shortener/` | planned |

One tag stamps all of them: `vX.Y.Z` releases the documents, the schemas, the
Go module, the TypeScript package and the example's charts and images
together. A consumer pins one version.

## Who it is for

A team running services on Kubernetes that wants the same shape in every
repository and every language, without a framework in the way. The contracts
bind at the process boundary — a configuration file, a set of probe
endpoints, a log format, a signal — so a Go service, a Node service and a JVM
service satisfy them the same way and share no code.

It deliberately does not ship a framework, a dependency-injection container,
a code generator or a base image. The example is the reference
implementation; there is nothing to import in order to conform.

## The model

Three things, in order of how much they bind:

- **Contracts** (`docs/contracts/`) are normative. Each one is short enough
  to read in a sitting and carries a conformance section saying how it is
  checked mechanically.
- **Canons** (`docs/canon/`) are the allowed lists: which libraries, which
  toolchain versions, which build tools, and the scope they apply to.
- **Guides** (`docs/guides/`) are how to satisfy the contracts, walking the
  example.

Decisions (`docs/decisions/`) record why a contract reads the way it does.
They are never edited after acceptance; they are superseded by a later one
that links back.

## Documentation

[`docs/`](docs/README.md) is the index, by audience: authoring a service,
reviewing one, operating one.

## The rule that makes this repository public

Mechanism only. Nothing here names an organisation, an account, a cluster, a
hostname, an environment, a team, a person, an internal repository or a
ticket. Every such thing is an input with a neutral default, supplied by the
consuming estate from its own private repository. `hack/leak-canary.sh`
enforces the mechanical part of that rule on every commit and in CI; the rest
is a review rule.

## Status

Pre-release: the contracts are being assembled and nothing is tagged yet.
Used in production by its maintainers once `v0.1.0` ships.

## Development

```sh
devbox shell   # or direnv, which does it on cd
just check     # the gate: exactly what CI runs
```

A service loads its configuration in three lines:

```go
var cfg Config
if err := config.Load(path, schemaBytes, &cfg); err != nil {
    return err // names the file and every failing key
}
```

`just check` needs nothing but this checkout — no network, no containers, no
cluster. Recipes that need more will be added as their own CI jobs and named
as such.

## Releasing

Manual tags for now: every first release, minor and major is a tag pushed by
a person after the CHANGELOG heading for that version has merged. Automatic
patch releases are not armed.

## Licence

MIT. See [LICENSE](LICENSE).
