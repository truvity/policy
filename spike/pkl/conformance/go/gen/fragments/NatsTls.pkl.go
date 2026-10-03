// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// Connect over TLS and authenticate with the service's own workload identity instead of a token. The certificate presented is the one the service's own top-level `tls` block loads (its mode must not be `off`); the broker maps the identity in it to a user with its own permissions, so what the service may do is the broker's decision and nothing in this file. The certificate is re-read on every connect, so the platform's rotation is picked up without a restart and a rotation drops no connection already made. Only a service that declares this key in its own schema acts on it.
type NatsTls struct {
	// Path to the trust bundle the BROKER's certificate is verified against. The broker's, not the workload identity's: a broker has a name and no workload identity, and usually a different chain.
	CaFile string `pkl:"caFile"`

	// The name the broker's certificate was issued for, when that is not the host in `url`. Empty verifies the host dialled.
	ServerName *string `pkl:"serverName"`
}
