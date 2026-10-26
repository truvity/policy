# Cloud Identity at Pod Admission

Self-contained summary of platform contract 1.4 §11 and decision 0013.
This file owns the checker **v0.34.0+** minimum. The worked include/tests
belong to [url-shortener][product].
Sources: [platform](https://github.com/truvity/policy/blob/master/docs/contracts/platform.md),
[decision 0013](https://github.com/truvity/policy/blob/master/docs/decisions/0013-pods-are-admitted-with-their-identity.md),
and `charts/service-lib/templates/_identity_check.tpl` in the reference checkout.

## Why waves are necessary but insufficient

An EKS Pod Identity association can report `Synced` before the pod-admission
webhook sees it. Credentials are injected only when a pod is created. Restarting
a container in the same pod repeats neither admission nor injection, so it can
crash-loop indefinitely with no credentials. A sleep or a controller's readiness
wait measures elapsed time/status, not whether admission works.

The required sequence is:

1. The **infra release** renders role → association/binding → ServiceAccount in
   increasing waves (the example uses 0, 1, 2). Keep
   `argocd.argoproj.io/sync-options: Prune=false,Delete=false` on the account.
2. Install the infra release and wait for it to be healthy. A hook cannot use
   an account that its own release has yet to create.
3. The **application release** references that account and calls
   `service-lib.identityCheck` before its workloads are created.

The binding mechanism is the platform's. The public `truvity/k8s`
`cmd/identity-check` supports Pod Identity and IRSA, asks STS `GetCallerIdentity`,
and optionally checks a role name and an expected AWS account. It refuses
missing injected identity before constructing the SDK and does not fall back
to the node's instance metadata.

## Library inputs and resulting Job

Pass the component's `context`, `name`, `platform`, plus:

| Input | Meaning |
|---|---|
| `cloudIdentity` | true renders the check; false/absent renders nothing. The including chart derives this from its identity/tier facts, not an independent deployer switch |
| `expectRole` | optional role name to verify |
| `expectAccount` | optional AWS account identifier, a quoted 12-digit string; malformed nonempty values fail the render when the check is enabled; absent/empty adds no flag |
| `image` | optional pinned image override; otherwise top-level `images.identity-check` |

The product's [include][product-include] is:

```gotemplate
{{- $log := include "url-shortener.platform" (dict "root" $ "component" "log") | fromJson }}
{{ include "service-lib.identityCheck" (dict "context" $ "name" "log" "platform" $log "cloudIdentity" (not (not (include "url-shortener.identityInInfra" .))) "expectAccount" (.Values.identityCheck | default dict).expectAccount) }}
```

This is the example's component mapping, not a reason to name every product's
component `log`. Adapt the derivation and keep the library include.

The example's product-level value is `identityCheck.expectAccount`, empty by
default. A platform sets the expected account as a **string**; do not commit
a real account identifier or a literal twelve-digit placeholder into this
public repository. The library forwards nonempty input as `-expect-account`.
The command checks both STS `Account` and the caller ARN's account, so an
identity in the wrong account fails the hook instead of failing later on
application permissions. `expectRole` remains a separate optional assertion.

**Image compatibility is required:** `-expect-account` first appears in
`truvity/k8s` v0.34.0. An earlier checker cannot parse it and fails every
attempt; move `images.identity-check` to a verified v0.34.0+ tag/digest pair
before enabling the account assertion. Leaving the value empty preserves
the earlier behavior.

| Job field | Current library contract |
|---|---|
| `helm.sh/hook` | `pre-install,pre-upgrade` (Argo CD PreSync) |
| hook weight | `-20`, before the migration's weight 0 |
| deletion policy | `before-hook-creation` |
| `restartPolicy` | `Never`: each retry creates a newly admitted pod |
| `backoffLimit` / `activeDeadlineSeconds` | 6 / 300 |
| account | the component's existing cloud-identity ServiceAccount |

The command and image are owned/released by
[truvity/k8s](https://github.com/truvity/k8s), not policy or the adopting product.
The selected product's `images.identity-check` owns the exact tag/digest pin;
v0.34.0 is the minimum for `expectAccount`, not an instruction to downgrade a
newer verified pin. The product release does not rebuild or restamp this image.

## Selectors and network access

The check pod has component label `<component>-identity-check` and
`service-lib.io/identity-check: "true"`. Never give it the component's ordinary
selector labels: a Job pod can be Ready while running, so a Service would send
real traffic to it and a disruption budget would count it.

The infrastructure chart supplies egress for these separate labels: DNS, the
Pod Identity agent and the cloud API. A policy selecting only the ordinary
component does not admit the check. Do not fix that by widening the component's
Service selector or by injecting cloud credentials in the chart.

## Checks and limits

The product's `charts/identity_check_test.go` checks the hook, account, fresh-pod
retries, tier behavior, waves and selector exclusion. Infra renders include
check-specific egress. `TestExpectAccountIsPassedToTheCheckOnlyWhenSet` checks
optional flag forwarding; `TestAMalformedExpectAccountFailsTheRender` checks
refusal at render time. The command's account/ARN comparison tests belong to
`truvity/k8s`, not these template tests.

Policy's Go tests and advisory canary are not a live identity test. The product
render tests prove wiring; its protected shared-cluster integration can prove
injected credentials. A consumer's kind tier has no EKS admission webhook/IAM.
The hook measures admission at that moment, not the correctness of a later
association change and not every permission the application will need.

[product]: https://github.com/truvity/url-shortener/tree/76770607aa08baa58331668c228dacd62a13a89a
[product-include]: https://github.com/truvity/url-shortener/blob/76770607aa08baa58331668c228dacd62a13a89a/charts/url-shortener/templates/identity-check.yaml
