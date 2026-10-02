# policy

The engineering contracts a service is held to: how it is configured, how it
starts and stops, how it is built and released — and one worked example that
runs them.

A contract here is short, normative and testable. The reasoning lives beside
it as a decision record, the allowed libraries as a canon, and the whole
thing is proven by an example service in this repository rather than by
assertion.

## What ships

| Artifact | Where | Published as |
|---|---|---|
| the contracts, canons, guides, decisions and [glossary](docs/glossary.md) | [`docs/`](docs/README.md) | the tagged source |
| configuration schemas (JSON Schema) | [`schemas/`](schemas/README.md) | the tagged source, and inside every loader |
| the Go configuration loader and conformance helpers | `config/`, `conformance/` | Go module `github.com/truvity/policy` |
| the TypeScript loader | [`ts/`](ts/package.json) | `@truvity/policy` on GitHub Packages |
| the Python loader | [`python/`](python/README.md) | a wheel attached to each GitHub Release |
| the Kotlin loader | [`kotlin/`](kotlin/README.md) | not published: built and tested here, used from a checkout |
| the shared lint configuration | [`lint/`](lint/README.md) | the tagged source, copied into a repository |
| the local cluster the example is tested on | [`hack/kind/`](hack/kind/README.md) | the tagged source, fetched at a pinned tag |
| the worked example, a URL shortener | [`examples/url-shortener/`](examples/url-shortener/README.md) | charts `url-shortener`, `url-shortener-infra` and `url-shortener-e2e` under `oci://ghcr.io/truvity/charts`; images under `ghcr.io/truvity/policy/url-shortener/` |

One tag stamps all of them: `vX.Y.Z` releases the documents, the schemas, the
Go module, the TypeScript package, the Python wheel and the example's charts
and images together. A consumer pins one version.

## Who it is for

A team running services on Kubernetes that wants the same shape in every
repository and every language, without a framework in the way. The contracts
bind at the process boundary — a configuration file, a set of probe
endpoints, a log format, a signal — so a Go service, a Node service and a JVM
service satisfy them the same way and share no code.

The [component contract](docs/contracts/component.md) extends the same idea
to the repositories that ship mechanism rather than services — charts,
images, Go libraries, CLIs, actions — so that every public repository an
estate consumes looks the same from the outside.

It deliberately does not ship a framework, a dependency-injection container,
a code generator or a base image. The example is the reference
implementation; there is nothing to import in order to conform.

This repository is itself public, for the reasons in
[0004](docs/decisions/0004-policy-is-public.md); a private repository
consumes the same contracts and the same libraries, with a different gate
and a different release destination — see
[docs/guides/private-consumer.md](docs/guides/private-consumer.md).

## The model

Three things, in order of how much they bind:

- **Contracts** (`docs/contracts/`) are normative. Each one is short enough
  to read in a sitting, carries a version, and has a conformance section
  saying how it is checked mechanically.
- **Canons** (`docs/canon/`) are the allowed lists: which libraries, which
  toolchain versions, which build tools, and the scope they apply to.
- **Guides** (`docs/guides/`) are how to satisfy the contracts, walking the
  example.

Decisions (`docs/decisions/`) record why a contract reads the way it does.
They are never edited after acceptance; they are superseded by a later one
that links back. [The glossary](docs/glossary.md) pins down the words the
documents use narrowly — estate, platform, ring, tier, lane.

## Install and a worked example

