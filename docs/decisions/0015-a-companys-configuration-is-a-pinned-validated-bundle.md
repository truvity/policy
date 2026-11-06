# 0015 — A company's configuration is a validated bundle, accepted whole and pinned by digest

**Status:** accepted

## Context

A platform shared by several companies has to read each company's requests:
the projects it runs, the people it admits, the alerts it wants, the clients
it registers. Two shapes are common, and both fail the same way.

- **The company edits the platform's own files.** A reviewer then reads a
  diff in the platform's repository to learn whether a row is the company's to
  ask for, and a mistake in one company's row is a fault in everyone's state.
- **The company runs its own tooling against the platform.** Each company then
  carries its own idea of what is allowed, and the platform learns what it
  accepted by looking at what happened.

What is wanted is a boundary a stranger could implement: the company writes
data, the platform holds the facts about the company, and a check that needs
nothing else says yes or no.

## Decision

**A company's configuration is a bundle of data rows, checked against the
platform's record of that company, accepted whole or not at all, and pinned by
a digest.** The contract is
[tenant.md](../contracts/tenant.md). The pieces, and the alternative each
one was chosen over:

- **One schema per section, strict.** Unknown fields are refused. *Not* a
  single document: a section changes for its own reasons, and a reader of one
  should not carry the others.
- **The registration is an input, never part of the bundle.** *Not* a bundle
  that states its own limits, which a company could widen in the same commit.
- **Ceilings are opaque to this repository.** Each domain's library defines its
  own ceiling and enforces it; the platform maps its records onto them. *Not*
  a ceiling type here, which would make this repository import every domain it
  bounds and would put the definition of "more access" in a place that cannot
  see the access service.
- **Source-hosting rows are checked for their organisation only.** *Not* a copy
  of their shape, which would fork the loader that owns it. The company's CI
  runs that loader's own check as a second step.
- **A refusal is all-or-nothing and reports every failure with a stable
  code.** *Not* a partial accept, which leaves a company whose state cannot be
  read off its repository, and *not* first-error-only, which costs a round trip
  per mistake.
- **The pin is the OCI manifest digest of the bundle as `ocictl` packs it.**
  *Not* a version control tree hash, because the published artifact is not a
  tree, and a pin that the publisher and the synchroniser compute differently
  is two pins. A company publishes with `ocictl static push`, checks with
  `ocictl digest`, and the platform pins the manifest digest a registry
  reports; the `tenant` package's `Digest` calls the same packer, so the check
  and the pin read the same bytes. *Not* an archive format of this repository's
  own: a tar built here would pin the digest of a layer that no registry
  reports and no tool but this package could produce, so a publisher would have
  to call this package instead of the tool that publishes. *Not* an archive any
  common archiver can reproduce: that would mean matching one archiver's
  habits, and the next archiver would still differ. The price is that the pin
  covers the manifest, so a bundle is published with no annotation and no
  source date.
- **A project name another company holds is `duplicate`, not
  `cross-company-ref`.** The name is a claim on a shared namespace, and a second
  claim is a collision. `cross-company-ref` is kept for a row that points at
  something the bundle does not define.
- **The bundle declares an integer interface, and the checker reads its own
  and the one before it.** The rule is [0011](0011-config-source-versions-and-secret-delivery.md)'s,
  applied to a second kind of document.

## Consequences

- **Good.** A company's request is reviewable in the company's own repository
  by the people who own it, and the platform's check is the same code whether it
  runs in the company's CI or in the platform. A refusal leaves the previous
  bundle in force, so a bad commit freezes one company's configuration and
  nobody else's.
- **Bad: one bad row blocks the good ones.** All-or-nothing means a company
  fixes the row to land the rest. That is the price of a state that can be read
  off its repository.
- **Bad: the pin follows another repository's packer.** The bytes of a pin are
  `ocictl`'s: a release of it that changes how a directory is packed or
  compressed moves every pin, and it is adopted here on purpose, with the
  golden digests in this repository's tests changed in the same step. The
  pinned compressor keeps a Go toolchain change from doing it by accident.
- **Bad: opaque ceilings leave a gap.** This repository can show that a row was
  carried to the ceiling's check; it cannot show the check is the right one.
  That is the domain library's test, and it is named in the contract as
  unchecked here.
- **Bad: a second meaning of "tenant".** The tenancy contract's tenant is a
  namespace; this contract's is a company. The glossary says which is which.

## Related

- [tenant.md](../contracts/tenant.md), the contract this records.
- [tenancy.md](../contracts/tenancy.md), the contract it is not.
