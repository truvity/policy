// Code generated from Pkl module `spike.vocab.Vocab`. DO NOT EDIT.
package tlsmode

import (
	"encoding"
	"fmt"
)

type TlsMode string

const (
	Off        TlsMode = "off"
	Permissive TlsMode = "permissive"
	Strict     TlsMode = "strict"
)

// String returns the string representation of TlsMode
func (rcv TlsMode) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(TlsMode)

// UnmarshalBinary implements encoding.BinaryUnmarshaler for TlsMode.
func (rcv *TlsMode) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "off":
		*rcv = Off
	case "permissive":
		*rcv = Permissive
	case "strict":
		*rcv = Strict
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid TlsMode`, str)
	}
	return nil
}
