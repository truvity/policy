// Code generated from Pkl module `spike.contract.urlshortener.Web`. DO NOT EDIT.
package web

// The service that owns the URL tables. An address and nothing else: which protocol the caller speaks is in its code, and whether the connection is authenticated is the tls block's business.
type Urls struct {
	Address string `pkl:"address"`
}
