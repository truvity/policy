// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// Where the platform mounts the workload identity. The files the component's own `config.tls` names must be under `mountPath`; the render refuses a file that is not.
type PlatformTls struct {
	// The driver that mounts the identity. The platform's, so there is no default; required once the identity is mounted.
	CsiDriver *string `pkl:"csiDriver"`

	// Defaults to /var/run/identity.
	MountPath *string `pkl:"mountPath"`

	// Defaults to whether `config.tls.mode` is permissive or strict. True mounts the identity into a component that presents none of its own, because the release does; false never mounts it.
	Mount *bool `pkl:"mount"`
}