Pin one version; every artifact carries the same one. v1.29.0 is the latest
tag at the time of writing — the
[releases page](https://github.com/truvity/policy/releases) lists every one.

```sh
go get github.com/truvity/policy@v1.29.0

# GitHub Packages: map the @truvity scope to npm.pkg.github.com, with a
# token that can read packages — the registry asks for one even here.
yarn add @truvity/policy@1.29.0

uv add https://github.com/truvity/policy/releases/download/v1.29.0/truvity_policy-1.29.0-py3-none-any.whl
```

A service loads its configuration in three lines, and a misconfiguration is
refused before anything is constructed:

```go
var cfg Config
if err := config.Load(path, schemaBytes, &cfg); err != nil {
    return err // names the file and every failing key
}
```

The worked example runs every contract end to end, and its application
chart renders from the release with only the names it refuses to guess —
the database's host, the root its certificate chains to and the two Secrets
holding its credentials, the broker, the bucket:

```yaml
# values.yaml
database:
  host: pg
  tls:
    rootCA:
      configMapName: pg-root-ca
  owner:
    passwordSecret: owner
  app:
    passwordSecret: app
events:
  url: nats://nats:4222
archive:
  bucket:
    name: archive
```

```sh
helm template example oci://ghcr.io/truvity/charts/url-shortener --version 1.29.0 -f values.yaml
```

What provides those names is the infrastructure chart's job, installed per
install beside it ([platform.md §11](docs/contracts/platform.md));
[the example's README](examples/url-shortener/README.md) walks what each
component proves.

## Consumers

The repositories that consume a release, and through which surface. No
versions: each consumer's own pin is the record of that.

| Consumer | Surface |
|---|---|
| `truvity/ci-actions` | the `cluster` action fetches `hack/kind/` at a pinned release tag |
| 14 repositories | copy [`lint/golangci-depguard.yaml`](lint/golangci-depguard.yaml) into their `.golangci.yaml` |
| `truvity/gitops` | deploys the url-shortener example's charts |
| every public component repository | is held to [the component contract](docs/contracts/component.md) |

## Neighbours

- **[ci-workflows](https://github.com/truvity/ci-workflows)** runs this
  repository's gate (`check.yaml`), its release (`release-public.yaml`) and
  its cluster lane (`integration.yaml`, whose kind tier stands up
  `hack/kind/` at the `policy-version` it is given).
- **[ci-actions](https://github.com/truvity/ci-actions)** holds the `cluster`
  action that fetches `hack/kind/` from a policy release; the box is defined
  here and nowhere else.
- **[gemaal](https://github.com/truvity/gemaal)** provides the `harness`
  package the example's end-to-end suite reaches every Service through.
- **[ocictl](https://github.com/truvity/ocictl)** provides `helmctl`, which
  bakes image digests into the example's charts at release.
- Every other public component repository is held to the contracts here;
  [component.md](docs/contracts/component.md#applies-to) lists them.

## Documentation

[`docs/`](docs/README.md) is the index, by audience: authoring a service,
authoring a component, reviewing one, operating one.
[`docs/landscape.md`](docs/landscape.md) is the other direction: one page
naming every public repository the contracts below apply to today, and how
they fit together. The contracts:

- [repository](docs/contracts/repository.md) — layout, toolchain, the gate
- [service](docs/contracts/service.md) — the process boundary
- [config](docs/contracts/config.md) — one typed configuration per binary
- [platform](docs/contracts/platform.md) — what a service asks of what runs it
- [release](docs/contracts/release.md) — one tag, what a version means
- [component](docs/contracts/component.md) — the rules C1–C14 for every
  public repository that ships charts, images, libraries, CLIs or actions

## The rule that makes this repository public

Mechanism only. Nothing here names an organisation, an account, a cluster, a
hostname, an environment, a team, a person, an internal repository or a
ticket. Every such thing is an input with a neutral default, supplied by the
consuming estate from its own private repository. `hack/leak-canary.sh`
enforces the mechanical part of that rule on every commit and in CI; the rest
is a review rule.

The one exception is [Consumers](#consumers): it names a consuming
repository and the surface it uses, and never a version, an environment or
a cluster ([component.md C8](docs/contracts/component.md)).

## Status

Used in production by its maintainers. Released from `v0.1.0`; tags follow
semver from `v1.25.0` ([release.md §2](docs/contracts/release.md)). Every
contract is at version 1.0, effective 2026-09-29.

What is not done: the Kotlin loader is built and tested but not published,
and two guides — keys and signing, and migrating off a framework — arrive
with the component that proves them
([guides/README.md](docs/guides/README.md)).

## Development

```sh
devbox shell   # or direnv, which does it on cd
just check     # the gate: exactly what CI runs
just golden    # regenerate the chart goldens — read the diff first
```

`just check` needs no credentials, no container runtime and no cluster.
The cluster tier needs a container runtime, stands up a local cluster
([`hack/kind/`](hack/kind/README.md)) and installs the example from the
same artifacts a release publishes:

```sh
just cluster          # the box: about a minute from nothing
just cluster-verify   # ask each server a real question
just cluster-all      # the whole tier, from nothing, as CI runs it
```

[AGENTS.md](AGENTS.md) is the entry point for anyone — or anything —
arriving without context: what to read first, what the gate checks that
surprises people, and the separate set of rules for bringing another
repository to this shape.

## Releasing

Manual tags: every first release, minor and major is a tag pushed by a
person after the CHANGELOG heading for that version has merged
([release.md §4](docs/contracts/release.md)). Automatic patch releases are
cut by a bot and need no CHANGELOG heading: their notes are the GitHub
release's, and the next hand-cut heading covers them.

## Licence

MIT. See [LICENSE](LICENSE).
