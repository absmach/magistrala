// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !msg_fluxmq

package writers

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/nats"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	AllTopic = "writers/#"

	prefix = "writers"
)

var cfg = jetstream.StreamConfig{
	Name:              "writers",
	Description:       "Magistrala Rules Engine stream for handling internal messages",
	Subjects:          []string{"writers.>"},
	Retention:         jetstream.LimitsPolicy,
	MaxMsgsPerSubject: 1e6,
	MaxAge:            time.Hour * 24,
	MaxMsgSize:        1024 * 1024,
	Discard:           jetstream.DiscardOld,
	Storage:           jetstream.FileStorage,
}

// InternalMetadata is a no-op for the NATS backend. It exists for compile-time
// compatibility with the FluxMQ variant; NATS carries metadata in the protobuf
// message.
func InternalMetadata(_, _, _ string) messaging.Option {
	return func(_ any) error { return nil }
}

func NewPubSub(ctx context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	brokerOpts := []messaging.Option{
		broker.Prefix(prefix),
		broker.JSStreamConfig(cfg),
	}
	brokerOpts = append(brokerOpts, opts...)
	pb, err := broker.NewPubSub(ctx, url, logger, brokerOpts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	brokerOpts := []messaging.Option{
		broker.Prefix(prefix),
		broker.JSStreamConfig(cfg),
	}
	brokerOpts = append(brokerOpts, opts...)
	pb, err := broker.NewPublisher(ctx, url, brokerOpts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
