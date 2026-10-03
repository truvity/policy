// Code generated from Pkl module `spike.contract.urlshortener.Web`. DO NOT EDIT.
package web

import "spike.invalid/pklconf/gen/vocab/cspmode"

// The Content-Security-Policy this server sends with every response. Absent means report-only with no extra origins: the header is sent, nothing is blocked, so turning it on cannot break a page.
type Csp struct {
	// `report-only` sends Content-Security-Policy-Report-Only: a browser reports a violation to its console (and to reportUri) and blocks nothing. `enforce` sends Content-Security-Policy. `off` sends neither.
	Mode cspmode.CspMode `pkl:"mode"`

	// Origins the page may connect to besides its own, added to `connect-src 'self'`. Each is a bare origin, scheme and host and optional port, never a path and never a keyword.
	ConnectSrc *[]string `pkl:"connectSrc"`

	// Where a browser POSTs violation reports. A path on this origin or an absolute URL; empty or absent adds no report-uri directive.
	ReportUri *string `pkl:"reportUri"`
}
