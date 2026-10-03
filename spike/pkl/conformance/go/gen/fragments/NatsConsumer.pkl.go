// Code generated from Pkl module `spike.contract.Fragments`. DO NOT EDIT.
package fragments

// What a consumer binds to: a stream, a durable name, and the subject it filters. Durable by name, because a consumer that forgets its position on restart replays or loses whatever arrived while it was gone.
type NatsConsumer struct {
	// The stream to consume from. It exists already: a service does not create the stream it reads.
	Stream string `pkl:"stream"`

	// The durable consumer name. Shared by every replica of this component, which is what makes them one consumer group rather than several.
	Durable string `pkl:"durable"`

	// Filter the stream to this subject. Unset consumes everything the stream holds.
	Subject *string `pkl:"subject"`
}
