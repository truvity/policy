// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// An object store addressed by the S3 API. The vendor is not part of the configuration: an endpoint, a region and a path-style flag are enough to reach a cloud service, an in-cluster store or a test double, and a service that hard-codes one of them cannot be tested without it.
type Bucket struct {
	// The bucket. It exists already: a service does not create its own store.
	Name string `pkl:"name"`

	// The region the bucket is in.
	Region *string `pkl:"region"`

	// Override the API endpoint. Unset means the SDK's own resolution for the region.
	Endpoint *string `pkl:"endpoint"`

	// Path to a certificate authority bundle the platform mounts, for a store whose endpoint is not signed by a public root. A store inside somebody's own network is the ordinary case, not the exotic one. Unset means the system trust store.
	Ca *string `pkl:"ca"`

	// Address the bucket as a path rather than as a host. Required by most non-cloud implementations.
	PathStyle bool `pkl:"pathStyle"`

	// The NAMES of the environment variables holding the credentials, never the values. Unset means the SDK's ambient credentials, which is what a workload identity provides.
	CredentialsEnv *BucketCredentialsEnv `pkl:"credentialsEnv"`
}
