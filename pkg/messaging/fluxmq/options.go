// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"errors"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	// ErrInvalidType is returned when the provided value is not of the expected type.
	ErrInvalidType = errors.New("invalid type")
)

const msgPrefix = "m"

type options struct {
	prefix string
}

func defaultOptions() options {
	return options{
		prefix: msgPrefix,
	}
}

// Prefix sets the topic prefix for publisher and subscriber.
func Prefix(prefix string) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.prefix = prefix
		case *pubsub:
			v.prefix = prefix
		default:
			return ErrInvalidType
		}

		return nil
	}
}

// JSStreamConfig is a no-op for FluxMQ AMQP backend and exists only to keep
// option-compatibility with legacy NATS broker wrappers.
func JSStreamConfig(_ jetstream.StreamConfig) messaging.Option {
	return func(val any) error {
		switch val.(type) {
		case *publisher, *pubsub:
			return nil
		default:
			return ErrInvalidType
		}
	}
}
