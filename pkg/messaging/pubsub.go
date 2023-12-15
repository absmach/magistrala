// Copyright (c) Abstract Machines
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
//
//go:generate mockery --name PubSub --filename pubsub.go --quiet --note "Copyright (c) Abstract Machines"
type PubSub interface {
	Publisher
	Subscriber
}

// Option represents optional configuration for message broker.
//
// This is used to provide optional configuration parameters to the
// underlying publisher and pubsub implementation so that it can be
// configured to meet the specific needs.
//
// For example, it can be used to set the message prefix so that
// brokers can be used for event sourcing as well as internal message broker.
// Using value of type interface is not recommended but is the most suitable
// for this use case as options should be compiled with respect to the
// underlying broker which can either be RabbitMQ or NATS.
//
// The example below shows how to set the prefix and jetstream stream for NATS.
//
// Example:
//
//	broker.NewPublisher(ctx, url, broker.Prefix(eventsPrefix), broker.JSStream(js))
type Option func(vals interface{}) error
