// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

type PlatformConfigMap struct {
	// For a ConfigMap that must be a hook resource: a pre-install job cannot mount one the release has not created yet.
	Annotations *map[string]string `pkl:"annotations"`
}
