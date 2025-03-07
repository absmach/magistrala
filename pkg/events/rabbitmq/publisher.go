// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/messaging"
	broker "github.com/absmach/supermq/pkg/messaging/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type pubEventStore struct {
	conn      *amqp.Connection
	publisher messaging.Publisher
	stream    string
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
		conn:      conn,
		publisher: publisher,
		stream:    stream,
	}

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

func (es *pubEventStore) Close() error {
	es.conn.Close()

	return es.publisher.Close()
}
