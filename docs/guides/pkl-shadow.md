# The Pkl shadow

**The rule.** The url-shortener's data contracts are also written in Pkl, and
everything restating them is generated from that one source into
[`examples/url-shortener/contract/generated/`](../../examples/url-shortener/contract/generated/).
CI compares the generated schemas with the hand-written ones and reports every
difference. **The hand-written schemas stay authoritative:** nothing reads the
generated files, and no difference fails a build.
[0010](../decisions/0010-data-contracts-are-written-in-pkl.md) is the decision
and argues it; this is its first rollout phase, "shadow first".

**Why.** The spike behind 0010 bootstrapped its Pkl from the JSON it was
compared with, so "identical" proved the generator loses nothing and not that a
person writing Pkl first lands on the same schemas. The shadow phase is that
second measurement, on the real contracts and over a real stretch of changes,
and it measures the two costs the spike did not: what generation adds to a
build, and what authoring in Pkl costs the person making a change.

## What is in it

| | Where |
|---|---|
| The contract | [`examples/url-shortener/contract/`](../../examples/url-shortener/contract/): a Pkl project, beside the schemas it shadows |
| Its pins | `PklProject` names the `truvity/pkl-contracts` packages at one exact version; `PklProject.deps.json` records their checksums and is committed |
| Each component's config | `config/*.pkl`: the six binaries' configuration, the archiver's, and the test chart's `echo` |
| Each chart's values | `charts/*.pkl`: the product chart, its infra chart, its e2e chart, and the smallest service chart. The product chart is the one [0009](../decisions/0009-charts-pass-config-through-and-share-a-library.md) calls the exception; 0010 ends the exception, so it is modelled here like the others, and `charts/ServiceExampleValues.pkl` carries one chart's defaults |
| What it generates | `generated/schemas/` (JSON Schema), `generated/charts/<chart>/` (values schema, values table, defaults where modelled), `generated/ts/` (zod), `generated/py/` (pydantic), `generated/docs/` (reference) |
| Pkl itself | [`bin/pkl`](../../bin/pkl): Pkl 0.32.1, downloaded once and checked against a pinned sha256. **Temporary**: nixpkgs ships an older Pkl, and the wrapper goes when it ships 0.32 |

Go and Kotlin types are not generated yet. They come from Pkl's official
generators, each needing a further download, and carry none of the constraints,
so they would add nothing to a comparison of schemas. They are the first thing
the switch would add.

The contract uses **only the vocabulary** (0010: no ad-hoc constraints). Where
the vocabulary has no type for a rule the hand-written schema states, the field
uses the nearest type, or a plain one, and the difference shows in the report
as a gap. That is the point: a gap is a finding for the contracts repository,
not something to hide with a constraint written in place.

## Commands

| | |
|---|---|
| `just shadow-generate` | regenerate `generated/` |
| `just generate` | the same, under the name the contracts pin-bump job runs |
| `just shadow-generated` | **strict**: the Pkl is formatted, the pins are resolved, and `generated/` is exactly what the contract generates. CI runs it |
| `just shadow-diff` | **report only**: regenerate, compare, print the report. In CI it also goes to the job summary |

They need the network (the packages are fetched from their GitHub release the
first time) and are not part of `check`. After a change to a contract, run
`just shadow-generate` and commit what it wrote, as for any generated file.

## Reading the report

`just shadow-diff` prints, per document and per chart, every difference with a
**category** (what differs), a **class** and a **cause**.

| Class | Meaning |
|---|---|
| expected | a semantic rule 0010 states: a field with a default is optional (`required-to-default`), `null` is not a value (`null-refused`), a pattern refuses every line break (`newline-guard`) and spells white space and `.` as explicit classes (`pattern-spelling`), a value only an install can give is required and left out of the defaults (`set-at-install`) |
| structural | a by-design consequence of generating from one source, gone at the switch, so not a finding: the generated schemas state the defaults (`default-added`) and the hand-written ones keep them in `values.yaml`. Listed in a collapsed section of the report |
| gap | a real difference: the contract does not say what the hand-written schema says. Fix it in the contract or the vocabulary, or accept it into the hand-written schema at the switch |
| doc | the description text only |

