package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// export runs one span with the given attributes through the allow-list
// filter this package installs, and returns what actually reached the
// exporter. It is the shape every test below asks the same question of:
// what survives, not what went in.
func export(t *testing.T, opts []Option, attrs ...attribute.KeyValue) map[attribute.Key]attribute.Value {
	t.Helper()

	inner := tracetest.NewInMemoryExporter()
	filtered := &filteringExporter{SpanExporter: inner, allowed: newOptions(opts...).allowed}
	provider := tracesdk.NewTracerProvider(tracesdk.WithSyncer(filtered))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	tracer := provider.Tracer("test")
	_, span := tracer.Start(context.Background(), "span")
	span.SetAttributes(attrs...)
	span.End()

	spans := inner.GetSpans()
	require.Len(t, spans, 1)

	got := map[attribute.Key]attribute.Value{}
	for _, kv := range spans[0].Attributes {
		got[kv.Key] = kv.Value
	}

	return got
}

func TestAnUnlistedAttributeIsAbsentFromWhatExports(t *testing.T) {
	got := export(t, nil, attribute.String("url.path", "/users/42"))

	_, ok := got["url.path"]
	require.False(t, ok, "an attribute nobody allow-listed must be ABSENT, not merely unexamined")
}

func TestADefaultAllowedAttributeSurvives(t *testing.T) {
	got := export(t, nil, attribute.String("http.route", "/users/:id"))

	require.Equal(t, "/users/:id", got["http.route"].AsString())
}

func TestACallerExtensionSurvives(t *testing.T) {
	got := export(t, []Option{WithAllowedAttributes("archive.key")},
		attribute.String("archive.key", "2026/09/26/00001.ndjson"),
		attribute.String("url.path", "/should/not/survive"),
	)

	require.Equal(t, "2026/09/26/00001.ndjson", got["archive.key"].AsString())
	_, ok := got["url.path"]
	require.False(t, ok)
}

func TestExtendingTheListNeverRemovesADefault(t *testing.T) {
	got := export(t, []Option{WithAllowedAttributes("archive.key")},
		attribute.String("http.route", "/users/:id"),
		attribute.String("archive.key", "k"),
	)

	require.Equal(t, "/users/:id", got["http.route"].AsString())
	require.Equal(t, "k", got["archive.key"].AsString())
}
