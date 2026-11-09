# The delegation model

Who runs what, when a company shares a platform with others. Short on
purpose. If a rule here needs a long explanation, it does not belong here.

Read this if you are an engineer or an AI agent working for a company that
runs on the shared platform, or on the platform itself. Words such as
platform, kernel and company are in the [glossary](glossary.md).

## Three sentences

1. **The platform owns everything shared.** That is the clusters, the Envoy
   fleet, Argo CD, observability, the identity issuer and the CI runners.
   It gives each company a box: its namespaces, its Envoy port, and its
   paths in secrets and state.
2. **A company owns its own accounts and runs them by hand.** That is its
   Cloudflare account, its GitHub organisation, its directory, and later its
   AWS organisation. People run the Pulumi stacks from the company's own
   `{company}/nexus` repository, with one-hour tokens from the identity issuer.
   They run them only when the configuration changes. CI only checks.
3. **A company deploys into its own namespaces through its gitops
   repository.** Argo CD does the day-to-day work: versions, values, routes
   and new hostnames.

## Repositories

| Repository | Holds | Applied by |
|---|---|---|
| the platform repository (private) | the platform | platform SREs, by hand |
| `{company}/nexus` | the company's own accounts and zones | company SREs, by hand |
| `{company}/gitops` | the company's applications | Argo CD |
| public `truvity/*` components | generic building blocks used by all of the above | nobody: they are consumed at a pinned version |

A `{company}/nexus` repository keeps its data apart from its code:

- `cfg/` holds plain YAML: `company.yaml` (what the company tells the platform,
  such as status-page entries and alert receivers), `cloudflare.yaml`,
  `aws.yaml`, `github.yaml`.
- One folder per component (`cloudflare/`, `aws/`, `github/`), each its own
  Pulumi project that reads its file in `cfg/`, run by hand with
  `just preview <component> <stack>` and `just up <component> <stack>`.

A `{company}/gitops` repository holds `pins/<project>/<cluster>.yaml` and
`values/<project>/<cluster>.yaml`. Promotion: new builds reach devel by
themselves; stage and prod are promoted only on request.

## Who does what, per area

| Area | Public component | Platform (by hand) | Company nexus (by hand) | Company gitops (Argo CD) | Isolated by |
|---|---|---|---|---|---|
| North-south gateway | [edge](https://github.com/truvity/edge) | the Envoy fleet; per company its own port and, per environment, a wildcard listener and certificate | the company's Cloudflare account: tunnels, wildcard DNS, zones | routes for its apps | its own Cloudflare account; its own Envoy port; the listener accepts only its names and its namespaces |
| Identity and access | [sluis](https://github.com/truvity/sluis), [aws-structure](https://github.com/truvity/aws-structure) | the issuer; which company group gets which access | none | none | company groups live in the company's own directory |
| Secrets | [secrets](https://github.com/truvity/secrets), External Secrets | a path and a store per company | puts secret values in | its ExternalSecrets | path prefix; the store is limited to its namespaces |
| Data services | [cnpg](https://github.com/truvity/cnpg), [nats](https://github.com/truvity/nats) | the operators | none | its databases and streams, in its charts | namespace |
| Cloud resources (AWS) | aws-structure, ACK | a limit role per project; later the company's organisation baseline | later: its own AWS organisation | its ACK objects | limit role per project |
| Observability | [observability](https://github.com/truvity/observability) | stores, Alertmanager, Grafana; stamps the owner label | none | dashboards and rules, in its charts | owner label set by the platform |
| Delivery | [cd](https://github.com/truvity/cd), [gemaal](https://github.com/truvity/gemaal), [artifacts](https://github.com/truvity/artifacts) | Argo CD, Kargo; the company's Argo project | none | apps, versions, values, promotions | one Argo project per company |
| CI | [ci-workflows](https://github.com/truvity/ci-workflows), [ci-actions](https://github.com/truvity/ci-actions), [grun](https://github.com/truvity/grun) | runners, caches | none | its repositories call the shared CI | one runner group per GitHub organisation |
| Source hosting | [github-structure](https://github.com/truvity/github-structure) | the company's GitHub organisation and its repositories | none | none | one GitHub organisation per company |
| Foundation | aws-structure, [pulumi-pipeline](https://github.com/truvity/pulumi-pipeline) | AWS accounts, state storage, company and project registration | its stacks, under its own state prefix | none | one state prefix per company |

## Environments

devel is open for rapid change. Stage and prod are strict and isolated.

## What a company must not touch

Anything shared: the Envoy fleet, the clusters, the identity issuer, another
company's namespaces, paths or state. The isolation column above is how the
platform enforces this. Do not rely on it. If you are not sure whether
something is yours, it is not: ask.

## How to ask the platform for something

Open an issue or a pull request against the platform repository. Say what
you need, in which environment, and why. Platform SREs review it and apply
it by hand. If you cannot read the repository, open the issue through your
platform contact and name the area from the table above.

Typical asks: a new namespace, another Envoy port, a new secret path, a
change to who gets which access. Day-to-day changes to your own applications
are not asks. Make them in your gitops repository.
