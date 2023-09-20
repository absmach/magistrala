// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/mainflux/mainflux/pkg/events"
	"github.com/mainflux/mainflux/pkg/events/redis"
)

const streamID = "mainflux.mqtt"

type EventStore interface {
	Connect(ctx context.Context, clientID string) error
	Disconnect(ctx context.Context, clientID string) error
}

// EventStore is a struct used to store event streams in Redis.
type eventStore struct {
	events.Publisher
	instance string
}

// NewEventStore returns wrapper around mProxy service that sends
// events to event store.
func NewEventStore(ctx context.Context, url, instance string) (EventStore, error) {
	publisher, err := redis.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		instance:  instance,
		Publisher: publisher,
	}, nil
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
