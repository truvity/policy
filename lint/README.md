# lint

The configuration a repository copies, so that a rule written in
[`docs/contracts`](../docs/README.md) is enforced rather than remembered.

| File | Copy into |
|---|---|
| [golangci-depguard.yaml](golangci-depguard.yaml) | the `linters.settings` block of a repository's `.golangci.yaml` |
| [biome.base.jsonc](biome.base.jsonc) | referenced from a package's own `biome.jsonc` via `extends` — not copied |
| [.editorconfig](.editorconfig) | the repository root, as `.editorconfig` |

## Why a copy

golangci-lint v2 cannot extend a configuration from a URL, so there is no
version of `golangci-depguard.yaml` that is a reference, and EditorConfig has
no `extends` mechanism at all — so `.editorconfig` is a copy for the same
reason a depguard block is. A copy can fall behind, and nothing in the
repository that holds it will say so; the parity check across repositories is
what closes that gap.

The alternative — generating each repository's configuration from a template —
costs more than it saves. A configuration nobody can read in place is one
nobody edits when their repository genuinely differs, and then the generator
grows an exception mechanism.

`biome.base.jsonc` is the one exception: Biome's own `extends` resolves a
relative path (or an installed npm package), so a package's `biome.jsonc` can
point straight at this file rather than copying it. It still is not a URL —
extends never fetches over the network — so it only works from inside a
checkout that holds both files, which every consumer of this repository's
`lint/` directory does.

## Adopting the import ban

1. Paste the block under `linters.settings` in `.golangci.yaml`, and add
   `depguard` to `linters.enable`.
2. Run `golangci-lint config verify` **first**. A settings block in the wrong
   place is accepted silently by `run` and rejected only by `verify`, so
   without it a lint rule can spend releases doing nothing.
3. Run `golangci-lint run ./...`. Every finding is either a migration or an
   exception, and an exception is a line in the repository's own
   configuration with a comment saying why.

## Adopting the Biome base

Give each TypeScript/JavaScript package its own `biome.jsonc` (not
`biome.json` — see the trap below) beside its `package.json`:

```jsonc
{
  "$schema": "https://biomejs.dev/schemas/2.5.14/schema.json",
  "extends": ["<path to this repository's checkout>/lint/biome.base.jsonc"]
}
```

The path is relative to the config that names it, exactly like an import —
`../lint/biome.base.jsonc` from a package that sits one level under the
repository root, `../../../lint/biome.base.jsonc` from one three levels
down. Add a package's own overrides and a `files.includes` exclusion for
whatever it generates (this repository's `ts/biome.jsonc` and
`examples/url-shortener/web/biome.jsonc` are the worked examples) — they
still `extends` the base rather than replacing it, the same rule
`golangci-depguard.yaml`'s adopters follow for their own exceptions.

Run `biome check` from the package's own directory. It needs no
`node_modules`: Biome reads source files directly, so this stays in a
repository's hermetic gate (`just lint`, part of `just check`) rather than
becoming a job that needs a registry.

**The `.json` trap.** A `biome.json` (not `.jsonc`) that contains a `//`
comment fails to parse — silently, on at least Biome 2.5.14: no diagnostic,
just every setting from `extends` and every `files.includes` exclusion
quietly not applied, formatted with Biome's own tab-indent default instead
of whatever `extends` said. If a package's own config needs a comment (an
exclusion is worth explaining, the same way this repository explains every
depguard entry), name the file `biome.jsonc`. If it has no comment, either
name works, but `.jsonc` is the safer default because the next edit is the
one that adds one.

**Pin the schema version to what the repository's toolchain manifest pins**
(`devbox.json`'s `biome` entry here). Biome's `$schema` URL is
version-specific, and a mismatched one still runs — it just validates
against the wrong config shape, wrongly.

## Adopting the EditorConfig base

Copy `.editorconfig` to the repository root. Unlike the other two files nothing
here is consumed by a tool at build time — an editor reads it directly — so
there is no adoption step beyond the copy. If the repository can afford
[editorconfig-checker](https://github.com/editorconfig-checker/editorconfig-checker.go)
(devbox: `editorconfig-checker`), wire the universal checks — final newline,
line ending, charset, trailing whitespace — into `lint`, the same way this
repository does. Leave indentation WIDTH to each language's own formatter
(gofmt, Biome, ruff, ktlint): a line-count check cannot tell a real violation
from a raw string, a docstring or a template that legitimately indents
differently, which this repository has all three of — `-disable-indentation
-disable-indent-size` in the invocation is what turns that noise off without
giving up the checks that have no false-positive class.

## What is deliberately not here

A full `.golangci.yaml`, and any TypeScript/JavaScript rule beyond Biome's own
`recommended` preset. Repositories differ in which linters they can afford to
enable and which rules their existing code already violates; a shared file
that half of them override is a file nobody trusts. What is shared is the
part that encodes a contract — an import ban naming what replaced it, or a
formatting shape (indent, quotes, line width) chosen to match code nobody
wrote against this file rather than to express a preference.
