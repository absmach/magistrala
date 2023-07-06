// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
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
	client   *redis.Client
	instance string
}

// NewEventStore returns wrapper around mProxy service that sends
// events to event store.
func NewEventStore(client *redis.Client, instance string) EventStore {
	return eventStore{
		client:   client,
		instance: instance,
	}
}

func (es eventStore) storeEvent(ctx context.Context, clientID, eventType string) error {
	event := mqttEvent{
		clientID:  clientID,
		eventType: eventType,
		instance:  es.instance,
	}

	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}

	return es.client.XAdd(ctx, record).Err()
}

// Connect issues event on MQTT CONNECT.
func (es eventStore) Connect(ctx context.Context, clientID string) error {
	return es.storeEvent(ctx, clientID, "connect")
}

// Disconnect issues event on MQTT CONNECT.
func (es eventStore) Disconnect(ctx context.Context, clientID string) error {
	return es.storeEvent(ctx, clientID, "disconnect")
}
