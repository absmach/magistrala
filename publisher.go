package mainflux

// MessagePublisher specifies a message publishing API.
type MessagePublisher interface {
	// Publishes message to the stream. A non-nil error is returned to indicate
	// operation failure.
	Publish(RawMessage) error
}
