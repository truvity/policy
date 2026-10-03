// Code generated from Pkl module `spike.contract.urlshortener.LogArchiver`. DO NOT EDIT.
package logarchiver

import "spike.invalid/pklconf/gen/fragments"

type Events struct {
	Nats fragments.Nats `pkl:"nats"`

	Consumer fragments.NatsConsumer `pkl:"consumer"`
}
