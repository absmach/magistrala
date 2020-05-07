// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messaging

// Publisher specifies message publishing API.
type Publisher interface {
	// Publishes message to the stream.
	Publish(topic string, msg Message) error
}

// MessageHandler represents Message handler for Subscriber.
type MessageHandler func(msg Message) error

// Subscriber specifies message subscription API.
type Subscriber interface {
	// Subscribe subscribes to the message stream and consumes messages.
	Subscribe(topic string, handler MessageHandler) error

	// Unsubscribe unsubscribes from the message stream and
	// stops consuming messages.
	Unsubscribe(topic string) error
}

// PubSub  represents aggregation interface for publisher and subscriber.
type PubSub interface {
	Publisher
	Subscriber
}
