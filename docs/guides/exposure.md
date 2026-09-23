# Exposure

**The rule.** A chart that is reachable from outside renders **one route**,
whose parent is a value and whose rules are **named**. It renders no gateway,
no listener, no certificate and no DNS record.
[platform.md §7](../contracts/platform.md).

**Why.** The edge is shared between services and described by a catalogue the
platform owns. A chart that rendered its own listener or certificate would
compete with that catalogue and win intermittently, which is worse than
losing. And rules are named because a policy attaches to a rule *by name*.

## Where to look

| Concern | Where |
|---|---|
| the route | the `HTTPRoute` at the end of [`templates/redirect.yaml`](../../examples/url-shortener/charts/url-shortener/templates/redirect.yaml) |
| its values | the `route` block in [`values.yaml`](../../examples/url-shortener/charts/url-shortener/values.yaml) |
| the refusal when a parent is missing | the negative fixture in `charts/testdata/invalid/` |
| the test that every rule is named | the chart tests in [`charts/`](../../examples/url-shortener/charts/) |

## The parent

A route with no parent attaches to whatever the cluster's default happens to
be — which is nothing in most clusters and the wrong gateway in the rest. It
renders as healthy, reports zero attached routes, and the only symptom is a
404 nobody can explain.

So the parent is **required** when the route is enabled, and the chart says
so in the error rather than defaulting.

## Why the rules are named

A policy — for sign-in, for CSRF, for rate limiting — targets a rule by name.

**A policy whose target names no rule that exists is not refused.** It is
simply not attached. The route keeps serving, the render looks correct, and
the endpoint that was supposed to require sign-in does not. There is no error
anywhere in the chain; the policy reports that its target was not found, in a
status nobody reads.

That is why the rule name is a value, why the chart always emits one, and why
there is a test asserting every rendered rule carries a name.

## Traps

**Off by default.** A chart that exposes something the moment it is installed
exposes something nobody decided to expose. The example's route is disabled
until asked for.

**The parent's namespace is the exposure's, not the application's.** Getting
this wrong produces the silent non-attachment above.

**Attachment is a grant, not a declaration.** Naming a parent does not
entitle a route to it: the parent decides which namespaces may attach. A
route can be perfectly correct and still attach to nothing, so verify against
what the gateway reports, not against the render.
