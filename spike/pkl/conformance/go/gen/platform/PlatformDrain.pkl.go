// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// The service's own shutdown budget is `config.drain.seconds`; the library derives the grace period from it and this delay, so the three numbers cannot disagree.
type PlatformDrain struct {
	// Fail readiness, then wait this long before the drain starts, so that whatever routes traffic has removed this endpoint first. Defaults to 5.
	PreStopSeconds *int `pkl:"preStopSeconds"`
}
