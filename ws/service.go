package ws

import (
	"errors"

	"github.com/mainflux/mainflux"
)

var (
	// ErrFailedMessagePublish indicates that message publishing failed.
	ErrFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel.
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// ErrFailedConnection indicates that service couldn't connect to message broker.
	ErrFailedConnection = errors.New("failed to connect to message broker")
)

// Service specifies web socket service API.
type Service interface {
	mainflux.MessagePublisher
	// Subscribes to channel with specified id.
	Subscribe(string, Channel) error
}

// Channel is used for recieving and sending messages.
type Channel struct {
	Messages chan mainflux.RawMessage
	Closed   chan bool
}

// Close channel and stop message transfer.
func (channel Channel) Close() {
	close(channel.Messages)
	close(channel.Closed)
}
