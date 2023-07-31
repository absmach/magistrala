// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
	mfredis "github.com/mainflux/mainflux/internal/clients/redis"
)

const (
	streamID  = "mainflux.mqtt"
	streamLen = 1000
)

type EventStore interface {
	Connect(ctx context.Context, clientID string) error
	Disconnect(ctx context.Context, clientID string) error
}

// EventStore is a struct used to store event streams in Redis.
type eventStore struct {
	mfredis.Publisher
	client   *redis.Client
	instance string
}

// NewEventStore returns wrapper around mProxy service that sends
// events to event store.
func NewEventStore(ctx context.Context, client *redis.Client, instance string) EventStore {
	es := &eventStore{
		client:    client,
		instance:  instance,
		Publisher: mfredis.NewEventStore(client, streamID, streamLen),
	}

	go es.StartPublishingRoutine(ctx)

	return es
}

// Connect issues event on MQTT CONNECT.
func (es *eventStore) Connect(ctx context.Context, clientID string) error {
	ev := mqttEvent{
		clientID:  clientID,
		eventType: "connect",
		instance:  es.instance,
	}

	return es.Publish(ctx, ev)
}

// Disconnect issues event on MQTT CONNECT.
func (es *eventStore) Disconnect(ctx context.Context, clientID string) error {
	ev := mqttEvent{
		clientID:  clientID,
		eventType: "disconnect",
		instance:  es.instance,
	}

	return es.Publish(ctx, ev)
}
