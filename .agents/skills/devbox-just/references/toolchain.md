# Toolchain Pins and Groups (Truvity policy canon)

Why every tool is pinned, what the pins are, and what moves together. Condensed and self-contained; normative source (needs network): [canon/toolchain.md](https://github.com/truvity/policy/blob/master/docs/canon/toolchain.md), [component contract C6/C10](https://github.com/truvity/policy/blob/master/docs/contracts/component.md).

## Why pins, in terms of the failures they prevent

A pin is not conservatism. Three things go wrong without one, and all three are quiet:

- **A tool that moves under CI** makes a build reproducible only until the next run. "It passed yesterday" stops being evidence.
- **Two tools that move independently** disagree. A linter builds the code with its own compiler; when that compiler is older than the one the repository targets, the linter rejects syntax that compiles, and the error blames the code.
- **A tool that moves on a laptop but not in CI** produces a diff that only one person can reproduce, usually in generated output or formatting.

## Go: one triad, moved together

The following versions are the reviewed manifest snapshot, not permanent minimums.
When updating a skill copy, consult `devbox.json` and `devbox.lock` together.

The separately versioned product owns Node/Yarn/uv/Gradle and the non-Go
loaders; its declarations are not the policy manifest. A lock-only update is
still a toolchain change; compare each repository's manifest, lock and
module/toolchain directives together.

| Tool | Version (the reference repository) |
|---|---|
| Go | 1.27.0 |
| golangci-lint | 2.13.1 |
| govulncheck | 1.7.0 |

These three move as one. The linter and the vulnerability scanner each compile the code with a Go of their own, and a mismatch shows up as a failure that names the wrong thing: syntax the linter cannot parse, or a scanner that cannot resolve the standard library. The language line is capped by whatever the linter can build; the patch is the newest of that line; the `go` directive in `go.mod` moves with them.

The root module now declares **`go 1.27.0`** and requests
**`toolchain go1.27.3`**, while devbox still declares Go 1.27.0. These are
different declarations: with automatic toolchain selection Go can select or
download the requested toolchain; `GOTOOLCHAIN=local` prevents that selection.
Check `go version` under the actual command's environment when diagnosing
parity. The manifest check does not establish which compiler ran, and the
module's `toolchain` directive is not a reason to silently rewrite a devbox pin.

## Node and TypeScript (consumer guidance)

Policy no longer installs Node or Yarn. These canon patterns apply when the
adopting product actually has Node components; do not add that runtime to a
Go-only producer just to imitate this section.

| Tool | Version |
|---|---|
| Node | 26.x |
| TypeScript | 7.0.x, for type checking |
| TypeScript | 6.0.x, where a tool needs the compiler's API |
| Yarn | 4.x |

**Two TypeScript lines, on purpose, and only for as long as it takes.** The 7.0 compiler is a native port: it type-checks the same language several times faster, and it ships **no programmatic API**. Every tool that drives the compiler rather than invoking it — a framework's build command, a transpiling test runner, a type-aware linter — calls that API and cannot run on 7.0. Type checking is 7.0, invoked as a command; anything that needs the API stays on 6.0, installed so it does not become the package the editor resolves. Emit needs neither: a service's bundle is produced by a transpiler that reads the syntax and never type-checks.

A repository moving to 7.0 removes the settings the line dropped — a base URL for module resolution, the older resolution modes, the down-level targets — before it flips, because they are hard errors rather than warnings.

The Node line is the newest the environment manifest can pin, not the newest that exists. A published library declares the OLDEST line it supports in `engines`: a consumer on an earlier line is a real consumer, and a package that quietly requires a newer runtime fails at their install rather than at ours. TypeScript is pinned to a patch line, not a range, because a minor changes what the compiler accepts.

Node's types and the runtime's HTTP client vendor a shared package between them; when one moves ahead, request and body types stop unifying and every fetch call fails to typecheck. They bump together or not at all.

## Yarn is installed by the manifest, and node's plugin has to be off for that to be true

The historical co-located example exposed this silent failure: Node's plugin
put corepack shims ahead of the pinned Yarn, and the shim downloaded the nearest
package manifest's `packageManager` version instead. Every command succeeded:

    $ yarn --version
    4.18.0
    $ command -v yarn
    .../virtenv/nodejs/corepack-bin/yarn      # not the pin
    $ .../nix/profile/default/bin/yarn --version
    4.14.1                                    # the pin, never run

Disabling the plugin (`"nodejs": {"version": "26", "disable_plugin": true}`) makes the pinned yarn the one that runs, and removes a download from every environment. `packageManager` stays in the manifest — somebody not using this toolchain reads it — and the two are checked against each other, because a repository that declares its yarn twice will eventually declare it differently.

## The environment is declared, not installed

Every repository declares its toolchain in a manifest that a single command materialises — the same versions on a laptop and in CI, with a lock file committed. Nothing is installed by hand, nothing is on a PATH somebody exported, and a missing tool is a line in the manifest rather than an instruction in a README. When a tool is missing, add it to the manifest. Never fetch it into a shell.

## The manifest's shape (excerpt)

The policy producer's current `devbox.json`, cut down to the shape — the manifest and its
committed lock are the authority, not this excerpt:

```jsonc
{
  "packages": {
    "go": "1.27.0",
    "golangci-lint": "2.13.1",
    "just": "1.58.0",
    "lefthook": "2.1.10",
    "kubectl": "1.36.3",
    "python": "3.14.4",
    "kubernetes-helm": "4.2.4",
    "editorconfig-checker": "4.0.2"
  },
  "env": { "PATH": "$DEVBOX_PROJECT_ROOT/bin:$PATH" },
  "shell": { "init_hook": ["lefthook install >/dev/null 2>&1 || true"] }
}
```

Policy also pins `govulncheck`, `kind`, `gopls`, `just-lsp`, `openssl` and
`goreleaser`; Python runs maintenance checks, not a policy Python loader.
It no longer declares Node, Yarn, uv, Gradle/JDK, ko, buf/protoc plugins,
kubeconform or Biome. Product-owned language/image/generation jobs declare
those tools in the product's manifest. Commit each lock; ignore `.devbox/`.

## Moving a pin

A dependency bot proposes every bump and the whole gate runs against it. Bumps that are grouped, because they cannot move separately:

- the Go triad above;
- a language's types and the runtime packages that vendor them;
- a build tool and its plugins, which declare peer ranges on each other and deadlock when proposed apart;
- anything whose binary artifact is pinned elsewhere — a browser automation library whose driver is installed by the environment manifest can only move to a version the manifest also has.

A group that is wrong is discovered the same way every time: two pull requests that each cannot pass until the other merges.

Policy's Renovate configuration leaves `devbox.json`, `devbox.lock` and the
Go directive to the fleet's devbox parity updater. The Node product also
coordinates Yarn's `packageManager`; that rule is not in policy's current preset.
Confirm that updater is enrolled before disabling an existing update path in a
migrating repository. Its job is to move the manifest, resolved lock and
followers together, not to give two bots competing ownership.

## What checks it

Run version and executable-resolution checks inside the owning devbox when
diagnosing parity: Go/linter versions, and Yarn/path checks only for a Node
consumer. Keep `GOTOOLCHAIN=local` on the reference's golangci-lint
invocations. An installed but shadowed binary is not the tool that ran.

For a Gradle consumer, an environment-supplied Gradle and a wrapper fetching a
second copy reopen the same "which version actually runs" problem. Choose one
declared authority; this is not a claim policy still has a Gradle build.

- **C6** (`policy-conformance`, mechanical): every entry of `packages` in `devbox.json` names a version — `name@X…` in the list form, a non-empty version other than `latest` in the map form.
- **Policy's `hack/toolchain-check.py`**, run by `just lint`, checks package version declarations only. The removed `ts/package.json` and Node plugin/Yarn comparison are no longer checked here. A Node consumer needs its own corresponding parity check; do not credit policy's check with coverage it no longer has.
- **Runtime parity is a separate observation.** That Python check reads manifests;
  it does not run Go, golangci-lint or Yarn. The current lint recipe invokes
  golangci-lint with `GOTOOLCHAIN=local`. Check versions and executable paths in
  devbox against the resolved lock when diagnosing a mismatch; do not call the
  manifest check proof of the compiler or package manager that actually ran.
- **C10**: `vuln` is never reachable from `check` — not a dependency, not through a recipe `check` depends on, and no body line of either running `just vuln`. It runs in `security.yaml`, on push, on pull requests and on a daily schedule.
