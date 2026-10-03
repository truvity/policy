// Code generated from Pkl module `spike.contract.urlshortener.Prober`. DO NOT EDIT.
package prober

// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is a platform decision this component does not make for itself.
type Urls struct {
	Address string `pkl:"address"`
}
