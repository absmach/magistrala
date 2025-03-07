// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/absmach/supermq/pkg/events"
	"github.com/redis/go-redis/v9"
)

type pubEventStore struct {
	client            *redis.Client
	unpublishedEvents chan *redis.XAddArgs
	stream            string
	mu                sync.Mutex
	flushPeriod       time.Duration
}

func NewPublisher(ctx context.Context, url, stream string, flushPeriod time.Duration) (events.Publisher, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	es := &pubEventStore{
		client:            redis.NewClient(opts),
		unpublishedEvents: make(chan *redis.XAddArgs, events.MaxUnpublishedEvents),
		stream:            eventsPrefix + stream,
		flushPeriod:       flushPeriod,
	}

	go es.flushUnpublished(ctx)

	return es, nil
}

func (es *pubEventStore) Publish(ctx context.Context, event events.Event) error {
	values, err := event.Encode()
	if err != nil {
		return err
	}
	values["occurred_at"] = time.Now().UnixNano()

	data, err := json.Marshal(values)
	if err != nil {
		return err
	}

	record := &redis.XAddArgs{
		Stream: es.stream,
		MaxLen: events.MaxEventStreamLen,
		Approx: true,
		Values: map[string]interface{}{"data": string(data)},
	}

	switch err := es.checkConnection(ctx); err {
	case nil:
		return es.client.XAdd(ctx, record).Err()
	default:
		es.mu.Lock()
		defer es.mu.Unlock()

		// If the channel is full (rarely happens), drop the events.
		if len(es.unpublishedEvents) == int(events.MaxUnpublishedEvents) {
			return nil
		}

		es.unpublishedEvents <- record

		return nil
	}
}

// flushUnpublished periodically checks the Redis connection and publishes
// the events that were not published due to a connection error.
func (es *pubEventStore) flushUnpublished(ctx context.Context) {
	defer close(es.unpublishedEvents)

	ticker := time.NewTicker(es.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := es.checkConnection(ctx); err == nil {
				es.mu.Lock()
				for i := len(es.unpublishedEvents) - 1; i >= 0; i-- {
					record := <-es.unpublishedEvents
					if err := es.client.XAdd(ctx, record).Err(); err != nil {
						es.unpublishedEvents <- record

						break
					}
				}
				es.mu.Unlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

func (es *pubEventStore) Close() error {
	return es.client.Close()
}

func (es *pubEventStore) checkConnection(ctx context.Context) error {
	// A timeout is used to avoid blocking the main thread
	ctx, cancel := context.WithTimeout(ctx, events.ConnCheckInterval)
	defer cancel()

	return es.client.Ping(ctx).Err()
}
