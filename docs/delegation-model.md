# The delegation model

Who runs what, when a company shares a platform with others. Short on
purpose. If a rule here needs a long explanation, it does not belong here.

Read this if you are an engineer or an AI agent working for a company that
runs on the shared platform, or on the platform itself. Words such as
platform, box and company are in the [glossary](glossary.md). How people and
agents work together is in the [working agreement](working-agreement.md).

## Three sentences

1. **The platform owns everything shared.** That is the clusters, the gateway
   fleet, Argo CD and Kargo, observability, the identity issuer and the CI
   runners. It gives each company a box: its namespaces, its gateway port, and
   its paths in secrets and state.
2. **A company owns its own accounts and runs them by hand.** That is its
   Cloudflare account, its GitHub organisation, its directory, its project
   stacks in the shared AWS accounts, and later its own AWS organisation.
   People run the Pulumi stacks from the company's own `{company}/nexus`
   repository, with one-hour tokens from the identity issuer. They run them
   only when the configuration changes. CI only checks.
3. **A company deploys into its own namespaces through its gitops
   repository.** Argo CD does the day-to-day work: versions, values, routes
   and new hostnames.

## Repositories

| Repository | Holds | Applied by |
|---|---|---|
| the platform repository (private) | the platform | platform admins, by hand |
| `{company}/nexus` | the company's own accounts and zones | company SREs, by hand |
| `{company}/gitops` | the company's applications | Argo CD |
| public `truvity/*` components | generic building blocks used by all of the above | nobody: they are consumed at a pinned version |

A `{company}/nexus` repository keeps its data apart from its code:

- `cfg/` holds plain YAML: `company.yaml` (what the company tells the platform,
  such as status-page entries and alert receivers), `cloudflare.yaml`,
  `aws.yaml`, `github.yaml` (planned).
- One folder per component (`cloudflare/`, `aws/`, `github/` (planned)), each its own
  Pulumi project that reads its file in `cfg/`, run by hand with
  `just preview <component> <stack>` and `just up <component> <stack>`.
- State lives under a prefix per company and component.

The project stacks (registries, release roles and limit roles) and the GitHub
organisation stack move to the company's nexus, step by step. Until a stack has
moved, it is applied where it is applied today. The permission boundaries stay
with the platform. A company names its boundary in a block the platform
approves and guards with code owners. A guard in the AWS organisation comes
first: until it exists, only the platform team applies these stacks.

A `{company}/gitops` repository holds one folder per project:

```text
projects/<project>/project.yaml            source chart, clusters, pipeline shape
projects/<project>/<cluster>/values.yaml   changed by people
projects/<project>/<cluster>/pin.yaml      written by the promotion tool
```

The platform renders one bootstrap per registered project.

## Who does what, per area

There are twelve areas.

