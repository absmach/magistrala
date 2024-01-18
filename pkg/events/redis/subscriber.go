// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !nats && !rabbitmq
// +build !nats,!rabbitmq

package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/go-redis/redis/v8"
)

const (
	eventCount = 100
	exists     = "BUSYGROUP Consumer Group name already exists"
	group      = "magistrala"
)

var _ events.Subscriber = (*subEventStore)(nil)

var (
	// ErrEmptyStream is returned when stream name is empty.
	ErrEmptyStream = errors.New("stream name cannot be empty")

	// ErrEmptyConsumer is returned when consumer name is empty.
	ErrEmptyConsumer = errors.New("consumer name cannot be empty")
)

type subEventStore struct {
	client   *redis.Client
	stream   string
	consumer string
	logger   *slog.Logger
}

func NewSubscriber(url, stream, consumer string, logger *slog.Logger) (events.Subscriber, error) {
	if stream == "" {
		return nil, ErrEmptyStream
	}

	if consumer == "" {
		return nil, ErrEmptyConsumer
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return &subEventStore{
		client:   redis.NewClient(opts),
		stream:   stream,
		consumer: consumer,
		logger:   logger,
	}, nil
}

func (es *subEventStore) Subscribe(ctx context.Context, handler events.EventHandler) error {
	err := es.client.XGroupCreateMkStream(ctx, es.stream, group, "$").Err()
	if err != nil && err.Error() != exists {
		return err
	}

	go func() {
		for {
			msgs, err := es.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: es.consumer,
				Streams:  []string{es.stream, ">"},
				Count:    eventCount,
			}).Result()
			if err != nil {
				es.logger.Warn(fmt.Sprintf("failed to read from Redis stream: %s", err))

				continue
			}
			if len(msgs) == 0 {
				continue
			}

			es.handle(ctx, msgs[0].Messages, handler)
		}
	}()

	return nil
}

func (es *subEventStore) Close() error {
	return es.client.Close()
}

type redisEvent struct {
	Data map[string]interface{}
}

func (re redisEvent) Encode() (map[string]interface{}, error) {
	return re.Data, nil
}

func (es *subEventStore) handle(ctx context.Context, msgs []redis.XMessage, h events.EventHandler) {
	for _, msg := range msgs {
		event := redisEvent{
			Data: msg.Values,
		}

		if err := h.Handle(ctx, event); err != nil {
			es.logger.Warn(fmt.Sprintf("failed to handle redis event: %s", err))

			return
		}

		if err := es.client.XAck(ctx, es.stream, group, msg.ID).Err(); err != nil {
			es.logger.Warn(fmt.Sprintf("failed to ack redis event: %s", err))

			return
		}
	}
}
