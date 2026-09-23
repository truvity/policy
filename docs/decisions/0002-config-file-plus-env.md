# 0002 — Configuration is a file; the environment carries secrets

**Status:** accepted

## Context

Three ways to configure a service are in common use, and services in one
estate had all three: flags with environment fallbacks, environment
variables mapped onto a nested structure by a library, and a file.

The environment is attractive because a container platform sets it. What it
cannot do:

- **Express structure.** A list of trusted issuers, a map of stores, becomes
  a naming convention with indices in it.
- **Be typed.** Everything is a string, so every value is parsed twice: once
  by whatever renders it and once by the service.
- **Be validated before the process starts.** A wrong variable is found by
  starting the service, which in an orchestrator means a crash loop that
  reports "unhealthy", not "you spelled the key wrong".
- **Be diffed.** Reviewing a change to eleven environment variables spread
  across a deployment template tells you what changed, not what it means.

A mapping library papers over the first two and makes a fourth problem: the
mapping itself becomes a thing to know. Variables that are not registered are
silently dropped, which is the worst possible failure — the service starts,
with a default nobody chose.

Secrets are the opposite case. A file is rendered into a config map, printed
when someone debugs a deployment, and committed as a test fixture. A secret
must survive all three being true, and the environment — injected from
whatever holds secrets, never rendered into an object that is read back — is
where it belongs.

## Decision

**A service reads one configuration file, validated against a schema before
anything else happens. Secrets, and only secrets, come from the
environment**, each one declared.

The path to the file is a single argument or a single environment variable.
A service takes no other structural input: no flag that overrides a field, no
variable that means the same thing as a key.

## Consequences

### Good

- What a service is configured with is one reviewable document.
- The same schema validates the file the service reads and the file the
  deployment renders, so drift between them is a test failure.
- Structure and types are free.
- Secrets are in exactly one place, and it is not the place that gets
  printed.

### Bad

- **A file must be rendered and mounted**, which is a step for anything that
  used to set two variables. For very small services this is genuinely more
  work.
- Changing configuration means re-rendering, not editing a variable in a
  console.
- Two mechanisms exist, so "where does this value come from" has two
  answers — mitigated by the rule that the split is exactly secrets and
  nothing else.

### Neutral

- Reloading configuration without a restart is not part of this decision. A
  service that needs it re-reads the same file and validates it the same way.
