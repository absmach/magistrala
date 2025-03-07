// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/messaging"
	broker "github.com/absmach/supermq/pkg/messaging/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

var _ events.Subscriber = (*subEventStore)(nil)

var (
	exchangeName = "events"
	eventsPrefix = "events"

	// ErrEmptyStream is returned when stream name is empty.
	ErrEmptyStream = errors.New("stream name cannot be empty")

	// ErrEmptyConsumer is returned when consumer name is empty.
	ErrEmptyConsumer = errors.New("consumer name cannot be empty")
)

type subEventStore struct {
	conn   *amqp.Connection
	pubsub messaging.PubSub
	logger *slog.Logger
}

func NewSubscriber(url string, logger *slog.Logger) (events.Subscriber, error) {
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

	pubsub, err := broker.NewPubSub(url, logger, broker.Channel(ch), broker.Exchange(exchangeName))
	if err != nil {
		return nil, err
	}

	return &subEventStore{
		conn:   conn,
		pubsub: pubsub,
		logger: logger,
	}, nil
}

func (es *subEventStore) Subscribe(ctx context.Context, cfg events.SubscriberConfig) error {
	if cfg.Stream == "" {
		return ErrEmptyStream
	}
	if cfg.Consumer == "" {
		return ErrEmptyConsumer
	}

	subCfg := messaging.SubscriberConfig{
		ID:    cfg.Consumer,
		Topic: cfg.Stream,
		Handler: &eventHandler{
			handler: cfg.Handler,
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
	logger  *slog.Logger
}

func (eh *eventHandler) Handle(msg *messaging.Message) error {
	event := event{
		Data: make(map[string]interface{}),
	}

	if err := json.Unmarshal(msg.GetPayload(), &event.Data); err != nil {
		return err
	}

	if err := eh.handler.Handle(eh.ctx, event); err != nil {
		eh.logger.Warn(fmt.Sprintf("failed to handle rabbitmq event: %s", err))
	}

	return nil
}

func (eh *eventHandler) Cancel() error {
	return nil
}
