// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"errors"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/nats-io/nats.go/jetstream"
)

// ErrInvalidType is returned when the provided value is not of the expected type.
var ErrInvalidType = errors.New("invalid type")

// Prefix sets the prefix for the publisher.
func Prefix(prefix string) messaging.Option {
	return func(val interface{}) error {
		p, ok := val.(*publisher)
		if !ok {
			return ErrInvalidType
		}

		p.prefix = prefix

		return nil
	}
}

// JSStream sets the JetStream for the publisher.
func JSStream(stream jetstream.JetStream) messaging.Option {
	return func(val interface{}) error {
		p, ok := val.(*publisher)
		if !ok {
			return ErrInvalidType
		}

		p.js = stream

		return nil
	}
}

// Stream sets the Stream for the subscriber.
func Stream(stream jetstream.Stream) messaging.Option {
	return func(val interface{}) error {
		p, ok := val.(*pubsub)
		if !ok {
			return ErrInvalidType
		}

		p.stream = stream

		return nil
	}
}
