# The repository contract

**Normative.** What a repository looks like, so that someone who has
worked in one can work in the next without being told anything.

## 1. One product per repository

A repository holds one thing that is released. Two products in one
repository share a version, a release cadence, a review population and a CI
budget, and none of those are things they have in common.

The exception is a product and the example or tooling that exists only to
serve it, which are the same thing released together.

## 2. The gate is one command, and it needs nothing

`just check` is the gate. It runs what CI runs, and it needs **nothing but
the checkout**: no network, no containers, no cluster, no credentials.

That constraint is the whole value. A gate that needs a cluster is a gate
people run once a week and then argue with. Everything heavier — a suite
that needs a database, a cross-platform build, a cluster test — is its own
recipe, run as its own CI job, and named so nobody is surprised by what it
needs.

CI runs recipes **by name**, never by re-implementing them in a workflow. A
check that passes locally and fails in CI is then a bug in a recipe rather
than a difference nobody can reproduce.

## 3. The toolchain is declared

One manifest names every tool and its version, with a lock file committed,
and one command materialises it. The same versions on a laptop and in CI.

A missing tool is a line added to the manifest. Never a fetch in a shell, and
never a PATH someone exported.

## 4. Documentation has fixed paths

| Path | Holds |
|---|---|
| `README.md` | what this is, what ships, who it is for, how to run it, how to develop it |
| `CHANGELOG.md` | one heading per released version, newest first, written for a consumer |
| `CONTRIBUTING.md` | the rules that are not obvious from the code |
| `SECURITY.md` | where to report a vulnerability, and what is in scope |
| `docs/` | everything else, indexed by `docs/README.md` |

A reader who has read one repository knows where to look in the next. The
paths are the contract; the length is not.

## 5. Required checks are named, and never renamed casually

A repository has exactly one required status context, and it is a job that
depends on everything else and reports a single verdict.

Matrix jobs and reusable-workflow jobs report under composed names that
change when the matrix changes, so requiring one of those directly wedges
every open pull request the moment anything moves. **Renaming a required job
wedges every open pull request**, which is worth knowing before it is
discovered.

## 6. Dependencies are bumped by a bot, on a schedule

A bot proposes every dependency bump and the whole gate runs against it.
Groups that must move together are declared, because two pull requests that
each cannot pass until the other merges is the failure mode, and it is
diagnosed as a flake the first three times.

## 7. Public repositories

A public repository is held to more, because its history cannot be
unpublished:

- **Mechanism only.** Nothing names an organisation, an account, a cluster, a
  hostname, an environment, a team, a person, an internal repository or a
  ticket — in code, documents, tests, commit messages or pull request text.
  Every particular is an input with a neutral default.
- **A canary enforces the mechanical half** of that rule on every commit and
  in CI. It reads tracked files, so it cannot catch a commit message: that is
  a review rule, and a message cannot be edited after the push.
- **Hosted runners only.** A public repository's pull requests come from
  forks, and a fork's code must never execute on the estate's own
  infrastructure. The reasoning is not about tokens — a fork's pull request
  gets none — it is that a self-hosted runner's identity and its shared
  build caches outlive the job, and a poisoned cache entry is read by trusted
  jobs afterwards.
- **No secrets in CI.** If work genuinely needs the estate, a private
  repository checks this one out at a pinned version and runs it there.
- **`pull_request_target` never checks out the pull request's head.**

## Conformance

| Rule | Mechanism |
|---|---|
| 2. the gate | CI runs recipes by name; a recipe that needs more is its own job |
| 3. toolchain | the lock file is committed; CI materialises the manifest |
| 4. paths | review |
| 5. required checks | one fan-in job per repository |
| 6. bumps | the bot's configuration, shared across repositories |
| 7. public | the leak canary, and the shared workflows' refusal of a self-hosted runner in a public repository |
