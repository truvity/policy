// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

import "spike.invalid/pklconf/gen/vocab/otelprotocol"

// OpenTelemetry's own environment variables (decision 0006): they leave the chart as variables, never as configuration keys. No endpoint means do not export.
type PlatformTelemetry struct {
	// Defaults to `<release>-<component>`.
	ServiceName *string `pkl:"serviceName"`

	Endpoint *string `pkl:"endpoint"`

	Protocol *otelprotocol.OtelProtocol `pkl:"protocol"`

	TracesSampler *string `pkl:"tracesSampler"`

	SampleRatio *any `pkl:"sampleRatio"`

	ResourceAttributes *map[string]string `pkl:"resourceAttributes"`
}
