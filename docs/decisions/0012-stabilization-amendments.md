# 0012 — Documents carry a product group, secrets are named and resolved through one source, and a stabilizing product may break in a minor

**Status:** accepted

Amends [0011](0011-config-source-versions-and-secret-delivery.md) (the
document's `apiVersion`, and how a secret arrives) and
[0002](0002-config-file-plus-env.md) (the environment as the secret's home).

## Context

Two products adopted these contracts at the same time and finished with the
same ideas spelled differently. A survey of them found four gaps.

**The group of `apiVersion` was left free.** [config.md §7](../contracts/config.md)
says `<group>/<kind>/v<N>` and not what the group is. Each product chose its
own, so a tool reading both could not tell a kind from a collision.

**A secret had three spellings and three answers.** A field named a
variable, a path or a store key, per field, and a service could mix all
three. One platform put secrets in the function's own environment, which is
capped at about 4 KB and is shown in plaintext in the platform's console.

**A service has more than one document.** Besides how it runs, it reads what
it decides, authored by an operator on another schedule. The contract spoke
of one document per binary.

**"Preset" meant two things.** One product used it for a bundle of adapter
choices and for a compliance bundle, and a preset could name an adapter that
had not been built.

Underneath is the release question: both products are still finding their
shapes, and [release.md §2](../contracts/release.md) makes every rename a
major.

## Decision

**A service document's `apiVersion` is `<product>.truvity.github.io/<kind>/v<N>`.**
Absent still means v1; N and N-1 are read. The schema's `$id` carries the
same version as the document.

**A secret is referenced by name, in a field ending `Secret`, and resolved
through one declared source per service:** `secrets.source` is `env`, `file`,
`ssm` or `openbao`, with a root. `…Env` is retired as a spelling. A
function's own environment is not a source. Resolved at start, a credential
may be re-read where a source refreshes; the configuration stays immutable.

**A service may read a service document and a policy document**, each with its
own kind, schema and `apiVersion`, both immutable per instance; the service
document names the policy file.

**A preset is a named bundle of adapter or deployment choices that expands
at load time and names only implemented choices.** A compliance bundle is a
profile.

**A product declares `stabilizing` or `stable` in its README.** While
stabilizing it may ship a breaking change in a minor, with a **Breaking:**
entry and migration steps, never in a patch; automation cuts patches only
for non-breaking `fix:` changes. The exit criteria are in
[release.md §2](../contracts/release.md). On 2026-10-05 the stabilizing
products are `sluis` and `audit`, the two that prompted this.

Documentation follows one shape, [docs.md](../contracts/docs.md).

## Roads not taken

- **Majors now.** Honest about the breaks, and rejected: in Go the major is
  in the module path, so every importer changes its import path at once,
  mid-rollout, for a break a minor with a **Breaking:** entry announces as
  well. The cost is that semver's promise is bent for a declared period.
- **`truvity.github.io` as the one common group.** One name to remember, and
  rejected: two products share kinds such as `config`, and a shared group
  makes `…/config/v2` ambiguous. The product in the group is the namespace.
- **Keeping `…Env`.** Nothing to migrate, and rejected: the field names how a
  value arrives instead of which secret it is, so moving a service to a file
  or a store changes its schema. The cost is a breaking change to three
  shared fragment fields, done in v1.45 ([upgrade steps](../how-to/upgrade/v1.45.md)).
- **A function's environment as a source, for small secrets.** Rejected: the
  budget is shared with everything else, and plaintext in a console is the
  failure the rule exists for.

## Consequences

### Good

- A document names its owner, its kind and its version in one string.
- "Where does this secret come from" has one answer per service.
- A stabilizing product can fix a bad shape in a minor; a stable one cannot.

### Bad

- **Semver is bent** for stabilizing products: a consumer pinned to a minor
  range may meet a break. The README state and the **Breaking:** entry are the
  only warning, and both are read by people.
- **The shared fragments carried `…Env`** until v1.45 replaced them
  (`passwordSecret`, `credentialsSecret`, `platform.secretFiles`); the rule
  and the schemas disagreed in the meantime.
- **Several of the new rules are unchecked** and say so in the conformance
  tables; a rule with no check is a preference until one exists.
- **A product's state is a README claim**, not a tool's output.

### Neutral

- The `apiVersion` envelope pattern already accepts the new group; nothing in
  the loaders changes.
