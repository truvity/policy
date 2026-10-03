// Code generated from Pkl module `spike.contract.Platform`. DO NOT EDIT.
package platform

// Where the image is. A digest when there is one; a tag only when there is not. Left out, the library reads the chart's own top-level `images.<component>`, the map a release stamps and refuses to publish with an empty digest.
type PlatformImage struct {
	Registry *string `pkl:"registry"`

	Repository string `pkl:"repository"`

	Tag *string `pkl:"tag"`

	Digest *string `pkl:"digest"`
}
