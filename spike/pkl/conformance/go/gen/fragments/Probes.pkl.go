// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// The health listener. Separate from the service's own traffic, so that readiness is answerable when the service's listener is saturated, and so that a probe is not reachable from outside.
type Probes struct {
	// host:port for /health/live and /health/ready.
	Address string `pkl:"address"`
}
