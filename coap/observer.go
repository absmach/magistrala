// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux/pkg/messaging"
	broker "github.com/nats-io/nats.go"
)

// Observer represents an internal observer used to handle CoAP observe messages.
type Observer interface {
	Cancel() error
}

// NewObserver returns a new Observer instance.
func NewObserver(subject string, c Client, conn *broker.Conn) (Observer, error) {
	sub, err := conn.Subscribe(subject, func(m *broker.Msg) {
		var msg messaging.Message
		if err := proto.Unmarshal(m.Data, &msg); err != nil {
			return
		}
		// There is no error handling, but the client takes care to log the error.
		c.SendMessage(msg)
	})
	if err != nil {
		return nil, err
	}
	ret := &observer{
		client: c,
		sub:    sub,
	}
	return ret, nil
}

type observer struct {
	client Client
	sub    *broker.Subscription
}

func (o *observer) Cancel() error {
	if err := o.sub.Unsubscribe(); err != nil && err != broker.ErrConnectionClosed {
		return err
	}
	return o.client.Cancel()
}
