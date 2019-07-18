//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package nats contains NATS message publisher implementation.
package nats

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	broker "github.com/nats-io/go-nats"
)

const prefix = "channel"

var _ mainflux.MessagePublisher = (*natsPublisher)(nil)

type natsPublisher struct {
	nc *broker.Conn
}

// NewMessagePublisher instantiates NATS message publisher.
func NewMessagePublisher(nc *broker.Conn) mainflux.MessagePublisher {
	return &natsPublisher{nc: nc}
}

func (pub *natsPublisher) Publish(_ context.Context, _ string, msg mainflux.RawMessage) error {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", prefix, msg.Channel)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
	}
	return pub.nc.Publish(subject, data)
}
