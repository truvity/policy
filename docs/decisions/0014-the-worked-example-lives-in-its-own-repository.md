# 0014 — The worked example lives in its own repository

**Status:** accepted · 2026-10-22

## Context

The worked example, a URL shortener, was a nested Go module in this
repository, together with the TypeScript, Python and Kotlin configuration
loaders it was the only consumer of. That put two things in one place that
share nothing but a history. This repository's releases are the contracts, the
schemas, the Go module and the library chart; the example's releases are its
own charts and images. One tag stamped both, so a documentation fix cut a new
version of an application, and most of the commits to this repository were to
the example.

## Decision

**The example and the three loaders moved to a repository of their own,
[truvity/url-shortener](https://github.com/truvity/url-shortener).** Its Go
module is `github.com/truvity/url-shortener`; the loaders keep their
directories (`ts/`, `python/`, `kotlin/`) at its root. It depends on this
repository's Go module and the `service-lib` chart by released version, like
any other consumer, and releases its charts and images on its own tags.

**This repository keeps what the contracts are made of:** the documents, the
schemas and their fixtures under `config/testdata`, the Go packages, the
library chart `charts/service-lib`, the lint configuration and the local
cluster definition (`hack/kind/`). It stops publishing the example's charts and
images, the npm package and the Python wheel; versions published before the
move stay where they are.

**The guides keep teaching through the example,** by absolute link into the
new repository.

**A downstream canary keeps the one thing the split would lose.** The
`example-canary` job in this repository's CI builds and tests the example's
default branch against the pull request's checkout of this module, so a
contract change that breaks the example is seen in the pull request that made
it. It is advisory: a change that must break the example cannot wait for the
example to adopt a release that does not exist yet. A job that is advisory can
be made required by adding it to the `needs` list of the `check` job.

## Consequences

### Good

- A release here is about the contracts, and a release there is about an
  application. Each has the cadence it needs.
- The example is held to the same rules as any other consumer: it pins a
  released version and adopts a new one in a pull request.

### Bad

- A change to a contract no longer changes the example in the same pull
  request. The canary reports the break; the fix lands in the other
  repository afterwards.
- The loaders are tested against fixtures and schemas they now get from a
  released version of this module, so a change here that breaks a loader is
  seen in the example's own dependency bump, with the canary as the early
  warning.
- Prose in the guides and contracts points at a repository that moves
  independently, so a link into it can go stale without this repository's
  link check noticing.
