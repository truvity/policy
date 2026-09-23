# Node and TypeScript canon

**Scope:** every Node service, library or front end held to these contracts.

| Area | Library | Notes |
|---|---|---|
| Language | TypeScript, `strict` | a service with `strict` off is not held to this canon; it is a service nobody can refactor |
| Service framework | Nest | its own dependency injection is the framework's, and stays inside it — [0001](../decisions/0001-no-di-containers.md) draws the line at the framework boundary, not through it |
| HTTP client | the runtime's `fetch` | |
| RPC | Connect (`@connectrpc/connect`), with the browser and Node transports | the same schema as the Go side, generated at build time |
| Schema validation | Zod for values that cross a boundary at run time | the configuration file is validated against its JSON Schema by this repository's loader, not by a second description of the same shape |
| Configuration | `@truvity/policy` | [config.md](../contracts/config.md) |
| Logging | a structured logger writing JSON to stdout, at one level | [service.md](../contracts/service.md) |
| Testing | Vitest | |
| Linting and formatting | Biome | one tool for both, so formatting is never a second opinion |
| Bundling | Vite | |
| Package manager | Yarn 4 | [build-tools.md](build-tools.md) |

## The front-end rows

| Area | Library |
|---|---|
| UI | React |
| Component library | a single one per product, named in that product's README |
| Router and data fetching | the framework's own, or one choice per product |

A product that renders HTML has more freedom here than a service does,
because what it must not do — invent a second way to be configured, a second
way to log, a second way to be probed — is already fixed by the service
contract. The rest is design.

## Peer-locked groups

Three sets that cannot be upgraded separately, learned each time by a pair of
pull requests that each could not pass until the other merged:

- the bundler, its framework plugin, and the test runner built on it;
- the language's type package and the runtime packages that vendor a copy of
  its types;
- a browser automation library and the browser binaries the environment
  manifest installs, which must be a version that manifest has.
