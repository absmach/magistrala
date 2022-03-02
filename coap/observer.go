// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	broker "github.com/nats-io/nats.go"
)

// Observer represents an internal observer used to handle CoAP observe messages.
type Observer interface {
	Cancel(topic string) error
}

// NewObserver returns a new Observer instance.
func NewObserver(subject string, c Client, pubsub nats.PubSub) (Observer, error) {
	err := pubsub.Subscribe(subject, c.SendMessage)
	if err != nil {
		return nil, err
	}
	ret := &observer{
		client: c,
		pubsub: pubsub,
	}
	return ret, nil
}

type observer struct {
	client Client
	pubsub nats.PubSub
}

func (o *observer) Cancel(topic string) error {
	if err := o.pubsub.Unsubscribe(topic); err != nil && err != broker.ErrConnectionClosed {
		return err
	}
	return o.client.Cancel()
}
