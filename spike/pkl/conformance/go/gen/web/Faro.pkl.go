// Code generated from Pkl module `spike.contract.urlshortener.Web`. DO NOT EDIT.
package web

// Browser telemetry (Grafana Faro). Absent, or `enabled: false`, means the page sends nothing and the Faro libraries are never downloaded. Every field here is written into the page for every visitor to read: `apiKey` is a PUBLIC identifier of this app at the collector (one key per app, rotatable, paired with the collector's origin allow-list), not a secret, and no credential may be put in this block.
type Faro struct {
	Enabled bool `pkl:"enabled"`

	// Where the page POSTs telemetry: a path on its own origin (the default, `/faro/collect`, which the gateway routes to the collector, so connect-src 'self' is enough) or an absolute HTTPS URL, whose origin is then added to connect-src.
	CollectorUrl string `pkl:"collectorUrl"`

	// The app's public key at the collector, sent as `x-api-key`. A public identifier, not a secret.
	ApiKey *string `pkl:"apiKey"`

	// The app name the collector sees.
	AppName string `pkl:"appName"`

	// A label for where this install runs, for example `devel`.
	Environment *string `pkl:"environment"`

	// The fraction of browser SESSIONS that report anything, 0 to 1.
	SampleRate float64 `pkl:"sampleRate"`
}
