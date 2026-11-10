# The box contract

Status: **Proposed** · Version: 0.1 · Effective: when accepted ([CHANGELOG](../../CHANGELOG.md))

**Proposal.** A shared platform gives each company a **box**. This contract
says what the platform publishes about a box, where a company writes inside it,
and what each of those places needs. It is mechanism only: it names no company,
and the facts of a particular box arrive as inputs
([repository.md §7](repository.md)).

It follows the [delegation model](../delegation-model.md#the-platform-and-a-company-meet-at-the-box):
the platform repository and a company's repositories do not depend on each
other at source level, and they meet at the box.

The test is the platform contract's: **a second platform, run by someone else,
should be able to give a company the same box without a patch to the company's
repositories.**

## 1. What the platform publishes about a box

The platform publishes these facts to a company. A company reads them and does
not guess them.

| Fact | What it is |
|---|---|
| Slug | the company's bare name, used in every other name below |
| Clusters | the clusters the company deploys to, by role (management, devel, stage, prod) |
| Gateway port | the company's own port on the shared gateway fleet |
| Namespaces | the namespaces the company owns, or the pattern that names them |
| Secret path | the company's path in the parameter store, and the key alias that encrypts it |
| State prefix | the prefix under which the company's Pulumi state lives, per component |
| Registry accounts | the registries the company may publish to |
| Boundary names | the permission boundaries the company may name in its own roles |
| Issuer | the identity issuer a company's people and CI sign in with |

These facts are the platform's record of the company. A company cannot widen its
own box by editing its own repository. This is the same rule as the
registration in [tenant.md §2](tenant.md).

## 2. The places a company writes to

A company changes the behaviour of a platform-run component by writing a typed
document into a named place in its box. The places are the rows of the table in
the [delegation model](../delegation-model.md#the-one-pattern): applications,
pipeline stages, routes, external secret requests, secret values, and (planned)
status entries, alert receivers and access rows.

A place is not a place until it meets the four requirements.

## 3. The four requirements

Every place a company writes to needs:

1. **A named place that only that company can write to.** The check is the
   platform's: a namespace and its role bindings, or a path and its IAM policy.
2. **A type.** A real resource kind is best, because the API server checks it.
   A config document needs a schema.
3. **A guard that the platform owns.** An allow-list, an admission policy, a
   listener that admits only the company's names, or a schema check. A company
   cannot edit its own guard.
4. **Feedback to the company on whether its document was taken.** How a config
   document reports back is an open question (see below).

A place that lacks one of the four is not ready for a company to write to.

## 4. The store follows the reader

Where the reader runs in a cluster, the document is an object in the company's
namespace. The API server types it, and the cluster's own access control guards
it.

Where the reader runs outside the clusters, the document is a config document
under the company's path. A schema that the platform owns types it, and the
path's access policy guards it.

One reader uses one store: the one where it runs.

## 5. How this relates to the tenant contract

[tenant.md](tenant.md) describes a bundle that a company copies into the
platform's side, where the platform checks it and pins it by digest. This
contract does not adopt that step. A company's files are not copied into the
platform repository, and the platform repository does not read a company's
nexus.

The row schemas of the tenant contract can still serve as types. A status
entry, an alert receiver or an access row can use the schema under
`schemas/tenant.v1/` as the type of its config document. The refusal codes in
[tenant.md §3](tenant.md) can serve as the feedback of the fourth requirement.

Until this contract is accepted, tenant.md stays as it is.

## How it is checked

| Rule | Check |
|---|---|
| §1 the published facts | **unchecked here**: there is no schema for them yet |
| §3.1 a place only the company can write to | **unchecked here**: the platform's access rules for each place |
| §3.2 a type | the API server for a resource kind; for a config document, a schema that the platform owns |
| §3.3 a guard | **unchecked here**: the platform's admission policies, allow-lists and listener rules |
| §3.4 feedback | **unchecked here**: see the open questions |

This is a proposal, so most of its rules have no check yet. A rule with no
check is a preference, and this contract says so about itself.

## Open questions

1. **Where are the facts of a box published?** The candidates are a file in
   the platform repository that a company is allowed to read, a config document
   in the company's own box, and a release artifact of the platform. Each of
   them changes what "do not depend on each other at source level" means in
   practice.
2. **How does a config document report back?** A resource kind has a status
   that the reader writes. A config document has nowhere to hold one. The
   candidates are a status document beside it, a check in the company's CI that
   runs the same schema, and a message to the company's alert receiver.
