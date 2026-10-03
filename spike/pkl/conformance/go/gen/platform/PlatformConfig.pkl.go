// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// How the file reaches the process. The path is ONE argument or ONE environment variable, never both (decision 0002).
type PlatformConfig struct {
	// Defaults to `<component>.yaml`.
	FileName *string `pkl:"fileName"`

	// The directory the ConfigMap is mounted at. Defaults to `/etc/<chart name>`.
	MountPath *string `pkl:"mountPath"`

	// The argument that carries the path. Defaults to `-config`.
	PathFlag *string `pkl:"pathFlag"`

	// When set, the path is passed in this environment variable instead of an argument.
	PathEnv *string `pkl:"pathEnv"`
}
