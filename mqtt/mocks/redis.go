// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/mqtt/events"
)

type MockEventStore struct{}

func NewEventStore() events.EventStore {
	return MockEventStore{}
}

func (es MockEventStore) Connect(ctx context.Context, clientID string) error {
	return nil
}

func (es MockEventStore) Disconnect(ctx context.Context, clientID string) error {
	return nil
}
