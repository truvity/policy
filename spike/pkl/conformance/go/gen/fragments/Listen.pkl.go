// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// A TCP listener. `address` is a host:port the service binds; an empty host binds every interface.
type Listen struct {
	// host:port, for example ":8080" or "127.0.0.1:8080".
	Address string `pkl:"address"`
}
