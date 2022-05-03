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
	Forward(id string, sub messaging.Subscriber, pub messaging.Publisher) error
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

func (f forwarder) Forward(id string, sub messaging.Subscriber, pub messaging.Publisher) error {
	return sub.Subscribe(id, f.topic, handle(pub, f.logger))
}

func handle(pub messaging.Publisher, logger log.Logger) handleFunc {
	return func(msg messaging.Message) error {
		if msg.Protocol == protocol {
			return nil
		}
		// Use concatenation instead of fmt.Sprintf for the
		// sake of simplicity and performance.
		topic := channels + "/" + msg.Channel + "/" + messages
		if msg.Subtopic != "" {
			topic += "/" + strings.ReplaceAll(msg.Subtopic, ".", "/")
		}
		go func() {
			if err := pub.Publish(topic, msg); err != nil {
				logger.Warn(fmt.Sprintf("Failed to forward message: %s", err))
			}
		}()
		return nil
	}
}

type handleFunc func(msg messaging.Message) error

func (h handleFunc) Handle(msg messaging.Message) error {
	return h(msg)

}

func (h handleFunc) Cancel() error {
	return nil
}
