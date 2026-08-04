// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build msg_fluxmq
// +build msg_fluxmq

package writers

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/fluxmq"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	AllTopic = "writers/#"

	prefix = "writers"
)

var cfg = jetstream.StreamConfig{
	Name:              "writers",
	Description:       "Magistrala Rules Engine stream for handling internal messages",
	Subjects:          []string{"writers/#"},
	Retention:         jetstream.LimitsPolicy,
	MaxMsgsPerSubject: 1e6,
	MaxAge:            time.Hour * 24,
	MaxMsgSize:        1024 * 1024,
	Discard:           jetstream.DiscardOld,
	Storage:           jetstream.FileStorage,
}

// InternalMetadata returns an option for a trusted local-service connection
// that carries internal metadata over mTLS and consumes the broker-provisioned
// writers stream. It lives here rather than being taken from
// pkg/messaging/brokers because that package selects its backend on a different
// build tag: an untagged build would pair a FluxMQ option with a NATS PubSub.
func InternalMetadata(certFile, keyFile, caFile string) messaging.Option {
	return broker.InternalMetadata(certFile, keyFile, caFile)
}

func NewPubSub(ctx context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	brokerOpts := []messaging.Option{
		broker.Prefix(prefix),
		broker.JSStreamConfig(cfg),
		broker.ConnectionName("writers-msg-pubsub"),
	}
	brokerOpts = append(brokerOpts, opts...)
	pb, err := broker.NewPubSub(ctx, url, logger, brokerOpts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewPublisher(ctx context.Context, url string) (messaging.Publisher, error) {
	pb, err := broker.NewPublisher(ctx, url, broker.Prefix(prefix), broker.JSStreamConfig(cfg), broker.ConnectionName("writers-msg-pub"))
	if err != nil {
		return nil, err
	}

	return pb, nil
}