Schemas are compared **semantically**: every `$ref` is followed (a shared
sub-schema referenced on one side and inlined on the other is not a
difference), an `allOf` of objects is merged, ordering is ignored, and a
pattern's companion `not: {pattern: "\n"}` is a flag, not a second pattern. A
reference to the same document on both sides is not followed: that document is
compared on its own, so a difference inside a fragment is one line, not one
line per schema that uses it.

Three more sections follow the differences:

- **Defaults.** A hand-written chart keeps its defaults in `values.yaml`, not in
  its schema, so `values.yaml` is compared with the defaults the contract
  states. Where the defaults are modelled (`ServiceExampleValues.pkl`), the
  generated `values.yaml` is compared with the hand-written file.
- **Verdicts.** The existing fixtures (the example configurations, every
  chart's test values, the invalid ones) go through both schemas of each pair.
  A disagreement is the strongest signal in the report.
- **The semantic rules, probed.** One instance per rule of 0010, through both
  schemas, so that "expected" is a measurement and not a label.

The last lines are a **machine-readable summary**: counts by class, category
and document, in JSON, and a single `SHADOW_SUMMARY {...}` line on stderr. The
numbers to watch over the phase are the `gap` count (`structural` is not a finding and is left out of the trend) (it should fall) and the
verdict disagreements (it should be none that are not an expected rule).

## Writing the contract

What the contracts packages offer, and where the contract uses it (the rules
are in the packages' `docs/authoring.md`):

- **A value only an install can give** (a host, a bucket, the Secret that holds a
  password) is `@A.SetAtInstall` on a `V.NonEmptyString?`, with no default. The
  schema requires it; the generated `values.yaml` leaves it out. The
  hand-written `values.yaml` ships `""` for it, which the report counts as
  `set-at-install`.
- **A bound that only one field has** is `@A.Range` / `@A.Length` on the
  property, over the vocabulary type. `Check.checked(module)` is Pkl's own
  enforcement of it; a chart's module that extends neither template calls it in
  its `output`.
- **A default may be an object** (the `resources` blocks).
- **A literal union in two modules** is refused by the generator: it belongs in
  the vocabulary as an enum. Where there is none (`off` | `permissive`), the
  contract uses the nearest enum and the report shows the wider set as an `enum`
  gap.

## The time box, and what decides

The phase is time-boxed to **four to six weeks** from the release that carries
it. Nothing a consumer depends on changes in it.

It ends in one decision, **switch or stop**, taken on three things:

1. **Do the generated artifacts agree?** Every `gap` is either closed (the
   contract or the vocabulary was fixed), or named as a deliberate change to
   the hand-written schema, to be made at the switch. No verdict disagreement
   remains that is not one of 0010's rules.
2. **What does generation add to a build?** Measured: the two recipes' wall
   time in CI, and what a contributor pays locally.
3. **What does authoring cost?** Measured on the changes made during the phase:
   how many contract changes there were, and how long each took in Pkl against
   the same change by hand.

If they hold, the generated files replace the hand-written ones and the
contracts that describe the hand-written rule are amended. If not, 0010 is
itself superseded and [0003](../decisions/0003-schemas-not-generators.md)
stands again for data contracts.

## Traps

- **Do not edit `generated/`.** `just shadow-generated` fails on it. Change the
  Pkl and regenerate.
- **A change to a hand-written schema now has a second place to look.** During
  the phase the hand-written schema is the source of truth, so a change goes to
  it first and the Pkl follows; `just shadow-diff` shows whether it did.
- **Run Pkl as `bin/pkl`.** A different Pkl is a different language: it is
  pre-1.0 and breaks between minors.
- **`pkl` needs `--project-dir`.** Pkl does not find a `PklProject` by itself;
  the recipes do it.
- **A field with a default is optional in the generated schema**, where the
  hand-written one lists it under `required`. That is 0010's rule, and it is
  the largest single category in the report.
- **The generated JSON Schemas carry no `default` yet.** The generators of the
  pinned release write the key where Pkl reads it as something else, so the
  zod and pydantic output carry every default and the schemas none. The report
  names it on each default; it is a finding for the contracts repository.
