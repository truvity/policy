// Code generated from Pkl module `spike.contract.urlshortener.LogArchiver`. DO NOT EDIT.
package logarchiver

import "spike.invalid/pklconf/gen/fragments"

type Archive struct {
	Bucket fragments.Bucket `pkl:"bucket"`

	// What every key this component writes begins with. A bucket is usually shared, and a component that writes to the root of one cannot be given permission to write only its own objects.
	Prefix string `pkl:"prefix"`

	// When to close an object and write it. Both limits apply: whichever is reached first. A size-only rule means a quiet hour is never archived; a time-only rule means a busy one is archived in objects too small to be worth reading.
	Batch *Batch `pkl:"batch"`
}
