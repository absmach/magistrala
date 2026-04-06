// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build es_nats
// +build es_nats

package store

import (
	"context"
	"log"
	"log/slog"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/nats"
)

// StreamAllEvents represents subject to subscribe for all the events.
const StreamAllEvents = "events/#"

func init() {
	log.Println("The binary was build using Nats as the events store")
}

func NewPublisher(ctx context.Context, url, _ string) (events.Publisher, error) {
	pb, err := nats.NewPublisher(ctx, url)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewSubscriber(ctx context.Context, url, _ string, logger *slog.Logger) (events.Subscriber, error) {
	pb, err := nats.NewSubscriber(ctx, url, logger)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
