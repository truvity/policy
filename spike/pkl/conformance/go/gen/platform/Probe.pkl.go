// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

type Probe struct {
	Path *string `pkl:"path"`

	PeriodSeconds *int `pkl:"periodSeconds"`

	InitialDelaySeconds *int `pkl:"initialDelaySeconds"`

	TimeoutSeconds *int `pkl:"timeoutSeconds"`

	SuccessThreshold *int `pkl:"successThreshold"`

	FailureThreshold *int `pkl:"failureThreshold"`
}
