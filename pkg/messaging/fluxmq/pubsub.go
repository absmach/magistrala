// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
			ps.logInfo("FluxMQ message pub/sub connected", "prefix", ps.prefix)
		})
	if ps.tlsConfig != nil {
		amqpOpts = amqpOpts.SetTLSConfig(ps.tlsConfig)
	}

	client, err := fluxamqp.New(amqpOpts)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if !ps.preprovisioned {
		if err := declareStream(client, ps.prefix); err != nil {
			_ = client.Close()
			return nil, err
		}
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
	sub := subscription{}

	if !ps.directTopicOnly {
		opts := streamConsumeOptions(ps.prefix, group, cfg)

		if err := ps.client.SubscribeToStream(opts, func(msg *fluxamqp.QueueMessage) {
			if err := ps.handle(cfg.Handler, msg); err != nil {
				ps.logWarn("failed to process FluxMQ stream message", "error", err, "topic", cfg.Topic, "consumer_group", group)
			}
		}); err != nil {
			return err
		}

		sub.streamTopic = queueFilter(ps.prefix, cfg.Topic)
	}
	// A preprovisioned connection is a local principal on the mTLS service
	// listener, and that listener only serves queue addresses: FluxMQ resolves
	// a bare topic filter to a pub/sub route and refuses it, because no
	// subscribe ACL entry can name one. The direct subscription is also
	// redundant there -- the broker-provisioned stream binds the same topic
	// patterns, so a direct publish reaches the stream consumer anyway.
	if ps.directTopicIngress && !ps.preprovisioned {
		// Subscribe to regular MQTT topics so that messages published directly
		// by MQTT clients (not through the stream queue) are also received.
		sub.mqttTopic = topicFilter(ps.prefix, cfg.Topic)
		if err := ps.client.Subscribe(sub.mqttTopic, func(msg *fluxamqp.Message) {
			if err := ps.handleTopicMessage(cfg.Handler, msg); err != nil {
				ps.logWarn("failed to process FluxMQ topic message", "error", err, "topic", sub.mqttTopic)
			}
		}); err != nil {
			if sub.streamTopic != "" {
				_ = ps.client.UnsubscribeFromStream(sub.streamTopic)
			}

			return err
		}
	}

	ps.mu.Lock()
	ps.subscriptions[subscriptionKey(cfg.ID, cfg.Topic)] = sub
	ps.mu.Unlock()
	return nil
}

func streamConsumeOptions(prefix, group string, cfg messaging.SubscriberConfig) *fluxamqp.StreamConsumeOptions {
	opts := &fluxamqp.StreamConsumeOptions{
		QueueName:     prefix,
		Filter:        streamFilter(prefix, cfg.Topic),
		ConsumerGroup: group,
	}
	if cfg.AckPolicy == messaging.AckExplicit {
		autoCommit := false
		opts.AutoCommit = &autoCommit
	}

	switch cfg.DeliveryPolicy {
	case messaging.DeliverNewPolicy:
		opts.Offset = "last"
	case messaging.DeliverAllPolicy:
		opts.Offset = "first"
	}
	return opts
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

	var streamErr error
	if sub.streamTopic != "" {
		streamErr = ps.client.UnsubscribeFromStream(sub.streamTopic)
	}
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
		ps.logWarn("failed to handle message",
			"channel", m.Channel,
			"workspace", m.Workspace,
			"subtopic", m.Subtopic,
			"publisher", m.Publisher,
			"error", handleErr,
		)
	}

	if ackErr := ps.handleAck(ackType, msg); ackErr != nil {
		return fmt.Errorf("failed to %s message: %w", ackType.String(), ackErr)
	}

	return nil
}

func messageFromDelivery(body []byte, headers map[string]any, ts time.Time, prefix, mqttTopic string) (*messaging.Message, error) {
	workspace, channel, subtopic, err := parseMQTTTopic(prefix, mqttTopic)
	if err != nil {
		return nil, err
	}

	clientID := stringHeader(headers, "client_id")
	publisher := stringHeader(headers, headerExternalID)
	deviceID := stringHeader(headers, headerDeviceID)

	protocol := stringHeader(headers, headerProtocol)
	if protocol == "" {
		protocol = protocolMQTT
	}

	created := time.Now().UnixNano()
	if !ts.IsZero() {
		created = ts.UnixNano()
	}
	if v, ok := int64Header(headers, "created"); ok {
		created = v
	}

	// Allocated lazily: this runs for every delivered message, and carrying
	// metadata is the exception rather than the rule.
	var metadata map[string]string
	for key, value := range headers {
		if key == headerDeviceID {
			// Already extracted above into deviceID — it shares the
			// metadata namespace for broker-side forgery protection, not
			// because it is user metadata.
			continue
		}
		metadataKey, ok := strings.CutPrefix(key, headerMetadataPrefix)
		if !ok || metadataKey == "" {
			continue
		}
		metadataValue, ok := value.(string)
		if !ok {
			continue
		}
		if metadata == nil {
			metadata = make(map[string]string)
		}
		metadata[metadataKey] = metadataValue
	}

	return &messaging.Message{
		Workspace: workspace,
		Channel:   channel,
		Subtopic:  subtopic,
		Payload:   body,
		Publisher: publisher,
		ClientId:  clientID,
		DeviceId:  deviceID,
		Protocol:  protocol,
		Created:   created,
		Metadata:  metadata,
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
	case messaging.Nack:
		return msg.Nack()
	case messaging.InProgress:
		// FluxMQ has no delivery lease to extend. Leaving the delivery
		// outstanding preserves the only safe meaning of InProgress.
		return nil
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
