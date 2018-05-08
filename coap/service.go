package coap

import (
	"errors"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap/nats"
)

var (
	// ErrFailedMessagePublish indicates that message publishing failed.
	ErrFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel.
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// ErrFailedConnection indicates that service couldn't connect to message broker.
	ErrFailedConnection = errors.New("failed to connect to message broker")
)

// Service specifies coap service API.
type Service interface {
	mainflux.MessagePublisher
	// Subscribes to channel with specified id and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(string, string, nats.Channel) error
	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(string)
	// SetTimeout sets timeout to wait CONF messages.
	SetTimeout(string, *time.Timer, int) (chan bool, error)
	// RemoveTimeout removes timeout when ACK message is received from client
	// if timeout existed.
	RemoveTimeout(string)
}
