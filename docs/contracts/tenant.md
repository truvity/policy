# The tenant bundle contract

Version: 1.0 · Effective: with the release that carries it ([CHANGELOG](../../CHANGELOG.md)) · Interface: `tenant.v1`

**Normative.** A shared platform serves more than one company. Each company
keeps its own configuration in a repository of its own and hands the platform
a **bundle** of data rows; the platform decides what each row may do. This
contract says what a bundle is, what the platform checks before it accepts
one, and how the accepted bundle is pinned. It is mechanism only: it names no
company, and the platform's facts about a company arrive as inputs
([repository.md §7](repository.md)).

It is not [tenancy.md](tenancy.md). That contract says what a **tenant
namespace** on a cluster must be. This one says what a **company's
configuration** must be before the platform reads it. A company usually has
many tenant namespaces; the bundle is where it asks for them.

The test is the platform contract's: **a second platform, run by someone else,
should be able to accept the same bundle without a patch to it.**

## 1. A bundle is a directory of data

A bundle is a directory named `tenant/` at the root of a company's
configuration repository. It holds a manifest and any of the section files
below, and nothing else. Every file is YAML; every section is a list of rows.

| File | Rows |
|---|---|
| `tenant.yaml` | the manifest: `schema: tenant.v1`, `company: <slug>`, `interface: <int>` |
| `projects.yaml` | projects and the endpoints they publish |
| `clients.yaml` | OIDC clients |
| `access.yaml` | groups, and the grants those groups hold |
| `alerts.yaml` | alert receivers and the routes that reach them |
| `status.yaml` | status entities |
| `ci.yaml` | requests for CI to run for a project |
| `registry.yaml` | container-registry project rows hosted by the platform |
| `github.yaml` | source-hosting organisation rows (§4) |

The shapes are the schemas under `schemas/tenant.v1/`, one per file.

1. **A section that is omitted means no rows.** A bundle with only a manifest
   is valid and asks for nothing.
2. **Unknown fields are refused**, in a row or as a file the table does not
   list. A field this contract does not name is a field no reader will read, and
   a quiet one is how a request goes unreviewed.
3. **A name is one name, never a pattern.** A project, group, client or
   receiver name is a bare lower-case name. A glob in a name is a grant to
   everything it matches, including what does not exist yet.
4. **Every grant carries a `why`.** A grant with no reason is the shape a stale
   grant takes ten months later.
5. **A host is claimed once.** Two endpoints of a bundle, in one project or in
   two, may not name the same host: the route to it would be ambiguous.
6. **A file holds one YAML document, and is a regular file.** A second
   document after `---` and a link in place of a file are refused, so that a
   tool reading the files itself cannot read what the checker never saw.

## 2. What the platform holds about a company

The bundle is a request; what the company **may** ask for is the platform's
own record of it, passed to the checks as a **registration**:

- the company's slug;
- its source-hosting organisation, by name and by numeric id;
- the e-mail domains its people sign in with;
- its network name;
- the DNS zones it owns, written in lower case without a trailing dot, as the
  hosts in a bundle are;
- the container registries it may publish to;
- one **ceiling** per domain it may delegate in (access, network, source
  hosting, and so on).

The ceilings are **opaque** to this contract. Each domain's ceiling is defined
by the library that enforces that domain, which owns its type; the platform maps
its own records onto them. This contract carries a ceiling to the check
untouched and treats a refusal from it as a refusal of the row. A ceiling that
is omitted admits nothing. A ceiling for a domain the checker does not enforce
is carried and ignored: only the access ceiling is enforced in interface 1.

The registration never comes from the bundle. A company cannot widen its own
ceiling by editing its own repository, and the checks refuse a bundle that
tries to say so.

## 3. Refusal

A bundle is accepted **whole or not at all**. One refused row refuses the
bundle, and the platform keeps the last bundle it accepted. A half-accepted
bundle is a company whose state nobody can read off its repository.

The check reports **every** failure it finds in one pass, not the first,
because a company that has to fix and resubmit once per row will stop reading
the message. Each failure carries a stable **code**, the **path** of the row,
and a **reason**:

```
tenant acme: access.yaml: grants[2]: scope "admin" exceeds ceiling "read" [ceiling-exceeded]
```

| Code | The bundle was refused because |
|---|---|
| `unknown-field` | a field, or a file, that this contract does not name |
| `invalid` | a value of the wrong type or shape, or a name that is not a bare name |
| `unknown-interface` | `interface` is newer than the checker reads, or `schema` is not `tenant.v1` |
| `duplicate` | two rows claim one name, or a name the platform already gave to another company |
| `missing-reason` | a grant carries no `why` |
| `cross-company-ref` | a row refers to something the company does not own: another company's project, or a project, group or receiver the bundle does not define; or the manifest names another company than the registered one |
| `provider-owned` | a row takes a name the platform reserves for itself |
| `ceiling-exceeded` | a row asks for more than the registration allows: a zone, an e-mail domain or a registry it does not hold, or a request a domain's ceiling refuses |
| `wrong-org` | the source-hosting organisation row names an organisation other than the registered one |

