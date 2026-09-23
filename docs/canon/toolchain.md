# Toolchain

The version of each tool a repository builds with, and why each is a pin
rather than "latest".

A pin is not conservatism. Three things go wrong without one, and all three
are quiet:

- **A tool that moves under CI** makes a build reproducible only until the
  next run. "It passed yesterday" stops being evidence.
- **Two tools that move independently** disagree. A linter builds the code
  with its own compiler; when that compiler is older than the one the
  repository targets, the linter rejects syntax that compiles, and the error
  blames the code.
- **A tool that moves on a laptop but not in CI** produces a diff that only
  one person can reproduce, usually in generated output or formatting.

So: every version below is pinned, and moved deliberately by a dependency
bot whose pull request runs the whole gate.

## Go: one triad, moved together

| Tool | Version |
|---|---|
| Go | 1.27.0 |
| golangci-lint | 2.13.1 |
| govulncheck | 1.7.0 |

These three move as one. The linter and the vulnerability scanner each
compile the code with a Go of their own, and a mismatch shows up as a
failure that names the wrong thing: syntax the linter cannot parse, or a
scanner that cannot resolve the standard library. Bumping the language
without them is the most common way to spend an afternoon on a non-problem.

The language line is capped by whatever the linter can build. The patch is
the newest of that line. One owner moves all three in one change, and the
`go` directive in `go.mod` moves with them.

## Node and TypeScript

| Tool | Version |
|---|---|
| Node | 26.x |
| TypeScript | 7.0.x, for type checking |
| TypeScript | 6.0.x, where a tool needs the compiler's API |
| Yarn | 4.x |

**Two TypeScript lines, on purpose, and only for as long as it takes.** The
7.0 compiler is a native port: it type-checks the same language several times
faster, and it ships **no programmatic API**. Every tool that drives the
compiler rather than invoking it — a framework's build command, a
transpiling test runner, a type-aware linter — calls that API and cannot run
on 7.0.

So the split is by job, not by preference. **Type checking is 7.0**, invoked
as a command against the project. **Anything that needs the API stays on the
6.0 line**, installed in a way that does not become the package the editor
and the tools resolve. Emit does not need either: a service's bundle is
produced by a transpiler that reads the syntax and never type-checks, which
is also why emit was never the slow part.

A repository on 7.0 removes the settings the line dropped — a base URL for
module resolution, the older resolution modes, and the down-level targets —
before it flips, because they are hard errors rather than warnings.

The Node line is the newest the environment manifest can pin, not the newest
that exists. A published library is a separate question: it declares the
OLDEST line it supports in its `engines` field, because a consumer on an
earlier line is a real consumer and a package that quietly requires a newer
runtime fails at their install rather than at ours.

TypeScript is pinned to a patch line, not a range, because a minor changes
what the compiler accepts, and a compiler that changes on its own turns a
dependency bump into a red build in an unrelated pull request.

Node's types and the runtime's HTTP client vendor a shared package between
them; when one moves ahead, request and body types stop unifying and every
fetch call fails to typecheck. They are bumped together or not at all.

## The environment is declared, not installed

Every repository declares its toolchain in a manifest that a single command
materialises — the same versions on a laptop and in CI, with a lock file
committed. Nothing is installed by hand, nothing is on a PATH somebody
exported, and a missing tool is a line in the manifest rather than an
instruction in a README.

The practical rule that follows: when a tool is missing, add it to the
manifest. Never fetch it into a shell.

## Moving a pin

A dependency bot proposes every bump and the whole gate runs against it.
Bumps that are grouped, because they cannot move separately:

- the Go triad above;
- a language's types and the runtime packages that vendor them;
- a build tool and its plugins, which declare peer ranges on each other and
  deadlock when proposed apart;
- anything whose binary artifact is pinned elsewhere — a browser automation
  library whose driver is installed by the environment manifest can only
  move to a version the manifest also has.

A group that is wrong is discovered the same way every time: two pull
requests that each cannot pass until the other merges.
