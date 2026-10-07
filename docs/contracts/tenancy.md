# The tenancy contract

Version: 1.0 · Effective: 2026-10-07 · Changes: see [CHANGELOG](../../CHANGELOG.md)

**Normative.** [platform.md](platform.md) says what a service asks of its
platform. This says what a **tenant namespace** on a shared cluster must be,
and who provides each part. It is guidelines that a tenant's own chart and the
platform's per-tenant charts are both held to, not a chart.

A tenant is a project's footprint on a shared cluster: one or more namespaces
and everything that confines them. Earlier, one chart rendered a tenant's whole
kit and so decided policy for many concerns at once. The model here splits
them: what a tenant namespace must be is this contract, the per-tenant inputs
are a few rows of data, and what a tenant may do outside the cluster is its own
identity.

The test is the platform contract's: **a second platform, run by someone else,
should be able to satisfy it without a patch to the tenant's chart.**

## 1. A tenant is a row of data

The facts about a tenant are values the platform reads: its name, its tier (the
profile that carries policy, quota and labels), the namespaces it owns, and the
names of the identities it needs. Nothing about a tenant is written into a
template. A rule that applies to a class of tenants is a property of the tier,
not of any one tenant.

## 2. The namespace

Every tenant namespace carries, from the platform and never from the tenant's
own chart:

- **Labels.** The tenant's name, its tier and its layer, under one label prefix
  the platform defines. Selectors in policies and admission rules read these
  labels, so the set of values is fixed by the platform and a tenant cannot
  invent a tier.
- **Pod Security.** A level and its version, and the modes that enforce it. The
  default is the strictest level the platform offers. A tenant that departs from
  the default records a **reason** next to the departure; a departure with no
  reason is refused. A rollout that tightens the level is warn-first, and the
  enforcing label ships only after the tenant's spec is compliant.
- **Protection.** A namespace's loss loses everything in it, so it is never
  pruned by the tool that renders it unless the tenant is explicitly marked
  deletable.

Only the platform's administrators and its delivery tool may change these
labels. A tenant cannot loosen its own namespace.

## 3. The baseline NetworkPolicy

Each tenant namespace has a default-deny NetworkPolicy in both directions, with
three kinds of allow:

- traffic between the namespace's own pods;
- the platform's shared allows (DNS, the shared gateway, observability
  scrapes), which are the platform's data and apply to every tenant of the
  matching kind;
- the tenant's own allows, which its own chart adds as additional policies.

The baseline is the floor. A tenant's chart may add policies; it cannot remove
the baseline, and the baseline does not name a tenant.

## 4. Quota and limits

Every tenant namespace has a ResourceQuota and a LimitRange chosen by its tier.
A tenant's workloads therefore always declare, or inherit, requests and limits.
A tenant that needs more asks for a different tier, which is a change to data,
reviewed as one.

## 5. Roles

Access to a tenant namespace is a small, fixed **role spine**: a role that may
administer the namespace's own objects, a role that may edit, and a role that
may read. People and automation are bound to these by group, never by a
per-person rule. Rights on operator custom resources (a database cluster, a
message-broker account) are granted by aggregating a cluster role into the
spine, once, not per tenant.

## 6. Identity

A tenant that needs rights outside the cluster gets them through **an identity
service account per component**, as [platform.md §2](platform.md) describes:
the tenant's chart names the account and nothing more; the platform binds that
account to a role. Two components of one tenant do not share an account, and a
tenant namespace's default service account carries no rights.

The rights themselves are a ceiling set by the platform for the tenant: a
tenant may be granted anything under its ceiling and nothing above it.

## 7. Per-tenant services

Where a tenant uses a shared service that has its own notion of an account (a
message broker, a database operator), the tenant's account in that service is
created from the same tenant row, named from it, and wired to the tenant's own
secrets by **name**, per [platform.md §1](platform.md). The shared service's
operator, not the tenant's chart, holds the credentials.

## 8. What a tenant's chart must not do

- Create or relabel its own namespace, or change its Pod Security labels.
- Remove or replace the baseline NetworkPolicy, the quota or the LimitRange.
- Bind roles outside its own namespace, or create cluster-scoped objects other
  than those its operator's custom resources require.
- Name a cluster, an organisation, an account or a region.

## 9. What the platform owes

- The namespace, with its labels, Pod Security level, baseline policy, quota
  and role spine, **before** the tenant's workloads start.
- The identity binding for every account the tenant's chart names.
- A refusal, at admission, of any change to the platform-owned labels by anyone
  but the platform's administrators.
- A way to see the tenant rows in one place, so a reviewer can read what any
  tenant is allowed.

## How it is checked

The platform's charts that render these objects take the rows as values and
refuse a row that departs from a default without a reason. A tenant's chart is
checked against §8 by the same render tests the other charts use; see
[guides/conformance.md](../guides/conformance.md).
