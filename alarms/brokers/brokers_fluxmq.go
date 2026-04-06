// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build msg_fluxmq
// +build msg_fluxmq

package brokers

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/fluxmq"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	AllTopic = "alarms/#"

	prefix = "alarms"
)

var cfg = jetstream.StreamConfig{
	Name:              "alarms",
	Description:       "Magistrala stream alarms",
	Subjects:          []string{"alarms/#"},
	Retention:         jetstream.LimitsPolicy,
	MaxMsgsPerSubject: 1e6,
	MaxAge:            time.Hour * 24,
	MaxMsgSize:        1024 * 1024,
	Discard:           jetstream.DiscardOld,
	Storage:           jetstream.FileStorage,
}

func NewPubSub(ctx context.Context, url string, logger *slog.Logger) (messaging.PubSub, error) {
	pb, err := broker.NewPubSub(ctx, url, logger, broker.Prefix(prefix), broker.JSStreamConfig(cfg), broker.ConnectionName("alarms-msg-pubsub"))
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewPublisher(ctx context.Context, url string) (messaging.Publisher, error) {
	pb, err := broker.NewPublisher(ctx, url, broker.Prefix(prefix), broker.JSStreamConfig(cfg), broker.ConnectionName("alarms-msg-pub"))
	if err != nil {
		return nil, err
	}

	return pb, nil
}
