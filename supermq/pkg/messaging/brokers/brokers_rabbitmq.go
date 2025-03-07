// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build rabbitmq
// +build rabbitmq

package brokers

import (
	"context"
	"log"
	"log/slog"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/rabbitmq"
)

// SubjectAllChannels represents subject to subscribe for all the channels.
const SubjectAllChannels = "channels.#"

func init() {
	log.Println("The binary was build using RabbitMQ as the message broker")
}

func NewPublisher(_ context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	pb, err := rabbitmq.NewPublisher(url, opts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewPubSub(_ context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	pb, err := rabbitmq.NewPubSub(url, logger, opts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
