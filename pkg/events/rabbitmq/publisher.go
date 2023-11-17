// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type pubEventStore struct {
	conn              *amqp.Connection
	publisher         messaging.Publisher
	unpublishedEvents chan amqp.Return
	stream            string
	mu                sync.Mutex
}

func NewPublisher(ctx context.Context, url, stream string) (events.Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchangeName, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return nil, err
	}

	publisher, err := broker.NewPublisher(url, broker.Prefix(eventsPrefix), broker.Exchange(exchangeName), broker.Channel(ch))
	if err != nil {
		return nil, err
	}

	es := &pubEventStore{
		conn:              conn,
		publisher:         publisher,
		unpublishedEvents: make(chan amqp.Return, events.MaxUnpublishedEvents),
		stream:            stream,
	}

	ch.NotifyReturn(es.unpublishedEvents)

	go es.StartPublishingRoutine(ctx)

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

	record := &messaging.Message{
		Payload: data,
	}

	return es.publisher.Publish(ctx, es.stream, record)
}

func (es *pubEventStore) StartPublishingRoutine(ctx context.Context) {
	defer close(es.unpublishedEvents)

	ticker := time.NewTicker(events.UnpublishedEventsCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ok := es.conn.IsClosed(); !ok {
				es.mu.Lock()
				for i := len(es.unpublishedEvents) - 1; i >= 0; i-- {
					record := <-es.unpublishedEvents
					msg := &messaging.Message{
						Payload: record.Body,
					}
					if err := es.publisher.Publish(ctx, es.stream, msg); err != nil {
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
	es.conn.Close()

	return es.publisher.Close()
}
