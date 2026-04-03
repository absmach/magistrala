// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build msg_nats
// +build msg_nats

package brokers

import (
	"context"
	"log"
	"log/slog"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/nats"
)

// SubjectAllMessages represents subject to subscribe for all the messages.
const SubjectAllMessages = string(messaging.MsgTopicPrefix) + "/#"

func init() {
	log.Println("The binary was built using NATS as the message broker")
}

// ConnectionName is a no-op for the NATS backend. It exists for API
// compatibility with the FluxMQ variant.
func ConnectionName(_ string) messaging.Option {
	return func(_ any) error { return nil }
}

func NewPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	pb, err := nats.NewPublisher(ctx, url, opts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewPubSub(ctx context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	pb, err := nats.NewPubSub(ctx, url, logger, opts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
