# The working agreement

How people and agents work together on a shared platform. It applies to the
platform repository and to every company repository, in the same words.

Read this before your first change. It is short on purpose. Where a rule has a
reason, the reason is in the sentence. Where it has no exception, the page says
"never". Everything else is a default: prefer it, and say so when you do
something else.

For who owns what, read [the delegation model](delegation-model.md). Words such
as platform, company and box are in the [glossary](glossary.md).

## Who decides

Every estate names an owner. People and agents propose. The owner decides.

A decision counts when it is written down in the decider's own words. The
places are the tracker's decision log and a decision record. A decision record
holds what was decided and why.

An approval that someone else passed on is not yet a decision. Record it as
waiting until the decider confirms it in writing.

## Where each kind of fact lives

| Kind of fact | Where it lives |
|---|---|
| Work, and who holds it | the tracker |
| What was decided, and why | a decision record |
| How to do something | a runbook |
| What a component is | the component's README |
| The reasoning before a decision | a wiki |

Prefer a link to a copy. A copy goes stale without anyone noticing, and a link
does not.

## Questions

Write a question on the work item that waits for it. Give the question, the
date, and who answers.

If work goes ahead on a default while the question is open, the item says which
default. The person who answers can then see what to undo.

## Changing things

- Prefer one pull request per repository per step.
- Preview first and read the preview. Apply as a separate step.
- A state handover shows a preview with no difference, or it stops. A
  difference means the old and the new side disagree about what exists.
- Prefer moving what exists to building a new mechanism. A move keeps what
  already works and has a preview that can be read.

## Boundaries

The platform repository and a company's repositories do not read each other.
They meet at the box, which is described in
[the delegation model](delegation-model.md#the-platform-and-a-company-meet-at-the-box).

To change something that another holder owns, ask. Open an issue or a pull
request against their repository, and say what you need and why.

## Pull requests

- Never push to the default branch.
- Read the review threads before every push.
- Once review has started, prefer a follow-up commit to a force-push, so that
  reviewers can see what changed.
- Checks are green and threads are resolved before a merge.
- Rely on CI. Prefer it to long local runs.

## Secrets

Never print or commit a secret value.

A secret found in a repository is rotated and reported. Removing it from the
latest commit is not enough, because history keeps it.

## Agents

An agent works for a named person who answers for its work. The tracker shows
that person as the holder of the item.

An agent:

- reads the repository's `AGENTS.md` and this page before its first change;
- says what it did not verify;
- stops and asks when a permission is refused, and does not look for another
  route to the same result;
- hands over in writing (see below).

## Handover

A handover note has these sections, in this order:

1. **Done.** What is finished.
2. **In progress.** What is started, and where it stopped.
3. **Leftovers.** What remains, in the order to do it.
4. **Decisions waiting.** Each decision, and who has to make it.
5. **Environment.** Notes on the place the work ran, such as a token that
   expires or a tool at a particular version.

## The tracker

- A started item has a holder.
- One item is one reviewable deliverable.
- Name a milestone by what it delivers.
- The lead of a project that has an end posts one line every week: on track, at
  risk or off track.
