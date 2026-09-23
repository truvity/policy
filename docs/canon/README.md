# Canon

The allowed lists: which library for which job, and which version of each
toolchain.

**Scope is a rule, never a list of repositories.** Every service or library
held to these contracts follows the canon for its language. An estate
adopting them names the set in its own configuration. A canon that
enumerates repositories goes stale the week it is written, and a stale canon
is one people stop trusting — which costs more than the drift it was meant
to prevent.

**Changing an entry means changing this document first.** A migration that
starts in a pull request and reaches the canon afterwards is how two
libraries for one job become permanent.

**One job, one library.** Not because variety is bad, but because the second
library for a job is never removed: it is added by someone with a deadline,
and it survives every later reader who assumes it was a decision.

| Language | Canon |
|---|---|
| Go | [go.md](go.md) |
| Node and TypeScript | [node.md](node.md) |
| Kotlin and the JVM | [kotlin.md](kotlin.md) |
| Python | [python.md](python.md) |

| Cross-cutting | |
|---|---|
| toolchain versions, and why each is pinned | [toolchain.md](toolchain.md) |
| build systems, package managers, container builds | [build-tools.md](build-tools.md) |
