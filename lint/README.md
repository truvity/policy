# lint

The configuration a repository copies, so that a rule written in
[`docs/contracts`](../docs/README.md) is enforced rather than remembered.

| File | Copy into |
|---|---|
| [golangci-depguard.yaml](golangci-depguard.yaml) | the `linters.settings` block of a repository's `.golangci.yaml` |

## Why a copy

golangci-lint v2 cannot extend a configuration from a URL, so there is no
version of this that is a reference. A copy can fall behind, and nothing in
the repository that holds it will say so; the parity check across repositories
is what closes that gap.

The alternative — generating each repository's configuration from a template —
costs more than it saves. A configuration nobody can read in place is one
nobody edits when their repository genuinely differs, and then the generator
grows an exception mechanism.

## Adopting it

1. Paste the block under `linters.settings` in `.golangci.yaml`, and add
   `depguard` to `linters.enable`.
2. Run `golangci-lint config verify` **first**. A settings block in the wrong
   place is accepted silently by `run` and rejected only by `verify`, so
   without it a lint rule can spend releases doing nothing.
3. Run `golangci-lint run ./...`. Every finding is either a migration or an
   exception, and an exception is a line in the repository's own
   configuration with a comment saying why.

## What is deliberately not here

A formatter configuration, and a full `.golangci.yaml`. Repositories differ in
which linters they can afford to enable, and a shared file that half of them
override is a file nobody trusts. What is shared is the part that encodes a
contract: which imports are not allowed, and what to use instead.
