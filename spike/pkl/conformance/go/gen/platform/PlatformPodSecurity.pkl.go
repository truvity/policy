// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// Who the process is. Every field defaults to 65532, the unprivileged user the runtime images run as; `fsGroup` is the one that matters, because a CSI driver writes what it mounts owned by root.
type PlatformPodSecurity struct {
	RunAsUser *int `pkl:"runAsUser"`

	RunAsGroup *int `pkl:"runAsGroup"`

	FsGroup *int `pkl:"fsGroup"`
}
