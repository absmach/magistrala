// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/absmach/supermq/pkg/messaging"
)

// Forwarder specifies MQTT forwarder interface API.
type Forwarder interface {
	// Forward subscribes to the Subscriber and
	// publishes messages using provided Publisher.
	Forward(ctx context.Context, id string, sub messaging.Subscriber, pub messaging.Publisher) error
}

type forwarder struct {
	topic  string
	logger *slog.Logger
}

// NewForwarder returns new Forwarder implementation.
func NewForwarder(topic string, logger *slog.Logger) Forwarder {
	return forwarder{
		topic:  topic,
		logger: logger,
	}
}

func (f forwarder) Forward(ctx context.Context, id string, sub messaging.Subscriber, pub messaging.Publisher) error {
	subCfg := messaging.SubscriberConfig{
		ID:      id,
		Topic:   f.topic,
		Handler: handle(ctx, pub, f.logger),
	}

	return sub.Subscribe(ctx, subCfg)
}

func handle(ctx context.Context, pub messaging.Publisher, logger *slog.Logger) handleFunc {
	return func(msg *messaging.Message) error {
		if msg.GetProtocol() == protocol {
			return nil
		}
		// Use concatenation instead of fmt.Sprintf for the
		// sake of simplicity and performance.
		topic := "channels/" + msg.GetChannel() + "/messages"
		if msg.GetSubtopic() != "" {
			topic = topic + "/" + strings.ReplaceAll(msg.GetSubtopic(), ".", "/")
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
