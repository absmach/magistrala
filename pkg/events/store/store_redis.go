// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !es_nats && !es_rabbitmq && !es_fluxmq
// +build !es_nats,!es_rabbitmq,!es_fluxmq

package store

import (
	"context"
	"log"
	"log/slog"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/redis"
)

// StreamAllEvents represents subject to subscribe for all the events.
const StreamAllEvents = ">"

func init() {
	log.Println("The binary was build using redis as the events store")
}

func NewPublisher(ctx context.Context, url, _ string) (events.Publisher, error) {
	pb, err := redis.NewPublisher(ctx, url, events.UnpublishedEventsCheckInterval)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewSubscriber(_ context.Context, url, _ string, logger *slog.Logger) (events.Subscriber, error) {
	pb, err := redis.NewSubscriber(url, logger)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
