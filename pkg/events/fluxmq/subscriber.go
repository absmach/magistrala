// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"

	fluxamqp "github.com/absmach/fluxmq/client/amqp"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/messaging"
)

var _ events.Subscriber = (*subEventStore)(nil)

var (
	// ErrEmptyStream is returned when stream name is empty.
	ErrEmptyStream = errors.New("stream name cannot be empty")
	// ErrEmptyConsumer is returned when consumer name is empty.
	ErrEmptyConsumer = errors.New("consumer name cannot be empty")
	// ErrMissingStreamOffset is returned when a stream delivery does not include an offset.
	ErrMissingStreamOffset = errors.New("missing FluxMQ stream offset")
)

type subEventStore struct {
	client *fluxamqp.Client
	logger *slog.Logger
}

// NewSubscriber creates a FluxMQ-backed event subscriber.
func NewSubscriber(_ context.Context, url string, logger *slog.Logger) (events.Subscriber, error) {
	opts := fluxamqp.NewOptions().SetURL(url)

	client, err := fluxamqp.New(opts)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	if err := declareEventsStream(client); err != nil {
		return nil, err
	}

	return &subEventStore{
		client: client,
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

	autoCommit := false
	opts := &fluxamqp.StreamConsumeOptions{
		QueueName:     eventsQueue,
		Filter:        streamFilter(cfg.Stream),
		ConsumerGroup: cfg.Consumer,
		AutoCommit:    &autoCommit,
	}

	if cfg.DeliveryPolicy == messaging.DeliverNewPolicy {
		opts.Offset = "last"
	}

	return es.client.SubscribeToStream(opts, func(msg *fluxamqp.QueueMessage) {
		if err := es.handle(ctx, cfg.Consumer, cfg.Handler, msg); err != nil {
			es.logWarn("failed to process FluxMQ event", "error", err)
		}
	})
}

func (es *subEventStore) Close() error {
	return es.client.Close()
}

func (es *subEventStore) handle(ctx context.Context, consumer string, handler events.EventHandler, msg *fluxamqp.QueueMessage) error {
	event := event{
		Data: make(map[string]any),
	}

	if err := json.Unmarshal(msg.Body, &event.Data); err != nil {
		if rejectErr := msg.Reject(); rejectErr != nil {
			return errors.Join(err, rejectErr)
		}
		return err
	}

	offset, ok := msg.StreamOffset()
	if !ok {
		if rejectErr := msg.Reject(); rejectErr != nil {
			return errors.Join(ErrMissingStreamOffset, rejectErr)
		}
		return ErrMissingStreamOffset
	}
	if offset == math.MaxUint64 {
		err := fmt.Errorf("invalid FluxMQ stream offset %d", offset)
		if rejectErr := msg.Reject(); rejectErr != nil {
			return errors.Join(err, rejectErr)
		}
		return err
	}

	if err := handler.Handle(ctx, event); err != nil {
		if nackErr := msg.Nack(); nackErr != nil {
			return errors.Join(fmt.Errorf("failed to handle FluxMQ event: %w", err), nackErr)
		}
		return fmt.Errorf("failed to handle FluxMQ event: %w", err)
	}

	if err := es.client.CommitOffset(eventsQueue, consumer, offset+1); err != nil {
		return err
	}

	if err := msg.Ack(); err != nil {
		return err
	}

	return nil
}

func (es *subEventStore) logWarn(msg string, args ...any) {
	if es.logger != nil {
		es.logger.Warn(msg, args...)
		return
	}

	slog.Warn(msg, args...)
}

type event struct {
	Data map[string]any
}

func (re event) Encode() (map[string]any, error) {
	return re.Data, nil
}
