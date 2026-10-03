// Code generated from Pkl module `spike.vocab.Vocab`. DO NOT EDIT.
package pullpolicy

import (
	"encoding"
	"fmt"
)

type PullPolicy string

const (
	Always       PullPolicy = "Always"
	IfNotPresent PullPolicy = "IfNotPresent"
	Never        PullPolicy = "Never"
)

// String returns the string representation of PullPolicy
func (rcv PullPolicy) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(PullPolicy)

// UnmarshalBinary implements encoding.BinaryUnmarshaler for PullPolicy.
func (rcv *PullPolicy) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "Always":
		*rcv = Always
	case "IfNotPresent":
		*rcv = IfNotPresent
	case "Never":
		*rcv = Never
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid PullPolicy`, str)
	}
	return nil
}
