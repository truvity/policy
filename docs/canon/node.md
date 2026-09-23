# Node and TypeScript canon

**Scope:** every Node service, library or front end held to these contracts.

| Area | Library | Notes |
|---|---|---|
| Language | TypeScript, `strict` | a service with `strict` off is not held to this canon; it is a service nobody can refactor. Two lines, split by job — see [toolchain.md](toolchain.md) |
| Service framework | Nest | its own dependency injection is the framework's, and stays inside it — [0001](../decisions/0001-no-di-containers.md) draws the line at the framework boundary, not through it |
| HTTP client | the runtime's `fetch` | |
| RPC | Connect (`@connectrpc/connect`), with the browser and Node transports | the same schema as the Go side, generated at build time |
| Schema validation | Zod for values that cross a boundary at run time | the configuration file is validated against its JSON Schema by this repository's loader, not by a second description of the same shape |
| Configuration | `@truvity/policy` | [config.md](../contracts/config.md) |
| Logging | a structured logger writing JSON to **stderr**, at one level | [service.md §4](../contracts/service.md) |
| Telemetry | the OpenTelemetry SDK, configured by its own environment | [0006](../decisions/0006-telemetry-is-the-sdk-environment.md); never gated on an environment name |
| Transpiler | SWC | emit only; it reads syntax and never type-checks, which is why it is fast and why the type checker is a separate job |
| Testing | Vitest | |
| Linting and formatting | Biome | one tool for both, so formatting is never a second opinion |
| Bundling | Vite | |
| Package manager | Yarn 4 | [build-tools.md](build-tools.md) |

## How a service is built

**Type checking and emit are separate jobs, run by different tools.** That is
not a preference; it is what the 7.0 compiler's lack of a programmatic API
forces, and it happens to be the arrangement that was already fastest.

| Job | Tool |
|---|---|
| Type check | the 7.0 compiler, as a command, no emit |
| Emit (service) | SWC, through the framework's builder or directly |
| Emit (front end) | Vite, which uses the same transpiler underneath |
| Test | Vitest, which transpiles the same way |
| Lint and format | Biome |

**The framework's build command is not in the canon.** It imports the
compiler to type-check while its builder emits, so it needs the API that 7.0
does not ship. A service either turns that type check off and runs the
checker as its own step — which is what the table above describes — or drives
the transpiler directly. Both are fine; what is not fine is discovering the
constraint during a version bump.

**The framework's code-generating build plugins are out of scope**, for the
same reason and with no workaround: they exist only as compiler transforms.
A service that decorates its API description by hand does not need them, and
a service that does need them stays on the older line until they have a
successor.

Decorators and their metadata are unaffected: the transpiler emits them, and
so does the 7.0 compiler when asked.

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
