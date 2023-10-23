// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

import "context"

type DeliveryPolicy uint8

const (
	// DeliverNewPolicy will only deliver new messages that are sent after the consumer is created.
	// This is the default policy.
	DeliverNewPolicy DeliveryPolicy = iota

	// DeliverAllPolicy starts delivering messages from the very beginning of a stream.
	DeliverAllPolicy
)

// Publisher specifies message publishing API.
type Publisher interface {
	// Publishes message to the stream.
	Publish(ctx context.Context, topic string, msg *Message) error

	// Close gracefully closes message publisher's connection.
	Close() error
}

// MessageHandler represents Message handler for Subscriber.
type MessageHandler interface {
	// Handle handles messages passed by underlying implementation.
	Handle(msg *Message) error

	// Cancel is used for cleanup during unsubscribing and it's optional.
	Cancel() error
}

type SubscriberConfig struct {
	ID             string
	Topic          string
	Handler        MessageHandler
	DeliveryPolicy DeliveryPolicy
}

// Subscriber specifies message subscription API.
type Subscriber interface {
	// Subscribe subscribes to the message stream and consumes messages.
	Subscribe(ctx context.Context, cfg SubscriberConfig) error

	// Unsubscribe unsubscribes from the message stream and
	// stops consuming messages.
	Unsubscribe(ctx context.Context, id, topic string) error

	// Close gracefully closes message subscriber's connection.
	Close() error
}

// PubSub  represents aggregation interface for publisher and subscriber.
type PubSub interface {
	Publisher
	Subscriber
}
