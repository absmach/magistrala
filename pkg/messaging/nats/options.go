// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"errors"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	// ErrInvalidType is returned when the provided value is not of the expected type.
	ErrInvalidType = errors.New("invalid type")

	jsStreamConfig = jetstream.StreamConfig{
		Name:              "m",
		Description:       "Magistrala stream for sending and receiving messages in between Magistrala channels",
		Subjects:          []string{"m.>"},
		Retention:         jetstream.LimitsPolicy,
		MaxMsgsPerSubject: 1e6,
		MaxAge:            time.Hour * 24,
		MaxMsgSize:        1024 * 1024,
		Discard:           jetstream.DiscardOld,
		Storage:           jetstream.FileStorage,
	}
)

const msgPrefix = "m"

type options struct {
	prefix         string
	jsStreamConfig jetstream.StreamConfig
}

func defaultOptions() options {
	return options{
		prefix:         msgPrefix,
		jsStreamConfig: jsStreamConfig,
	}
}

// Prefix sets the prefix for the publisher or subscriber.
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

// JSStreamConfig sets the JetStream for the publisher or subscriber.
func JSStreamConfig(jsStreamConfig jetstream.StreamConfig) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.jsStreamConfig = jsStreamConfig
		case *pubsub:
			v.jsStreamConfig = jsStreamConfig
		default:
			return ErrInvalidType
		}

		return nil
	}
}
