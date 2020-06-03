// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"fmt"
	"strings"

	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
)

const (
	channels = "channels"
	messages = "messages"
)

// Forwarder specifies MQTT forwarder interface API.
type Forwarder interface {
	// Forward subscribes to the Subscriber and
	// publishes messages using provided Publisher.
	Forward(sub messaging.Subscriber, pub messaging.Publisher) error
}

type forwarder struct {
	topic  string
	logger log.Logger
}

// NewForwarder returns new Forwarder implementation.
func NewForwarder(topic string, logger log.Logger) Forwarder {
	return forwarder{
		topic:  topic,
		logger: logger,
	}
}

func (f forwarder) Forward(sub messaging.Subscriber, pub messaging.Publisher) error {
	return sub.Subscribe(f.topic, f.handle(pub))
}

func (f forwarder) handle(pub messaging.Publisher) messaging.MessageHandler {
	return func(msg messaging.Message) error {
		if msg.Protocol == protocol {
			return nil
		}
		// Use concatenation instead of mft.Sprintf for the
		// sake of simplicity and performance.
		topic := channels + "/" + msg.Channel + "/" + messages
		if msg.Subtopic != "" {
			topic += "/" + strings.ReplaceAll(msg.Subtopic, ".", "/")
		}
		go func() {
			if err := pub.Publish(topic, msg); err != nil {
				f.logger.Warn(fmt.Sprintf("Failed to forward message: %s", err))
			}
		}()
		return nil
	}
}
