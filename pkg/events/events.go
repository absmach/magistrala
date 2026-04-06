// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
)

const (
	UnpublishedEventsCheckInterval        = 1 * time.Minute
	ConnCheckInterval                     = 100 * time.Millisecond
	MaxUnpublishedEvents           uint64 = 1e4
	MaxEventStreamLen              int64  = 1e6
)

// Event represents an event.
type Event interface {
	// Encode encodes event to map.
	Encode() (map[string]any, error)
}

// Publisher specifies events publishing API.
type Publisher interface {
	// Publish publishes event to stream.
	Publish(ctx context.Context, stream string, event Event) error

	// Close gracefully closes event publisher's connection.
	Close() error
}

// EventHandler represents event handler for Subscriber.
type EventHandler interface {
	// Handle handles events passed by underlying implementation.
	Handle(ctx context.Context, event Event) error
}

// SubscriberConfig represents event subscriber configuration.
type SubscriberConfig struct {
	Consumer       string
	Stream         string
	Handler        EventHandler
	Ordered        bool
	DeliveryPolicy messaging.DeliveryPolicy
}

// Subscriber specifies event subscription API.
type Subscriber interface {
	// Subscribe subscribes to the event stream and consumes events.
	Subscribe(ctx context.Context, cfg SubscriberConfig) error

	// Close gracefully closes event subscriber's connection.
	Close() error
}

// Read reads value from event map.
// If value is not of type T, returns default value.
func Read[T any](event map[string]any, key string, def T) T {
	val, ok := event[key].(T)
	if !ok {
		return def
	}

	return val
}

// ReadStringSlice reads string slice from event map.
// If value is not a string slice, returns empty slice.
func ReadStringSlice(event map[string]any, key string) []string {
	var res []string

	vals, ok := event[key].([]any)
	if !ok {
		return res
	}

	for _, v := range vals {
		if s, ok := v.(string); ok {
			res = append(res, s)
		}
	}

	return res
}
