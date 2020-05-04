// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux/messaging"
	broker "github.com/nats-io/nats.go"
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	conn *broker.Conn
}

// Publisher wraps messaging Publisher exposing
// Close() method for NATS connection.
type Publisher interface {
	messaging.Publisher
	Close()
}

// NewPublisher returns NATS message Publisher.
func NewPublisher(url string) (Publisher, error) {
	conn, err := broker.Connect(url)
	if err != nil {
		return nil, err
	}
	ret := &publisher{
		conn: conn,
	}
	return ret, nil
}

func (pub *publisher) Publish(topic string, msg messaging.Message) error {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, topic)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
	}
	if err := pub.conn.Publish(subject, data); err != nil {
		return err
	}

	return nil
}

func (pub *publisher) Close() {
	pub.conn.Close()
}
