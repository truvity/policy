// Code generated from Pkl module `spike.vocab.Vocab`. DO NOT EDIT.
package loglevel

import (
	"encoding"
	"fmt"
)

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

// String returns the string representation of LogLevel
func (rcv LogLevel) String() string {
	return string(rcv)
}

var _ encoding.BinaryUnmarshaler = new(LogLevel)

// UnmarshalBinary implements encoding.BinaryUnmarshaler for LogLevel.
func (rcv *LogLevel) UnmarshalBinary(data []byte) error {
	switch str := string(data); str {
	case "debug":
		*rcv = Debug
	case "info":
		*rcv = Info
	case "warn":
		*rcv = Warn
	case "error":
		*rcv = Error
	default:
		return fmt.Errorf(`illegal: "%s" is not a valid LogLevel`, str)
	}
	return nil
}
