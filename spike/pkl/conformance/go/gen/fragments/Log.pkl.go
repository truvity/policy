// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

import "spike.invalid/pklconf/gen/vocab/loglevel"

// Structured logging. One level for the whole service: per-package levels are deliberately not part of the contract.
type Log struct {
	// The lowest level that is written.
	Level loglevel.LogLevel `pkl:"level"`
}
