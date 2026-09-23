// Package policy publishes the configuration shapes these contracts define,
// and nothing else. The loaders are in the packages beside it.
package policy

import "embed"

// Schemas holds the shared configuration schemas, rooted at `schemas/`.
//
// They are embedded rather than fetched: a `$id` in this repository is an
// identifier, not an address. Validation therefore needs no network, works
// in a test with no egress, and pins the shapes to the version of this
// module a service depends on.
//
//go:embed schemas
var Schemas embed.FS

// SchemaBase is the prefix every `$id` in [Schemas] carries. A document at
// `schemas/fragments/log.json` is identified by SchemaBase +
// "fragments/log.json".
const SchemaBase = "https://github.com/truvity/policy/schemas/"
