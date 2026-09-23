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
