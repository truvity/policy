// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// The rolling update. The defaults make a rollout gapless: the replacement is READY before the incumbent is touched.
type PlatformStrategy struct {
	// Defaults to 0.
	MaxUnavailable *any `pkl:"maxUnavailable"`

	// Defaults to 1.
	MaxSurge *any `pkl:"maxSurge"`
}
