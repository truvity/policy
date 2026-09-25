package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// fieldTraceID and fieldSpanID are the names the OpenTelemetry specification
// recommends for trace context in a log format that is not OTLP: lower-case
// hex, the W3C forms. Every language in this repository uses exactly these
// two names, so a line can be grepped for either without knowing which
// service wrote it.
const (
	fieldTraceID = "trace_id"
	fieldSpanID  = "span_id"
)

// NewLogHandler wraps a slog.Handler so that a record written while a span
// is current carries that span's trace_id and span_id.
//
// It reads the span from the ctx the Handle call itself receives, not from
// a package-level variable — which is what makes the two halves of the
// contract hold at once: log.InfoContext(ctx, ...) carries the fields
// because that ctx is the one a boundary (an RPC, a queue consumer) derived
// from an active span, and a call with no context, or a background one,
// carries nothing, even while a span is current somewhere else in the
// process. slog itself guarantees ctx is never nil here: the context-less
// logging methods hand Handle a context.Background().
//
// When no span is current the fields are left off entirely — not empty
// strings, not zeroes — so a line without them reads as "no trace", not as
// "trace zero".
type logHandler struct {
	next slog.Handler
}

// NewLogHandler returns h, or a handler equivalent to it, with trace
// correlation added.
func NewLogHandler(h slog.Handler) slog.Handler {
	return &logHandler{next: h}
}

func (h *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *logHandler) Handle(ctx context.Context, record slog.Record) error {
	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		record.AddAttrs(
			slog.String(fieldTraceID, span.TraceID().String()),
			slog.String(fieldSpanID, span.SpanID().String()),
		)
	}
	return h.next.Handle(ctx, record)
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &logHandler{next: h.next.WithAttrs(attrs)}
}

func (h *logHandler) WithGroup(name string) slog.Handler {
	return &logHandler{next: h.next.WithGroup(name)}
}
