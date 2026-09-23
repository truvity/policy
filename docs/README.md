# Documentation

The index, by what you are here to do.

## Authoring a service

- `contracts/repository.md` — what a repository looks like: layout, toolchain,
  the gate, required checks.
- [contracts/service.md](contracts/service.md) — the process boundary:
  configuration, probes, logs, shutdown, version, images, and how services
  talk to each other.
- [contracts/config.md](contracts/config.md) — one typed configuration per
  binary, its schema, and how the chart is held to the same schema.
- `guides/` — how to satisfy the above, walking the example.

## Reviewing a service

- `contracts/release.md` — one tag stamps every artifact; what a version
  means; what a consumer's adoption must show.
- `guides/conformance.md` — the checklist, and what CI checks for you.

## Choosing a library

- `canon/` — the allowed lists per ecosystem, and the scope they bind.

## Understanding why

- [decisions/](decisions/README.md) — the decision records behind the
  contracts.

---

The tree above is being assembled; this index is written first so that every
page has a place to land. Pages arrive with the change that fills them.
