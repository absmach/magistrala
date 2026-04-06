// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
)

var (
	ErrMissingOccurredAt = errors.New("missing occurred_at")
	errMissingOperation  = errors.New("missing operation")
	errMissingAttributes = errors.New("missing attributes")
	errMsg               = "failed to save journal"
)

// Start method starts consuming messages received from Event store.
func Start(ctx context.Context, consumer string, sub events.Subscriber, service journal.Service) error {
	subCfg := events.SubscriberConfig{
		Consumer: consumer,
		Stream:   store.StreamAllEvents,
		Handler:  Handle(service),
	}

	return sub.Subscribe(ctx, subCfg)
}

func Handle(service journal.Service) handleFunc {
	return func(ctx context.Context, event events.Event) error {
		data, err := event.Encode()
		if err != nil {
			return err
		}

		operation, ok := data["operation"].(string)
		if !ok {
			// Error is logged instead of being returned to avoid redelivering of the event.
			slog.Error(errMsg, "error", errMissingOperation)
			return nil
		}
		delete(data, "operation")

		if operation == "" {
			slog.Error(errMsg, "error", errMissingOperation)
			return nil
		}

		occurredAt, ok := data["occurred_at"].(float64)
		if !ok {
			slog.Error(errMsg, "error", ErrMissingOccurredAt)
			return nil
		}
		delete(data, "occurred_at")

		if occurredAt == 0 {
			slog.Error(errMsg, "error", ErrMissingOccurredAt)
			return nil
		}

		metadata, ok := data["metadata"].(map[string]any)
		if !ok {
			metadata = make(map[string]any)
		}
		delete(data, "metadata")

		if len(data) == 0 {
			slog.Error(errMsg, "error", errMissingAttributes)
			return nil
		}

		j := journal.Journal{
			Operation:  operation,
			OccurredAt: time.Unix(0, int64(occurredAt)),
			Attributes: data,
			Metadata:   metadata,
		}
		if err := service.Save(ctx, j); err != nil {
			slog.Error(errMsg, "error", err)
		}

		return nil
	}
}

type handleFunc func(ctx context.Context, event events.Event) error

func (h handleFunc) Handle(ctx context.Context, event events.Event) error {
	return h(ctx, event)
}

func (h handleFunc) Cancel() error {
	return nil
}
