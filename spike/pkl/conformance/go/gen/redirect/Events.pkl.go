// Code generated from Pkl module `spike.contract.urlshortener.Redirect`. DO NOT EDIT.
package redirect

import "spike.invalid/pklconf/gen/fragments"

type Events struct {
	Nats fragments.Nats `pkl:"nats"`

	// Where a resolved redirect is published. One event kind per subject, so a consumer never has to guess what it decoded.
	RedirectSubject string `pkl:"redirectSubject"`

	// Where the request log is published. A separate subject from the redirect, for the same reason.
	RequestSubject string `pkl:"requestSubject"`
}
