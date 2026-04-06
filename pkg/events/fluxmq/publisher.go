// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	eventsQueue  = "events"
	eventsPrefix = "events."
	queuePrefix  = "$queue/"
)

type pubEventStore struct {
	client *fluxamqp.Client
}

// NewPublisher creates a FluxMQ-backed event publisher.
func NewPublisher(_ context.Context, url, connectionName string) (events.Publisher, error) {
	logger := slog.Default()
	opts := fluxamqp.NewOptions().SetURL(url).
		SetConnectionName(connectionName).
		SetOnConnectionLost(func(err error) {
			logger.Warn("FluxMQ event publisher connection lost", "error", err)
		}).
		SetOnReconnecting(func(attempt int) {
			logger.Info("FluxMQ event publisher reconnecting", "attempt", attempt)
		}).
		SetOnConnect(func() {
			logger.Info("FluxMQ event publisher connected")
		})

	client, err := fluxamqp.New(opts)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if err := declareEventsStream(client); err != nil {
		return nil, err
	}

	return &pubEventStore{client: client}, nil
}

func (es *pubEventStore) Publish(ctx context.Context, stream string, event events.Event) error {
	values, err := event.Encode()
	if err != nil {
		return err
	}

	values["occurred_at"] = time.Now().UnixNano()
	values["stream"] = canonicalStream(stream)

	data, err := json.Marshal(values)
	if err != nil {
		return err
	}

	return es.client.PublishContext(ctx, queueTopic(stream), data)
}

func (es *pubEventStore) Close() error {
	return es.client.Close()
}
