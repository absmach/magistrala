// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"log/slog"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
)

const stream = "events.magistrala.*.*"

type eventHandler struct {
	svc bootstrap.Service
}

// BootstrapEventsSubscribe subscribes bootstrap config-state handlers to the event store.
func BootstrapEventsSubscribe(ctx context.Context, svc bootstrap.Service, esURL, esConsumerName string, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, esURL, "bootstrap-es-sub", logger)
	if err != nil {
		return err
	}

	subConfig := events.SubscriberConfig{
		Stream:   stream,
		Consumer: esConsumerName,
		Handler:  NewEventHandler(svc),
		Ordered:  true,
	}
	return subscriber.Subscribe(ctx, subConfig)
}

// NewEventHandler returns bootstrap events handler.
func NewEventHandler(svc bootstrap.Service) events.EventHandler {
	return &eventHandler{
		svc: svc,
	}
}

func (es *eventHandler) Handle(_ context.Context, _ events.Event) error {
	return nil
}
