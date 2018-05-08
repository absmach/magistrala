package nats

import "github.com/mainflux/mainflux"

// Service specifies NATS service API.
type Service interface {
	mainflux.MessagePublisher
	// Subscribe is used to subscribe to channel with specified id.
	Subscribe(string, Channel) error
}

// Channel is used for receiving and sending messages.
type Channel struct {
	Messages chan mainflux.RawMessage
	Closed   chan bool
	Timer    chan bool
	Notify   chan bool
}

// Close channel and stop message transfer.
func (channel Channel) Close() {
	close(channel.Messages)
	close(channel.Closed)
	close(channel.Timer)
	close(channel.Notify)
}
