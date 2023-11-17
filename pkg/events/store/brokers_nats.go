// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build nats
// +build nats

package store

import (
	"context"
	"log"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/nats"
)

func init() {
	log.Println("The binary was build using nats as the events store")
}

func NewPublisher(ctx context.Context, url, stream string) (events.Publisher, error) {
	pb, err := nats.NewPublisher(ctx, url, stream)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewSubscriber(ctx context.Context, url, stream, consumer string, logger mglog.Logger) (events.Subscriber, error) {
	pb, err := nats.NewSubscriber(ctx, url, stream, consumer, logger)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
