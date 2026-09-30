# The public repository landscape

One page for the 20 public repositories in the `truvity` GitHub organisation,
so that someone new can see the whole set without opening each one. It is
generated on 2026-09-29 by [`hack/landscape.sh`](../hack/landscape.sh)
(`just landscape`); see
["Keeping this page true"](#keeping-this-page-true) below.

Every repository here is a **component**, held to
[the component contract](contracts/component.md): it ships charts, images, a
Go library, a CLI, or GitHub Actions and workflows, for an estate to
install, import or call at a pinned version. None of them is a service in
its own right except through the mechanism it ships (`gemaal` and the
`audit`/`observability`/`nats-auth-callout`/`argocd-ecr-updater`/
`amazon-eks-pod-identity-webhook` images run as services once a chart
installs them — the repository still ships mechanism, not a deployment).

## The repositories

<!-- landscape:table:start -->

| Layer | Repository | Purpose | Ships | Latest tag |
|---|---|---|---|---|
| Identity and secrets | [`access-roster`](https://github.com/truvity/access-roster) | The policy is the product. One file in git turns the groups your people already have in the corporate directory, and the identities your machines already hold — a GitHub Actions job, a Kubernetes ServiceAccount — into one vocabulary of internal groups, gated per audience at one small OpenID provider. | charts:1; Go module (importable); images; CLI: acceptance, accessctl, access-issuer, github-roster; action | v1.39.2 |
| Identity and secrets | [`openbao`](https://github.com/truvity/openbao) | OpenBAO for Kubernetes estates, as reusable mechanism: the two halves the upstream server chart leaves out, the desired state of OpenBAO's own configuration and its apply, and the KMS-rooted CA ceremony its PKI hangs from. | charts:2; Go module (importable); CLI: openbaoctl, openbao-hostcert | v0.20.0 |
| Identity and secrets | [`audit`](https://github.com/truvity/audit) | An audit trail an application owns: one record format, one write path, an immutable archive, and projections for security, billing and history. | charts:1; Go module (importable); images; CLI: audit, audit-query, audit-writer, protoc-gen-audit-jsonschema | v0.4.0 |
| Edge | [`gateway`](https://github.com/truvity/gateway) | Envoy Gateway for Kubernetes estates, as reusable mechanism: the plane the controller manages, split into the half that is vendor-specific and the half that is not. | charts:4 | v1.5.3 |
| Edge | [`cloudflare`](https://github.com/truvity/cloudflare) | Cloudflare for Kubernetes estates, as reusable mechanism: accounts, zone settings, tunnels and R2 buckets as Pulumi Go components; the in-cluster end of a tunnel and an R2 temporary-credentials broker as Helm charts. | charts:2; Go module (importable); images; CLI: r2broker | v2.7.1 |
| Edge | [`tailscale`](https://github.com/truvity/tailscale) | Tailscale for Kubernetes estates, as reusable mechanism: the subnet router that puts a cluster on a tailnet, and the tailnet's policy, keys and split DNS as code. | charts:1; Go module (importable) | v1.15.0 |
| Data | [`cnpg`](https://github.com/truvity/cnpg) | Helm charts for running PostgreSQL on CloudNativePG, split by who installs them. | charts:3; Go module (not importable) | v2.1.1 |
| Observability | [`observability`](https://github.com/truvity/observability) | A self-hosted observability stack for Kubernetes estates, as reusable mechanism: the VictoriaMetrics family as the store, every query scoped at the door to the clusters and namespaces the caller's own token allows, the collectors that stamp those under OpenTelemetry's names, and the alerting rules that catch a backup, a store or a volume failing while everything still looks green. | charts:6; Go module (importable); images; CLI: alert-ingress, dashboardlint | v0.9.1 |
| CI | [`ci-workflows`](https://github.com/truvity/ci-workflows) | The reusable GitHub Actions workflows every repository in both Truvity organisations calls: the merge gate, integration suites, releases, automatic patch tags, and the estate-wide renovate and parity jobs. | reusable workflows: auto-release.yaml, check.yaml, integration.yaml, parity-fleet.yaml, release-private.yaml, release-public.yaml, renovate-fleet.yaml | v3.14.2 |
| CI | [`ci-actions`](https://github.com/truvity/ci-actions) | The composite actions truvity/ci-workflows is built from: bootstrapping a runner, running a recipe, guarding pins and runners, discovering and comparing the fleet, standing up an end-to-end cluster, and checking a repository against the component contract. | action | v1.3.0 |
| CI | [`ci-plane`](https://github.com/truvity/ci-plane) | The CI plane for GitHub Actions on Kubernetes, released as one versioned unit: the ARC runner image, the nix-worker image, and the two Helm charts around them — `ci-builders` (the in-cluster builders and caches) and `arc-runners` (the runner scale sets). | charts:2 | v3.0.0 |
| CI | [`ci-cache`](https://github.com/truvity/ci-cache) | Build caches for CI, as one released bundle: the setup action that looks at a repository and wires its caches for a job, and a cache server — a disk tier over an S3-compatible bucket, serving the Go build cache and the Go module proxy — with its Helm chart. | charts:1; Go module (importable); images; CLI: ci-cache, ci-cache-bench; action | v0.3.1 |
| Cluster add-ons | [`nats-auth-callout`](https://github.com/truvity/nats-auth-callout) | Workload identity for NATS on Kubernetes: an auth-callout responder that validates a connecting client's ServiceAccount token via TokenReview and answers the broker with a signed user JWT that places the client into the NATS account named after its namespace. | charts:1; Go module (importable); images; CLI: responder | v1.1.0 |
| Cluster add-ons | [`argocd-ecr-updater`](https://github.com/truvity/argocd-ecr-updater) | Keeps the ECR credential in Argo CD's repo-creds Secrets fresh: a CronJob that rewrites each Secret's password with a new authorization token, and a PostSync hook that seeds the Secrets a fresh cluster does not have yet. | charts:1; Go module (importable); images; CLI: updater | v2.0.0 |
| Cluster add-ons | [`amazon-eks-pod-identity-webhook`](https://github.com/truvity/amazon-eks-pod-identity-webhook) | Fork of aws/amazon-eks-pod-identity-webhook with Kubernetes 1.35+ compatibility. | charts:1; Go module (importable); images; CLI: webhook | v1.0.9 |
| Developer tooling | [`ocictl`](https://github.com/truvity/ocictl) | Deterministic OCI chart packaging and CRD repack tooling. | charts:3; Go module (importable); CLI: crdctl, helmctl | v0.6.2 |
| Developer tooling | [`workstation`](https://github.com/truvity/workstation) | Developer machine provisioning for the Truvity estate. | Go module (not importable); CLI: awsctl, direnvctl, dockerctl, licencectl, playwrightctl | v0.1.2 |
| Developer tooling | [`gemaal`](https://github.com/truvity/gemaal) | A *gemaal* is a Dutch pumping station — the machine that keeps a polder dry. | charts:1; Go module (importable); images; CLI: gemaalctl, server | v0.25.0 |
| Developer tooling | [`github-structure`](https://github.com/truvity/github-structure) | A GitHub organization's structure as code — with the discipline that makes it survivable. | Go module (importable) | v0.12.2 |
| Doctrine | [`policy`](https://github.com/truvity/policy) | The engineering contracts a service is held to: how it is configured, how it starts and stops, how it is built and released — and one worked example that runs them. | Go module (importable); images | v1.29.1 |

<!-- landscape:table:end -->

`Ships` is a short mechanical summary — chart count, whether a Go module is
importable outside `cmd/`/`internal/`, CLI binary names, container images,
a composite action, reusable workflows — not the repository's full artifact
table; read the repository's own README for that.

## How they fit

**Identity and secrets: access-roster, openbao, audit.** access-roster is
the issuer: it turns directory groups and machine identities into one
vocabulary of internal groups and mints tokens or reconciles memberships.
openbao is a relying party of it, never the other way round — it trusts
access-roster's tokens and issues certificates from a KMS-rooted CA
([openbao's own integration doc](https://github.com/truvity/openbao/blob/master/docs/integrations/access-roster.md)).
audit is the record every one of them writes: every decision, sign-in,
refusal, exchange and console action is one record in the audit trail that
both the issuer and its relying parties read from and write to.

**Edge: gateway, cloudflare, tailscale.** These three are layers of one
exposure path, not alternatives. cloudflare's tunnel carries public traffic
to the estate's origin; that origin is a gateway exposure (a ClusterIP
behind the tunnel); tailscale routes whole CIDRs for private access.
gateway's private exposure and tailscale's Service-CIDR route are the two
ways to reach a private service.

**CI: ci-workflows, ci-actions, ci-cache, ci-plane.** ci-workflows is the
only thing a caller pins — every other public repository's `check`,
`release-public` and `auto-release` call into it. ci-actions holds the
composite steps ci-workflows' workflows call. ci-cache owns cache wiring
(its `setup` action, called from ci-actions' `setup-devbox`) and a cache
server; as of this page's generation, ci-cache's own server, chart and agent
have no production installation, and ci-plane's `ci-builders` chart still
carries the in-cluster Nix, npm and Bazel caches it was meant to replace —
both repositories describe that migration as tracked separately, not done.
ci-plane is where the work actually executes: the runner and nix-worker
images, and the `arc-runners` and `ci-builders` charts.

**Developer machine credentials: workstation, access-roster.** `accessctl`
(from access-roster) and `awsctl` (from workstation) both mint AWS
credentials on a developer machine; `accessctl` is the estate path, and
`awsctl` is the SSO fallback for when access-roster is unreachable. Both
tools live on the same machine and are called as alternatives, not layers.

**Chart packaging: ocictl, gemaal.** gemaal's release pipeline shells out to
`helmctl` (`go tool helmctl`, from ocictl) to package and push its own
chart — the same tool every other component repository's release uses to
turn a build into a digest-pinned, immutable chart.

**Doctrine: policy.** Every repository above is held to
[the component contract](contracts/component.md) this repository publishes;
none of them restates it. policy's own worked example (the URL shortener)
proves the contract's service-side rules the same way, using gemaal's
`harness` package and ocictl's `helmctl`.

## Known inconsistencies

None found as of 2026-09-29: for every boundary above, the repositories on
both sides describe it the same way. This heading stays even when it has
nothing under it — the next regeneration is where a real disagreement would
first show up, and a reader should not have to guess whether the check ran.

## Keeping this page true

Run `just landscape` (`hack/landscape.sh`) to re-fetch the table above from
the GitHub API — the latest tag, what each repository ships, and its
one-line purpose — and read the diff before committing it. It is not part
of `just check`: it needs the network and a `gh` session, and the gate does
not. The "How they fit" prose above is not regenerated; it is drawn by hand
from each repository's own `Neighbours` section and needs a human re-read
when one of them changes.
