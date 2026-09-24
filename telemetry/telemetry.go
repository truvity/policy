// Package telemetry starts the OpenTelemetry SDK from OpenTelemetry's own
// environment, and stops it cleanly.
//
// There is deliberately nothing to configure here. Decision 0006: the
// specification defines the variables, every language's SDK reads them
// without being asked, and every document about OpenTelemetry is written
// in terms of them. A second vocabulary in a configuration file would be
// a ceiling on what an operator can set, and a precedence question at
// three in the morning.
//
// NO ENDPOINT MEANS EXPORT NOTHING. That is not a special case written
// here: `OTEL_TRACES_EXPORTER=none` is a value the specification defines
// and autoexport honours, and a deployment that configures no endpoint
// sets it. So a laptop, a test and a cluster with no collector all do the
// same thing — nothing — without this package or its callers carrying an
// enable flag.
//
// That flag is the failure decision 0006 records. A service exported only
// when an environment name matched "production"; nothing set that
// variable in any deployment, so it ran with a console exporter, writing
// spans onto the same stream as its logs, in every environment including
// the one it was written for.
package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

// Shutdown flushes what is buffered and releases the exporters. A caller
// that skips it loses whatever had not been sent, which on a short-lived
// process is usually everything.
type Shutdown func(context.Context) error

// Start installs the global tracer and meter providers.
//
// The resource, the sampler, the endpoint and the protocol all come from
// the environment. Service name included: it is a log STREAM field where
// these signals land, so it has to be stable for the life of the process
// and carry no request, tenant or version — which is a deployment's
// decision to make, not a program's.
func Start(ctx context.Context) (Shutdown, error) {
	spans, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}

	metrics, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("metric reader: %w", err)
	}

	tracer := tracesdk.NewTracerProvider(tracesdk.WithBatcher(spans))
	meter := metricsdk.NewMeterProvider(metricsdk.WithReader(metrics))

	otel.SetTracerProvider(tracer)
	otel.SetMeterProvider(meter)

	// The Go runtime's own metrics: heap, goroutines, GC pauses, scheduler
	// latency. They cost nothing to collect and they are the ones somebody
	// actually wants at two in the morning, so a service that exports no
	// metrics of its own still exports these.
	//
	// It also means an empty metric store is unambiguous. Without a series
	// that is always present, "nothing is arriving" and "this service is
	// quiet" look identical, and the store cannot tell you which.
	if err := runtimemetrics.Start(runtimemetrics.WithMeterProvider(meter)); err != nil {
		return nil, fmt.Errorf("runtime metrics: %w", err)
	}

	// W3C, so a trace survives a hop between two services that agree on
	// nothing else. Without this every service starts its own trace and
	// the store fills with one-span traces that look like success.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		return errors.Join(tracer.Shutdown(ctx), meter.Shutdown(ctx))
	}, nil
}
