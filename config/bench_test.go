package config_test

import (
	"testing"

	"github.com/truvity/policy/config"
)

// BenchmarkLoad is the cost of one Load of a valid file against a schema that
// references the published shapes: the read, the validation and the decode.
// Compiling the schemas once per process is what keeps it flat.
func BenchmarkLoad(b *testing.B) {
	schema := schema(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg shortener
		if err := config.Load("testdata/valid.yaml", schema, &cfg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidate is the validation alone, of an already-decoded document.
func BenchmarkValidate(b *testing.B) {
	schema := schema(b)
	doc := map[string]any{
		"listen":   map[string]any{"address": ":8080"},
		"probes":   map[string]any{"address": ":7070"},
		"log":      map[string]any{"level": "info"},
		"baseURL":  "https://example.com",
		"database": map[string]any{"url": "postgres://shortener@db:5432/shortener", "passwordSecret": "db/password", "maxConnections": float64(20)},
		"secrets":  map[string]any{"source": "file", "root": "/var/run/secrets/shortener"},
	}
	if err := config.Validate(doc, schema); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate(doc, schema)
	}
}
