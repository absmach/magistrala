// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	fluxtopics "github.com/absmach/fluxmq/topics"
	"github.com/absmach/magistrala/pkg/messaging"
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
	subscriptions map[string]subscription
}

type subscription struct {
	streamTopic string
	mqttTopic   string
}

// NewPubSub creates a FluxMQ-backed message publisher/subscriber.
func NewPubSub(_ context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	ps := &pubsub{
		publisher: publisher{
			options: defaultOptions(),
		},
		logger:        logger,
		subscriptions: make(map[string]subscription),
	}

	for _, opt := range opts {
		if err := opt(ps); err != nil {
			return nil, err
		}
	}

	amqpOpts := fluxamqp.NewOptions().SetURL(url).
		SetConnectionName(ps.connectionName).
		SetOnConnectionLost(func(err error) {
			ps.logWarn("FluxMQ message pub/sub connection lost", "error", err)
		}).
		SetOnReconnecting(func(attempt int) {
			ps.logInfo("FluxMQ message pub/sub reconnecting", "attempt", attempt)
		}).
		SetOnConnect(func() {
			ps.logInfo("FluxMQ message pub/sub connected", url, ps.prefix)
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
		QueueName:     ps.prefix,
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
			ps.logWarn("failed to process FluxMQ stream message", "error", err, "topic", cfg.Topic, "consumer_group", group)
		}
	}); err != nil {
		return err
	}

	sub := subscription{
		streamTopic: queueFilter(ps.prefix, cfg.Topic),
	}
	if ps.directTopicIngress {
		// Subscribe to regular MQTT topics so that messages published directly
		// by MQTT clients (not through the stream queue) are also received.
		sub.mqttTopic = topicFilter(ps.prefix, cfg.Topic)
		if err := ps.client.Subscribe(sub.mqttTopic, func(msg *fluxamqp.Message) {
			if err := ps.handleTopicMessage(cfg.Handler, msg); err != nil {
				ps.logWarn("failed to process FluxMQ topic message", "error", err, "topic", sub.mqttTopic)
			}
		}); err != nil {
			_ = ps.client.UnsubscribeFromStream(sub.streamTopic)

			return err
		}
	}

	ps.mu.Lock()
	ps.subscriptions[subscriptionKey(cfg.ID, cfg.Topic)] = sub
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
	sub, ok := ps.subscriptions[key]
	ps.mu.Unlock()
	if !ok {
		return ErrNotSubscribed
	}

	streamErr := ps.client.UnsubscribeFromStream(sub.streamTopic)
	var topicErr error
	if sub.mqttTopic != "" {
		topicErr = ps.client.Unsubscribe(sub.mqttTopic)
	}

	ps.mu.Lock()
	delete(ps.subscriptions, key)
	ps.mu.Unlock()

	return errors.Join(streamErr, topicErr)
}

func (ps *pubsub) handleTopicMessage(h messaging.MessageHandler, msg *fluxamqp.Message) error {
	mqttTopic := fluxtopics.AMQPTopicToMQTT(msg.Topic)
	m, err := messageFromDelivery(msg.Body, msg.Headers, msg.Timestamp, ps.prefix, mqttTopic)
	if err != nil {
		return fmt.Errorf("failed to parse MQTT topic %q: %w", msg.Topic, err)
	}

	if err := h.Handle(m); err != nil {
		ps.logWarn("failed to handle topic message", "error", err)
	}

	return nil
}

func (ps *pubsub) handle(h messaging.MessageHandler, msg *fluxamqp.QueueMessage) error {
	mqttTopic := strings.TrimPrefix(msg.RoutingKey, queuePrefix)
	m, err := messageFromDelivery(msg.Body, msg.Headers, msg.Timestamp, ps.prefix, mqttTopic)
	if err != nil {
		if rejectErr := msg.Reject(); rejectErr != nil {
			return errors.Join(err, rejectErr)
		}
		return err
	}

	handleErr := h.Handle(m)
	ackType := ps.errAckType(handleErr)
	if handleErr != nil {
		ps.logWarn("failed to handle message", "ack_type", ackType.String(), "error", handleErr)
	}

	if ackErr := ps.handleAck(ackType, msg); ackErr != nil {
		return fmt.Errorf("failed to %s message: %w", ackType.String(), ackErr)
	}

	return nil
}

func messageFromDelivery(body []byte, headers map[string]any, ts time.Time, prefix, mqttTopic string) (*messaging.Message, error) {
	domain, channel, subtopic, err := parseMQTTTopic(prefix, mqttTopic)
	if err != nil {
		return nil, err
	}

	clientID := stringHeader(headers, "client_id")
	publisher := stringHeader(headers, "external_id")

	protocol := stringHeader(headers, "protocol")
	if protocol == "" {
		protocol = "mqtt"
	}

	created := ts.UnixNano()
	if s := stringHeader(headers, "created"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			created = v
		}
	}

	return &messaging.Message{
		Domain:    domain,
		Channel:   channel,
		Subtopic:  subtopic,
		Payload:   body,
		Publisher: publisher,
		ClientId:  clientID,
		Protocol:  protocol,
		Created:   created,
	}, nil
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
