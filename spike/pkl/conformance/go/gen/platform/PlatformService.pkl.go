// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// A component that listens has a Service on the ports of its own `config.listen`; one that does not has none.
type PlatformService struct {
	// Defaults to whether `config.listen` exists. Enabling one for a component that listens on nothing is refused.
	Enabled *bool `pkl:"enabled"`
}
