// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

import "spike.invalid/pklconf/gen/vocab/tlsmode"

// Mutually authenticated transport, where the platform provides the identity. A workload presents a certificate it did not mint, reloads it without restarting, and admits peers by the account they run as rather than by the address they call from. Absent, or mode 'off', means cleartext: a service must be installable on a platform that provides none of this.
type TlsFields struct {
	// 'off' serves cleartext only. 'permissive' serves BOTH, on two ports, so that an edge can migrate one side at a time without a coordinated window. 'strict' serves only the authenticated port. One listener cannot be both in every runtime, which is why permissive is two ports rather than one that sniffs.
	Mode tlsmode.TlsMode `pkl:"mode"`

	// Where the authenticated listener binds under 'permissive', beside the cleartext one. Under 'strict' there is one listener and it is the service's own, so this is unused: the protocol changes, the address does not, and nothing downstream has to be told.
	Address *string `pkl:"address"`

	// The certificate this workload presents, mounted and rotated by the platform. Re-read when it changes, never cached for the process's lifetime: a one-hour certificate outlives no deployment.
	CertFile *string `pkl:"certFile"`

	// Its private key. It lives in the pod and never in a secret, so a workload that can read secrets in its namespace still cannot read a neighbour's key.
	KeyFile *string `pkl:"keyFile"`

	// The authority peers are verified against, distributed by the platform as a trust bundle.
	CaFile *string `pkl:"caFile"`

	// The root of every identity this service will admit, for example 'example.internal'. A peer whose identity belongs to another trust domain is refused before its account is even considered.
	TrustDomain *string `pkl:"trustDomain"`

	// Who may call. Each entry is an ACCOUNT, not an address: an address resolves to whoever holds it today. An empty list admits no one, which is the correct default for a service nobody has been granted.
	Peers *[]TlsPeer `pkl:"peers"`
}
