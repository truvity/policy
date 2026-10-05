# Documentation

The index, by what you are here to do.

## Authoring a service

- [contracts/repository.md](contracts/repository.md) — what a repository looks
  like: layout, toolchain, the gate, required checks, and what a public
  repository is held to on top.
- [contracts/service.md](contracts/service.md) — the process boundary:
  configuration, probes, logs, shutdown, version, images, and how services
  talk to each other.
- [contracts/config.md](contracts/config.md) — one typed configuration per
  binary, its schema, and how the chart is held to the same schema.
- [contracts/platform.md](contracts/platform.md) — the other side of the
  seam: what a service asks of whatever runs it, and what that platform owes
  back. Read it before writing a chart.
- [contracts/delivery-interface.md](contracts/delivery-interface.md) — the
  order in which the platform contract's menu grew, and the one number a
  chart declares so that a platform hands it only what it can take.
- [contracts/docs.md](contracts/docs.md) — how a product's documentation is
  organised, which of it is generated, and what an estate repository may
  restate.
- [guides/](guides/README.md) — how to satisfy the above, one guide per
  aspect, each pointing at the file in the worked example where it is done.

## Upgrading

- [how-to/upgrade/v1.45.md](how-to/upgrade/v1.45.md) — secrets by name through
  one declared source; `…Env` removed from the fragments and the library chart
  (**breaking**).

## Authoring a component

- [contracts/component.md](contracts/component.md) — every public
  repository that ships charts, images, Go libraries, Pulumi components,
  CLIs or actions: the rules C1–C17, how each is checked, and the forms
  they retired.
- [landscape.md](landscape.md) — the other direction: every public
  repository this contract applies to today, grouped by layer, and how they
  fit together.

## Reviewing a service

- [contracts/release.md](contracts/release.md) — one tag stamps every
  artifact; what a version means; what a consumer's adoption must show.
- [guides/conformance.md](guides/conformance.md) — the checklist, and what
  CI checks for you.

## Testing a service

- [hack/kind/](../hack/kind/README.md) — the local cluster the charts are
  tested against, what is in it and what deliberately is not.

## Configuring a service

- [schemas/](../schemas/README.md) — the shared shapes, and how a service
  references them from its own schema.

## Choosing a library

- [canon/](canon/README.md) — the allowed lists per ecosystem, the pinned
  toolchain versions, and the build tools.

## Understanding why

- [decisions/](decisions/README.md) — the decision records behind the
  contracts.
- [glossary.md](glossary.md) — the words these documents use narrowly:
  estate, platform, ring, tier, lane, component, service, preset, consumer, and
  the environment names.

## How a contract changes

Each contract carries a header with its version and the date it took
effect. A change to one is an entry under `### Contracts` in the
[CHANGELOG](../CHANGELOG.md), in the version it ships in
([release.md §3](contracts/release.md)).

---

Two guides are still to come — keys and signing, and migrating off a
framework. Each arrives with the component that proves it, because writing
one earlier means describing code nobody has run.
[guides/README.md](guides/README.md) says which is which.