The codes are the interface. A message may be reworded; a code is renamed only
by a new interface number.

## 4. What this contract does not check

- **Source-hosting rows.** `github.yaml` is checked for its organisation key
  and nothing else. The shape of the rest of the row belongs to the tool that
  applies it, which has its own loader and its own refusals; copying that shape
  here would fork it. A company's CI runs the applying tool's own check as a
  second step.
- **The effect of a row.** Whether an access grant, once merged with the
  platform's own, widens anyone's rights is the access service's question and is
  answered by it. This contract bounds the row; the service bounds the result.
- **Secrets.** A bundle names a secret and never holds one. The check enforces
  this where the shape allows it: a webhook receiver's target must be a bare
  name, never a URL.
- **Where a client may redirect.** Endpoint hosts are bounded by the company's
  zones and e-mail addresses by its domains; a client's redirect URIs are
  checked for their shape only (an https URL, or a plain-http loopback URL for a
  native client, and never a wildcard). Bounding the host a client may send a
  user to is left to the identity provider that holds the client.

## 5. The pin

The platform pins the bundle it accepted by a **digest**, so the company's CI
and the platform's synchroniser name the same thing.

1. **The pin is the OCI manifest digest of the bundle as `ocictl` packs it.**
   The bundle is published as an OCI artifact with `ocictl static push`, which
   packs the directory into one deterministic tar+gzip layer (files at their
   paths relative to the bundle root, sorted, with a zero modification time,
   owner `0:0` and mode `0644`, and a fixed compressor) under the default
   artifact shape. The pin is the digest of the manifest that push writes, the
   value a registry reports for the tag. A company's CI checks the bundle with
   `ocictl digest <dir>`, which prints that digest without touching a registry
   (its `digest` field with `--output json`, its `manifest` line in text), and
   the platform computes the same value with the `tenant` package's `Digest`,
   which calls the same packer. A consumer that records the pin (a lock file,
   for example) records this manifest digest.
2. **The pin is the digest of a plain publication.** The manifest digest also
   covers what is added to the manifest, so a bundle is published with the
   default artifact shape and with no `--annotation`, no named annotation flag,
   no `--annotations-from-git`, and no source date (`--created`, or the
   `SOURCE_DATE_EPOCH` environment variable, which `ocictl` reads). Any of
   these publishes a valid artifact with another manifest digest, and that
   digest is not the pin. The artifact is valid, but it is a different pin,
   and a check that compares it with the platform's finds the difference.
3. **It does not depend on the checkout.** File modes, modification times,
   directory order and the operating system do not change it; the content and
   the path of each file do. The compressor is pinned by `ocictl`'s own
   dependency, so a Go toolchain change does not move it; a new `ocictl` may,
   and the version the platform's checker was built with is the one that fixes
   the bytes. A company's CI pins the same `ocictl` version it publishes with.
4. **A file that is not a regular file is refused**, so a link cannot make two
   checkouts of one commit differ, and so is a bundle with no file.

A digest that cannot be reproduced is not a pin; the check that accepts a bundle
and the one that pins it read the same bytes, packed by the same code.

## 6. The interface number

A bundle declares the **interface** it was written for as a whole number. The
rules are [delivery-interface.md §2](delivery-interface.md)'s: the number only
grows, and a step only adds. A checker reads its own interface and the one
before it ([decision 0011](../decisions/0011-config-source-versions-and-secret-delivery.md)
applies the same rule to configuration), and refuses a bundle that declares a
newer one with `unknown-interface`. Interface 1 is the first.

## 7. What a company must not do

- Name another company's project, group or zone in a row.
- Take a name the platform reserves.
- Put a pattern where a name belongs.
- Carry a registration, a ceiling or a secret in the bundle.
- Reach past the section files: a hand-edited copy of the platform's state is
  not a bundle.

## How it is checked

| Rule | Check |
|---|---|
| §1 shapes, unknown fields, bare names | the section schemas under `schemas/tenant.v1/`, loaded by the `tenant` package |
| §1.4, §3 codes | one refusal fixture per code, each a valid bundle with one change, asserting the code and the path; a sweep guard fails if a code has no fixture |
| §2 registration | the `tenant` package's `Check`, against an `Estate` the caller implements |
| §1, §3 and §5 from a company's own CI | the `tenant-validate` command (`go run github.com/truvity/policy/cmd/tenant-validate@<version> <dir>`): `Load`, `Validate` and `Digest` over the directory, every failure printed with its code, exit status 1 for a refused bundle and 2 when the check could not run. It does not run `Check`: the registration is the platform's record, not the company's |
| §5 digest | tests that `Digest` equals what `ocictl digest` prints for the same files, that it is stable under file order, modes and times, and that a source date changes it; golden digests pin the packing to the `ocictl` version in `go.mod` |
| §4 source-hosting rows beyond the organisation key | **unchecked here**: the applying tool's own check, run by the company's CI |
| §4 effect of a merged row | **unchecked here**: the access service |
