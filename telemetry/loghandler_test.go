package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/truvity/policy/telemetry"
)

func newLogger(buf *bytes.Buffer) *slog.Logger {
	handler := telemetry.NewLogHandler(slog.NewJSONHandler(buf, nil))
	return slog.New(handler)
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var record map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	return record
}

func TestARecordWrittenInsideASpanCarriesItsIDs(t *testing.T) {
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(tracetest.NewSpanRecorder()))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	ctx, span := provider.Tracer("test").Start(context.Background(), "request")
	defer span.End()

	var buf bytes.Buffer
	newLogger(&buf).InfoContext(ctx, "handled")

	record := decode(t, &buf)
	require.Equal(t, span.SpanContext().TraceID().String(), record["trace_id"],
		"the record's trace_id does not match the current span")
	require.Equal(t, span.SpanContext().SpanID().String(), record["span_id"],
		"the record's span_id does not match the current span")
}

func TestARecordWrittenOutsideAnySpanCarriesNeitherField(t *testing.T) {
	var buf bytes.Buffer
	newLogger(&buf).InfoContext(context.Background(), "handled")

	record := decode(t, &buf)
	require.NotContains(t, record, "trace_id", "no span was current: the field must be absent, not empty")
	require.NotContains(t, record, "span_id", "no span was current: the field must be absent, not empty")
}

// TestALogCallWithNoContextCarriesNothingEvenWithASpanElsewhere is the Go-
// specific half of the contract: a span being current somewhere in the
// process must not leak into a log call that was not handed that span's
// context. Without this a package that still calls Logger.Info instead of
// InfoContext would get fields anyway, by accident of whichever request
// happened to be in flight, and the two would look interchangeable until a
// log line pointed at the wrong trace.
func TestALogCallWithNoContextCarriesNothingEvenWithASpanElsewhere(t *testing.T) {
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(tracetest.NewSpanRecorder()))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	_, span := provider.Tracer("test").Start(context.Background(), "elsewhere")
	defer span.End()

	var buf bytes.Buffer
	// Info, not InfoContext: that is the point of the test. slog hands the
	// handler context.Background(), which carries no span regardless of
	// what is current in this goroutine -- there is no ambient/thread-local
	// span in Go to leak.
	newLogger(&buf).Info("handled") //nolint:sloglint // deliberately: proves a context-less call carries nothing

	record := decode(t, &buf)
	require.NotContains(t, record, "trace_id", "a call with no context must not pick up a span from elsewhere")
	require.NotContains(t, record, "span_id", "a call with no context must not pick up a span from elsewhere")
}
