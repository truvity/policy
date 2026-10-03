// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// How long the service may take to finish in-flight work after SIGTERM. It is one number shared with whatever deploys the service: the grace period granted to the process and the pre-stop delay before it are derived from this, so that a draining process is never killed at the moment it would have finished.
type Drain struct {
	// Seconds to finish in-flight work. Unset means the service's own default, which is only correct if nothing external is counting.
	Seconds int `pkl:"seconds"`
}
