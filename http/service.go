package http

import "github.com/mainflux/mainflux/writer"

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// Publish accepts the raw SenML message and publishes it to the event bus
	// for post processing.
	Publish(writer.RawMessage) error
}
