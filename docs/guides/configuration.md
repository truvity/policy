# Configuration

**The rule.** A service reads one file, validates it against a schema it
ships, and decodes it into a type it wrote by hand. Secrets are not in the
file: the file names the *variable* each secret arrives in.
[contracts/config.md](../contracts/config.md) is normative;
[0002](../decisions/0002-config-file-plus-env.md) and
[0003](../decisions/0003-schemas-not-generators.md) argue it.

**Why.** A flat environment cannot express a list, cannot be typed, and
cannot be checked before the process starts, so a mistake in it becomes a
crash loop reported as "unhealthy" rather than "you spelled the key wrong". A
file can be rendered by whatever deploys the service, validated against the
same schema the service uses, reviewed as a diff, and committed as a fixture.
Secrets are the opposite case, because a file gets printed, mounted and
committed.

## Where to look

| Language | Load it | Declare it |
|---|---|---|
| Go | `config.Load(path, schema, &cfg)` in [`config/`](../../config/) | the types in [`examples/url-shortener/internal/config/config.go`](../../examples/url-shortener/internal/config/config.go), one per binary |
| TypeScript | `load(path, schema)` in [`ts/src/config.ts`](../../ts/src/config.ts) | the front end follows |
| Kotlin | follows, with the counter | |
| Python | `load(path, schema)` in [`python/src/truvity_policy/config.py`](../../python/src/truvity_policy/config.py) | a `TypedDict` per binary, with the log component |

The schemas are in [`examples/url-shortener/schemas/`](../../examples/url-shortener/schemas/),
one per binary, each referencing the shared envelope with `allOf`. The shared
fragments are in [`schemas/`](../../schemas/README.md).

## The shape of a schema

A service's schema references the envelope and adds its own properties beside
it:

```json
{
  "allOf": [{ "$ref": ".../schemas/service.json" }],
  "properties": { "database": { "$ref": ".../fragments/postgres.json" } },
  "unevaluatedProperties": false
}
```

`unevaluatedProperties: false`, not `additionalProperties: false`. The second
one does not see through `allOf`: with it, every property the envelope
contributes is "additional" and every valid document is refused.

## Traps

**Validation happens before decoding, and that is the whole point.** A
decoder that silently drops an unknown key turns a typo into a default nobody
chose. Both loaders validate first and name the failing key, not the file.

**A schema that is not strict is decoration.** Every schema needs a negative
fixture whose only defect is an unknown key. If that fixture passes, the
strictness has been lost somewhere in the references and nothing will tell
you.

**The chart and the binary read one schema.** The chart's test pulls each
configuration file out of a render and validates it with the *binary's*
schema — see the chart tests in
[`examples/url-shortener/charts/`](../../examples/url-shortener/charts/).
Without that, the chart keeps setting a key the binary stopped reading and
the service runs on a default, with no signal but behaviour.

**A secret's NAME is configuration; its value never is.** The file says which
variable to read. A rendered file that contains a password is a password in
the release's stored manifest, readable by anyone who can read a release.
