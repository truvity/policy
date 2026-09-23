# Probes and rollout

**The rule.** Two endpoints on a listener of their own, and a deployment that
replaces instances without a gap.
[service.md §3 and §9](../contracts/service.md) are normative.

**Why.** Readiness has to be answerable when the service's own listener is
saturated, which is exactly when somebody is asking; and a probe endpoint on
a routed port is reachable by people who were never meant to reach it. As for
the rollout: a drain in the code is decoration unless whatever deploys the
service gives it the time it asks for and keeps one instance serving
throughout.

## Where to look

| Language | Probes | Drain |
|---|---|---|
| Go | `runtime.Probes` and `runtime.Serve` in [`internal/runtime/runtime.go`](../../examples/url-shortener/internal/runtime/runtime.go) | `runtime.Drain`, and `signal.NotifyContext` in each `main` |
| TypeScript | follows | follows |
| Kotlin | follows | the framework's graceful shutdown |
| Python | `Probes` in [`runtime.py`](../../examples/url-shortener/log/src/url_shortener_log/runtime.py), on the standard library's server | the consume loop in `__main__.py`: stop fetching, write what is held, acknowledge, bounded by the same number the chart gives the platform |

The chart side is
[`examples/url-shortener/charts/url-shortener/templates/`](../../examples/url-shortener/charts/url-shortener/templates/):
the probe blocks in each workload, the shared drain and spread helpers in
`_helpers.tpl`, and `poddisruptionbudget.yaml`.

## The two endpoints

| Path | Answers | Checks |
|---|---|---|
| `/health/live` | the process is alive | **nothing else** |
| `/health/ready` | this instance can serve now | what it needs to serve |

**Liveness must not check a dependency.** A liveness probe that fails when a
database is slow restarts a healthy process and makes an outage worse: every
instance restarts at once, none of them fixes the database, and the restarts
become the incident. The example's liveness handler closes over nothing, and
says so.

## The four numbers, which are one number

| Setting | Where | Derived from |
|---|---|---|
| the service's shutdown timeout | its configuration file | `drain.seconds` |
| `terminationGracePeriodSeconds` | the pod | `drain.seconds` + the delay + a margin |
| the pre-stop delay | the pod | `drain.preStopSeconds` |
| `maxUnavailable: 0` | the deployment | — |

They disagree by default, and both ways of disagreeing look like a network
fault. A grace period equal to the service's timeout kills it at the exact
moment it would have finished. No pre-stop delay means traffic keeps arriving
for as long as it takes the routing layer to notice the endpoint is gone.

## Traps

**A rollout and a drain are different events, and so are a rollout and a node
drain.** `maxUnavailable: 0` makes the first gapless. Only a disruption
budget makes the second gapless, and nothing in a deployment's strategy
implies one.

**A budget of zero disruptions does not protect a service.** It wedges the
drain of the node underneath it, and the service goes down anyway — later,
and in a way nobody connects to the budget.

**Spread with `ScheduleAnyway`, not `DoNotSchedule`.** A single-node cluster
is the ordinary case for a gate and a laptop, and a constraint that cannot be
met there leaves pods Pending with an event nobody reads.

**One replica is not a rollout strategy.** Whatever the strategy says, the
instance goes away before its replacement is ready.

**Replacing everything at once is allowed, and the reason goes next to the
setting.** A single writer or an exclusive volume is a real reason. The next
reader's first assumption will be that it was an oversight, so answer them in
the chart.
