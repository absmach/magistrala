//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package nats contains NATS message publisher implementation.
package nats

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	broker "github.com/nats-io/go-nats"
)

const prefix = "channel"

var _ mainflux.MessagePublisher = (*natsPublisher)(nil)

type natsPublisher struct {
	nc *broker.Conn
}

// New instantiates NATS message publisher.
func New(nc *broker.Conn) coap.Broker {
	return &natsPublisher{nc}
}

func (pubsub *natsPublisher) Publish(msg mainflux.RawMessage) error {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", prefix, msg.Channel)
	return pubsub.nc.Publish(subject, data)
}

func (pubsub *natsPublisher) Subscribe(chanID, obsID string, observer *coap.Observer) error {
	sub, err := pubsub.nc.Subscribe(fmt.Sprintf("%s.%s", prefix, chanID), func(msg *broker.Msg) {
		if msg == nil {
			return
		}
		var rawMsg mainflux.RawMessage
		if err := proto.Unmarshal(msg.Data, &rawMsg); err != nil {
			return
		}
		observer.Messages <- rawMsg
	})
	if err != nil {
		return err
	}

	go func() {
		<-observer.Cancel
		sub.Unsubscribe()
	}()

	return nil
}