| Area | Public component | Platform (by hand) | Company nexus (by hand) | Company gitops (Argo CD) | Isolated by |
|---|---|---|---|---|---|
| Edge | [edge](https://github.com/truvity/edge) | the gateway fleet; per company its own gateway port; registers each zone and where it is served; one certificate per zone and cluster | the company's Cloudflare account: tunnels, DNS, the zones it owns | routes for its apps, under its zones | its own Cloudflare account; its own gateway port; the listener accepts only its namespaces and the names of its zones |
| Identity and access | [sluis](https://github.com/truvity/sluis), [aws-structure](https://github.com/truvity/aws-structure) | the issuer; which company group gets which access | none | none | company groups live in the company's own directory |
| Secrets | [secrets](https://github.com/truvity/secrets), External Secrets | a path and a store per company | puts secret values in | its ExternalSecrets | path prefix; the store is limited to its namespaces |
| Data services | [cnpg](https://github.com/truvity/cnpg), [nats](https://github.com/truvity/nats) | the operators | none | its databases and streams, in its charts | namespace |
| AWS | aws-structure, [k8s](https://github.com/truvity/k8s), [asg](https://github.com/truvity/asg), ACK | the shared accounts; the permission boundaries; later the company's organisation baseline | its limit roles; later its own AWS organisation | its ACK objects | limit role per project |
| Artifacts | [artifacts](https://github.com/truvity/artifacts), [ocictl](https://github.com/truvity/ocictl) | where each kind lives: the public registry, the private registry and the package registry | its registries and release roles | none | registry and release role per project |
| Pulumi support | [pulumi-pipeline](https://github.com/truvity/pulumi-pipeline), aws-structure (the state backend) | the state storage, a state prefix per company and component, the pipeline runner | its stacks, under its own state prefix | none | one state prefix per company and component |
| Observability | [observability](https://github.com/truvity/observability) | stores, Alertmanager, Grafana; stamps the owner label | none | dashboards and rules, in its charts | owner label set by the platform |
| Delivery | [cd](https://github.com/truvity/cd), [gemaal](https://github.com/truvity/gemaal) | Argo CD, Kargo; the company's Argo project | none | apps, versions, values, promotions | one Argo project per company |
| CI | [ci-workflows](https://github.com/truvity/ci-workflows), [ci-actions](https://github.com/truvity/ci-actions), [grun](https://github.com/truvity/grun) | runners, caches | none | its repositories call the shared CI | one runner group per GitHub organisation |
| Source hosting | [github-structure](https://github.com/truvity/github-structure) | the github-structure program, and the GitHub organisation stack until it moves to the company's nexus | its GitHub organisation stack, once it has moved | none | one GitHub organisation per company |
| Foundation | aws-structure | the clusters and the accounts they run in; company and project registration | none | none | the platform registers each company and each project |

### Edge

A company's edge derives from the zones it owns. The platform registers a zone
and where it is served. Names inside the zone belong to the company and need no
change on the platform. The platform keeps one certificate per zone and
cluster, and sets no WAF defaults for a company's zones.

## The platform and a company meet at the box

The platform repository and a company's repositories do not depend on each
other at source level. Neither imports code from the other or pins a commit of
the other. The platform reads what a company writes only in the places listed
in the table below, and a company reads what the platform publishes about
its box. People may read any repository they have access to. The two sides
meet at the box.

Each public component has two parts with a contract between them, kept in the
public repositories:

- the **platform part**, applied by the platform admin;
- the **company part**, applied by the company's SRE.

### The one pattern

The company writes a typed document into a named place in its own box. A
component that the platform runs reads it and changes its behaviour. The
platform owns the guard on that place. Most of the platform already works this
way:

| What the company writes | Where | Platform-run reader | Guard |
|---|---|---|---|
| Applications | its own namespace on the management cluster | Argo CD | the company's Argo project allow-list |
| Pipeline warehouse and stages | its pipeline namespaces | Kargo | an admission policy on stage steps |
| Routes | its project namespaces | the gateway | the listener admits only its namespaces and the names of its zones |
| External secret requests | its project namespaces | External Secrets | the store admits only its namespaces |
| Secret values | its own path in the parameter store | External Secrets, through a reader role | IAM on the path |
| Status entries (planned) | a config document under its own path | the status box | a schema check the platform owns |
| Alert receivers (planned) | its namespace | the alert engine | admission, and a forced owner label |
| Access rows (planned) | a config map where the issuer runs on a cluster, a config document where it runs outside | the identity issuer | the company's ceiling |

Every row needs four things:

1. A named place that only that company can write to.
2. A type. A real resource kind is best, because the API server checks it.
3. A guard that the platform owns.
4. Feedback to the company on whether its document was taken.

### The store follows the reader

Where the reader runs in a cluster, the document is an object in the
company's namespace. Where the reader runs outside the clusters, the document
is a config document under the company's path.

The facts of a box and the places a company writes to are in the
[box contract](contracts/box.md). It is a proposal.

## Environments

devel is open for rapid change. Stage and prod are strict and isolated.

New builds reach devel by themselves. Stage and prod move on a promotion
request. Stage and prod values need a code owner. Pins need none, so the
promotion tool can write them.

## What a company must not touch

Anything shared: the gateway fleet, the clusters, the identity issuer, another
company's namespaces, paths or state. The isolation column above is how the
platform enforces this. Do not rely on it. If you are not sure whether
something is yours, it is not: ask.

## How to ask the platform for something

Open an issue or a pull request against the platform repository. Say what
you need, in which environment, and why. Platform admins review it and apply
it by hand. If you cannot read the repository, open the issue through your
platform contact and name the area from the table above. The
[working agreement](working-agreement.md) says how to write the question down.

Typical asks: a new namespace, another gateway port, a new secret path, a
change to who gets which access. Day-to-day changes to your own applications
are not asks. Make them in your gitops repository.
