// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
)

const streamID = "magistrala.mqtt"

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
	publisher, err := store.NewPublisher(ctx, url, streamID)
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
