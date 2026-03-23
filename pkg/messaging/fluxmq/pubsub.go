// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/supermq/pkg/messaging"
	"google.golang.org/protobuf/proto"
)

// Publisher and Subscriber errors.
var (
	ErrNotSubscribed = errors.New("not subscribed")
	ErrEmptyTopic    = errors.New("empty topic")
	ErrEmptyID       = errors.New("empty id")
)

var _ messaging.PubSub = (*pubsub)(nil)

type pubsub struct {
	publisher
	logger *slog.Logger

	mu            sync.Mutex
	subscriptions map[string]string
}

// NewPubSub creates a FluxMQ-backed message publisher/subscriber.
func NewPubSub(_ context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	ps := &pubsub{
		publisher: publisher{
			options: defaultOptions(),
		},
		logger:        logger,
		subscriptions: make(map[string]string),
	}

	for _, opt := range opts {
		if err := opt(ps); err != nil {
			return nil, err
		}
	}

	amqpOpts := fluxamqp.NewOptions().SetURL(url).
		SetOnConnectionLost(func(err error) {
			ps.logWarn("FluxMQ message pub/sub connection lost", "error", err)
		}).
		SetOnReconnecting(func(attempt int) {
			ps.logInfo("FluxMQ message pub/sub reconnecting", "attempt", attempt)
		}).
		SetOnConnect(func() {
			ps.logInfo("FluxMQ message pub/sub connected")
		})

	client, err := fluxamqp.New(amqpOpts)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if err := declareStream(client, ps.prefix); err != nil {
		_ = client.Close()
		return nil, err
	}

	ps.client = client

	return ps, nil
}

func (ps *pubsub) Subscribe(_ context.Context, cfg messaging.SubscriberConfig) error {
	if cfg.ID == "" {
		return ErrEmptyID
	}
	if cfg.Topic == "" {
		return ErrEmptyTopic
	}

	group := formatConsumerName(cfg.Topic, cfg.ID)
	opts := &fluxamqp.StreamConsumeOptions{
		QueueName:     streamQueue(ps.prefix),
		Filter:        streamFilter(ps.prefix, cfg.Topic),
		ConsumerGroup: group,
	}

	switch cfg.DeliveryPolicy {
	case messaging.DeliverNewPolicy:
		opts.Offset = "last"
	case messaging.DeliverAllPolicy:
		opts.Offset = "first"
	}

	if err := ps.client.SubscribeToStream(opts, func(msg *fluxamqp.QueueMessage) {
		if err := ps.handle(cfg.Handler, msg); err != nil {
			ps.logWarn("failed to process FluxMQ message", "error", err, "topic", cfg.Topic, "consumer_group", group)
		}
	}); err != nil {
		return err
	}

	ps.mu.Lock()
	ps.subscriptions[subscriptionKey(cfg.ID, cfg.Topic)] = queueFilter(ps.prefix, cfg.Topic)
	ps.mu.Unlock()

	return nil
}

func (ps *pubsub) Unsubscribe(_ context.Context, id, topic string) error {
	if id == "" {
		return ErrEmptyID
	}
	if topic == "" {
		return ErrEmptyTopic
	}

	key := subscriptionKey(id, topic)

	ps.mu.Lock()
	streamTopic, ok := ps.subscriptions[key]
	ps.mu.Unlock()
	if !ok {
		return ErrNotSubscribed
	}

	if err := ps.client.UnsubscribeFromStream(streamTopic); err != nil {
		return err
	}

	ps.mu.Lock()
	delete(ps.subscriptions, key)
	ps.mu.Unlock()

	return nil
}

func (ps *pubsub) handle(h messaging.MessageHandler, msg *fluxamqp.QueueMessage) error {
	var m messaging.Message
	if err := proto.Unmarshal(msg.Body, &m); err != nil {
		if rejectErr := msg.Reject(); rejectErr != nil {
			return errors.Join(err, rejectErr)
		}
		return err
	}

	err := h.Handle(&m)
	ackType := ps.errAckType(err)
	if err != nil {
		ps.logWarn("failed to handle message", "ack_type", ackType.String(), "error", err)
	}

	if ackErr := ps.handleAck(ackType, msg); ackErr != nil {
		return fmt.Errorf("failed to %s message: %w", ackType.String(), ackErr)
	}

	return nil
}

func (ps *pubsub) errAckType(err error) messaging.AckType {
	if err == nil {
		return messaging.Ack
	}
	if e, ok := err.(messaging.Error); ok && e != nil {
		return e.Ack()
	}
	return messaging.NoAck
}

func (ps *pubsub) handleAck(at messaging.AckType, msg *fluxamqp.QueueMessage) error {
	switch at {
	case messaging.Ack, messaging.DoubleAck:
		return msg.Ack()
	case messaging.Nack, messaging.InProgress:
		return msg.Nack()
	case messaging.Term:
		return msg.Reject()
	case messaging.NoAck:
		return nil
	default:
		return nil
	}
}

func (ps *pubsub) logInfo(msg string, args ...any) {
	if ps.logger != nil {
		ps.logger.Info(msg, args...)
		return
	}

	slog.Info(msg, args...)
}

func (ps *pubsub) logWarn(msg string, args ...any) {
	if ps.logger != nil {
		ps.logger.Warn(msg, args...)
		return
	}

	slog.Warn(msg, args...)
}

func (ps *pubsub) Close() error {
	return ps.client.Close()
}

func subscriptionKey(id, topic string) string {
	return fmt.Sprintf("%s|%s", id, topic)
}
