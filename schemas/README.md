# Schemas

The shapes that mean the same thing in more than one service, so that every
tool which reads a configuration is not a special case.

| File | Shape |
|---|---|
| [service.json](service.json) | the envelope: listener, probes, logging, telemetry |
| [fragments/listen.json](fragments/listen.json) | a TCP listener |
| [fragments/probes.json](fragments/probes.json) | the health listener |
| [fragments/log.json](fragments/log.json) | one level for the whole service |
| [fragments/otel.json](fragments/otel.json) | OpenTelemetry export |
| [fragments/bucket.json](fragments/bucket.json) | an object store addressed by the S3 API |
| [fragments/postgres.json](fragments/postgres.json) | a PostgreSQL connection |
| [fragments/nats.json](fragments/nats.json) | a NATS connection and its stream |

## How a service uses them

A service ships its own schema, which references the envelope and adds its
own properties:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.com/schemas/shortener.json",
  "allOf": [{ "$ref": "https://github.com/truvity/policy/schemas/service.json" }],
  "type": "object",
  "unevaluatedProperties": false,
  "required": ["database"],
  "properties": {
    "database": { "$ref": "https://github.com/truvity/policy/schemas/fragments/postgres.json" }
  }
}
```

`unevaluatedProperties: false` rather than `additionalProperties: false`: the
latter does not see through `allOf`, so it would reject every property the
envelope contributes. This is the single most common way a strict schema with
a `$ref` in it goes wrong.

## The `$id`s are identifiers, not addresses

Nothing fetches them. The loader resolves every `$id` above from schemas
compiled into it, so validation needs no network, and a service pins the
shapes by pinning this repository's version.

## Rules

- Everything defined here is strict. A typo must fail.
- A fragment describes a *shape*, never a particular: a bucket's name is a
  value, its vendor is not a field.
- Secrets are named, never carried: a fragment holds the NAME of an
  environment variable, and the value never appears in a configuration file.
