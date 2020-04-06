// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package broker

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux/errors"
	"github.com/nats-io/nats.go"
)

// Nats specifies a NATS message API.
type Nats interface {
	// Publish publishes message to the msessage broker.
	Publish(ctx context.Context, token string, msg Message) error

	// Subscribe subscribes to a message broker subject.
	Subscribe(subject string, consumer func(msg *nats.Msg)) (*nats.Subscription, error)

	// Subscribe subscribes to the message broker for a given channel ID and subtopic.
	QueueSubscribe(subject, queue string, f func(msg *nats.Msg)) (*nats.Subscription, error)

	// Close closes NATS connection.
	Close()
}

const (
	chansPrefix = "channels"

	// SubjectAllChannels define the subject to subscribe to all channels messages
	SubjectAllChannels = "channels.>"
)

var (
	errNatsConn     = errors.New("Failed to connect to NATS")
	errNatsPub      = errors.New("Failed to publish to NATS")
	errNatsSub      = errors.New("Failed to subscribe to NATS")
	errNatsQueueSub = errors.New("Failed to queue subscribe to NATS")
)

var _ Nats = (*broker)(nil)

type broker struct {
	conn *nats.Conn
}

// New returns NATS message broker.
func New(url string) (Nats, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, errors.Wrap(errNatsConn, err)
	}

	return &broker{
		conn: nc,
	}, nil
}

func (b broker) Publish(_ context.Context, _ string, msg Message) error {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, msg.Channel)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
	}
	if err := b.conn.Publish(subject, data); err != nil {
		return errors.Wrap(errNatsPub, err)
	}

	return nil
}

func (b broker) Subscribe(subject string, f func(msg *nats.Msg)) (*nats.Subscription, error) {
	ps := fmt.Sprintf("%s.%s", chansPrefix, subject)
	sub, err := b.conn.Subscribe(ps, f)
	if err != nil {
		return nil, errors.Wrap(errNatsSub, err)
	}

	return sub, nil
}

func (b broker) QueueSubscribe(subject, queue string, f func(msg *nats.Msg)) (*nats.Subscription, error) {
	sub, err := b.conn.QueueSubscribe(subject, queue, f)
	if err != nil {
		return nil, errors.Wrap(errNatsQueueSub, err)
	}

	return sub, nil
}

func (b broker) Close() {
	b.conn.Close()
}
