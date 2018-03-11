// Package nats contains NATS message publisher implementation.
package nats

import (
	"encoding/json"

	"github.com/mainflux/mainflux"
	broker "github.com/nats-io/go-nats"
)

const topic string = "src.http"

var _ mainflux.MessagePublisher = (*natsPublisher)(nil)

type natsPublisher struct {
	nc *broker.Conn
}

// NewMessagePublisher instantiates NATS message publisher.
func NewMessagePublisher(nc *broker.Conn) mainflux.MessagePublisher {
	return &natsPublisher{nc}
}

func (pub *natsPublisher) Publish(msg mainflux.RawMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return pub.nc.Publish(topic, data)
}
