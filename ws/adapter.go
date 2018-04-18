package ws

import (
	"github.com/mainflux/mainflux"
	broker "github.com/nats-io/go-nats"
)

var _ Service = (*adapterService)(nil)

type adapterService struct {
	pubsub Service
}

// New instantiates the domain service implementation.
func New(pubsub Service) Service {
	return &adapterService{pubsub}
}

func (as *adapterService) Publish(msg mainflux.RawMessage) error {
	if err := as.pubsub.Publish(msg); err != nil {
		switch err {
		case broker.ErrConnectionClosed, broker.ErrInvalidConnection:
			return ErrFailedConnection
		default:
			return ErrFailedMessagePublish
		}
	}
	return nil
}

func (as *adapterService) Subscribe(chanID string, channel Channel) error {
	if err := as.pubsub.Subscribe(chanID, channel); err != nil {
		return ErrFailedSubscription
	}
	return nil
}
