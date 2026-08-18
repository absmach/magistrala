// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/magistrala/pkg/events"
)

var _ events.Subscriber = (*subQueueStore)(nil)

var (
	// ErrEmptyExchange is returned when the exchange name is empty.
	ErrEmptyExchange = errors.New("exchange name cannot be empty")
	// ErrEmptyRoutingKey is returned when the routing key is empty.
	ErrEmptyRoutingKey = errors.New("routing key cannot be empty")
	// ErrNilHandler is returned when no event handler is configured.
	ErrNilHandler = errors.New("event handler cannot be nil")
)

// QueueSubscriberConfig names the AMQP topology NewQueueSubscriber binds
// against: a durable topic exchange and the one fixed routing key every
// message on it carries. Atom's outbox publisher (absmach/atom
// src/events/publisher.rs) publishes every domain event to one such
// (exchange, routing_key) pair regardless of event type -- it never routes
// per event type, so consumers tell events apart from the payload's own
// "event" field, not from AMQP routing (see pkg/atom/events).
type QueueSubscriberConfig struct {
	Exchange   string
	RoutingKey string
}

type subQueueStore struct {
	client     *fluxamqp.Client
	exchange   string
	routingKey string
	logger     *slog.Logger
}

// NewQueueSubscriber creates an AMQP subscriber that binds a durable queue
// directly to a topic exchange and consumes it with plain AMQP 0-9-1
// semantics (basic.consume), rather than FluxMQ's Stream Queue extension
// used by NewSubscriber. That extension requires the publisher to speak
// FluxMQ's own "$queue/..." naming convention; a foreign AMQP client such as
// Atom's does not, so this constructor exists to consume exactly what such a
// publisher produces.
//
// Subscribe treats cfg.Stream as the name of the queue to declare (durable,
// so it survives a restart and so Atom's publish is routable even before any
// consumer has started) and bind to (exchange, routingKey). Callers that
// want their own full, independent copy of every event -- rather than
// competing with other consumers for one shared queue -- should each pass a
// distinct queue name.
func NewQueueSubscriber(_ context.Context, url, connectionName string, topology QueueSubscriberConfig, logger *slog.Logger) (events.Subscriber, error) {
	if topology.Exchange == "" {
		return nil, ErrEmptyExchange
	}
	if topology.RoutingKey == "" {
		return nil, ErrEmptyRoutingKey
	}

	opts := fluxamqp.NewOptions().SetURL(url).
		SetConnectionName(connectionName).
		SetOnConnectionLost(func(err error) {
			logger.Warn("Atom events AMQP subscriber connection lost", "error", err)
		}).
		SetOnReconnecting(func(attempt int) {
			logger.Info("Atom events AMQP subscriber reconnecting", "attempt", attempt)
		}).
		SetOnConnect(func() {
			logger.Info("Atom events AMQP subscriber connected", "url", url, "connection", connectionName)
		})

	client, err := fluxamqp.New(opts)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}

	// Durable and idempotent: whichever side -- Atom or this subscriber --
	// declares the exchange first creates it, and the other's identical
	// declare is a no-op. Atom only declares it when its own
	// ATOM_EVENTS_AMQP_EXCHANGE is non-empty (see the module doc in
	// absmach/atom src/events/publisher.rs); declaring it here too means
	// this subscriber works regardless of which side starts first.
	if err := client.DeclareExchange(&fluxamqp.ExchangeDeclareOptions{
		Name:    topology.Exchange,
		Kind:    "topic",
		Durable: true,
	}); err != nil {
		return nil, err
	}

	return &subQueueStore{
		client:     client,
		exchange:   topology.Exchange,
		routingKey: topology.RoutingKey,
		logger:     logger,
	}, nil
}

func (es *subQueueStore) Subscribe(ctx context.Context, cfg events.SubscriberConfig) error {
	if cfg.Stream == "" {
		return ErrEmptyStream
	}
	if cfg.Handler == nil {
		return ErrNilHandler
	}

	if _, err := es.client.DeclareQueue(&fluxamqp.QueueDeclareOptions{
		Name:    cfg.Stream,
		Durable: true,
	}); err != nil {
		return err
	}
	if err := es.client.BindQueue(cfg.Stream, es.routingKey, es.exchange, false, nil); err != nil {
		return err
	}

	return es.client.SubscribeWithOptions(&fluxamqp.SubscribeOptions{
		Topic:       cfg.Stream,
		AutoAck:     false,
		ConsumerTag: cfg.Consumer,
	}, func(msg *fluxamqp.Message) {
		if err := es.handle(ctx, cfg.Handler, msg); err != nil {
			es.logWarn("failed to process Atom event", "error", err)
		}
	})
}

func (es *subQueueStore) Close() error {
	return es.client.Close()
}

func (es *subQueueStore) handle(ctx context.Context, handler events.EventHandler, msg *fluxamqp.Message) error {
	ev := event{
		Data: make(map[string]any),
	}

	if err := json.Unmarshal(msg.Body, &ev.Data); err != nil {
		if rejectErr := msg.Reject(); rejectErr != nil {
			return errors.Join(err, rejectErr)
		}
		return err
	}

	if err := handler.Handle(ctx, ev); err != nil {
		if nackErr := msg.Nack(); nackErr != nil {
			return errors.Join(fmt.Errorf("failed to handle Atom event: %w", err), nackErr)
		}
		return fmt.Errorf("failed to handle Atom event: %w", err)
	}

	return msg.Ack()
}

func (es *subQueueStore) logWarn(msg string, args ...any) {
	if es.logger != nil {
		es.logger.Warn(msg, args...)
		return
	}

	slog.Warn(msg, args...)
}
