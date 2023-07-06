package mocks

import (
	"context"

	"github.com/mainflux/mainflux/mqtt/redis"
)

type MockEventStore struct{}

func NewEventStore() redis.EventStore {
	return MockEventStore{}
}

func (es MockEventStore) Connect(ctx context.Context, clientID string) error {
	return nil
}

func (es MockEventStore) Disconnect(ctx context.Context, clientID string) error {
	return nil
}
