// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// How the component is probed, on the listener its own `config.probes` binds. Liveness is nothing but the process: a probe that checks a dependency restarts a healthy process and makes an outage worse.
type PlatformProbes struct {
	Liveness *Probe `pkl:"liveness"`

	Readiness *Probe `pkl:"readiness"`

	// Absent means no startup probe. Present, it needs at least one field.
	Startup *Probe `pkl:"startup"`
}
