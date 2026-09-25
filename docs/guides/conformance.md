# Checking a repository against the contracts

Not normative. This is how to tell whether a repository satisfies
[the contracts](../README.md), and which parts a machine does for you.

## What CI already checks

| Contract | Rule | Mechanism |
|---|---|---|
| [service](../contracts/service.md) | no dependency-injection container | the [depguard block](../../lint/README.md) refuses the imports |
| service | configuration is a validated file | the loader refuses anything else; the schema golden fails on drift |
| service | no build step in a runtime image | the image lint; the release builds every platform in one job |
| [config](../contracts/config.md) | the binary and the chart read one schema | the chart's tests validate what they render with the binary's schema |
| config | strictness | a negative fixture per schema: an unknown key must fail |
| [repository](../contracts/repository.md) | the gate needs nothing but the checkout | CI runs recipes by name; a recipe needing more is its own job |
| repository | public repositories name no particulars | the leak canary, on every commit and in CI |
| repository | public repositories run hosted | the shared workflow refuses a self-hosted runner for a public caller |
| [release](../contracts/release.md) | one tag stamps everything | the release workflow, from the tag, in CI |

## What a reviewer checks

The rest, in the order that finds problems fastest:

1. **Does `just check` pass from a clean clone, with the network off?** If it
   needs a registry, a container or a cluster, the gate is not the gate and
   the rest of this list is guesswork.
2. **Is every key the deployment sets a key the service reads?** The schema
   answers it. If the chart writes environment variables the service parses
   by hand, the contract is not in force whatever the documents say.
3. **Does a secret appear in a rendered file anywhere?** Configuration is
   rendered, logged and committed as fixtures. A secret must survive all
   three being true, so it is named in the file and carried in the
   environment.
4. **Does `/health/ready` check what the service needs to serve, and
   `/health/live` check nothing?** A liveness probe that touches a database
   turns a slow dependency into a restart loop, which is how a small outage
   becomes a large one.
5. **Does SIGTERM drain?** Nothing in a test suite notices when it does not.
   The signal is what a rolling upgrade sends.
6. **Are the log lines JSON, on stdout, at one level?** This is checked by
   eye today, and the conformance table in the service contract says so
   rather than implying otherwise.
7. **Is each service-to-service call the right shape?** An ownership boundary
   is an RPC, fan-out is an event, and a direct read of another service's
   store is written down where it is used, with the latency argument that
   justifies it.
8. **Is one request one trace?** Fetch a single trace by its id from the trace
   store and read the tree: every service the request touched is in it, and
   no span but the root is missing its parent. Spans that exist per service
   but do not connect are the failure, and every exporter's health check
   passes while it is happening. The boundaries that drop context are listed
   in [logging-and-telemetry.md](logging-and-telemetry.md#a-trace-that-stays-whole).

## Adopting the import ban

```sh
# 1. paste lint/golangci-depguard.yaml under linters.settings
# 2. add `depguard` to linters.enable
golangci-lint config verify   # FIRST: a misplaced block is accepted silently by `run`
golangci-lint run ./...
```

Every finding is either a migration or an exception. An exception is a line
in the repository's own configuration with a comment saying why, never a
silent removal of a rule.

## The honest gaps

Two rules are checked by review alone: the shape of a log line, and whether a
boundary should be an RPC or an event. Both are named in the service
contract's conformance table. A table with a gap in it is worth more than one
that implies coverage it does not have, because a gap is a thing somebody can
close.
## The recipe names are the check

`repository.md` §2 fixes them: `check build test lint vuln charts golden
leak-canary`. A repository that has the job under another name has made
every caller special — anything automating across repositories has to learn
its exception, and a person moving between them has to read a recipe file to
find out what the linter is called.

It is worth listing them against `just --list` rather than assuming, because
the failure is invisible from inside the repository: everything runs, the
gate is green, and only a caller from outside notices. This repository
shipped the chart check as `kubeconform` — an accurate name for the tool and
the wrong name for the recipe — and did not notice until the contract was
read back against it.

## The two charts, and the tier

A repository whose service owns a database or a stream ships **two**
charts: the infrastructure one and the application one. The reasons are in
`contracts/platform.md` rules 6 and 11, and the checklist is short.

- **Does the infrastructure chart provision per INSTALL?** Its resources
  are functions of this release in this namespace. If any of them is
  really per cluster, it belongs to the platform instead, and putting it
  here means every test install gets a copy it should not have.
- **Can something that runs per cluster supply any of it?** If you are
  tempted to move a resource into a provisioning stack, check that a test
  install still gets one. Those stacks run per cluster; there is no
  "install" at that level, so the answer is usually no.
- **Does it take a tier?** An install that is not the deployment should
  provision less: a shared store with its own prefix, and the namespace's
  standing identity. An engineer's namespace often cannot create custom
  resources at all, so a chart that always mints them is a chart they
  cannot install.
- **Does anything need a credential somebody has to generate?** If so, a
  test install has no provider for it. Prefer a client certificate — but
  check what your operator actually offers first, because the one here
  mints none for a managed role. `contracts/platform.md` has what that
  costs and which of the two CA arrangements is cheaper.
- **Does it name a vendor's kinds?** It may, if they are objects the
  install owns. The bar is that the chart RENDERS without that cloud, not
  that it installs on one — so those kinds belong behind the tier, and
  the tier that renders none of them is the default. A chart whose cloud
  objects render unconditionally is a chart for one cloud.
- **Is there more than one installer?** There will be: a deployment
  installs it, and so does whatever installs a copy per engineer and per
  CI run. Both pass the same values. If one of them needs a different
  chart, the interface is wrong.
