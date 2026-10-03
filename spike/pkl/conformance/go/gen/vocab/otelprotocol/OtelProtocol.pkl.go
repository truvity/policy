// Code generated from Pkl module `spike.vocab.Vocab`. DO NOT EDIT.
package otelprotocol

import (
	"encoding"
	"fmt"
)

type OtelProtocol string

const (
	Grpc         OtelProtocol = "grpc"
	HttpProtobuf OtelProtocol = "http/protobuf"
	HttpJson     OtelProtocol = "http/json"
)

// String returns the string representation of OtelProtocol
func (rcv OtelProtocol) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(OtelProtocol)

// UnmarshalBinary implements encoding.BinaryUnmarshaler for OtelProtocol.
func (rcv *OtelProtocol) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "grpc":
		*rcv = Grpc
	case "http/protobuf":
		*rcv = HttpProtobuf
	case "http/json":
		*rcv = HttpJson
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid OtelProtocol`, str)
	}
	return nil
}
