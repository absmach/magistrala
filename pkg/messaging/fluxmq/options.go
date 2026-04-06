// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"errors"
	"strings"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/nats-io/nats.go/jetstream"
)

// ErrInvalidType is returned when the provided value is not of the expected type.
var ErrInvalidType = errors.New("invalid type")

const msgPrefix = "m"

type options struct {
	prefix             string
	connectionName     string
	directTopicIngress bool
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
			v.prefix = strings.TrimSpace(prefix)
			if v.prefix == "" {
				v.prefix = msgPrefix
			}
		case *pubsub:
			v.prefix = strings.TrimSpace(prefix)
			if v.prefix == "" {
				v.prefix = msgPrefix
			}
		default:
			return ErrInvalidType
		}

		return nil
	}
}

// ConnectionName sets a human-readable connection name sent to FluxMQ
// for identifying this client in the broker's admin UI.
func ConnectionName(name string) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.connectionName = name
		case *pubsub:
			v.connectionName = name
		default:
			return ErrInvalidType
		}

		return nil
	}
}

// DirectTopicIngress enables direct MQTT topic delivery in addition to stream
// queue delivery. This is opt-in because direct topic messages are normalized
// from broker-native metadata instead of the protobuf queue envelope.
func DirectTopicIngress() messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			return nil
		case *pubsub:
			v.directTopicIngress = true
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
