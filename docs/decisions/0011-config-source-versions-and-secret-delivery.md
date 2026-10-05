# 0011 — Configuration is immutable per instance, versioned, and found the same way everywhere; secrets arrive by name three ways

**Status:** accepted. Secret spelling and sources amended by [0012](0012-stabilization-amendments.md)

Amends [0002](0002-config-file-plus-env.md) in two places — where secrets
come from, and reloading — and keeps the rest of it.

## Context

[0002](0002-config-file-plus-env.md) made configuration one file validated
against a schema, and the environment the one place secrets come from. It
left three things open, and services adopting these contracts on more than
one platform ran into all three.

**Where the file is.** 0002 says "a single argument or a single environment
variable", and nothing more. Each service writes the lookup itself, and
nothing holds them to one spelling of the flag, one convention for the
variable, or one answer to which wins when both are set. A platform that runs
more than one service then has to know each one's, which is the special case
the contracts exist to remove.

**What happens when the shape changes.** A configuration's schema is not
fixed for ever. A key moves into a block, a list replaces a scalar, a field
changes meaning. With nothing in the document saying which shape it is, a
binary validates whatever it is given against the only schema it knows. A
new file beside an old binary is checked against the old schema: it either
fails on keys that are perfectly good in the new shape, or passes because a
key that changed meaning kept its name. Neither says "this file is newer than
this binary". And a change of shape had to reach the binary and the file at
the same moment, which no rollout does.

**Secrets on a platform that is not a cluster.** The environment works on
Kubernetes. It does not survive a serverless platform: a function's
environment is one small, fixed budget shared by every variable, and a service
with several credentials and its telemetry settings runs out of it. There is
no larger budget to ask for. Some secrets are also per client — a credential
for each party the service deals with — and a set that grows with the
business cannot be a variable each, because each new client would be a
deployment change.

Underneath all three is a question 0002 left neutral: whether a running
instance's configuration may change. 0002 said a service that needs to reload
"re-reads the same file and validates it the same way". The platforms these
contracts run on make that unnecessary: a Deployment rolls a changed
ConfigMap out as new pods, a server restarts its unit, a serverless function
publishes a new version.

## Decision

**A running instance's configuration does not change.** The file is read
once, at start-up. Changing it means new instances, by whatever rollout the
platform has. What does change under a running instance — a rotated
credential, the state a service keeps in its own store — is read where it
lives, when it is used, and is not configuration.

**The configuration is a file on every platform, found the same way.** A
Kubernetes ConfigMap mounted into the pod, a file on a server's disk, a
layer a serverless function mounts (on one common platform, under `/opt`):
the binary does not know which. It takes the path from `--config <path>`
(`-config` is the same flag), otherwise from one environment variable it
names, conventionally `<APP>_CONFIG`; the argument wins. The loaders resolve
it (`config.PathFrom` in Go, `configPath` in TypeScript), against one shared
table of cases.

**A document says which version it is, and a binary reads two.** A document
may carry `apiVersion: <group>/<kind>/v<N>` at its root; absent means v1. The
loader reads it before validating anything and picks that version's schema.
A binary declares the version it is written against, N, and may also read
N-1, with a conversion from N-1 to N that the service writes and registers
for its kind. Anything else — newer than N, older than N-1, another kind, not
of the form — is refused at start-up, naming `apiVersion`. The schemas stay
authored, one per version, as
[config.md §2](../contracts/config.md) requires.

**A secret is a declared name, delivered by environment variable, a mounted
file, or a declared secret source resolved at start-up; never a value in the
configuration file.** The source is for a platform with no secret object to
deliver through. On Kubernetes, [platform.md §3](../contracts/platform.md) is
unchanged: the secret is a Secret, delivered as a variable or a file.

## Consequences

### Good

- **A version change is two ordinary deployments.** The binary that reads N
  and N-1 ships first, still fed the N-1 file; then the file moves to N.
  Nothing has to happen at the same moment.
- **A file newer than its binary says so** — `apiVersion: v3 is newer than
  this binary reads (v2, v1)` — instead of failing on keys that are right,
  or passing on keys that changed meaning.
- **Every existing document is already valid**: no `apiVersion` is v1.
- **One spelling for where the file is**, so a platform passes the path the
  same way to every service, and the library chart's `-config` already is it.
- **No reload path.** A reload is a second code path that runs only in
  production, under load, half-way through a request. A new instance runs the
  one path every test ran.
- **A serverless service can hold its secrets**, including one per client,
  without breaking the rule that they never sit in the file.

### Bad

- **A rollback has an order.** The binary before N does not read N, so the
  file goes back first, then the binary. Getting it backwards is a start-up
  refusal — loud, but an outage for that rollout.
- **Every conversion is code the service carries** for one version, and must
  be deleted when N+1 ships. A conversion that is not deleted is not a
  correctness problem, only a reader's.
- **Three ways for a secret to arrive** where 0002 had one, so "where does
  this secret come from" has three answers. Each is declared by name in the
  schema, which is what keeps the answer findable.
- **A secret source read at start-up is a dependency at start-up**: a store
  that is down means a service that does not start. That is the right
  failure — loud, and before serving — but it is a new one.
- **Changing configuration costs a rollout**, never a signal. For a value an
  operator wants to turn often, that is the cost of the rule, and the answer
  is that it was never configuration.
- **Two of the four loaders do not read two versions yet.** Python and Kotlin
  read v1 and refuse anything later; a service in either cannot change shape
  without a loader change first.

### Neutral

- Hot reloading, which 0002 left open, is now closed. A service that needs a
  value to change while it runs is reading state, and state lives in a store.
- Telemetry stays where [0006](0006-telemetry-is-the-sdk-environment.md) put
  it: OpenTelemetry's own environment, outside the file.
