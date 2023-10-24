// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build rabbitmq
// +build rabbitmq

package store

import (
	"context"
	"log"

	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/events"
	"github.com/mainflux/mainflux/pkg/events/rabbitmq"
)

func init() {
	log.Println("The binary was build using rabbitmq as the events store")
}

func NewPublisher(ctx context.Context, url, stream string) (events.Publisher, error) {
	pb, err := rabbitmq.NewPublisher(ctx, url, stream)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewSubscriber(_ context.Context, url, stream, consumer string, logger mflog.Logger) (events.Subscriber, error) {
	pb, err := rabbitmq.NewSubscriber(url, stream, consumer, logger)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
