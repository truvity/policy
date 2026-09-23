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
- [guides/](guides/conformance.md) — how to satisfy the above. The rest of
  the guides walk the worked example, and arrive with it.

## Reviewing a service

- [contracts/release.md](contracts/release.md) — one tag stamps every
  artifact; what a version means; what a consumer's adoption must show.
- [guides/conformance.md](guides/conformance.md) — the checklist, and what
  CI checks for you.

## Configuring a service

- [schemas/](../schemas/README.md) — the shared shapes, and how a service
  references them from its own schema.

## Choosing a library

- [canon/](canon/README.md) — the allowed lists per ecosystem, the pinned
  toolchain versions, and the build tools.

## Understanding why

- [decisions/](decisions/README.md) — the decision records behind the
  contracts.

---

The guides are being assembled: they walk the worked example, and arrive with
it.
