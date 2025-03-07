// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"time"

	"github.com/absmach/supermq/journal"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
)

var ErrMissingOccurredAt = errors.New("missing occurred_at")

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
			return errors.New("missing operation")
		}
		delete(data, "operation")

		if operation == "" {
			return errors.New("missing operation")
		}

		occurredAt, ok := data["occurred_at"].(float64)
		if !ok {
			return ErrMissingOccurredAt
		}
		delete(data, "occurred_at")

		if occurredAt == 0 {
			return ErrMissingOccurredAt
		}

		metadata, ok := data["metadata"].(map[string]interface{})
		if !ok {
			metadata = make(map[string]interface{})
		}
		delete(data, "metadata")

		if len(data) == 0 {
			return errors.New("missing attributes")
		}

		j := journal.Journal{
			Operation:  operation,
			OccurredAt: time.Unix(0, int64(occurredAt)),
			Attributes: data,
			Metadata:   metadata,
		}

		return service.Save(ctx, j)
	}
}

type handleFunc func(ctx context.Context, event events.Event) error

func (h handleFunc) Handle(ctx context.Context, event events.Event) error {
	return h(ctx, event)
}

func (h handleFunc) Cancel() error {
	return nil
}
