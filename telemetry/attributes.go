package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

// DefaultAllowedAttributes is the set of span attribute keys exported by
// default: OpenTelemetry semantic-convention keys that describe a call's
// shape rather than its content. An attribute nobody thought about is
// ABSENT, not exported by accident, so this list is short on purpose and
// never grows to accommodate one caller — a caller with more to say uses
// [WithAllowedAttributes].
//
// Deliberately NOT here: url.path, url.query, url.full (the request line
// itself, which carries ids, search terms and tokens), any header, any
// database or messaging PAYLOAD, and any peer address. Those are exactly
// the attributes an instrumentation library adds on its own, which is why
// this list is an ALLOW list rather than a set of things to strip.
var DefaultAllowedAttributes = []attribute.Key{
	"http.request.method",
	"http.route",
	"http.response.status_code",
	"rpc.system",
	"rpc.service",
	"rpc.method",
	"rpc.grpc.status_code",
	"rpc.connect_rpc.error_code",
	"db.system",
	"db.system.name",
	"db.operation",
	"db.operation.name",
	"db.query.text",
	"db.response.returned_rows",
	"messaging.system",
	"messaging.destination.name",
	"messaging.operation",
	"messaging.operation.type",
	"server.port",
	"error.type",
	"otel.status_code",
	"otel.status_description",
}

// options collects what Start needs beyond the environment.
type options struct {
	allowed map[attribute.Key]struct{}
}

// Option configures Start. The only kind today is [WithAllowedAttributes].
type Option func(*options)

// WithAllowedAttributes extends the default allow-list with keys this
// service's own code adds. It is additive: it never removes a default key,
// and it is the ONLY way to add one — there is no configuration key and no
// environment variable for it (decision 0006 is about the SDK's own
// environment, not this list).
func WithAllowedAttributes(keys ...attribute.Key) Option {
	return func(o *options) {
		for _, key := range keys {
			o.allowed[key] = struct{}{}
		}
	}
}

func newOptions(opts ...Option) *options {
	o := &options{allowed: make(map[attribute.Key]struct{}, len(DefaultAllowedAttributes))}
	for _, key := range DefaultAllowedAttributes {
		o.allowed[key] = struct{}{}
	}
	for _, apply := range opts {
		apply(o)
	}

	return o
}

// filteringExporter drops every span attribute not on the allow-list before
// handing spans to the real exporter.
//
// The Go SDK's span processor sees a span at OnEnd as a [tracesdk.ReadOnlySpan]
// that already carries whatever instrumentation added — there is no hook
// that lets a processor edit it. The exporter is the one place downstream of
// that where the attributes are still visible AND the shape is still ours to
// change before it leaves the process, so the filter lives here, wrapping
// whatever exporter autoexport built from the environment.
type filteringExporter struct {
	tracesdk.SpanExporter

	allowed map[attribute.Key]struct{}
}

func (f *filteringExporter) ExportSpans(ctx context.Context, spans []tracesdk.ReadOnlySpan) error {
	filtered := make([]tracesdk.ReadOnlySpan, len(spans))
	for i, span := range spans {
		filtered[i] = filteredSpan{ReadOnlySpan: span, attributes: f.keep(span.Attributes())}
	}

	return f.SpanExporter.ExportSpans(ctx, filtered)
}

func (f *filteringExporter) keep(attrs []attribute.KeyValue) []attribute.KeyValue {
	kept := make([]attribute.KeyValue, 0, len(attrs))
	for _, kv := range attrs {
		if _, ok := f.allowed[kv.Key]; ok {
			kept = append(kept, kv)
		}
	}

	return kept
}

// filteredSpan is a [tracesdk.ReadOnlySpan] with its attributes replaced.
// Embedding the interface means every other method — name, kind, status,
// links, resource, and so on, which are out of scope for this filter — is
// unchanged.
type filteredSpan struct {
	tracesdk.ReadOnlySpan

	attributes []attribute.KeyValue
}

func (f filteredSpan) Attributes() []attribute.KeyValue { return f.attributes }
