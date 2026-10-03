// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// The account this component runs as. EVERY component has its own, always; `default` is refused. The library grants nothing: annotations are where a platform binds the account to rights outside the cluster, and which mechanism does that is the platform's.
type PlatformServiceAccount struct {
	// False where the platform creates the accounts; they must then exist. Defaults to true.
	Create *bool `pkl:"create"`

	// Defaults to `<release>-<component>`.
	Name *string `pkl:"name"`

	Annotations *map[string]string `pkl:"annotations"`
}
