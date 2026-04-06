// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/nats"
	"github.com/nats-io/nats.go/jetstream"
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
	pubsub messaging.PubSub
}

func NewSubscriber(ctx context.Context, url string, logger *slog.Logger) (events.Subscriber, error) {
	pubsub, err := broker.NewPubSub(ctx, url, logger, broker.JSStreamConfig(jsStreamConfig))
	if err != nil {
		return nil, err
	}

	return &subEventStore{
		pubsub: pubsub,
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
		},
		DeliveryPolicy: cfg.DeliveryPolicy,
		Ordered:        cfg.Ordered,
	}

	return es.pubsub.Subscribe(ctx, subCfg)
}

func (es *subEventStore) Close() error {
	return es.pubsub.Close()
}

type event struct {
	Data map[string]any
}

func (re event) Encode() (map[string]any, error) {
	return re.Data, nil
}

type eventHandler struct {
	handler events.EventHandler
	ctx     context.Context
}

func (eh *eventHandler) Handle(msg *messaging.Message) error {
	event := event{
		Data: make(map[string]any),
	}

	if err := json.Unmarshal(msg.GetPayload(), &event.Data); err != nil {
		return err
	}

	err := eh.handler.Handle(eh.ctx, event)
	if err != nil {
		return fmt.Errorf("failed to handle nats event: %s", err)
	}

	return nil
}

func (eh *eventHandler) Cancel() error {
	return nil
}
