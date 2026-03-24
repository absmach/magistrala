// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"log/slog"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/supermq/pkg/messaging"
	"google.golang.org/protobuf/proto"
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	client *fluxamqp.Client
	options
}

// NewPublisher creates a FluxMQ-backed message publisher.
func NewPublisher(_ context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	pub := &publisher{
		options: defaultOptions(),
	}

	for _, opt := range opts {
		if err := opt(pub); err != nil {
			return nil, err
		}
	}

	logger := slog.Default()
	amqpOpts := fluxamqp.NewOptions().SetURL(url).
		SetConnectionName(pub.connectionName).
		SetOnConnectionLost(func(err error) {
			logger.Warn("FluxMQ message publisher connection lost", "error", err)
		}).
		SetOnReconnecting(func(attempt int) {
			logger.Info("FluxMQ message publisher reconnecting", "attempt", attempt)
		}).
		SetOnConnect(func() {
			logger.Info("FluxMQ message publisher connected")
		})

	client, err := fluxamqp.New(amqpOpts)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if err := declareStream(client, pub.prefix); err != nil {
		_ = client.Close()
		return nil, err
	}

	pub.client = client

	return pub, nil
}

func (pub *publisher) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	return pub.client.PublishContext(ctx, queueTopic(pub.prefix, topic), data)
}

func (pub *publisher) Close() error {
	return pub.client.Close()
}
