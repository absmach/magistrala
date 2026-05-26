// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/magistrala/pkg/messaging"
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

	props := map[string]string{
		"external_id": msg.GetPublisher(),
		"protocol":    msg.GetProtocol(),
	}
	if clientID := msg.ClientIdentity(); clientID != "" {
		props["client_id"] = clientID
	}
	if msg.GetCreated() != 0 {
		props["created"] = strconv.FormatInt(msg.GetCreated(), 10)
	}

	cleanTopic := strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if cleanTopic == "" {
		return ErrEmptyTopic
	}

	// $queue/-prefixed topics are routed to the durable stream queue.
	if queueName, ok := strings.CutPrefix(cleanTopic, queuePrefix); ok {
		return pub.client.PublishToQueueWithOptionsContext(ctx, &fluxamqp.QueuePublishOptions{
			QueueName:  queueName,
			Payload:    msg.GetPayload(),
			Properties: props,
		})
	}

	// Normalize to "prefix/subTopic" — strip any existing prefix to avoid doubling.
	subTopic := strings.TrimPrefix(cleanTopic, pub.prefix+"/")
	publishTopic := pub.prefix + "/" + subTopic

	// Non-default prefix publishers (e.g. "writers", "alarms") are stream-backed:
	// route to the durable queue so stream subscribers receive the message.
	if pub.prefix != msgPrefix {
		return pub.client.PublishToQueueWithOptionsContext(ctx, &fluxamqp.QueuePublishOptions{
			QueueName:  publishTopic,
			Payload:    msg.GetPayload(),
			Properties: props,
		})
	}

	return pub.client.PublishWithOptionsContext(ctx, &fluxamqp.PublishOptions{
		Topic:      publishTopic,
		Payload:    msg.GetPayload(),
		Properties: props,
	})
}

func (pub *publisher) Close() error {
	return pub.client.Close()
}
