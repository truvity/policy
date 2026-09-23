# The release contract

**Normative.** How a version is cut, what it means, and what a consumer
must see before adopting it.

## 1. One tag stamps everything

Pushing `vX.Y.Z` releases every artifact the repository produces at that
version: the images, the charts, the modules, the packages, the documents.
Committed chart versions are a placeholder that never moves; the release
stamps them.

A consumer therefore pins **one version per repository**. Two artifacts from
one repository at two versions is a lag to close, never a choice.

## 2. Versions mean what they mean to the consumer

Read from the outside, not from the diff:

- **Major** — something that worked stops working: a removed or renamed
  value, a changed object name, a changed selector (immutable on a live
  object, so the upgrade becomes a delete and a recreate), a new required
  field with no default.
- **Minor** — a capability is added, and its default renders exactly what the
  previous version rendered.
- **Patch** — no rendered output changes and no interface changes.

A module whose language requires a major suffix in its path carries it
**before** the tag is pushed. A tag whose metadata disagrees with its path
can never be fetched, and a tag that has been fetched once cannot be taken
back.

## 3. The changelog is written for the consumer

`CHANGELOG.md`, newest first, one heading per version, prose bullets. The
pull request that changes something a consumer can see adds its bullet under
the heading it will be released as, creating the heading if it is the first.
A breaking bullet starts with **Breaking:** and names the step to take.

Not the commit subjects: those are written for the reviewer of a diff. A
consumer reading a pin bump needs a different sentence — what changes in the
output, what must be done first, whether a default moved. The file also
travels with the source where a hosted release page does not.

A version with no heading changed nothing a consumer can see. That is
information, so it is worth keeping true.

## 4. Who cuts a tag

**Every first release, every minor and every major is cut by a person**,
after the changelog heading for that version has merged. A minor or major
carries a decision, and the tag is the moment someone made it.

Automation may cut **patches only**: immediately for a security fix, and
otherwise on a schedule when merged dependency bumps have moved the default
branch past the latest tag.

The trap in that split, worth stating because it has caught people: the
scheduled lane asks whether the branch has moved, not *what* moved it. A
feature merged and left untagged ships in the next automatic **patch**. So a
minor is tagged when the feature merges, not when it is convenient.

## 5. The build is the release

Artifacts are built once, in CI, from the tagged commit, and published from
that build. Nothing is built on a laptop and pushed. A release that can be
produced by one person's machine is a release nobody else can reproduce.

Images are referenced by digest in everything the release publishes, so what
a consumer installs cannot be moved under them by a tag being repointed.

## 6. Adoption is proved, not asserted

A consumer adopts a release only when the output it produces is
**byte-identical to what runs**, or differs exactly by the change the
release announced. That render, or that plan, is the evidence attached to
the pull request that moves the pin.

Moving from hand-written objects to a chart is one change whose diff is
empty. Tightening a default is a separate release, adopted separately, with
its diff visible.

This is the rule that makes the rest of the contract worth anything: without
it, "no rendered output changes" in a patch is a claim rather than a fact.

## Conformance

| Rule | Mechanism |
|---|---|
| 1. one tag | the release workflow stamps every artifact from one tag |
| 2. versioning | review |
| 3. changelog | review; a version with no heading is a deliberate statement |
| 4. who cuts | automation is armed for patches only |
| 5. built in CI | the release runs only from a tag, in CI |
| 6. adoption | the consumer's pin bump carries the diff |
