// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"errors"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var errPublishTimeout = errors.New("failed to publish due to timeout reached")

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	client  mqtt.Client
	timeout time.Duration
	qos     uint8
}

// NewPublisher returns a new MQTT message publisher.
func NewPublisher(address string, qos uint8, timeout time.Duration) (messaging.Publisher, error) {
	client, err := newClient(address, "mqtt-publisher", timeout)
	if err != nil {
		return nil, err
	}

	ret := publisher{
		client:  client,
		timeout: timeout,
		qos:     qos,
	}
	return ret, nil
}

func (pub publisher) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	// Publish only the payload and not the whole message.
	token := pub.client.Publish(topic, byte(pub.qos), false, msg.GetPayload())
	if token.Error() != nil {
		return token.Error()
	}

	if ok := token.WaitTimeout(pub.timeout); !ok {
		return errPublishTimeout
	}

	return nil
}

func (pub publisher) Close() error {
	pub.client.Disconnect(uint(pub.timeout))
	return nil
}
