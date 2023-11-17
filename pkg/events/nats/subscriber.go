// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/nats"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	maxReconnects = -1
)

var _ events.Subscriber = (*subEventStore)(nil)

var (
	eventsPrefix = "events"

	jsStreamConfig = jetstream.StreamConfig{
		Name:              "events",
		Description:       "Magistrala stream for sending and receiving messages in between Magistrala events",
		Subjects:          []string{"events.>"},
		Retention:         jetstream.LimitsPolicy,
		MaxMsgsPerSubject: 1e9,
		MaxAge:            time.Hour * 24,
		MaxMsgSize:        1024 * 1024,
		Discard:           jetstream.DiscardOld,
		Storage:           jetstream.FileStorage,
	}

	// ErrEmptyStream is returned when stream name is empty.
	ErrEmptyStream = errors.New("stream name cannot be empty")

	// ErrEmptyConsumer is returned when consumer name is empty.
	ErrEmptyConsumer = errors.New("consumer name cannot be empty")
)

type subEventStore struct {
	conn     *nats.Conn
	pubsub   messaging.PubSub
	stream   string
	consumer string
	logger   mglog.Logger
}

func NewSubscriber(ctx context.Context, url, stream, consumer string, logger mglog.Logger) (events.Subscriber, error) {
	if stream == "" {
		return nil, ErrEmptyStream
	}

	if consumer == "" {
		return nil, ErrEmptyConsumer
	}

	conn, err := nats.Connect(url, nats.MaxReconnects(maxReconnects))
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	jsStream, err := js.CreateStream(ctx, jsStreamConfig)
	if err != nil {
		return nil, err
	}

	pubsub, err := broker.NewPubSub(ctx, url, logger, broker.Stream(jsStream))
	if err != nil {
		return nil, err
	}

	return &subEventStore{
		conn:     conn,
		pubsub:   pubsub,
		stream:   stream,
		consumer: consumer,
		logger:   logger,
	}, nil
}

func (es *subEventStore) Subscribe(ctx context.Context, handler events.EventHandler) error {
	subCfg := messaging.SubscriberConfig{
		ID:    es.consumer,
		Topic: eventsPrefix + "." + es.stream,
		Handler: &eventHandler{
			handler: handler,
			ctx:     ctx,
			logger:  es.logger,
		},
		DeliveryPolicy: messaging.DeliverNewPolicy,
	}

	return es.pubsub.Subscribe(ctx, subCfg)
}

func (es *subEventStore) Close() error {
	es.conn.Close()
	return es.pubsub.Close()
}

type event struct {
	Data map[string]interface{}
}

func (re event) Encode() (map[string]interface{}, error) {
	return re.Data, nil
}

type eventHandler struct {
	handler events.EventHandler
	ctx     context.Context
	logger  mglog.Logger
}

func (eh *eventHandler) Handle(msg *messaging.Message) error {
	event := event{
		Data: make(map[string]interface{}),
	}

	if err := json.Unmarshal(msg.GetPayload(), &event.Data); err != nil {
		return err
	}

	if err := eh.handler.Handle(eh.ctx, event); err != nil {
		eh.logger.Warn(fmt.Sprintf("failed to handle redis event: %s", err))
	}

	return nil
}

func (eh *eventHandler) Cancel() error {
	return nil
}
