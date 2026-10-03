// Code generated from Pkl module `spike.contract.urlshortener.LogArchiver`. DO NOT EDIT.
package logarchiver

// When to close an object and write it. Both limits apply: whichever is reached first. A size-only rule means a quiet hour is never archived; a time-only rule means a busy one is archived in objects too small to be worth reading.
type Batch struct {
	// Write once this many records are held.
	MaxRecords int `pkl:"maxRecords"`

	// Write this long after the first record of a batch arrived, however few there are.
	MaxSeconds int `pkl:"maxSeconds"`
}
