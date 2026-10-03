// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// The NAMES of the environment variables holding the credentials, never the values. Unset means the SDK's ambient credentials, which is what a workload identity provides.
type BucketCredentialsEnv struct {
	AccessKeyID string `pkl:"accessKeyID"`

	SecretAccessKey string `pkl:"secretAccessKey"`
}
