# The documentation contract

Version: 1.0 · Effective: 2026-10-05 · Changes: see [CHANGELOG](../../CHANGELOG.md)

**Normative.** How a product's documentation is organised, which of it is
generated, and what an estate's repository may say about a product.

The failure this prevents is specific and common: a page written once by hand
describes the configuration of a version that no longer exists, a runbook
omits the step that bit the last person, and an estate repository carries a
copy of the product's table that stopped being true two releases ago. Nobody
is wrong; nothing was checked.

## 1. A product is organised by what the reader is doing

Documentation follows [Diátaxis](https://diataxis.fr/), in these directories:

| Directory | The reader is | Holds |
|---|---|---|
| `docs/getting-started/` | learning | one tutorial per deployment shape, from nothing to working |
| `docs/how-to/` | doing a task | one task per page; upgrade pages live under `docs/how-to/upgrade/` |
| `docs/reference/` | looking something up | config keys, chart values, CLI, adapter matrices |
| `docs/explanation/` | understanding | design and the why |
| `docs/decisions/` | asking why it is so | the ADRs |

A page belongs to one directory. A page that is part tutorial and part
reference is two pages, because the reader of one is lost in the other.

## 2. What can be generated is generated

Reference that can be produced from code or a schema **is**: configuration
keys, chart values, CLI help, adapter matrices. The generated file is
committed, and CI regenerates it and fails on a difference, the way
[repository.md](repository.md) treats generated code. A hand-written table of
what a schema already says is a second copy that will drift.

## 3. The ADR index has a Status, and a status is changed where it is read

The index in `docs/decisions/README.md` carries a **Status** column. A
superseded or amended decision has **its own Status line changed** to say so
and by what, in addition to any banner: a reader who opens one record must
not have to know that another exists. The body of a decision is still never
edited after acceptance.

## 4. A runbook has one template

1. **Purpose** — what this achieves, in a sentence.
2. **Preconditions** — what must already be true.
3. **Before you start** — a list of known traps, each a failure that has
   happened, with what it looks like.
4. **Steps** — each with the command, the expected output, how to verify it,
   and how to roll it back.
5. **Afterwards** — what to check later, and what to tell whom.

A step with no rollback says "none" and why. A runbook whose traps are
written after the fact, at the end, is read after the failure.

## 5. An estate holds estate values and links to the product

A repository that deploys a product holds **its own values** — which
hostnames, which sizes, which choices — and **links to the versioned product
page** for everything else. It does not restate a product's keys, defaults or
procedure: the restatement is true on the day it is written. The link names
the version the estate runs.

## 6. Links and obsolete names are checked

CI checks that every relative link resolves and that no page uses a **banned
obsolete name** — a spelling the product retired, listed in one file the
check reads. A renamed concept leaves no page that still teaches the old
name.

## 7. The CHANGELOG is not a manual

The changelog says **what changed**, in a line. The steps to take live in a
versioned page, `docs/how-to/upgrade/vX.Y.md`, and the changelog entry links
it ([release.md §3](release.md)). A **Breaking:** entry without that link is
a breaking change without its migration.

## 8. A page is small enough to read

Prefer a page under about 400 lines. A longer page is split **by audience**
(an operator's half and a developer's half), not by length, because a page
cut at an arbitrary line is two pages nobody can navigate.

## Conformance

| Rule | Mechanism |
|---|---|
| 1. Diátaxis layout | review — unchecked (a directory check is planned) |
| 2. generated reference | a drift recipe per product, as `just drift` does here; where a product has none, review — unchecked |
| 3. ADR status | review — unchecked; this repository's index carries the column ([decisions](../decisions/README.md)) |
| 4. runbook template | review — unchecked (a heading check is planned) |
| 5. estate links, not copies | review — unchecked |
| 6. links | this repository's lint: every relative link in a Markdown file resolves |
| 6. banned names | planned — no shared check exists yet |
| 7. upgrade page linked | review — unchecked |
| 8. page size | review — unchecked (a line count is planned) |

This repository follows the same layout where it applies: its contracts are
the reference, its guides the how-to and tutorial, its decisions the
explanation.
