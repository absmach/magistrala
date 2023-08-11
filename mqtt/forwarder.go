// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"fmt"
	"strings"

	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
)

// Forwarder specifies MQTT forwarder interface API.
type Forwarder interface {
	// Forward subscribes to the Subscriber and
	// publishes messages using provided Publisher.
	Forward(ctx context.Context, id string, sub messaging.Subscriber, pub messaging.Publisher) error
}

type forwarder struct {
	topic  string
	logger mflog.Logger
}

// NewForwarder returns new Forwarder implementation.
func NewForwarder(topic string, logger mflog.Logger) Forwarder {
	return forwarder{
		topic:  topic,
		logger: logger,
	}
}

func (f forwarder) Forward(ctx context.Context, id string, sub messaging.Subscriber, pub messaging.Publisher) error {
	return sub.Subscribe(ctx, id, f.topic, handle(ctx, pub, f.logger))
}

func handle(ctx context.Context, pub messaging.Publisher, logger mflog.Logger) handleFunc {
	return func(msg *messaging.Message) error {
		if msg.Protocol == protocol {
			return nil
		}
		// Use concatenation instead of fmt.Sprintf for the
		// sake of simplicity and performance.
		topic := fmt.Sprintf("channels/%s/messages", msg.Channel)
		if msg.Subtopic != "" {
			topic = fmt.Sprintf("%s/%s", topic, strings.ReplaceAll(msg.Subtopic, ".", "/"))
		}
		go func() {
			if err := pub.Publish(ctx, topic, msg); err != nil {
				logger.Warn(fmt.Sprintf("Failed to forward message: %s", err))
			}
		}()
		return nil
	}
}

type handleFunc func(msg *messaging.Message) error

func (h handleFunc) Handle(msg *messaging.Message) error {
	return h(msg)
}

func (h handleFunc) Cancel() error {
	return nil
}
