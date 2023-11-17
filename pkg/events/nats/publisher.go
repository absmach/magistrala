// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/nats"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Max message payload size is 1MB.
var reconnectBufSize = 1024 * 1024 * int(events.MaxUnpublishedEvents)

type pubEventStore struct {
	url       string
	conn      *nats.Conn
	publisher messaging.Publisher
	stream    string
}

func NewPublisher(ctx context.Context, url, stream string) (events.Publisher, error) {
	conn, err := nats.Connect(url, nats.MaxReconnects(maxReconnects), nats.ReconnectBufSize(reconnectBufSize))
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	if _, err := js.CreateStream(ctx, jsStreamConfig); err != nil {
		return nil, err
	}

	publisher, err := broker.NewPublisher(ctx, url, broker.Prefix(eventsPrefix), broker.JSStream(js))
	if err != nil {
		return nil, err
	}

	es := &pubEventStore{
		url:       url,
		conn:      conn,
		publisher: publisher,
		stream:    stream,
	}

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
	// Nats doesn't need to check for unpublished events
	// since the events are published to a buffer.
	// The buffer is flushed when the connection is reestablished.
	// https://docs.nats.io/using-nats/developer/connecting/reconnect/buffer

	<-ctx.Done()
}

func (es *pubEventStore) Close() error {
	es.conn.Close()

	return es.publisher.Close()
}
