// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build es_fluxmq
// +build es_fluxmq

package store

import (
	"context"
	"log"
	"log/slog"

	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/fluxmq"
)

// StreamAllEvents represents subject to subscribe for all the events.
const StreamAllEvents = ">"

func init() {
	log.Println("The binary was built using FluxMQ as the events store")
}

func NewPublisher(ctx context.Context, url, connectionName string) (events.Publisher, error) {
	pb, err := fluxmq.NewPublisher(ctx, url, connectionName)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewSubscriber(ctx context.Context, url, connectionName string, logger *slog.Logger) (events.Subscriber, error) {
	pb, err := fluxmq.NewSubscriber(ctx, url, connectionName, logger)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
