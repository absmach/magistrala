// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	fluxtopics "github.com/absmach/fluxmq/topics"
	"github.com/absmach/magistrala/pkg/messaging"
)

var _ messaging.Publisher = (*publisher)(nil)

const (
	headerExternalID = "external_id"
	headerProtocol   = "protocol"
	// headerMetadataPrefix sits inside FluxMQ's reserved "_flux." namespace, so
	// the broker treats message metadata as internal state passed between
	// first-party services: it is dropped when an untrusted connection sets it
	// and omitted when one subscribes. A service therefore cannot be fed forged
	// metadata by a device, and a device cannot read metadata a service set.
	headerMetadataPrefix = "_flux.mg."
	protocolMQTT         = "mqtt"
)

type publisher struct {
	client *fluxamqp.Client
	options
}

// NewPublisher creates a FluxMQ-backed message publisher.
func NewPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	return newPublisher(ctx, url, true, opts...)
}

// NewUndeclaredPublisher creates a FluxMQ-backed publisher without declaring
// the stream queue. Use it only when another service owns queue declaration.
func NewUndeclaredPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	return newPublisher(ctx, url, false, opts...)
}

func newPublisher(_ context.Context, url string, declare bool, opts ...messaging.Option) (messaging.Publisher, error) {
	pub := &publisher{
		options: defaultOptions(),
	}

	for _, opt := range opts {
		if err := opt(pub); err != nil {
			return nil, err
		}
	}

	logger := slog.Default()
	amqpOpts := fluxamqp.NewOptions().SetURL(url).
		SetConnectionName(pub.connectionName).
		SetOnConnectionLost(func(err error) {
			logger.Warn("FluxMQ message publisher connection lost", "error", err)
		}).
		SetOnReconnecting(func(attempt int) {
			logger.Info("FluxMQ message publisher reconnecting", "attempt", attempt)
		}).
		SetOnConnect(func() {
			logger.Info("FluxMQ message publisher connected")
		})
	if pub.tlsConfig != nil {
		amqpOpts = amqpOpts.SetTLSConfig(pub.tlsConfig)
	}

	client, err := fluxamqp.New(amqpOpts)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if declare && !pub.preprovisioned {
		if err := declareStream(client, pub.prefix); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	pub.client = client

	return pub, nil
}

func (pub *publisher) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	props := messageProperties(msg)

	cleanTopic := strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if cleanTopic == "" {
		return ErrEmptyTopic
	}

	// $queue/-prefixed topics are routed to the durable stream queue.
	if queueName, ok := strings.CutPrefix(cleanTopic, queuePrefix); ok {
		return pub.client.PublishToQueueWithOptionsContext(ctx, &fluxamqp.QueuePublishOptions{
			QueueName:  queueName,
			Payload:    msg.GetPayload(),
			Properties: props,
		})
	}

	// Normalize to "prefix/subTopic" — strip any existing prefix to avoid doubling.
	subTopic := strings.TrimPrefix(cleanTopic, pub.prefix+"/")
	publishTopic := pub.prefix + "/" + subTopic

	// Non-default prefix publishers (e.g. "writers", "alarms") are stream-backed:
	// route to the durable queue so stream subscribers receive the message.
	if pub.prefix != msgPrefix {
		return pub.client.PublishToQueueWithOptionsContext(ctx, &fluxamqp.QueuePublishOptions{
			QueueName:  publishTopic,
			Payload:    msg.GetPayload(),
			Properties: props,
		})
	}
	if pub.preprovisioned {
		publishTopic = mqttTopicToWireTopic(publishTopic)
	}

	return pub.client.PublishWithOptionsContext(ctx, &fluxamqp.PublishOptions{
		Topic:      publishTopic,
		Payload:    msg.GetPayload(),
		Properties: props,
	})
}

func mqttTopicToWireTopic(topic string) string {
	// AMQP uses dots as segment separators, while MQTT permits dots as ordinary
	// topic characters. Escape literal dots before translating slashes so the
	// subscriber can URL-decode them back instead of treating them as slashes.
	topic = strings.ReplaceAll(topic, ".", "%2E")
	return fluxtopics.MQTTTopicToAMQP(topic)
}

func messageProperties(msg *messaging.Message) map[string]string {
	props := map[string]string{
		headerExternalID: msg.GetPublisher(),
		headerProtocol:   msg.GetProtocol(),
	}
	if clientID := msg.ClientIdentity(); clientID != "" {
		props["client_id"] = clientID
	}
	if msg.GetCreated() != 0 {
		props["created"] = strconv.FormatInt(msg.GetCreated(), 10)
	}
	for key, value := range msg.GetMetadata() {
		props[headerMetadataPrefix+key] = value
	}

	return props
}

func (pub *publisher) Close() error {
	return pub.client.Close()
}
