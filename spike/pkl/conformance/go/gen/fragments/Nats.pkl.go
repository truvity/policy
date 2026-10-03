// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// A NATS connection, and nothing else. What a service does with the connection — publish to a subject, bind a durable consumer to a stream — differs per component and is described beside it: a publisher with a `consumer` field it never reads is a field somebody will eventually set.
type Nats struct {
	// The server URL, for example nats://nats:4222.
	Url string `pkl:"url"`

	// Path to a file holding the token the client authenticates with, mounted by the platform and re-read on every reconnect so that a rotated token is picked up without a restart. It is the workload's own account token: the broker asks an authorisation service who the bearer is, and that service answers from the account rather than from anything the client claims. Unset with `tls` unset means no authentication, which is a test configuration and not a deployment; unset with `tls` set means the certificate is the credential.
	TokenFile *string `pkl:"tokenFile"`

	// Connect over TLS and authenticate with the service's own workload identity instead of a token. The certificate presented is the one the service's own top-level `tls` block loads (its mode must not be `off`); the broker maps the identity in it to a user with its own permissions, so what the service may do is the broker's decision and nothing in this file. The certificate is re-read on every connect, so the platform's rotation is picked up without a restart and a rotation drops no connection already made. Only a service that declares this key in its own schema acts on it.
	Tls *NatsTls `pkl:"tls"`
}
