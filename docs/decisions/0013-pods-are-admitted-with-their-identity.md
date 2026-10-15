# 0013 — A pod is admitted with its cloud identity, and a hook Job proves it before the workloads exist

**Status:** accepted

## Context

On EKS a workload gets cloud credentials from a Pod Identity association: a
role, an association binding it to a namespace and a ServiceAccount, and the
account itself. The credentials reach a container as environment
(`AWS_CONTAINER_CREDENTIALS_FULL_URI` and `AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE`)
and a projected token volume, and **they are injected by the cluster's
pod-admission webhook, when the pod is created, and at no other time.** A
container restarted inside a pod never gets them. Only a new pod does.

The association is eventually consistent. The AWS API reference for
[CreatePodIdentityAssociation](https://docs.aws.amazon.com/eks/latest/APIReference/API_CreatePodIdentityAssociation.html)
says so in as many words: associations "may take several seconds to be
effective after the initial API call returns successfully", and recommends
making changes "in a separate initialization or setup routine". The controller
that creates the association reports it `Synced` when the API returns, which is
before the webhook can see it. The upstream issues record it as a known gap,
not a bug in any one cluster
([amazon-eks-pod-identity-webhook#264](https://github.com/aws/amazon-eks-pod-identity-webhook/issues/264):
wait "under five seconds";
[containers-roadmap#2507](https://github.com/aws/containers-roadmap/issues/2507):
there is no readiness API to wait on).

So a release that creates the association and then, a few seconds later,
creates a pod, can produce a pod admitted in the gap. It runs for its whole
life without credentials and crash-loops on its first call to the cloud, with
an error that names a missing credential a long way from the cause. It was
seen on a managed cluster where pods were created two to three seconds after
the association synced, twenty sync waves later. Nothing that orders a rollout
helps, because everything that orders a rollout waits on a resource's status,
and the status is already `Synced`: sync waves, Helm 4's readiness waits, a
Flux `dependsOn` and its health checks all pass. A scheduling gate does not
help either, because admission has already run by the time a gate opens.

[Platform rule 11](../contracts/platform.md) already splits a component with a
cloud identity into an infrastructure chart (the role, the association, then
the account) and an application chart. That orders the objects. It cannot order
the webhook.

## Decision

**A release whose component runs as a cloud-identity account starts with a
check that a pod admitted now has the identity.**

- `identity-check` is a small command, **owned by
  [truvity/k8s](https://github.com/truvity/k8s) (`cmd/identity-check`)**, with
  its image. This repository holds the rule, the template and the example, and
  releases no binary for it. The command works out which workload identity its
  own pod was admitted with (Pod Identity, or IRSA), asks STS who it is
  (`GetCallerIdentity`, which needs no permission), and exits non-zero if it
  has none, if STS refuses it, or if the role is not the one named by
  `-expect-role`. With no injected identity it fails **before** the SDK is
  built, and it never reads the instance metadata service: a node's role would
  answer, and a pod with no identity of its own would look as though it had
  one. A chart names the image in its own `images.identity-check`.
- The library chart renders it as a Job, `service-lib.identityCheck`, running
  as the component's ServiceAccount. It is a Helm hook (`pre-install,pre-upgrade`,
  deleted `before-hook-creation`, weight `-20`), which Argo CD runs as PreSync
  and a Helm-based controller runs as Helm does, so one template serves all
  three install paths. `restartPolicy: Never` is the load-bearing line: a Job
  creates a new pod for every attempt (`backoffLimit: 6`,
  `activeDeadlineSeconds: 300`), and each new pod is admitted afresh. When one
  comes up with the identity, the webhook can see the association, and the
  workloads created after the hook are admitted with it too.
- The including chart says when a component has a cloud identity
  (`cloudIdentity`); there is no switch for a deployer. The example renders it
  for its one such component on a primary install, where the account is the
  infrastructure chart's.
- Because a hook runs before anything else in its release, the account **must
  already exist**: it belongs to the infrastructure release, which is installed
  and healthy first. Platform rule 11 now says so.

## Consequences

### Good

- The failure moves from a crash loop in a workload, discovered by whoever is
  watching, to a failed hook with the cause in its first line: `no workload
  identity was injected: the pod was admitted before its association was
  visible; the Job retries with a new pod`.
- It is a measurement, not a guess: it waits exactly as long as the webhook
  takes, a second or a minute, and no longer.
- One mechanism for every install path, and no new moving part in the cluster:
  a Job, an image, and a flag-less command.
- The chart couples to nothing about Pod Identity's internals. It does not
  name the environment variables or the token volume; the command reads them.

### Bad

- **A release can fail before it has deployed anything,** after at most five
  minutes, if an association never becomes visible. That is the correct
  outcome and a new one: the failure used to come later and look like an
  application fault.
- **It proves the past, not the future.** A pod admitted at 12:00:00 shows the
  webhook can see the association at 12:00:00. An association that is later
  changed or deleted is a different event, and this does not cover it.
- **One more image to depend on, and one more Job in a hook phase.** The image
  is another repository's release, so a chart pins it by tag and digest like
  any other. Under Argo CD the PreSync phase runs on every sync, and a failed
  hook blocks it.
- **The check pod needs a network policy of its own.** Its labels are distinct
  on purpose: `app.kubernetes.io/component` is `<component>-identity-check` and
  it carries `service-lib.io/identity-check`. With the component's own labels,
  its Service would select it, and a Job pod with no readiness probe is Ready
  while it runs, so real traffic would reach a pod that serves nothing; a
  disruption budget would count it too. A policy aimed at the component does
  not match it either, so the infrastructure chart renders egress for the check
  (name resolution, the Pod Identity agent, the cloud API).
- Only Pod Identity has this gap in the form described here. An account
  annotated for IRSA is read from the account itself at admission and has no
  association to wait for; the command accepts it because a check that fails on
  a working identity would be worse than none.

### Neutral

- A cluster that does not use EKS Pod Identity renders none of this: the
  including chart does not set `cloudIdentity`.

## Roads not taken

- **An init container, a startup probe or a liveness probe in the workload.**
  A restart in the same pod never goes back through admission, so every one of
  them can only report, forever, that the pod is broken. The pod has to be
  replaced; a container cannot do that.
- **A fixed sleep after the association.** The cloud gives no bound, a number
  that is safe today slows every sync to protect against a case that most do
  not hit, and one that is too short fails silently, which is the failure being
  removed.
- **A policy engine (Kyverno or a ValidatingAdmissionPolicy) that rejects a pod
  admitted without the variables.** There is no precedent for one in these
  contracts. It blocks pods on a genuinely broken association as well as a late
  one, and the controller's retry backoff grows with every rejection, so the
  recovery gets slower exactly when it is wanted.
- **Injecting the environment and the token volume in the chart.** That couples
  every chart to Pod Identity's internals and contradicts
  [platform rule 2](../contracts/platform.md): a chart declares the account and
  names no mechanism. It would also paper over a real fault: a pod
  given credentials the cluster never granted it looks healthy until the
  first call is refused.
- **A manual rollout restart.** It is what happens today, by hand, after the
  page.

## Sources

- AWS, [CreatePodIdentityAssociation](https://docs.aws.amazon.com/eks/latest/APIReference/API_CreatePodIdentityAssociation.html):
  "Similar to AWS IAM behavior, EKS Pod Identity associations are eventually
  consistent, and may take several seconds to be effective after the initial
  API call returns successfully. You must design your applications to account
  for these potential delays. We recommend that you don't include association
  create/updates in the critical, high-availability code paths of your
  application. Instead, make changes in a separate initialization or setup
  routine that you run less frequently."
- [aws/amazon-eks-pod-identity-webhook#264](https://github.com/aws/amazon-eks-pod-identity-webhook/issues/264):
  a pod created right after its association is admitted without credentials.
- [aws/containers-roadmap#2507](https://github.com/aws/containers-roadmap/issues/2507):
  no way to ask whether an association is effective.
