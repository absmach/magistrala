// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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
	stream jetstream.Stream
}

// NewPubSub returns NATS message publisher/subscriber.
// Parameter queue specifies the queue for the Subscribe method.
// If queue is specified (is not an empty string), Subscribe method
// will execute NATS QueueSubscribe which is conceptually different
// from ordinary subscribe. For more information, please take a look
// here: https://docs.nats.io/developing-with-nats/receiving/queues.
// If the queue is empty, Subscribe will be used.
func NewPubSub(ctx context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	ps := &pubsub{
		publisher: publisher{
			options: defaultOptions(),
		},
		logger: logger,
	}

	for _, opt := range opts {
		if err := opt(ps); err != nil {
			return nil, err
		}
	}

	conn, err := broker.Connect(url, broker.MaxReconnects(maxReconnects))
	if err != nil {
		return nil, err
	}
	ps.conn = conn

	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}
	stream, err := js.CreateStream(ctx, ps.jsStreamConfig)
	if err != nil {
		return nil, err
	}
	ps.js = js
	ps.stream = stream

	return ps, nil
}

func (ps *pubsub) Subscribe(ctx context.Context, cfg messaging.SubscriberConfig) error {
	if cfg.ID == "" {
		return ErrEmptyID
	}
	if cfg.Topic == "" {
		return ErrEmptyTopic
	}

	// nolint:contextcheck
	nh := ps.natsHandler(cfg.Handler)

	natsTopic := toNATSTopic(cfg.Topic)
	consumerConfig := jetstream.ConsumerConfig{
		Name:          formatConsumerName(cfg.Topic, cfg.ID),
		Durable:       formatConsumerName(cfg.Topic, cfg.ID),
		Description:   fmt.Sprintf("Magistrala consumer of id %s for cfg.Topic %s", cfg.ID, cfg.Topic),
		DeliverPolicy: jetstream.DeliverNewPolicy,
		FilterSubject: natsTopic,
	}

	if cfg.Ordered {
		consumerConfig.MaxAckPending = 1
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
		args := []any{
			slog.String("subject", m.Subject()),
		}
		meta, err := m.Metadata()
		switch err {
		case nil:
			args = append(args,
				slog.String("stream", meta.Stream),
				slog.String("consumer", meta.Consumer),
				slog.Uint64("stream_seq", meta.Sequence.Stream),
				slog.Uint64("consumer_seq", meta.Sequence.Consumer),
			)
		default:
			args = append(args,
				slog.String("metadata_error", err.Error()),
			)
		}

		var msg messaging.Message
		if err := proto.Unmarshal(m.Data(), &msg); err != nil {
			ackType := messaging.Term
			args = append(args, slog.String("ack_type", ackType.String()), slog.String("error", err.Error()))
			ps.logger.Warn("failed to unmarshal message", args...)
			ps.handleAck(ackType, m)
			return
		}

		err = h.Handle(&msg)
		ackType := ps.errAckType(err)
		if err != nil {
			args = append(args, slog.String("ack_type", ackType.String()), slog.String("error", err.Error()))
			ps.logger.Warn("failed to handle message", args...)
		}
		ps.handleAck(ackType, m)
	}
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

func (ps *pubsub) handleAck(at messaging.AckType, m jetstream.Msg) {
	switch at {
	case messaging.Ack:
		if err := m.Ack(); err != nil {
			ps.logger.Warn(fmt.Sprintf("failed to ack message: %s", err))
		}
	case messaging.DoubleAck:
		if err := m.DoubleAck(context.Background()); err != nil {
			ps.logger.Warn(fmt.Sprintf("failed to double ack message: %s", err))
		}
	case messaging.Nack:
		if err := m.Nak(); err != nil {
			ps.logger.Warn(fmt.Sprintf("failed to negatively ack message: %s", err))
		}
	case messaging.InProgress:
		if err := m.InProgress(); err != nil {
			ps.logger.Warn(fmt.Sprintf("failed to set message in progress: %s", err))
		}
	case messaging.Term:
		if err := m.Term(); err != nil {
			ps.logger.Warn(fmt.Sprintf("failed to terminate message: %s", err))
		}
	}
}

// toNATSTopic converts a canonical /-separated topic with MQTT wildcards
// to NATS format using . separators and NATS wildcards.
var natsTopicReplacer = strings.NewReplacer("/", ".", "+", "*", "#", ">")

func toNATSTopic(topic string) string {
	return natsTopicReplacer.Replace(topic)
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
		"+", "_",
		"#", "_",
	}
	topic = strings.NewReplacer(chars...).Replace(topic)

	return fmt.Sprintf("%s-%s", topic, id)
}
