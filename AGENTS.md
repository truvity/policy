# Instructions for an agent working here

This file is for a model, human or otherwise, that arrives with no other
context. It is short on purpose. Everything it points at is normative; this
page is only the order to read it in and the traps that are not obvious from
the documents themselves.

There are two jobs, and they have different rules.

---

## Job one: changing this repository

### Read first

1. [docs/contracts/repository.md](docs/contracts/repository.md) — how this
   repository is laid out and what its gate is.
2. The contract you are touching, in full, before editing a word of it.
3. [docs/decisions/](docs/decisions/README.md) — a rule you are about to
   change may have a record saying why it is that way. A decision is never
   edited after acceptance; it is superseded by a new one that links back.

### The gate

`just check` is everything CI runs, and it needs nothing but this checkout:
no network, no containers, no credentials. Run it before you push. It is
build, test, lint, vulnerability scan, generated-code drift, and the leak
canary.

Two recipes are **not** in the gate because they need more than a checkout:
`just ts` fetches from a registry, and `just cluster-*` needs a container
runtime. CI runs them as their own jobs. The cluster lane is a pull-request
gate, so a change that breaks the example breaks the build.

Four things the gate checks that surprise people:

- **Every relative link in a Markdown file must resolve.** A link to a file
  that is still in an unmerged pull request fails the build. Add the file in
  the same change as the link.
- **Generated code is committed**, and regenerating must produce no diff. If
  you change a schema, run `just ts-schemas` and commit the result.
- **Nothing over a megabyte** may be committed. This repository is public and
  its history cannot be unpublished.
- **The leak canary runs on every commit**, not just in CI, via a hook.

### Writing rules

- **No particulars, ever.** No organisation name, no host name, no cluster
  name, no team name, no ticket key, no person, no incident, no internal
  repository. This repository is public and its history is permanent. Say
  "an estate adopting these contracts", "the platform", "a deployment". If
  you find yourself needing a real name to make a sentence work, the sentence
  is about a deployment and does not belong here.
- **A rule states what must be true, why, and how it is checked.** A rule
  with no check is a preference, and the contracts say so about themselves.
  If you cannot name the check, say that it is unchecked — a conformance
  table with an honest gap is worth more than one that implies a check that
  does not exist.
- **Name the failure, not the virtue.** "A liveness probe that checks a
  database restarts a healthy process and makes an outage worse" tells a
  reader what to avoid. "Probes should be lightweight" does not.
- **Every consumer-visible change adds a bullet to `CHANGELOG.md`**, written
  for someone consuming this repository, not for whoever wrote the change.

### Merging

Pull requests **merge by rebase**. A branch with merge commits in it is
refused; start a new branch with one squashed commit rather than fighting it.
Check `mergeable` before you try: a conflicting pull request does not just
fail to merge, it runs no checks at all, because there is no merge ref for
the workflows to run against.

---

## Job two: bringing another repository to this shape

This is the common case, and it is a survey followed by a series of small
changes, not one large one.

### The order

1. **Read [docs/guides/conformance.md](docs/guides/conformance.md).** It is
   the checklist, and it says which of its items CI can check for you.
2. **Survey before you change anything.** Produce the list of gaps first.
   Most repositories fail several rules in ways that interact, and a fix
   applied before the survey is a fix you will redo.
3. **One concern per pull request.** Logging is a pull request. Probes are a
   pull request. The rollout block is a pull request. A single change that
   touches all three is unreviewable and un-revertable.
4. **Prove each rule where the guide says it is proved.** Most have a check;
   use that check rather than asserting the rule is now satisfied.

### What is usually wrong, in the order it is usually wrong

This is what a survey of real services found, and it is a reasonable
expectation for the next one:

- **No rollout block at all** — no disruption budget, no grace period, no
  pre-stop delay, and a single replica. The drain in the code is real and
  nothing honours it.
- **Probe paths differ per service**, sometimes per component within a
  service, and liveness often checks a dependency or is literally the same
  handler as readiness.
- **Logs**: a level that is hardcoded, or a per-package level scheme, or no
  logger configured at all so the language's default text handler is what
  ships.
- **A library logging in its own format** past the service's logger.
- **Telemetry** configured by a condition on an environment name, which is
  never set, so it exports to a console in the environment it was written
  for.
- **A chart that hard-codes one grant mechanism**, which makes it installable
  on half the clusters it should run on.
- **A schema with no build configuration** beside it, and a hand-written
  client that drifted from it.
- **A toolchain manifest with "latest"** in it.

### What never to do

- **Do not weaken a schema to make a render pass.** The schema refusing
  something is the schema working. Fix the values.
- **Do not add a dependency-injection container**, or a configuration-mapping
  library, or a second logger. The import ban in [lint/](lint/README.md) is
  the mechanical half of this; the reasons are in
  [0001](docs/decisions/0001-no-di-containers.md) and
  [0002](docs/decisions/0002-config-file-plus-env.md).
- **Do not copy a chart's infrastructure resources into its application
  chart** to avoid an ordering problem. The ordering problem is real and the
  split is the answer;
  [platform.md §6](docs/contracts/platform.md) says why.
- **Do not turn transport identity on in a chart's defaults.** It is off in
  the chart forever; the platform turns it on.
- **Do not bring a finding from a private repository into this one.** Fix it
  there; state the rule here in general terms.

### When the contract is wrong

It happens, and it is the point of putting a real product through the
contracts. If a rule cannot be satisfied by a real service for a real reason,
**change the rule here first**, in its own pull request, with the reason —
then adopt it. A repository carrying an exception that the contract does not
acknowledge is how a contract stops being true.
