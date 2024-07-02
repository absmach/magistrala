// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/redis/go-redis/v9"
)

const (
	eventsPrefix = "events."
	eventCount   = 100
	exists       = "BUSYGROUP Consumer Group name already exists"
	group        = "magistrala"
)

var _ events.Subscriber = (*subEventStore)(nil)

var (
	// ErrEmptyStream is returned when stream name is empty.
	ErrEmptyStream = errors.New("stream name cannot be empty")

	// ErrEmptyConsumer is returned when consumer name is empty.
	ErrEmptyConsumer = errors.New("consumer name cannot be empty")
)

type subEventStore struct {
	client *redis.Client
	logger *slog.Logger
}

func NewSubscriber(url string, logger *slog.Logger) (events.Subscriber, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return &subEventStore{
		client: redis.NewClient(opts),
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

	err := es.client.XGroupCreateMkStream(ctx, cfg.Stream, group, "$").Err()
	if err != nil && err.Error() != exists {
		return err
	}

	go func() {
		for {
			msgs, err := es.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: cfg.Consumer,
				Streams:  []string{cfg.Stream, ">"},
				Count:    eventCount,
			}).Result()
			if err != nil {
				es.logger.Warn(fmt.Sprintf("failed to read from redis stream: %s", err))

				continue
			}
			if len(msgs) == 0 {
				continue
			}

			es.handle(ctx, cfg.Stream, msgs[0].Messages, cfg.Handler)
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

func (es *subEventStore) handle(ctx context.Context, stream string, msgs []redis.XMessage, h events.EventHandler) {
	for _, msg := range msgs {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Values["data"].(string)), &data); err != nil {
			es.logger.Warn(fmt.Sprintf("failed to unmarshal redis event: %s", err))

			return
		}

		event := redisEvent{
			Data: data,
		}

		if err := h.Handle(ctx, event); err != nil {
			es.logger.Warn(fmt.Sprintf("failed to handle redis event: %s", err))

			return
		}

		if err := es.client.XAck(ctx, stream, group, msg.ID).Err(); err != nil {
			es.logger.Warn(fmt.Sprintf("failed to ack redis event: %s", err))

			return
		}
	}
}
