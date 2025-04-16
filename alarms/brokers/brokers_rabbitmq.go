// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build rabbitmq
// +build rabbitmq

package brokers

import (
	"context"
	"log/slog"

	"github.com/absmach/supermq/pkg/messaging"
	broker "github.com/absmach/supermq/pkg/messaging/rabbitmq"
)

const (
	AllTopic = "alarms.#"

	exchangeName = "writers"
	prefix       = "writers"
)

func NewPubSub(_ context.Context, url string, logger *slog.Logger) (messaging.PubSub, error) {
	pb, err := broker.NewPubSub(url, logger, broker.Prefix("writers"), broker.Exchange(exchangeName))
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewPublisher(_ context.Context, url string) (messaging.Publisher, error) {
	pb, err := broker.NewPublisher(url, broker.Prefix("writers"), broker.Exchange(exchangeName))
	if err != nil {
		return nil, err
	}

	return pb, nil
}
