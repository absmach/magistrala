// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

const chansPrefix = "channels"

// Publisher and Subscriber errors.
var (
	ErrNotSubscribed = errors.New("not subscribed")
	ErrEmptyTopic    = errors.New("empty topic")
	ErrEmptyID       = errors.New("empty id")

	jsStreamConfig = jetstream.StreamConfig{
		Name:              "channels",
		Description:       "Magistrala stream for sending and receiving messages in between Magistrala channels",
		Subjects:          []string{"channels.>"},
		Retention:         jetstream.LimitsPolicy,
		MaxMsgsPerSubject: 1e6,
		MaxAge:            time.Hour * 24,
		MaxMsgSize:        1024 * 1024,
		Discard:           jetstream.DiscardOld,
		Storage:           jetstream.FileStorage,
	}
)

var _ messaging.PubSub = (*pubsub)(nil)

type pubsub struct {
	publisher
	logger mglog.Logger
	stream jetstream.Stream
}

// NewPubSub returns NATS message publisher/subscriber.
// Parameter queue specifies the queue for the Subscribe method.
// If queue is specified (is not an empty string), Subscribe method
// will execute NATS QueueSubscribe which is conceptually different
// from ordinary subscribe. For more information, please take a look
// here: https://docs.nats.io/developing-with-nats/receiving/queues.
// If the queue is empty, Subscribe will be used.
func NewPubSub(ctx context.Context, url string, logger mglog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	conn, err := broker.Connect(url, broker.MaxReconnects(maxReconnects))
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	stream, err := js.CreateStream(ctx, jsStreamConfig)
	if err != nil {
		return nil, err
	}

	ret := &pubsub{
		publisher: publisher{
			js:     js,
			conn:   conn,
			prefix: chansPrefix,
		},
		stream: stream,
		logger: logger,
	}

	for _, opt := range opts {
		if err := opt(ret); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (ps *pubsub) Subscribe(ctx context.Context, cfg messaging.SubscriberConfig) error {
	if cfg.ID == "" {
		return ErrEmptyID
	}
	if cfg.Topic == "" {
		return ErrEmptyTopic
	}

	nh := ps.natsHandler(cfg.Handler)

	consumerConfig := jetstream.ConsumerConfig{
		Name:          formatConsumerName(cfg.Topic, cfg.ID),
		Durable:       formatConsumerName(cfg.Topic, cfg.ID),
		Description:   fmt.Sprintf("Magistrala consumer of id %s for cfg.Topic %s", cfg.ID, cfg.Topic),
		DeliverPolicy: jetstream.DeliverNewPolicy,
		FilterSubject: cfg.Topic,
	}

	switch cfg.DeliveryPolicy {
	case messaging.DeliverNewPolicy:
		consumerConfig.DeliverPolicy = jetstream.DeliverNewPolicy
	case messaging.DeliverAllPolicy:
		consumerConfig.DeliverPolicy = jetstream.DeliverAllPolicy
	}

	consumer, err := ps.stream.CreateOrUpdateConsumer(ctx, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	if _, err = consumer.Consume(nh); err != nil {
		return fmt.Errorf("failed to consume: %w", err)
	}

	return nil
}

func (ps *pubsub) Unsubscribe(ctx context.Context, id, topic string) error {
	if id == "" {
		return ErrEmptyID
	}
	if topic == "" {
		return ErrEmptyTopic
	}

	err := ps.stream.DeleteConsumer(ctx, formatConsumerName(topic, id))
	switch {
	case errors.Is(err, jetstream.ErrConsumerNotFound):
		return ErrNotSubscribed
	default:
		return err
	}
}

func (ps *pubsub) natsHandler(h messaging.MessageHandler) func(m jetstream.Msg) {
	return func(m jetstream.Msg) {
		var msg messaging.Message
		if err := proto.Unmarshal(m.Data(), &msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))

			return
		}

		if err := h.Handle(&msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to handle Magistrala message: %s", err))
		}
		if err := m.Ack(); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to ack message: %s", err))
		}
	}
}

func formatConsumerName(topic, id string) string {
	// A durable name cannot contain whitespace, ., *, >, path separators (forward or backwards slash), and non-printable characters.
	chars := []string{
		" ", "_",
		".", "_",
		"*", "_",
		">", "_",
		"/", "_",
		"\\", "_",
	}
	topic = strings.NewReplacer(chars...).Replace(topic)

	return fmt.Sprintf("%s-%s", topic, id)
}
