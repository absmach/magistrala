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

const AllTopic = "alarms.#"

func NewPubSub(ctx context.Context, url string, logger *slog.Logger) (messaging.PubSub, error) {
	pb, err := broker.NewPubSub(ctx, url, logger, broker.Prefix("alarms"))
	if err != nil {
		return nil, err
	}

	return pb, nil
}
