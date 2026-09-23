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
| Node | 24.x |
| TypeScript | 6.0.x |
| Yarn | 4.x |

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
