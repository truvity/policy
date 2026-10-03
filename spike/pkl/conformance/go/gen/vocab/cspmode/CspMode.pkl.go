// Code generated from Pkl module `spike.vocab.Vocab`. DO NOT EDIT.
package cspmode

import (
	"encoding"
	"fmt"
)

type CspMode string

const (
	Off        CspMode = "off"
	ReportOnly CspMode = "report-only"
	Enforce    CspMode = "enforce"
)

// String returns the string representation of CspMode
func (rcv CspMode) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(CspMode)

// UnmarshalBinary implements encoding.BinaryUnmarshaler for CspMode.
func (rcv *CspMode) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "off":
		*rcv = Off
	case "report-only":
		*rcv = ReportOnly
	case "enforce":
		*rcv = Enforce
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid CspMode`, str)
	}
	return nil
}
